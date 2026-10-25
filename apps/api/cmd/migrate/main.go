// Command migrate aplica somente migrations up; configuração sensível vem do ambiente.
package main

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func run(databaseURL, path string, output io.Writer) int {
	logger := slog.New(slog.NewJSONHandler(output, nil))
	if databaseURL == "" || path == "" {
		logger.Error("migration configuration missing")
		return 1
	}
	migration, err := migrate.New("file://"+path, databaseURL)
	if err != nil {
		// Erros de conexão/source podem conter DSN; nunca registrar err/URL.
		logger.Error("migration initialization failed")
		return 1
	}
	defer migration.Close()
	migration.LockTimeout = 15 * time.Second
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	var interrupted atomic.Bool
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-signals:
			interrupted.Store(true)
			select {
			case migration.GracefulStop <- true:
			case <-finished:
			}
		case <-finished:
		}
	}()
	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("migration up failed; inspect schema state before retry")
		return 1
	}
	if interrupted.Load() {
		logger.Warn("migration interrupted")
		return 130
	}
	logger.Info("migrations up completed")
	return 0
}

func main() {
	path := os.Getenv("MIGRATION_PATH")
	if path == "" {
		path = "/migrations"
	}
	os.Exit(run(os.Getenv("MIGRATION_DATABASE_URL"), path, os.Stderr))
}
