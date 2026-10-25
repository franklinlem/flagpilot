package httpapi

import (
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/tokens"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

type tokenHandlers struct{ service *tokens.Service }

func WithTokens(session *auth.Service, service *tokens.Service, origins []string, secure bool) Option {
	return func(router chi.Router) {
		a := &authHandlers{service: session, origins: map[string]bool{}, secure: secure}
		for _, origin := range origins {
			a.origins[origin] = true
		}
		access := &accessHandlers{session: a}
		h := &tokenHandlers{service: service}
		router.Group(func(r chi.Router) {
			r.Use(access.requireSession)
			r.Get("/api/v1/environments/{envId}/tokens", h.list)
			r.Group(func(w chi.Router) {
				w.Use(access.requireMutation)
				w.Post("/api/v1/environments/{envId}/tokens", h.issue)
				w.Delete("/api/v1/environments/{envId}/tokens/{tokenId}", h.revoke)
			})
		})
	}
}
func tokenError(w http.ResponseWriter, r *http.Request, err error) {
	status := 503
	switch {
	case errors.Is(err, tokens.ErrNotFound):
		status = 404
	case errors.Is(err, tokens.ErrForbidden):
		status = 403
	case errors.Is(err, tokens.ErrValidation):
		status = 422
	}
	problem(w, r, status, err.Error())
}
func (h *tokenHandlers) list(w http.ResponseWriter, r *http.Request) {
	page, err := pageRequest(r)
	if err != nil {
		tokenError(w, r, err)
		return
	}
	out, err := h.service.List(r.Context(), actor(r), chi.URLParam(r, "envId"), page)
	if err != nil {
		tokenError(w, r, err)
		return
	}
	jsonResponse(w, 200, out)
}
func (h *tokenHandlers) issue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string    `json:"name"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	out, err := h.service.Issue(r.Context(), actor(r), chi.URLParam(r, "envId"), body.Name, body.ExpiresAt)
	if err != nil {
		tokenError(w, r, err)
		return
	}
	jsonResponse(w, 201, out)
}
func (h *tokenHandlers) revoke(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Revoke(r.Context(), actor(r), chi.URLParam(r, "envId"), chi.URLParam(r, "tokenId")); err != nil {
		tokenError(w, r, err)
		return
	}
	w.WriteHeader(204)
}
