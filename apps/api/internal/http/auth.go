package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/go-chi/chi/v5"
)

const sessionCookie = "flagpilot_session"

type Option func(chi.Router)
type authHandlers struct {
	service *auth.Service
	origins map[string]bool
	secure  bool
	limiter *auth.Limiter
}

func WithAuth(service *auth.Service, origins []string, secure bool) Option {
	return func(r chi.Router) {
		h := &authHandlers{service: service, origins: make(map[string]bool), secure: secure, limiter: auth.NewLimiter()}
		for _, origin := range origins {
			h.origins[origin] = true
		}
		r.Post("/api/v1/auth/login", h.login)
		r.Post("/api/v1/auth/logout", h.logout)
		r.Get("/api/v1/me", h.me)
	}
}
func (h *authHandlers) validOrigin(r *http.Request) bool {
	return len(r.Header.Values("Origin")) == 1 && h.origins[r.Header.Get("Origin")] && r.Header.Get("Sec-Fetch-Site") != "cross-site"
}
func cookieToken(r *http.Request) string {
	token := ""
	count := 0
	for _, cookie := range r.Cookies() {
		if cookie.Name == sessionCookie {
			count++
			token = cookie.Value
		}
	}
	if count != 1 {
		return ""
	}
	return token
}
func (h *authHandlers) setCookie(w http.ResponseWriter, token string, expires time.Time) {
	age := -1
	if token != "" {
		age = int(time.Until(expires).Seconds())
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: age})
}
func (h *authHandlers) authError(w http.ResponseWriter, r *http.Request, err error) {
	status := 503
	title := "Serviço de autenticação indisponível"
	switch {
	case errors.Is(err, auth.ErrCredentials):
		status = 401
		title = "Email ou senha inválidos"
	case errors.Is(err, auth.ErrSession):
		status = 401
		title = "Sessão inválida ou expirada"
		h.setCookie(w, "", time.Unix(1, 0))
	case errors.Is(err, auth.ErrValidation):
		status = 422
		title = "Informe um email válido e uma senha de até 256 bytes"
	case errors.Is(err, auth.ErrBusy):
		status = 429
		title = "Muitas tentativas. Aguarde e tente novamente"
		w.Header().Set("Retry-After", "60")
	}
	problem(w, r, status, title)
}
func (h *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(r) {
		problem(w, r, 403, "Origem não permitida")
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		problem(w, r, 415, "Envie application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&body); err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			problem(w, r, 413, "Corpo excede o limite permitido")
		} else {
			problem(w, r, 400, "JSON inválido")
		}
		return
	}
	if err = decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		problem(w, r, 400, "Envie apenas um objeto JSON")
		return
	}
	email, ok := auth.NormalizeEmail(body.Email)
	if !ok || body.Password == "" || len(body.Password) > 256 {
		h.authError(w, r, auth.ErrValidation)
		return
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	// Não confiar em Forwarded/X-Forwarded-For recebidos. Proxy confiável será configurado no destino.
	if !h.limiter.Allow(ip, strings.ToLower(email)) {
		h.authError(w, r, auth.ErrBusy)
		return
	}
	session, err := h.service.Login(r.Context(), email, body.Password, cookieToken(r))
	if err != nil {
		h.authError(w, r, err)
		return
	}
	h.setCookie(w, session.Token, session.ExpiresAt)
	jsonResponse(w, 200, session.Identity)
}
func (h *authHandlers) me(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		problem(w, r, 403, "Origem não permitida")
		return
	}
	identity, err := h.service.Find(r.Context(), cookieToken(r))
	if err != nil {
		h.authError(w, r, err)
		return
	}
	jsonResponse(w, 200, identity)
}
func (h *authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(r) {
		problem(w, r, 403, "Origem não permitida")
		return
	}
	token := cookieToken(r)
	if _, err := h.service.Find(r.Context(), token); err != nil {
		h.authError(w, r, err)
		return
	}
	if !auth.ValidCSRF(token, r.Header.Get("X-CSRF-Token")) {
		problem(w, r, 403, "Token CSRF inválido")
		return
	}
	if err := h.service.Logout(r.Context(), token); err != nil {
		h.authError(w, r, err)
		return
	}
	h.setCookie(w, "", time.Unix(1, 0))
	w.WriteHeader(http.StatusNoContent)
}
