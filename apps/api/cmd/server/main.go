package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/franklinlem/flagpilot/apps/api/internal/audit"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/config"
	"github.com/franklinlem/flagpilot/apps/api/internal/evaluation"
	"github.com/franklinlem/flagpilot/apps/api/internal/flags"
	httpapi "github.com/franklinlem/flagpilot/apps/api/internal/http"
	"github.com/franklinlem/flagpilot/apps/api/internal/observability"
	"github.com/franklinlem/flagpilot/apps/api/internal/organizations"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
	"github.com/franklinlem/flagpilot/apps/api/internal/tokens"
)

func main() { os.Exit(run()) }

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid_config", "error", err.Error())
		return 1
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := postgres.New(ctx, cfg.DatabaseURL, cfg.DBTimeout)
	if err != nil {
		logger.Error("database_init_failed")
		return 1
	}
	defer store.Close()
	authService, err := auth.New(store.Pool, cfg.SessionTTL, cfg.DBTimeout)
	if err != nil {
		logger.Error("auth_init_failed")
		return 1
	}
	var metrics *observability.Metrics
	var observer httpapi.Observer
	if cfg.MetricsAddr != "" {
		metrics = observability.New(store.Pool)
		observer = metrics
	}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.NewObserved(logger, store, observer, httpapi.WithAuth(authService, cfg.AllowedOrigins, cfg.SecureCookies), httpapi.WithAudit(authService, audit.New(store.Pool, cfg.DBTimeout), cfg.AllowedOrigins, cfg.SecureCookies), httpapi.WithOrganizations(authService, organizations.New(store.Pool, cfg.DBTimeout), cfg.AllowedOrigins, cfg.SecureCookies), httpapi.WithProjects(authService, projects.New(store.Pool, cfg.DBTimeout), cfg.AllowedOrigins, cfg.SecureCookies), httpapi.WithFlags(authService, flags.New(store.Pool, cfg.DBTimeout), cfg.AllowedOrigins, cfg.SecureCookies), httpapi.WithTokens(authService, tokens.New(store.Pool, cfg.DBTimeout), cfg.AllowedOrigins, cfg.SecureCookies), httpapi.WithEvaluation(evaluation.New(store.Pool, cfg.DBTimeout))), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}

	servers := []*http.Server{server}
	if metrics != nil {
		servers = append(servers, &http.Server{Addr: cfg.MetricsAddr, Handler: metrics.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8 * 1024})
	}
	done := make(chan error, len(servers))
	for _, current := range servers {
		go func() { done <- current.ListenAndServe() }()
	}
	logger.Info("server_start", "address", cfg.HTTPAddr, "environment", cfg.Environment)
	if metrics != nil {
		logger.Info("metrics_server_start", "address", cfg.MetricsAddr)
	}
	code := 0
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server_failed")
			code = 1
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	for _, current := range servers {
		if err := current.Shutdown(shutdown); err != nil {
			_ = current.Close()
			logger.Error("shutdown_timeout")
			code = 1
		}
	}
	logger.Info("server_stop")
	return code
}
