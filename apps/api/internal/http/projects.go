package httpapi

import (
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type projectHandlers struct{ service *projects.Service }

func WithProjects(session *auth.Service, service *projects.Service, origins []string, secure bool) Option {
	return func(router chi.Router) {
		a := &authHandlers{service: session, origins: make(map[string]bool), secure: secure}
		for _, origin := range origins {
			a.origins[origin] = true
		}
		access := &accessHandlers{session: a}
		h := &projectHandlers{service: service}
		router.Group(func(r chi.Router) {
			r.Use(access.requireSession)
			r.Get("/api/v1/organizations/{orgId}/projects", h.list)
			r.Get("/api/v1/projects/{projectId}", h.get)
			r.Get("/api/v1/projects/{projectId}/environments", h.environments)
			r.Group(func(w chi.Router) {
				w.Use(access.requireMutation)
				w.Post("/api/v1/organizations/{orgId}/projects", h.create)
				w.Post("/api/v1/projects/{projectId}/environments", h.createEnvironment)
			})
		})
	}
}
func projectError(w http.ResponseWriter, r *http.Request, err error) {
	status := 503
	switch {
	case errors.Is(err, projects.ErrNotFound):
		status = 404
	case errors.Is(err, projects.ErrForbidden):
		status = 403
	case errors.Is(err, projects.ErrValidation):
		status = 422
	case errors.Is(err, projects.ErrConflict):
		status = 409
	}
	problem(w, r, status, err.Error())
}
func (h *projectHandlers) list(w http.ResponseWriter, r *http.Request) {
	page, err := pageRequest(r)
	if err != nil {
		projectError(w, r, err)
		return
	}
	out, err := h.service.List(r.Context(), actor(r), chi.URLParam(r, "orgId"), page)
	if err != nil {
		projectError(w, r, err)
		return
	}
	jsonResponse(w, 200, out)
}
func (h *projectHandlers) get(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Get(r.Context(), actor(r), chi.URLParam(r, "projectId"))
	if err != nil {
		projectError(w, r, err)
		return
	}
	jsonResponse(w, 200, out)
}
func (h *projectHandlers) environments(w http.ResponseWriter, r *http.Request) {
	page, err := pageRequest(r)
	if err != nil {
		projectError(w, r, err)
		return
	}
	out, err := h.service.Environments(r.Context(), actor(r), chi.URLParam(r, "projectId"), page)
	if err != nil {
		projectError(w, r, err)
		return
	}
	jsonResponse(w, 200, out)
}
func (h *projectHandlers) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	out, err := h.service.Create(r.Context(), actor(r), chi.URLParam(r, "orgId"), body.Name, body.Slug)
	if err != nil {
		projectError(w, r, err)
		return
	}
	jsonResponse(w, 201, out)
}
func (h *projectHandlers) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		IsProduction *bool  `json:"isProduction"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.IsProduction == nil {
		projectError(w, r, projects.ErrValidation)
		return
	}
	out, err := h.service.CreateEnvironment(r.Context(), actor(r), chi.URLParam(r, "projectId"), body.Name, body.Slug, *body.IsProduction)
	if err != nil {
		projectError(w, r, err)
		return
	}
	jsonResponse(w, 201, out)
}
