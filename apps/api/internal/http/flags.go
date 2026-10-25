package httpapi

import (
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/flags"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type flagHandlers struct{ service *flags.Service }

func WithFlags(session *auth.Service, service *flags.Service, origins []string, secure bool) Option {
	return func(router chi.Router) {
		a := &authHandlers{service: session, origins: make(map[string]bool), secure: secure}
		for _, origin := range origins {
			a.origins[origin] = true
		}
		access := &accessHandlers{session: a}
		h := &flagHandlers{service: service}
		router.Group(func(r chi.Router) {
			r.Use(access.requireSession)
			r.Get("/api/v1/projects/{projectId}/flags", h.list)
			r.Get("/api/v1/projects/{projectId}/flags/{flagKey}", h.get)
			r.Get("/api/v1/environments/{envId}/flags", h.configs)
			r.Get("/api/v1/environments/{envId}/flags/{flagKey}", h.getConfig)
			r.Group(func(w chi.Router) {
				w.Use(access.requireMutation)
				w.Post("/api/v1/projects/{projectId}/flags", h.create)
				w.Patch("/api/v1/projects/{projectId}/flags/{flagKey}", h.change)
				w.Put("/api/v1/environments/{envId}/flags/{flagKey}", h.putConfig)
			})
		})
	}
}
func flagError(w http.ResponseWriter, r *http.Request, err error) {
	status := 503
	switch {
	case errors.Is(err, flags.ErrNotFound):
		status = 404
	case errors.Is(err, flags.ErrForbidden):
		status = 403
	case errors.Is(err, flags.ErrValidation), errors.Is(err, flags.ErrProduction):
		status = 422
	case errors.Is(err, flags.ErrConflict):
		status = 409
	case errors.Is(err, flags.ErrRequired):
		status = 428
	case errors.Is(err, flags.ErrPrecondition):
		status = 412
	}
	problem(w, r, status, err.Error())
}
func flagResponse(w http.ResponseWriter, status int, out flags.Flag) {
	w.Header().Set("ETag", flags.FlagETag(out))
	jsonResponse(w, status, out)
}
func (h *flagHandlers) list(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()["q"]) > 1 || len(r.URL.Query()["status"]) > 1 {
		flagError(w, r, flags.ErrValidation)
		return
	}
	page, err := pageRequest(r)
	if err != nil {
		flagError(w, r, err)
		return
	}
	out, err := h.service.List(r.Context(), actor(r), chi.URLParam(r, "projectId"), page, flags.Filter{Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status")})
	if err != nil {
		flagError(w, r, err)
		return
	}
	jsonResponse(w, 200, out)
}
func (h *flagHandlers) get(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Get(r.Context(), actor(r), chi.URLParam(r, "projectId"), chi.URLParam(r, "flagKey"))
	if err != nil {
		flagError(w, r, err)
		return
	}
	flagResponse(w, 200, out)
}
func (h *flagHandlers) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key         string `json:"key"`
		Description string `json:"description"`
	}
	if !decodeJSONLimit(w, r, &body, 32*1024) {
		return
	}
	out, err := h.service.Create(r.Context(), actor(r), chi.URLParam(r, "projectId"), body.Key, body.Description)
	if err != nil {
		flagError(w, r, err)
		return
	}
	flagResponse(w, 201, out)
}
func (h *flagHandlers) change(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Description       *string `json:"description"`
		Archived          *bool   `json:"archived"`
		ConfirmProduction bool    `json:"confirmProduction"`
	}
	if !decodeJSONLimit(w, r, &body, 32*1024) {
		return
	}
	out, err := h.service.Change(r.Context(), actor(r), chi.URLParam(r, "projectId"), chi.URLParam(r, "flagKey"), r.Header.Get("If-Match"), body.Description, body.Archived, body.ConfirmProduction)
	if err != nil {
		flagError(w, r, err)
		return
	}
	flagResponse(w, 200, out)
}
func (h *flagHandlers) configs(w http.ResponseWriter, r *http.Request) {
	page, err := pageRequest(r)
	if err != nil {
		flagError(w, r, err)
		return
	}
	out, etag, err := h.service.Configs(r.Context(), actor(r), chi.URLParam(r, "envId"), page)
	if err != nil {
		flagError(w, r, err)
		return
	}
	w.Header().Set("ETag", etag)
	jsonResponse(w, 200, out)
}
func (h *flagHandlers) getConfig(w http.ResponseWriter, r *http.Request) {
	out, etag, err := h.service.GetConfig(r.Context(), actor(r), chi.URLParam(r, "envId"), chi.URLParam(r, "flagKey"))
	if err != nil {
		flagError(w, r, err)
		return
	}
	w.Header().Set("ETag", etag)
	jsonResponse(w, 200, out)
}
func (h *flagHandlers) putConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled           *bool  `json:"enabled"`
		RolloutBps        *int32 `json:"rolloutBps"`
		ConfirmProduction bool   `json:"confirmProduction"`
	}
	if !decodeJSONLimit(w, r, &body, 32*1024) {
		return
	}
	if body.Enabled == nil || body.RolloutBps == nil {
		flagError(w, r, flags.ErrValidation)
		return
	}
	out, etag, err := h.service.PutConfig(r.Context(), actor(r), chi.URLParam(r, "envId"), chi.URLParam(r, "flagKey"), r.Header.Get("If-Match"), *body.Enabled, *body.RolloutBps, body.ConfirmProduction)
	if err != nil {
		flagError(w, r, err)
		return
	}
	w.Header().Set("ETag", etag)
	jsonResponse(w, 200, out)
}
