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

	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/config"
	httpapi "github.com/franklinlem/flagpilot/apps/api/internal/http"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
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
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.New(logger, store, httpapi.WithAuth(authService, cfg.AllowedOrigins, cfg.SecureCookies)), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	done := make(chan error, 1)
	go func() {
		logger.Info("server_start", "address", cfg.HTTPAddr, "environment", cfg.Environment)
		done <- server.ListenAndServe()
	}()
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server_failed")
			return 1
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			logger.Error("shutdown_timeout")
			return 1
		}
	}
	logger.Info("server_stop")
	return 0
}
