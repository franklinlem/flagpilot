package httpapi

import (
	"github.com/franklinlem/flagpilot/apps/api/internal/evaluation"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEvaluationRateAndBrowserBoundary(t *testing.T) {
	h := New(slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, WithEvaluation(evaluation.New(nil, time.Second)))
	call := func(origin bool) int {
		r := httptest.NewRequest("POST", "/api/v1/evaluate", strings.NewReader(`{"flagKey":"bad key"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer fp_eval_"+strings.Repeat("a", 64))
		if origin {
			r.Header.Set("Origin", "http://localhost")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == 429 && w.Header().Get("Retry-After") != "1" {
			t.Fatal("Retry-After ausente")
		}
		return w.Code
	}
	if call(true) != 403 {
		t.Fatal("browser aceito")
	}
	for range 100 {
		if call(false) != 422 {
			t.Fatal("validação/rate limit incorretos")
		}
	}
	if call(false) != 429 {
		t.Fatal("rate limit não aplicado antes de acessar banco")
	}
}
