package httpapi

import (
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/evaluation"
	"github.com/franklinlem/flagpilot/apps/api/internal/tokens"
	"github.com/go-chi/chi/v5"
	"net"
	"net/http"
	"strings"
)

type evaluationHandlers struct {
	service *evaluation.Service
	limiter *evaluation.Limiter
}

func WithEvaluation(service *evaluation.Service) Option {
	return func(router chi.Router) {
		h := &evaluationHandlers{service: service, limiter: evaluation.NewLimiter()}
		router.Group(func(r chi.Router) {
			r.Use(h.requireBearer)
			r.Post("/api/v1/evaluate", h.single)
			r.Post("/api/v1/evaluate/batch", h.batch)
		})
	}
}
func (h *evaluationHandlers) requireBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Segredo de servidor: nenhum acesso direto de navegador ou CORS público.
		if r.Header.Get("Origin") != "" || r.Header.Get("Sec-Fetch-Site") != "" {
			problem(w, r, 403, "Avaliação disponível somente para clientes de servidor")
			return
		}
		values := r.Header.Values("Authorization")
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(values) != 1 || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || !tokens.ValidSecret(parts[1]) {
			evaluationError(w, r, evaluation.ErrUnauthorized)
			return
		}
		peer, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			peer = r.RemoteAddr
		}
		if !h.limiter.Allow(peer, parts[1]) {
			w.Header().Set("Retry-After", "1")
			problem(w, r, 429, "Limite de avaliação atingido; tente novamente em um segundo")
			return
		}
		// O serviço verifica hash/expiração/revogação na transação do snapshot.
		next.ServeHTTP(w, r)
	})
}
func evaluationError(w http.ResponseWriter, r *http.Request, err error) {
	status := 503
	switch {
	case errors.Is(err, evaluation.ErrUnauthorized):
		status = 401
		w.Header().Set("WWW-Authenticate", `Bearer realm="flagpilot-evaluation"`)
	case errors.Is(err, evaluation.ErrValidation):
		status = 422
	}
	problem(w, r, status, err.Error())
}
func bearer(r *http.Request) string { return strings.Fields(r.Header.Get("Authorization"))[1] }
func (h *evaluationHandlers) single(w http.ResponseWriter, r *http.Request) {
	var body evaluation.Request
	if !decodeJSONLimit(w, r, &body, 64*1024) {
		return
	}
	out, err := h.service.Evaluate(r.Context(), bearer(r), []evaluation.Request{body})
	if err != nil {
		evaluationError(w, r, err)
		return
	}
	jsonResponse(w, 200, out.Results[0])
}
func (h *evaluationHandlers) batch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Evaluations []evaluation.Request `json:"evaluations"`
	}
	if !decodeJSONLimit(w, r, &body, 64*1024) {
		return
	}
	out, err := h.service.Evaluate(r.Context(), bearer(r), body.Evaluations)
	if err != nil {
		evaluationError(w, r, err)
		return
	}
	jsonResponse(w, 200, out)
}
