//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/evaluation"
	"github.com/franklinlem/flagpilot/apps/api/internal/flags"
	httpapi "github.com/franklinlem/flagpilot/apps/api/internal/http"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/organizations"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
	"github.com/franklinlem/flagpilot/apps/api/internal/tokens"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func exerciseEvaluation(t *testing.T, ctx context.Context, store *postgres.Store, owner *pgx.Conn) {
	ts := tokens.New(store.Pool, 3*time.Second)
	ev := evaluation.New(store.Pool, 3*time.Second)
	fs := flags.New(store.Pool, 3*time.Second)
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
	password := "evaluation-test-password-123"
	hash, err := auth.HashPassword(password)
	check(err, nil)
	actor, outside := identifiers.String(identifiers.New()), identifiers.String(identifiers.New())
	exec("INSERT INTO users(id,email,password_hash) VALUES($1,'evaluation-owner@example.invalid',$3),($2,'evaluation-outside@example.invalid',$3)", actor, outside, hash)
	org, err := os.Create(ctx, actor, "Evaluation", "evaluation-org")
	check(err, nil)
	foreign, err := os.Create(ctx, outside, "Evaluation outside", "evaluation-outside")
	check(err, nil)
	project, err := ps.Create(ctx, actor, org.ID, "Evaluation project", "evaluation-project")
	check(err, nil)
	other, err := ps.Create(ctx, outside, foreign.ID, "Other evaluation", "other-evaluation")
	check(err, nil)
	member, err := session.CreateUser(ctx, "evaluation-member@example.invalid", password, org.ID)
	check(err, nil)
	main, err := fs.Create(ctx, actor, project.ID, "checkout", "Checkout")
	check(err, nil)
	_, err = fs.Create(ctx, actor, project.ID, "disabled_flag", "")
	check(err, nil)
	_, err = fs.Create(ctx, outside, other.ID, "foreign_only", "")
	check(err, nil)
	envs, err := ps.Environments(ctx, actor, project.ID, pagination.PageRequest{Limit: 25})
	check(err, nil)
	envIDs := map[string]string{}
	for _, env := range envs.Items {
		envIDs[env.Slug] = env.ID
	}
	dev, prod := envIDs["development"], envIDs["production"]
	_, etag, err := fs.GetConfig(ctx, actor, dev, main.Key)
	check(err, nil)
	_, _, err = fs.PutConfig(ctx, actor, dev, main.Key, etag, true, 10000, false)
	check(err, nil)
	expiry := time.Now().Add(24 * time.Hour)
	issued, err := ts.Issue(ctx, actor, dev, "Application dev", expiry)
	check(err, nil)
	production, err := ts.Issue(ctx, actor, prod, "Application production", expiry)
	check(err, nil)
	if !tokens.ValidSecret(issued.Secret) || issued.Secret == production.Secret {
		t.Fatal("token inválido ou repetido")
	}
	digest := sha256.Sum256([]byte(issued.Secret))
	var stored []byte
	if err = owner.QueryRow(ctx, "SELECT token_hash FROM api_tokens WHERE id=$1", issued.Token.ID).Scan(&stored); err != nil || !bytes.Equal(stored, digest[:]) {
		t.Fatal("hash não corresponde")
	}
	requests := []evaluation.Request{{FlagKey: main.Key}, {FlagKey: "disabled_flag"}, {FlagKey: "missing_flag"}, {FlagKey: main.Key}, {FlagKey: "foreign_only"}}
	result, err := ev.Evaluate(ctx, issued.Secret, requests)
	check(err, nil)
	if len(result.Results) != 5 || !result.Results[0].Value || result.Results[0].Reason != "ENABLED" || result.Results[1].Value || result.Results[1].Reason != "DISABLED" || result.Results[2].Reason != "NOT_FOUND" || result.Results[4].Reason != "NOT_FOUND" || result.Results[3] != result.Results[0] {
		t.Fatal("avaliação ou ordem incorreta")
	}
	for _, item := range result.Results {
		if item.Revision != result.Revision || item.Environment != "development" {
			t.Fatal("snapshot misturado")
		}
	}
	prodResult, err := ev.Evaluate(ctx, production.Secret, []evaluation.Request{{FlagKey: main.Key}})
	check(err, nil)
	if prodResult.Results[0].Value {
		t.Fatal("token de dev atravessou ambiente")
	}
	listed, err := ts.List(ctx, actor, dev, pagination.PageRequest{Limit: 25})
	check(err, nil)
	metadata, _ := json.Marshal(listed)
	if strings.Contains(string(metadata), issued.Secret) || strings.Contains(string(metadata), "token_hash") {
		t.Fatal("listagem expôs segredo")
	}
	_, err = ts.Issue(ctx, actor, dev, "", expiry)
	check(err, tokens.ErrValidation)
	_, err = ts.Issue(ctx, actor, dev, "Past", time.Now().Add(-time.Hour))
	check(err, tokens.ErrValidation)
	_, err = ts.Issue(ctx, actor, dev, "Forever", time.Now().Add(366*24*time.Hour))
	check(err, tokens.ErrValidation)
	_, err = ts.Issue(ctx, outside, dev, "IDOR", expiry)
	check(err, tokens.ErrNotFound)
	_, err = ts.List(ctx, outside, dev, pagination.PageRequest{Limit: 25})
	check(err, tokens.ErrNotFound)
	check(ts.Revoke(ctx, actor, prod, issued.Token.ID), tokens.ErrNotFound)
	for _, role := range []string{"Viewer", "Editor", "Admin", "Owner"} {
		_, err = os.Change(ctx, actor, org.ID, member, role)
		check(err, nil)
		_, listErr := ts.List(ctx, member, dev, pagination.PageRequest{Limit: 25})
		trial, issueErr := ts.Issue(ctx, member, dev, "Role test", expiry)
		if role == "Viewer" || role == "Editor" {
			check(listErr, tokens.ErrForbidden)
			check(issueErr, tokens.ErrForbidden)
			check(ts.Revoke(ctx, member, dev, issued.Token.ID), tokens.ErrForbidden)
		} else {
			check(listErr, nil)
			check(issueErr, nil)
			check(ts.Revoke(ctx, member, dev, trial.Token.ID), nil)
		}
	}
	first, err := ts.List(ctx, actor, dev, pagination.PageRequest{Limit: 1})
	check(err, nil)
	if first.NextCursor == nil {
		t.Fatal("paginação ausente")
	}
	second, err := ts.List(ctx, actor, dev, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor})
	check(err, nil)
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatal("token repetido")
	}
	_, err = ts.List(ctx, actor, prod, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor})
	check(err, tokens.ErrValidation)
	exec("REVOKE INSERT ON audit_events FROM flagpilot_app")
	failed, err := ts.Issue(ctx, actor, dev, "Rollback issue", expiry)
	check(err, tokens.ErrUnavailable)
	if failed.Secret != "" {
		t.Fatal("segredo retornado sem commit")
	}
	check(ts.Revoke(ctx, actor, dev, issued.Token.ID), tokens.ErrUnavailable)
	exec("GRANT INSERT ON audit_events TO flagpilot_app")
	_, err = ev.Evaluate(ctx, issued.Secret, requests)
	check(err, nil)
	var count int
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM api_tokens WHERE name='Rollback issue'").Scan(&count); err != nil || count != 0 {
		t.Fatal("emissão parcial")
	}
	expired, err := ts.Issue(ctx, actor, dev, "Expiry", expiry)
	check(err, nil)
	exec("UPDATE api_tokens SET created_at=clock_timestamp()-interval '2 days',expires_at=clock_timestamp()-interval '1 day' WHERE id=$1", expired.Token.ID)
	_, err = ev.Evaluate(ctx, expired.Secret, requests)
	check(err, tokens.ErrUnauthorized)
	// Contexto não vai ao banco nem à auditoria.
	bucketingKey := "ephemeral-private-context-123"
	_, err = ev.Evaluate(ctx, issued.Secret, []evaluation.Request{{FlagKey: main.Key, Context: &evaluation.Context{BucketingKey: &bucketingKey}}})
	check(err, nil)
	var audits string
	if err = owner.QueryRow(ctx, "SELECT jsonb_agg(to_jsonb(audit_events))::text FROM audit_events WHERE organization_id=$1", org.ID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(audits, issued.Secret) || strings.Contains(audits, bucketingKey) || strings.Contains(audits, "token_hash") {
		t.Fatal("auditoria contém segredo/contexto")
	}
	// Um lote não mistura estados/revisões durante alterações concorrentes.
	_, err = fs.Create(ctx, actor, project.ID, "pair_one", "")
	check(err, nil)
	_, err = fs.Create(ctx, actor, project.ID, "pair_two", "")
	check(err, nil)
	writer := make(chan error, 1)
	go func() {
		for i := range 20 {
			tx, err := owner.Begin(ctx)
			if err != nil {
				writer <- err
				return
			}
			if _, err = tx.Exec(ctx, "SELECT id FROM environments WHERE id=$1 FOR NO KEY UPDATE", dev); err == nil {
				_, err = tx.Exec(ctx, "UPDATE flag_configs SET enabled=$2,version=flag_configs.version+1 FROM flags f WHERE flag_configs.flag_id=f.id AND flag_configs.environment_id=$1 AND f.key IN ('pair_one','pair_two')", dev, i%2 == 0)
			}
			if err == nil {
				_, err = tx.Exec(ctx, "UPDATE environments SET revision=revision+1 WHERE id=$1", dev)
			}
			if err != nil {
				_ = tx.Rollback(ctx)
				writer <- err
				return
			}
			if err = tx.Commit(ctx); err != nil {
				writer <- err
				return
			}
		}
		writer <- nil
	}()
	for range 20 {
		snapshot, err := ev.Evaluate(ctx, issued.Secret, []evaluation.Request{{FlagKey: "pair_one"}, {FlagKey: "pair_two"}})
		check(err, nil)
		if snapshot.Results[0].Value != snapshot.Results[1].Value || snapshot.Results[0].Revision != snapshot.Results[1].Revision {
			t.Fatal("batch misturou snapshot")
		}
	}
	check(<-writer, nil)
	// Arquivada retorna NOT_FOUND mesmo que tenha estado enabled=true persistido.
	archive := true
	_, err = fs.Change(ctx, actor, project.ID, main.Key, flags.FlagETag(main), nil, &archive, true)
	check(err, nil)
	archived, err := ev.Evaluate(ctx, issued.Secret, []evaluation.Request{{FlagKey: main.Key}})
	check(err, nil)
	if archived.Results[0].Value || archived.Results[0].Reason != "NOT_FOUND" {
		t.Fatal("arquivada avaliada como ativa")
	}
	// HTTP real: segredo não administra e cookie não avalia.
	handler := httpapi.New(slog.New(slog.NewJSONHandler(io.Discard, nil)), store, httpapi.WithTokens(session, ts, []string{"http://127.0.0.1:5173"}, false), httpapi.WithEvaluation(ev), httpapi.WithFlags(session, fs, []string{"http://127.0.0.1:5173"}, false))
	login, err := session.Login(ctx, "evaluation-owner@example.invalid", password, "")
	check(err, nil)
	call := func(method, path, body, credential, cookie, csrf, origin string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if credential != "" {
			r.Header.Set("Authorization", "Bearer "+credential)
		}
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: "flagpilot_session", Value: cookie})
		}
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s expected %d got %d", path, want, w.Code)
		}
		return w
	}
	call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag"}`, "", login.Token, "", "", 401)
	call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag"}`, login.Token, "", "", "", 401)
	call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag"}`, issued.Secret, "", "", "http://127.0.0.1:5173", 403)
	call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag"}`, "fp_eval_"+strings.Repeat("f", 64), "", "", "", 401)
	call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag"}`, issued.Secret, "", "", "", 200)
	call("POST", "/api/v1/evaluate/batch", `{"evaluations":[{"flagKey":"disabled_flag"},{"flagKey":"missing_flag"}]}`, issued.Secret, "", "", "", 200)
	call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag","environmentId":"other"}`, issued.Secret, "", "", "", 400)
	call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag","context":{"email":"private@example.invalid"}}`, issued.Secret, "", "", "", 400)
	call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag","context":{"bucketingKey":"`+strings.Repeat("x", 65536)+`"}}`, issued.Secret, "", "", "", 413)
	call("POST", "/api/v1/evaluate/batch", `{"evaluations":[]}`, issued.Secret, "", "", "", 422)
	batch := make([]evaluation.Request, 101)
	for i := range batch {
		batch[i].FlagKey = "disabled_flag"
	}
	data, _ := json.Marshal(map[string]any{"evaluations": batch})
	call("POST", "/api/v1/evaluate/batch", string(data), issued.Secret, "", "", "", 422)
	call("POST", "/api/v1/projects/"+project.ID+"/flags", `{"key":"intrusion"}`, issued.Secret, "", "", "", 401)
	path := "/api/v1/environments/" + dev + "/tokens"
	call("GET", path, "", issued.Secret, "", "", "", 401)
	call("GET", path, "", "", login.Token, "", "", 200)
	requestBody := `{"name":"HTTP","expiresAt":"` + expiry.UTC().Format(time.RFC3339) + `"}`
	call("POST", path, requestBody, "", login.Token, "", "http://127.0.0.1:5173", 403)
	call("POST", path, requestBody, "", login.Token, login.Identity.CSRFToken, "http://127.0.0.1:5173", 201)
	// Revogação afeta chamadas seguintes, sem cache positivo de autenticação.
	check(ts.Revoke(ctx, actor, dev, issued.Token.ID), nil)
	check(ts.Revoke(ctx, actor, dev, issued.Token.ID), nil)
	invalid := call("POST", "/api/v1/evaluate", `{"flagKey":"disabled_flag"}`, issued.Secret, "", "", "", 401)
	if invalid.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("desafio Bearer ausente")
	}
	_, err = ev.Evaluate(ctx, production.Secret, requests)
	check(err, nil)
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE resource_id=$1 AND action='token.revoked'", issued.Token.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("revogação duplicou auditoria")
	}
	sqlError := func(sql, code string, args ...any) {
		t.Helper()
		_, err := store.Pool.Exec(ctx, sql, args...)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != code {
			t.Fatalf("SQL esperava %s", code)
		}
	}
	sqlError("UPDATE api_tokens SET expires_at=now()+interval '1 year' WHERE id=$1", "42501", production.Token.ID)
	sqlError("UPDATE api_tokens SET token_hash=decode(repeat('00',32),'hex') WHERE id=$1", "42501", production.Token.ID)
	foreignEnvs, err := ps.Environments(ctx, outside, other.ID, pagination.PageRequest{Limit: 25})
	check(err, nil)
	sqlError("INSERT INTO api_tokens(id,project_id,environment_id,name,token_prefix,token_hash,scope,expires_at) VALUES($1,$2,$3,'Wrong scope','fp_eval_00000000',decode(repeat('00',32),'hex'),'evaluation',now()+interval '1 day')", "23503", identifiers.New(), project.ID, foreignEnvs.Items[0].ID)
	sqlError("INSERT INTO api_tokens(id,project_id,environment_id,name,token_prefix,token_hash,scope,expires_at) VALUES($1,$2,$3,'Wrong permission','fp_eval_00000000',decode(repeat('00',32),'hex'),'admin',now()+interval '1 day')", "23514", identifiers.New(), project.ID, dev)

}
