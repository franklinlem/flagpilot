//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	httpapi "github.com/franklinlem/flagpilot/apps/api/internal/http"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
)

func TestFoundation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	req := testcontainers.ContainerRequest{
		Image: "postgres:16.13-alpine", ExposedPorts: []string{"5432/tcp"},
		Env:        map[string]string{"POSTGRES_USER": "migrator", "POSTGRES_PASSWORD": "integration-only-owner", "POSTGRES_DB": "flagpilot_test", "APP_DB_PASSWORD": "integration-only-runtime"},
		Files:      []testcontainers.ContainerFile{{HostFilePath: filepath.Join(root, "deploy/postgres/init-runtime.sh"), ContainerFilePath: "/docker-entrypoint-initdb.d/10-runtime.sh", FileMode: 0755}},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(time.Minute),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Error(err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}
	ownerURL := fmt.Sprintf("postgres://migrator:integration-only-owner@%s:%s/flagpilot_test?sslmode=disable", host, port.Port())
	runtimeURL := fmt.Sprintf("postgres://flagpilot_app:integration-only-runtime@%s:%s/flagpilot_test?sslmode=disable", host, port.Port())
	m, err := migrate.New((&url.URL{Scheme: "file", Path: filepath.Join(root, "db/migrations")}).String(), ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = m.Close() }()
	store, err := postgres.New(ctx, runtimeURL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	router := httpapi.New(slog.New(slog.NewJSONHandler(io.Discard, nil)), store)
	check := func(path string, want int) {
		t.Helper()
		r := httptest.NewRecorder()
		router.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != want {
			t.Fatalf("%s: want %d got %d", path, want, r.Code)
		}
	}
	check("/api/v1/health/live", 200)
	check("/api/v1/health/ready", 503)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migration não idempotente: %v", err)
	}
	check("/api/v1/health/ready", 200)
	if _, err := store.Pool.Exec(ctx, "CREATE TABLE forbidden(id integer)"); err == nil {
		t.Fatal("runtime executou DDL")
	}
	_, err = store.Pool.Exec(ctx, "INSERT INTO users (id,email,password_hash) VALUES ('00000000-0000-0000-0000-000000000001','one@example.invalid','not-a-real-password')")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"INSERT INTO users (id,email,password_hash) VALUES ('00000000-0000-0000-0000-000000000002','ONE@example.invalid','x')",
		"INSERT INTO memberships (organization_id,user_id,role) VALUES ('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000001','Owner')",
		"INSERT INTO sessions (id,user_id,token_hash,expires_at) VALUES ('00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000001',decode('00','hex'),now()+interval '1 hour')",
	} {
		if _, err := store.Pool.Exec(ctx, q); err == nil {
			t.Fatal("restrição ausente")
		}
	}
	owner, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = owner.Close(context.Background()) }()
	if _, err := owner.Exec(ctx, "UPDATE schema_migrations SET dirty=true"); err != nil {
		t.Fatal(err)
	}
	check("/api/v1/health/ready", 503)
	if _, err := owner.Exec(ctx, "UPDATE schema_migrations SET dirty=false,version=2"); err != nil {
		t.Fatal(err)
	}
	check("/api/v1/health/ready", 503)
	if _, err := owner.Exec(ctx, "UPDATE schema_migrations SET version=1"); err != nil {
		t.Fatal(err)
	}
	if err := m.Down(); err != nil {
		t.Fatal(err)
	}
	check("/api/v1/health/ready", 503)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	check("/api/v1/health/ready", 200)
	if _, err := owner.Exec(ctx, "ALTER ROLE flagpilot_app NOLOGIN"); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename='flagpilot_app'"); err != nil {
		t.Fatal(err)
	}
	check("/api/v1/health/ready", 503)
	check("/api/v1/health/live", 200)
}
