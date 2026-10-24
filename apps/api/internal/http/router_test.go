package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type readinessFunc func(context.Context) error

func (f readinessFunc) Ready(ctx context.Context) error { return f(ctx) }
func TestHealthAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name, path, method string
		db                 readinessFunc
		status             int
	}{
		{"live without database", "/api/v1/health/live", "GET", func(context.Context) error { return errors.New("private") }, 200},
		{"ready", "/api/v1/health/ready", "GET", func(context.Context) error { return nil }, 200},
		{"unavailable", "/api/v1/health/ready", "GET", func(context.Context) error { return errors.New("private") }, 503},
		{"panic", "/api/v1/health/ready", "GET", func(context.Context) error { panic("private") }, 500},
		{"unknown path", "/private-secret?token=private", "GET", func(context.Context) error { return nil }, 404},
		{"method", "/api/v1/health/live", "POST", func(context.Context) error { return nil }, 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			h := New(slog.New(slog.NewJSONHandler(&logs, nil)), tc.db)
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Authorization", "Bearer private")
			req.Header.Set("X-Request-ID", "private")
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
			id := res.Header().Get("X-Request-ID")
			if len(id) != 32 {
				t.Fatalf("request ID inválido: %s", id)
			}
			if strings.Contains(res.Body.String(), "private") || strings.Contains(logs.String(), "private") {
				t.Fatal("segredo exposto")
			}
			var data map[string]any
			if err := json.Unmarshal(res.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
			if tc.status >= 400 {
				if res.Header().Get("Content-Type") != "application/problem+json" {
					t.Fatal("content-type incorreto")
				}
				if data["requestId"] != id {
					t.Fatal("request ID inconsistente")
				}
			}
			if res.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("health/error não pode ser cacheado")
			}
		})
	}
}
func TestRequestIDIsUnique(t *testing.T) {
	h := New(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)), readinessFunc(func(context.Context) error { return nil }))
	ids := map[string]bool{}
	for range 2 {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil))
		id := r.Header().Get("X-Request-ID")
		if ids[id] {
			t.Fatal("ID duplicado")
		}
		ids[id] = true
	}
}
