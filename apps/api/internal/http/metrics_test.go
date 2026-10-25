package httpapi

import (
	"context"
	"github.com/franklinlem/flagpilot/apps/api/internal/observability"
	"github.com/go-chi/chi/v5"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsRoutesPrivacyAndBoundedMethods(t *testing.T) {
	metrics := observability.New(nil)
	h := NewObserved(slog.New(slog.NewJSONHandler(io.Discard, nil)), readinessFunc(func(context.Context) error { return nil }), metrics, func(r chi.Router) {
		r.Get("/api/v1/projects/{projectId}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	})
	for _, path := range []string{"/api/v1/projects/private-id-one?token=secret", "/api/v1/projects/private-id-two", "/unknown-secret-one", "/unknown-secret-two"} {
		request := httptest.NewRequest("GET", path, strings.NewReader("private-context"))
		request.Header.Set("Authorization", "Bearer secret")
		h.ServeHTTP(httptest.NewRecorder(), request)
	}
	for _, method := range []string{"CUSTOM_ONE", "CUSTOM_TWO"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, "/missing", nil))
	}
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/v1/health/ready", nil))
	public := httptest.NewRecorder()
	h.ServeHTTP(public, httptest.NewRequest("GET", "/metrics", nil))
	if public.Code != 404 {
		t.Fatal("métricas no router público")
	}
	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	body := recorder.Body.String()
	if recorder.Code != 200 || !strings.Contains(body, `route="/api/v1/projects/{projectId}"`) || !strings.Contains(body, `method="OTHER"`) || !strings.Contains(body, "flagpilot_http_request_duration_seconds_bucket") {
		t.Fatal("métricas ausentes")
	}
	for _, value := range []string{"private-id", "private-context", "secret", "CUSTOM_ONE", "CUSTOM_TWO", "unknown-secret"} {
		if strings.Contains(body, value) {
			t.Fatal("métrica expôs dado de request")
		}
	}
	other := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(other, httptest.NewRequest("GET", "/api/v1/health/ready", nil))
	if other.Code != 404 {
		t.Fatal("listener interno expôs API")
	}
	// Um registry pertence à instância, sem conflito global ou estado compartilhado.
	fresh := httptest.NewRecorder()
	observability.New(nil).Handler().ServeHTTP(fresh, httptest.NewRequest("GET", "/metrics", nil))
	if strings.Contains(fresh.Body.String(), "route=") {
		t.Fatal("instâncias compartilham contadores")
	}
}
