package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/organizations"
	"github.com/go-chi/chi/v5"
	"io"
	"mime"
	"net/http"
	"strconv"
)

type principalKey struct{}
type organizationHandlers struct {
	*accessHandlers
	service *organizations.Service
}

func WithOrganizations(session *auth.Service, service *organizations.Service, origins []string, secure bool) Option {
	return func(router chi.Router) {
		a := &authHandlers{service: session, origins: make(map[string]bool), secure: secure}
		for _, origin := range origins {
			a.origins[origin] = true
		}
		h := &organizationHandlers{accessHandlers: &accessHandlers{session: a}, service: service}
		router.Group(func(r chi.Router) {
			r.Use(h.requireSession)
			r.Get("/api/v1/organizations", h.list)
			r.Get("/api/v1/organizations/{orgId}", h.get)
			r.Get("/api/v1/organizations/{orgId}/members", h.members)
			r.Get("/api/v1/organizations/{orgId}/member-candidates", h.candidates)
			r.Group(func(w chi.Router) {
				w.Use(h.requireMutation)
				w.Post("/api/v1/organizations", h.create)
				w.Post("/api/v1/organizations/{orgId}/members", h.add)
				w.Patch("/api/v1/organizations/{orgId}/members/{userId}", h.change)
				w.Delete("/api/v1/organizations/{orgId}/members/{userId}", h.remove)
			})
		})
	}
}
func actor(r *http.Request) string { return r.Context().Value(principalKey{}).(auth.Identity).User.ID }
func organizationError(w http.ResponseWriter, r *http.Request, err error) {
	status := 503
	switch {
	case errors.Is(err, organizations.ErrNotFound):
		status = 404
	case errors.Is(err, organizations.ErrForbidden):
		status = 403
	case errors.Is(err, organizations.ErrValidation):
		status = 422
	case errors.Is(err, organizations.ErrConflict), errors.Is(err, organizations.ErrLastOwner):
		status = 409
	}
	problem(w, r, status, err.Error())
}
func pageRequest(r *http.Request) (organizations.PageRequest, error) {
	p := organizations.PageRequest{Limit: 25, Cursor: r.URL.Query().Get("cursor")}
	if len(r.URL.Query()["limit"]) > 1 || len(r.URL.Query()["cursor"]) > 1 {
		return p, organizations.ErrValidation
	}
	if value, ok := r.URL.Query()["limit"]; ok {
		n, err := strconv.Atoi(value[0])
		if err != nil || n < 1 || n > 100 {
			return p, organizations.ErrValidation
		}
		p.Limit = n
	}
	if len(p.Cursor) > 512 {
		return p, organizations.ErrValidation
	}
	return p, nil
}
func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		problem(w, r, 415, "Envie application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	fail := func(err error) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			problem(w, r, 413, "Corpo excede o limite permitido")
		} else {
			problem(w, r, 400, "Envie apenas um objeto JSON válido")
		}
	}
	if err := decoder.Decode(destination); err != nil {
		fail(err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		fail(err)
		return false
	}
	return true
}
func (h *organizationHandlers) list(w http.ResponseWriter, r *http.Request) {
	page, err := pageRequest(r)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	result, err := h.service.List(r.Context(), actor(r), page)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	jsonResponse(w, 200, result)
}
func (h *organizationHandlers) members(w http.ResponseWriter, r *http.Request) {
	page, err := pageRequest(r)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	result, err := h.service.Members(r.Context(), actor(r), chi.URLParam(r, "orgId"), page)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	jsonResponse(w, 200, result)
}
func (h *organizationHandlers) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	result, err := h.service.Create(r.Context(), actor(r), body.Name, body.Slug)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	jsonResponse(w, 201, result)
}
func (h *organizationHandlers) add(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	result, err := h.service.Add(r.Context(), actor(r), chi.URLParam(r, "orgId"), body.UserID, body.Role)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	jsonResponse(w, 201, result)
}
func (h *organizationHandlers) change(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	result, err := h.service.Change(r.Context(), actor(r), chi.URLParam(r, "orgId"), chi.URLParam(r, "userId"), body.Role)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	jsonResponse(w, 200, result)
}
func (h *organizationHandlers) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Remove(r.Context(), actor(r), chi.URLParam(r, "orgId"), chi.URLParam(r, "userId")); err != nil {
		organizationError(w, r, err)
		return
	}
	w.WriteHeader(204)
}

func (h *organizationHandlers) get(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Get(r.Context(), actor(r), chi.URLParam(r, "orgId"))
	if err != nil {
		organizationError(w, r, err)
		return
	}
	jsonResponse(w, 200, result)
}

func (h *organizationHandlers) candidates(w http.ResponseWriter, r *http.Request) {
	page, err := pageRequest(r)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	result, err := h.service.Candidates(r.Context(), actor(r), chi.URLParam(r, "orgId"), page)
	if err != nil {
		organizationError(w, r, err)
		return
	}
	jsonResponse(w, 200, result)
}
