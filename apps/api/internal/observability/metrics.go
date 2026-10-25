// Package observability agrega medidas sem identidades, contexto ou credenciais.
package observability

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strconv"
	"time"
)

type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func New(pool *pgxpool.Pool) *Metrics {
	m := &Metrics{registry: prometheus.NewRegistry(), requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "flagpilot_http_requests_total", Help: "Completed API requests by bounded method, route pattern and status."}, []string{"method", "route", "status"}), duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "flagpilot_http_request_duration_seconds", Help: "API request duration in seconds by method and route pattern.", Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}}, []string{"method", "route"})}
	m.registry.MustRegister(m.requests, m.duration, collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	if pool != nil {
		for name, get := range map[string]func() float64{
			"connections":          func() float64 { return float64(pool.Stat().TotalConns()) },
			"acquired_connections": func() float64 { return float64(pool.Stat().AcquiredConns()) },
			"idle_connections":     func() float64 { return float64(pool.Stat().IdleConns()) },
			"max_connections":      func() float64 { return float64(pool.Stat().MaxConns()) },
		} {
			m.registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: "flagpilot_db_pool_" + name, Help: "Current database pool connection count."}, get))
		}
	}
	return m
}
func (m *Metrics) ObserveHTTP(method, route string, status int, duration time.Duration) {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
	default:
		method = "OTHER"
	}
	if route == "" {
		route = "unmatched"
	}
	if status < 100 || status > 599 {
		status = 500
	}
	m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(method, route).Observe(duration.Seconds())
}

// Handler só é montado no listener separado; nunca no router da aplicação.
func (m *Metrics) Handler() http.Handler {
	mux := http.NewServeMux()
	handler := promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{MaxRequestsInFlight: 2, Timeout: 5 * time.Second})
	mux.Handle("GET /metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		handler.ServeHTTP(w, r)
	}))
	return mux
}
