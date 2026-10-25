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
	"os/exec"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	// BuildKit via CLI mantém o mesmo Dockerfile corrigido usado em desenvolvimento/release.
	image := fmt.Sprintf("flagpilot-integration-%d:local", time.Now().UnixNano())
	if output, err := exec.CommandContext(ctx, "docker", "build", "--quiet", "-f", filepath.Join(root, "deploy/postgres/Dockerfile"), "-t", image, root).CombinedOutput(); err != nil {
		t.Fatalf("build da imagem PostgreSQL: %v: %s", err, output)
	}
	t.Cleanup(func() { _ = exec.Command("docker", "image", "rm", image).Run() })
	req := testcontainers.ContainerRequest{
		Image: image, ExposedPorts: []string{"5432/tcp"},
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
	if err := m.Steps(1); err != nil {
		t.Fatal(err)
	}
	check("/api/v1/health/ready", 503)

	if err := m.Steps(1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, "INSERT INTO organizations(id,name,slug) VALUES('00000000-0000-4000-8000-000000000010','Legacy identity','legacy-identity')"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, "INSERT INTO users(id,email,password_hash) VALUES('00000000-0000-4000-8000-000000000011','legacy@example.invalid','test-hash-preserved')"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, "INSERT INTO memberships(organization_id,user_id,role) VALUES('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000011','Owner')"); err != nil {
		t.Fatal(err)
	}

	if err := m.Steps(1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, "INSERT INTO projects(id,organization_id,name,slug) VALUES('00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000010','Legacy project','legacy-project')"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, "INSERT INTO environments(id,project_id,name,slug,is_production,revision) VALUES('00000000-0000-4000-8000-000000000013','00000000-0000-4000-8000-000000000012','Legacy production','production',true,7)"); err != nil {
		t.Fatal(err)
	}

	if err := m.Steps(1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, "INSERT INTO flags(id,project_id,key,description) VALUES('00000000-0000-4000-8000-000000000014','00000000-0000-4000-8000-000000000012','legacy_flag','Existing flag')"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool.Exec(ctx, "INSERT INTO flag_configs(id,project_id,flag_id,environment_id,enabled,salt,version) VALUES('00000000-0000-4000-8000-000000000015','00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000014','00000000-0000-4000-8000-000000000013',true,repeat('a',64),3)"); err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}

	var preserved int
	if err := store.Pool.QueryRow(ctx, "SELECT count(*) FROM memberships m JOIN users u ON u.id=m.user_id JOIN organizations o ON o.id=m.organization_id WHERE o.slug='legacy-identity' AND u.password_hash='test-hash-preserved' AND m.role='Owner'").Scan(&preserved); err != nil || preserved != 1 {
		t.Fatal("upgrade perdeu identidade existente", err)
	}

	if err := store.Pool.QueryRow(ctx, "SELECT count(*) FROM environments e JOIN projects p ON p.id=e.project_id WHERE p.slug='legacy-project' AND e.revision=7 AND e.is_production").Scan(&preserved); err != nil || preserved != 1 {
		t.Fatal("upgrade perdeu projeto/ambiente existente", err)
	}

	if err := store.Pool.QueryRow(ctx, "SELECT count(*) FROM flag_configs c JOIN flags f ON f.id=c.flag_id WHERE f.key='legacy_flag' AND c.enabled AND c.version=3 AND c.salt=repeat('a',64)").Scan(&preserved); err != nil || preserved != 1 {
		t.Fatal("upgrade perdeu flag/configuração existente", err)
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
	if _, err := owner.Exec(ctx, "UPDATE schema_migrations SET dirty=false,version=6"); err != nil {
		t.Fatal(err)
	}
	check("/api/v1/health/ready", 503)
	if _, err := owner.Exec(ctx, "UPDATE schema_migrations SET version=5"); err != nil {
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
	t.Run("authentication", func(t *testing.T) { exerciseAuth(t, ctx, store, owner) })
	t.Run("organizations", func(t *testing.T) { exerciseOrganizations(t, ctx, store, owner) })
	t.Run("projects", func(t *testing.T) { exerciseProjects(t, ctx, store, owner) })
	t.Run("flags", func(t *testing.T) { exerciseFlags(t, ctx, store, owner) })
	t.Run("evaluation", func(t *testing.T) { exerciseEvaluation(t, ctx, store, owner) })
	t.Run("dashboard", func(t *testing.T) { exerciseDashboard(t, ctx, store, owner) })
	if _, err := owner.Exec(ctx, "ALTER ROLE flagpilot_app NOLOGIN"); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename='flagpilot_app'"); err != nil {
		t.Fatal(err)
	}
	check("/api/v1/health/ready", 503)
	check("/api/v1/health/live", 200)
}
