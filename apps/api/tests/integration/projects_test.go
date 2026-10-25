//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	httpapi "github.com/franklinlem/flagpilot/apps/api/internal/http"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/organizations"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func exerciseProjects(t *testing.T, ctx context.Context, store *postgres.Store, owner *pgx.Conn) {
	service := projects.New(store.Pool, 3*time.Second)
	orgs := organizations.New(store.Pool, 3*time.Second)
	session, err := auth.New(store.Pool, 12*time.Hour, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	password := "projects-test-password-123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	actor, outsider := identifiers.String(identifiers.New()), identifiers.String(identifiers.New())
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	check := func(got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Fatalf("want %v got %v", want, got)
		}
	}
	exec("INSERT INTO users(id,email,password_hash) VALUES($1,'project-owner@example.invalid',$3),($2,'project-outsider@example.invalid',$3)", actor, outsider, hash)
	org, err := orgs.Create(ctx, actor, "Projects", "projects-test")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := orgs.Create(ctx, outsider, "Foreign projects", "foreign-projects-test")
	if err != nil {
		t.Fatal(err)
	}
	member, err := session.CreateUser(ctx, "project-member@example.invalid", password, org.ID)
	if err != nil {
		t.Fatal(err)
	}
	project, err := service.Create(ctx, actor, org.ID, " Loja ", "loja")
	if err != nil || project.Name != "Loja" {
		t.Fatal("projeto inválido", err)
	}
	envs, err := service.Environments(ctx, actor, project.ID, pagination.PageRequest{Limit: 25})
	if err != nil || len(envs.Items) != 3 {
		t.Fatal("presets ausentes", err)
	}
	presets := map[string]bool{}
	for _, env := range envs.Items {
		presets[env.Slug] = true
		if env.ProjectID != project.ID || env.Revision != 0 || env.IsProduction != (env.Slug == "production") {
			t.Fatal("ambiente inconsistente")
		}
	}
	for _, slug := range []string{"development", "staging", "production"} {
		if !presets[slug] {
			t.Fatal("preset ausente")
		}
	}
	for _, role := range []string{"Viewer", "Editor", "Admin", "Owner"} {
		if _, err = orgs.Change(ctx, actor, org.ID, member, role); err != nil {
			t.Fatal(err)
		}
		if got, err := service.Get(ctx, member, project.ID); err != nil || got.Role != role {
			t.Fatal("papel herdado", err)
		}
		_, err = service.List(ctx, member, org.ID, pagination.PageRequest{Limit: 25})
		check(err, nil)
		_, err = service.Environments(ctx, member, project.ID, pagination.PageRequest{Limit: 25})
		check(err, nil)
		_, err = service.Create(ctx, member, org.ID, "Role "+role, "role-"+map[string]string{"Viewer": "viewer", "Editor": "editor", "Admin": "admin", "Owner": "owner"}[role])
		_, envErr := service.CreateEnvironment(ctx, member, project.ID, "Role "+role, "role-"+map[string]string{"Viewer": "viewer", "Editor": "editor", "Admin": "admin", "Owner": "owner"}[role], false)
		if role == "Owner" || role == "Admin" {
			check(err, nil)
			check(envErr, nil)
		} else {
			check(err, projects.ErrForbidden)
			check(envErr, projects.ErrForbidden)
		}
	}
	// Slug único no escopo, nunca global; produção identificada explicitamente.
	_, err = service.Create(ctx, actor, org.ID, "Duplicado", "loja")
	check(err, projects.ErrConflict)
	other, err := service.Create(ctx, outsider, foreign.ID, "Outra loja", "loja")
	check(err, nil)
	_, err = service.CreateEnvironment(ctx, actor, project.ID, "Duplicado", "staging", false)
	check(err, projects.ErrConflict)
	_, err = service.CreateEnvironment(ctx, actor, project.ID, "Produção", "production", false)
	check(err, projects.ErrValidation)
	custom, err := service.CreateEnvironment(ctx, actor, project.ID, "Produção Europa", "eu-live", true)
	check(err, nil)
	if !custom.IsProduction {
		t.Fatal("classificação de produção perdida")
	}
	for _, nameSlug := range [][2]string{{" ", "valid-slug"}, {"Nome", "Invalid"}, {"Nome", "x"}} {
		_, err = service.Create(ctx, actor, org.ID, nameSlug[0], nameSlug[1])
		check(err, projects.ErrValidation)
	}
	_, err = service.Get(ctx, outsider, project.ID)
	check(err, projects.ErrNotFound)
	_, err = service.List(ctx, outsider, org.ID, pagination.PageRequest{Limit: 25})
	check(err, projects.ErrNotFound)
	_, err = service.Create(ctx, outsider, org.ID, "IDOR", "idor")
	check(err, projects.ErrNotFound)
	_, err = service.Environments(ctx, actor, other.ID, pagination.PageRequest{Limit: 25})
	check(err, projects.ErrNotFound)
	_, err = service.CreateEnvironment(ctx, actor, other.ID, "IDOR", "idor", false)
	check(err, projects.ErrNotFound)
	first, err := service.List(ctx, actor, org.ID, pagination.PageRequest{Limit: 1})
	check(err, nil)
	if first.NextCursor == nil {
		t.Fatal("paginação de projetos ausente")
	}
	second, err := service.List(ctx, actor, org.ID, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor})
	check(err, nil)
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatal("paginação repetida")
	}
	_, err = service.List(ctx, outsider, foreign.ID, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor})
	check(err, projects.ErrValidation)
	_, err = service.Environments(ctx, actor, project.ID, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor})
	check(err, projects.ErrValidation)
	seen := map[string]bool{}
	cursor := ""
	for {
		page, err := service.Environments(ctx, actor, project.ID, pagination.PageRequest{Limit: 1, Cursor: cursor})
		check(err, nil)
		for _, env := range page.Items {
			if seen[env.ID] {
				t.Fatal("ambiente duplicado")
			}
			seen[env.ID] = true
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if len(seen) != 6 {
		t.Fatal("ambiente omitido")
	}
	// Falha no terceiro preset reverte projeto, presets anteriores e toda auditoria.
	exec("CREATE FUNCTION reject_production_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.slug='production' THEN RAISE EXCEPTION 'test preset failure'; END IF; RETURN NEW; END $$")
	exec("CREATE TRIGGER reject_production_test BEFORE INSERT ON environments FOR EACH ROW EXECUTE FUNCTION reject_production_test()")
	_, err = service.Create(ctx, actor, org.ID, "Rollback presets", "rollback-presets")
	check(err, projects.ErrUnavailable)
	exec("DROP TRIGGER reject_production_test ON environments")
	exec("DROP FUNCTION reject_production_test()")
	var count int
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM projects WHERE slug='rollback-presets'").Scan(&count); err != nil || count != 0 {
		t.Fatal("projeto parcial")
	}
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE resource_type='project' AND after_redacted->>'slug'='rollback-presets'").Scan(&count); err != nil || count != 0 {
		t.Fatal("auditoria parcial")
	}
	exec("REVOKE INSERT ON audit_events FROM flagpilot_app")
	_, err = service.Create(ctx, actor, org.ID, "Rollback audit", "rollback-audit")
	check(err, projects.ErrUnavailable)
	_, err = service.CreateEnvironment(ctx, actor, project.ID, "Rollback audit", "rollback-audit", false)
	check(err, projects.ErrUnavailable)
	exec("GRANT INSERT ON audit_events TO flagpilot_app")
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM environments WHERE slug='rollback-audit'").Scan(&count); err != nil || count != 0 {
		t.Fatal("ambiente sem auditoria")
	}
	// Revogação enquanto aguarda lock precisa ser relida.
	tx, err := owner.Begin(ctx)
	check(err, nil)
	if _, err = tx.Exec(ctx, "SELECT id FROM organizations WHERE id=$1 FOR NO KEY UPDATE", org.ID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := service.CreateEnvironment(ctx, member, project.ID, "Stale", "stale-role", false)
		result <- err
	}()
	waiting := false
	for range 100 {
		var n int
		if err = tx.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE usename='flagpilot_app' AND wait_event_type='Lock'").Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			waiting = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !waiting {
		_ = tx.Rollback(ctx)
		t.Fatal("operação não aguardou lock")
	}
	if _, err = tx.Exec(ctx, "UPDATE memberships SET role='Viewer' WHERE organization_id=$1 AND user_id=$2", org.ID, member); err != nil {
		t.Fatal(err)
	}
	check(tx.Commit(ctx), nil)
	check(<-result, projects.ErrForbidden)
	// HTTP inclui guards reais e contrato estrito.
	handler := httpapi.New(slog.New(slog.NewJSONHandler(io.Discard, nil)), store, httpapi.WithProjects(session, service, []string{"http://127.0.0.1:5173"}, false))
	login, err := session.Login(ctx, "project-owner@example.invalid", password, "")
	check(err, nil)
	viewer, err := session.Login(ctx, "project-member@example.invalid", password, "")
	check(err, nil)
	call := func(method, path string, body any, token, csrf string, want int) {
		t.Helper()
		data, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://127.0.0.1:5173")
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "flagpilot_session", Value: token})
		}
		r.Header.Set("X-CSRF-Token", csrf)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, r)
		if res.Code != want {
			t.Fatalf("%s %s want %d got %d", method, path, want, res.Code)
		}
	}
	path := "/api/v1/projects/" + project.ID + "/environments"
	call("GET", path, nil, "", "", 401)
	call("GET", path, nil, viewer.Token, "", 200)
	call("POST", path, map[string]any{"name": "API", "slug": "api", "isProduction": false}, viewer.Token, viewer.Identity.CSRFToken, 403)
	call("POST", path, map[string]any{"name": "API", "slug": "api", "isProduction": false}, login.Token, "", 403)
	call("POST", path, map[string]any{"name": "API", "slug": "api"}, login.Token, login.Identity.CSRFToken, 422)
	call("POST", path, map[string]any{"name": "API", "slug": "api", "isProduction": false, "unexpected": true}, login.Token, login.Identity.CSRFToken, 400)
	call("POST", path, map[string]any{"name": "API", "slug": "api", "isProduction": false}, login.Token, login.Identity.CSRFToken, 201)
	call("GET", "/api/v1/projects/"+other.ID, nil, login.Token, "", 404)
	call("GET", path+"?limit=101", nil, login.Token, "", 422)
	call("GET", "/api/v1/organizations/"+org.ID+"/projects", nil, login.Token, "", 200)
	check(orgs.Remove(ctx, actor, org.ID, member), nil)
	call("GET", path, nil, viewer.Token, "", 404)

	// Integridade e menor privilégio: validar códigos, evitando falso positivo por duplicidade.
	checkSQL := func(sql, code string, args ...any) {
		t.Helper()
		_, err := store.Pool.Exec(ctx, sql, args...)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != code {
			t.Fatalf("restrição SQL: want %s got %v", code, err)
		}
	}
	checkSQL("UPDATE environments SET is_production=false WHERE id=$1", "42501", custom.ID)
	checkSQL("UPDATE projects SET name='Renamed' WHERE id=$1", "42501", project.ID)
	checkSQL("DELETE FROM projects WHERE id=$1", "42501", project.ID)
	checkSQL("INSERT INTO environments(id,project_id,name,slug,is_production) VALUES($1,$2,'Invalid','production',false)", "23514", identifiers.New(), other.ID)
	checkSQL("INSERT INTO environments(id,project_id,name,slug,is_production) VALUES($1,$2,'Invalid','valid',false)", "23503", identifiers.New(), identifiers.New())
}
