//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/flags"
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
	"strings"
	"sync"
	"testing"
	"time"
)

func exerciseFlags(t *testing.T, ctx context.Context, store *postgres.Store, owner *pgx.Conn) {
	service := flags.New(store.Pool, 3*time.Second)
	ps := projects.New(store.Pool, 3*time.Second)
	os := organizations.New(store.Pool, 3*time.Second)
	session, err := auth.New(store.Pool, 12*time.Hour, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	check := func(got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Fatalf("want %v got %v", want, got)
		}
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	password := "flags-test-password-123"
	hash, err := auth.HashPassword(password)
	check(err, nil)
	actor, outside := identifiers.String(identifiers.New()), identifiers.String(identifiers.New())
	exec("INSERT INTO users(id,email,password_hash) VALUES($1,'flags-owner@example.invalid',$3),($2,'flags-outside@example.invalid',$3)", actor, outside, hash)
	org, err := os.Create(ctx, actor, "Flags", "flags-org")
	check(err, nil)
	foreign, err := os.Create(ctx, outside, "Flags Outside", "flags-outside")
	check(err, nil)
	project, err := ps.Create(ctx, actor, org.ID, "Flags project", "flags-project")
	check(err, nil)
	other, err := ps.Create(ctx, outside, foreign.ID, "Other flags", "other-flags")
	check(err, nil)
	member, err := session.CreateUser(ctx, "flags-member@example.invalid", password, org.ID)
	check(err, nil)
	flag, err := service.Create(ctx, actor, project.ID, "new_checkout", "Checkout novo")
	check(err, nil)
	envs, err := ps.Environments(ctx, actor, project.ID, pagination.PageRequest{Limit: 25})
	check(err, nil)
	envIDs := map[string]string{}
	for _, env := range envs.Items {
		envIDs[env.Slug] = env.ID
		snap, _, err := service.Configs(ctx, actor, env.ID, pagination.PageRequest{Limit: 25})
		check(err, nil)
		if len(snap.Items) != 1 || snap.Items[0].Enabled || snap.Environment.Revision != 1 {
			t.Fatal("inicialização inconsistente")
		}
	}
	dev, stage, prod := envIDs["development"], envIDs["staging"], envIDs["production"]
	// Papéis herdados; Editor escreve flags, Viewer só lê.
	for _, role := range []string{"Viewer", "Editor", "Admin", "Owner"} {
		_, err = os.Change(ctx, actor, org.ID, member, role)
		check(err, nil)
		_, err = service.Get(ctx, member, project.ID, flag.Key)
		check(err, nil)
		_, err = service.Create(ctx, member, project.ID, "role_"+map[string]string{"Viewer": "viewer", "Editor": "editor", "Admin": "admin", "Owner": "owner"}[role], "")
		if role == "Viewer" {
			check(err, flags.ErrForbidden)
		} else {
			check(err, nil)
		}
	}
	_, err = service.Create(ctx, actor, project.ID, flag.Key, "")
	check(err, flags.ErrConflict)
	for _, key := range []string{"x", "Invalid", "bad key"} {
		_, err = service.Create(ctx, actor, project.ID, key, "")
		check(err, flags.ErrValidation)
	}
	_, err = service.Create(ctx, outside, project.ID, "forbidden", "")
	check(err, flags.ErrNotFound)
	_, err = service.Get(ctx, actor, other.ID, flag.Key)
	check(err, flags.ErrNotFound)
	_, _, err = service.Configs(ctx, outside, dev, pagination.PageRequest{Limit: 25})
	check(err, flags.ErrNotFound)
	before, tag, err := service.GetConfig(ctx, actor, stage, flag.Key)
	check(err, nil)
	_, canonicalTag, err := service.GetConfig(ctx, actor, strings.ToUpper(stage), flag.Key)
	check(err, nil)
	if canonicalTag != tag {
		t.Fatal("ETag depende da caixa do UUID recebido")
	}
	_, _, err = service.PutConfig(ctx, actor, stage, flag.Key, "", true, 10000, false)
	check(err, flags.ErrRequired)
	_, _, err = service.PutConfig(ctx, actor, stage, flag.Key, tag, true, 5000, false)
	check(err, flags.ErrValidation)
	changed, newTag, err := service.PutConfig(ctx, actor, stage, flag.Key, tag, true, 10000, false)
	check(err, nil)
	if !changed.Enabled || changed.Revision != before.Revision+1 || changed.Version != before.Version+1 || tag == newTag {
		t.Fatal("revisão não avançou")
	}
	_, _, err = service.PutConfig(ctx, actor, stage, flag.Key, tag, false, 10000, false)
	check(err, flags.ErrPrecondition)
	unchanged, prodTag, err := service.GetConfig(ctx, actor, prod, flag.Key)
	check(err, nil)
	if unchanged.Enabled {
		t.Fatal("staging alterou produção")
	}
	_, _, err = service.PutConfig(ctx, actor, prod, flag.Key, prodTag, true, 10000, false)
	check(err, flags.ErrProduction)
	_, _, err = service.PutConfig(ctx, actor, prod, flag.Key, prodTag, true, 10000, true)
	check(err, nil)
	// Slug personalizado de produção também exige confirmação.
	custom, err := ps.CreateEnvironment(ctx, actor, project.ID, "Live Europe", "eu-live", true)
	check(err, nil)
	cfg, customTag, err := service.GetConfig(ctx, actor, custom.ID, flag.Key)
	check(err, nil)
	if cfg.Enabled || cfg.Version != 1 {
		t.Fatal("novo ambiente herdou flag ligada")
	}
	_, _, err = service.PutConfig(ctx, actor, custom.ID, flag.Key, customTag, true, 10000, false)
	check(err, flags.ErrProduction)
	// Dois escritores da mesma revisão: exatamente um vence, sem perda silenciosa.
	_, raceTag, err := service.GetConfig(ctx, actor, dev, flag.Key)
	check(err, nil)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := service.PutConfig(ctx, actor, dev, flag.Key, raceTag, true, 10000, false)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success, stale := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, flags.ErrPrecondition) {
			stale++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatal("escrita perdida")
	}
	// Paginação ligada a snapshot; página seguinte de revisão antiga é rejeitada.
	first, firstTag, err := service.Configs(ctx, actor, dev, pagination.PageRequest{Limit: 1})
	check(err, nil)
	if first.NextCursor == nil {
		t.Fatal("paginação ausente")
	}
	second, _, err := service.Configs(ctx, actor, dev, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor})
	check(err, nil)
	if len(second.Items) != 1 || second.Items[0].FlagID == first.Items[0].FlagID {
		t.Fatal("página repetida")
	}
	_, _, err = service.PutConfig(ctx, actor, dev, flag.Key, firstTag, false, 10000, false)
	check(err, nil)
	_, _, err = service.Configs(ctx, actor, dev, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor})
	check(err, flags.ErrValidation)
	// Auditoria indisponível reverte configuração, versão, revisão e criação.
	old, oldTag, err := service.GetConfig(ctx, actor, stage, flag.Key)
	check(err, nil)
	exec("REVOKE INSERT ON audit_events FROM flagpilot_app")
	_, _, err = service.PutConfig(ctx, actor, stage, flag.Key, oldTag, false, 10000, false)
	check(err, flags.ErrUnavailable)
	_, err = service.Create(ctx, actor, project.ID, "rollback_flag", "")
	check(err, flags.ErrUnavailable)
	_, err = ps.CreateEnvironment(ctx, actor, project.ID, "Rollback environment", "rollback-env", false)
	check(err, projects.ErrUnavailable)
	exec("GRANT INSERT ON audit_events TO flagpilot_app")
	current, currentTag, err := service.GetConfig(ctx, actor, stage, flag.Key)
	check(err, nil)
	if current != old || currentTag != oldTag {
		t.Fatal("configuração parcial")
	}
	_, err = service.Get(ctx, actor, project.ID, "rollback_flag")
	check(err, flags.ErrNotFound)
	// Metadados versionados; não há alteração de chave ou restauração implícita.
	description := "Descrição editada"
	originalTag := flags.FlagETag(flag)
	flag, err = service.Change(ctx, actor, project.ID, flag.Key, originalTag, &description, nil, false)
	check(err, nil)
	_, err = service.Change(ctx, actor, project.ID, flag.Key, originalTag, &description, nil, false)
	check(err, flags.ErrPrecondition)
	archive := true
	_, err = service.Change(ctx, actor, project.ID, flag.Key, flags.FlagETag(flag), nil, &archive, false)
	check(err, flags.ErrProduction)
	exec("REVOKE INSERT ON audit_events FROM flagpilot_app")
	_, err = service.Change(ctx, actor, project.ID, flag.Key, flags.FlagETag(flag), nil, &archive, true)
	check(err, flags.ErrUnavailable)
	exec("GRANT INSERT ON audit_events TO flagpilot_app")
	// Corrida criar flag/criar ambiente não deixa lacunas.
	wg.Add(2)
	errs := make(chan error, 2)
	go func() {
		defer wg.Done()
		_, err := service.Create(ctx, actor, project.ID, "concurrent_flag", "")
		errs <- err
	}()
	go func() {
		defer wg.Done()
		_, err := ps.CreateEnvironment(ctx, actor, project.ID, "Concurrent", "concurrent-env", false)
		errs <- err
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		check(err, nil)
	}
	var missing int
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM flags f CROSS JOIN environments e LEFT JOIN flag_configs c ON c.flag_id=f.id AND c.environment_id=e.id WHERE f.project_id=$1 AND e.project_id=f.project_id AND f.archived_at IS NULL AND c.id IS NULL", project.ID).Scan(&missing); err != nil || missing != 0 {
		t.Fatal("lacunas após concorrência", err)
	}

	_, waitTag, err := service.GetConfig(ctx, member, stage, flag.Key)
	check(err, nil)
	tx, err := owner.Begin(ctx)
	check(err, nil)
	if _, err = tx.Exec(ctx, "SELECT id FROM organizations WHERE id=$1 FOR NO KEY UPDATE", org.ID); err != nil {
		t.Fatal(err)
	}
	denied := make(chan error, 1)
	go func() {
		_, _, err := service.PutConfig(ctx, member, stage, flag.Key, waitTag, false, 10000, false)
		denied <- err
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
		t.Fatal("configuração não aguardou lock")
	}
	if _, err = tx.Exec(ctx, "UPDATE memberships SET role='Viewer' WHERE organization_id=$1 AND user_id=$2", org.ID, member); err != nil {
		t.Fatal(err)
	}
	check(tx.Commit(ctx), nil)
	check(<-denied, flags.ErrForbidden)
	// Guards HTTP, If-Match e corpo estrito com cookie real.
	handler := httpapi.New(slog.New(slog.NewJSONHandler(io.Discard, nil)), store, httpapi.WithFlags(session, service, []string{"http://127.0.0.1:5173"}, false))
	login, err := session.Login(ctx, "flags-owner@example.invalid", password, "")
	check(err, nil)
	call := func(method, path string, body any, token, csrf, match string, want int) *httptest.ResponseRecorder {
		t.Helper()
		data, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://127.0.0.1:5173")
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("If-Match", match)
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "flagpilot_session", Value: token})
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, r)
		if res.Code != want {
			t.Fatalf("%s want %d got %d", path, want, res.Code)
		}
		return res
	}
	path := "/api/v1/environments/" + stage + "/flags/" + flag.Key
	call("GET", path, nil, "", "", "", 401)
	read := call("GET", path, nil, login.Token, "", "", 200)
	httpTag := read.Header().Get("ETag")
	if httpTag == "" {
		t.Fatal("ETag ausente")
	}
	body := map[string]any{"enabled": false, "rolloutBps": 10000}
	call("PUT", path, body, login.Token, "", httpTag, 403)
	call("PUT", path, body, login.Token, login.Identity.CSRFToken, "", 428)
	call("PUT", path, body, login.Token, login.Identity.CSRFToken, `"stale"`, 412)
	call("PUT", path, body, login.Token, login.Identity.CSRFToken, httpTag, 200)
	call("PUT", path, body, login.Token, login.Identity.CSRFToken, httpTag, 412)
	call("PUT", path, map[string]any{"enabled": true}, login.Token, login.Identity.CSRFToken, httpTag, 422)
	call("PATCH", "/api/v1/projects/"+project.ID+"/flags/"+flag.Key, map[string]any{"key": "changed"}, login.Token, login.Identity.CSRFToken, flags.FlagETag(flag), 400)
	// Viewer é negado pelo servidor mesmo com confirmação e revisão válidas.
	_, err = os.Change(ctx, actor, org.ID, member, "Viewer")
	check(err, nil)
	viewer, err := session.Login(ctx, "flags-member@example.invalid", password, "")
	check(err, nil)
	call("PUT", path, body, viewer.Token, viewer.Identity.CSRFToken, httpTag, 403)
	call("GET", "/api/v1/projects/"+other.ID+"/flags", nil, login.Token, "", "", 404)
	// Arquivamento altera snapshots, preserva chave e torna configuração inacessível.
	preArchive, _, err := service.GetConfig(ctx, actor, stage, flag.Key)
	check(err, nil)
	flag, err = service.Change(ctx, actor, project.ID, flag.Key, flags.FlagETag(flag), nil, &archive, true)
	check(err, nil)
	if flag.ArchivedAt == nil {
		t.Fatal("arquivamento perdido")
	}
	_, _, err = service.GetConfig(ctx, actor, stage, flag.Key)
	check(err, flags.ErrNotFound)
	_, _, err = service.PutConfig(ctx, actor, stage, flag.Key, flags.EnvironmentETag(stage, preArchive.Revision, "Owner"), true, 10000, false)
	check(err, flags.ErrNotFound)
	_, err = service.Create(ctx, actor, project.ID, flag.Key, "")
	check(err, flags.ErrConflict)
	after, _, err := service.Configs(ctx, actor, stage, pagination.PageRequest{Limit: 25})
	check(err, nil)
	if after.Environment.Revision != preArchive.Revision+1 {
		t.Fatal("snapshot arquivado não avançou")
	}
	for _, cfg := range after.Items {
		if cfg.FlagKey == flag.Key {
			t.Fatal("flag arquivada ainda ativa")
		}
	}
	late, err := ps.CreateEnvironment(ctx, actor, project.ID, "After archive", "after-archive", false)
	check(err, nil)
	_, _, err = service.GetConfig(ctx, actor, late.ID, flag.Key)
	check(err, flags.ErrNotFound)
	// FKs compostas e imutabilidade de chave/salt também são garantidas no banco.
	sqlError := func(sql, code string, args ...any) {
		t.Helper()
		_, err := store.Pool.Exec(ctx, sql, args...)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != code {
			t.Fatalf("constraint esperada %s got %v", code, err)
		}
	}
	sqlError("UPDATE flags SET key='changed' WHERE id=$1", "42501", flag.ID)
	sqlError("UPDATE flag_configs SET salt=repeat('0',64) WHERE flag_id=$1", "42501", flag.ID)
	otherEnvs, err := ps.Environments(ctx, outside, other.ID, pagination.PageRequest{Limit: 25})
	check(err, nil)
	sqlError("INSERT INTO flag_configs(id,project_id,flag_id,environment_id,salt) VALUES($1,$2,$3,$4,repeat('0',64))", "23503", identifiers.New(), other.ID, flag.ID, otherEnvs.Items[0].ID)
	var auditCount int
	if err = owner.QueryRow(ctx, "SELECT count(DISTINCT action) FROM audit_events WHERE organization_id=$1 AND action IN ('flag.created','flag.updated','flag.archived','flag.config_updated')", org.ID).Scan(&auditCount); err != nil || auditCount != 4 {
		t.Fatal("auditoria incompleta", err)
	}
}
