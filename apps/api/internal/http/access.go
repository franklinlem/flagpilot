package httpapi

import (
	"context"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"net/http"
)

type accessHandlers struct{ session *authHandlers }

func (h *accessHandlers) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			problem(w, r, 403, "Origem não permitida")
			return
		}
		identity, err := h.session.service.Find(r.Context(), cookieToken(r))
		if err != nil {
			h.session.authError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, identity)))
	})
}
func (h *accessHandlers) requireMutation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.session.validOrigin(r) {
			problem(w, r, 403, "Origem não permitida")
			return
		}
		if !auth.ValidCSRF(cookieToken(r), r.Header.Get("X-CSRF-Token")) {
			problem(w, r, 403, "Token CSRF inválido")
			return
		}
		next.ServeHTTP(w, r)
	})
}
