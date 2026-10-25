package httpapi

import (
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/audit"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func WithAudit(session *auth.Service, service *audit.Service, origins []string, secure bool) Option {
	return func(router chi.Router) {
		a := &authHandlers{service: session, origins: make(map[string]bool), secure: secure}
		for _, origin := range origins {
			a.origins[origin] = true
		}
		access := &accessHandlers{session: a}
		router.Group(func(r chi.Router) {
			r.Use(access.requireSession)
			r.Get("/api/v1/organizations/{orgId}/audit-events", func(w http.ResponseWriter, r *http.Request) {
				page, err := pageRequest(r)
				if err != nil {
					auditError(w, r, err)
					return
				}
				if len(r.URL.Query()["action"]) > 1 || len(r.URL.Query()["resourceType"]) > 1 {
					auditError(w, r, audit.ErrValidation)
					return
				}
				out, err := service.List(r.Context(), actor(r), chi.URLParam(r, "orgId"), page, audit.Filter{Action: r.URL.Query().Get("action"), ResourceType: r.URL.Query().Get("resourceType")})
				if err != nil {
					auditError(w, r, err)
					return
				}
				jsonResponse(w, 200, out)
			})
		})
	}
}

func auditError(w http.ResponseWriter, r *http.Request, err error) {
	status := 503
	switch {
	case errors.Is(err, audit.ErrValidation):
		status = 422
	case errors.Is(err, audit.ErrNotFound):
		status = 404
	case errors.Is(err, audit.ErrForbidden):
		status = 403
	}
	problem(w, r, status, err.Error())
}
