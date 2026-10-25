package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Readiness interface{ Ready(context.Context) error }
type requestIDKey struct{}

func RequestID(ctx context.Context) string { id, _ := ctx.Value(requestIDKey{}).(string); return id }

type Observer interface {
	ObserveHTTP(method, route string, status int, duration time.Duration)
}

func New(logger *slog.Logger, db Readiness, options ...Option) http.Handler {
	return NewObserved(logger, db, nil, options...)
}
func NewObserved(logger *slog.Logger, db Readiness, observer Observer, options ...Option) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			var b [16]byte
			if _, err := rand.Read(b[:]); err != nil {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
				return
			}
			id := hex.EncodeToString(b[:])
			req = req.WithContext(context.WithValue(req.Context(), requestIDKey{}, id))
			w.Header().Set("X-Request-ID", id)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Cache-Control", "no-store")
			wrapped := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			start := time.Now()
			defer func() {
				if recover() != nil {
					problem(wrapped, req, 500, "Erro interno")
				}
				route := chi.RouteContext(req.Context()).RoutePattern()
				if route == "" {
					route = "unmatched"
				}
				status := wrapped.Status()
				if status == 0 {
					status = 200
				}
				if observer != nil {
					observer.ObserveHTTP(req.Method, route, status, time.Since(start))
				}
				logger.Info("http_request", "request_id", id, "method", req.Method, "route", route, "status", status, "duration_ms", time.Since(start).Milliseconds())
			}()
			next.ServeHTTP(wrapped, req)
		})
	})
	r.Get("/api/v1/health/live", func(w http.ResponseWriter, req *http.Request) {
		jsonResponse(w, 200, map[string]string{"status": "ok"})
	})
	r.Get("/api/v1/health/ready", func(w http.ResponseWriter, req *http.Request) {
		if err := db.Ready(req.Context()); err != nil {
			problem(w, req, 503, "Serviço indisponível")
			return
		}
		jsonResponse(w, 200, map[string]string{"status": "ok"})
	})
	for _, option := range options {
		option(r)
	}
	r.NotFound(func(w http.ResponseWriter, req *http.Request) { problem(w, req, 404, "Recurso não encontrado") })
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) { problem(w, req, 405, "Método não permitido") })
	return r
}

func problem(w http.ResponseWriter, req *http.Request, status int, title string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": title, "status": status, "detail": title, "instance": "urn:request:" + RequestID(req.Context()), "requestId": RequestID(req.Context())})
}
func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
