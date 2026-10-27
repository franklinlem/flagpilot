//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	httpapi "github.com/franklinlem/flagpilot/apps/api/internal/http"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
	"github.com/jackc/pgx/v5"
)

func exercisePasswordChange(t *testing.T, ctx context.Context, store *postgres.Store, owner *pgx.Conn) {
	service, err := auth.New(store.Pool, 12*time.Hour, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	const email = "self-service@example.invalid"
	const oldPassword = "old-self-service-password"
	const newPassword = "new-self-service-password"
	var org string
	if err = owner.QueryRow(ctx, "SELECT id::text FROM organizations WHERE slug='e2e'").Scan(&org); err != nil {
		t.Fatal(err)
	}
	userID, err := service.CreateUser(ctx, email, oldPassword, org)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Login(ctx, email, oldPassword, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Login(ctx, email, oldPassword, "")
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	h := httpapi.New(slog.New(slog.NewJSONHandler(&logs, nil)), store, httpapi.WithAuth(service, []string{"http://127.0.0.1:5173"}, false))
	call := func(body, token, csrf, origin, content string, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest("POST", "/api/v1/auth/password", strings.NewReader(body))
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", content)
		req.Header.Set("X-CSRF-Token", csrf)
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "flagpilot_session", Value: token})
		}
		out := httptest.NewRecorder()
		h.ServeHTTP(out, req)
		if out.Code != want {
			t.Fatalf("password change: want %d got %d", want, out.Code)
		}
		return out
	}
	origin := "http://127.0.0.1:5173"
	body := func(current, next string) string {
		b, _ := json.Marshal(map[string]string{"currentPassword": current, "newPassword": next})
		return string(b)
	}
	valid := body(oldPassword, newPassword)
	call(valid, "", "", origin, "application/json", 401)
	call(valid, first.Token, first.Identity.CSRFToken, "https://evil.invalid", "application/json", 403)
	call(valid, first.Token, "", origin, "application/json", 403)
	call(valid, first.Token, first.Identity.CSRFToken, origin, "text/plain", 415)
	call(`{"userId":"other"}`, first.Token, first.Identity.CSRFToken, origin, "application/json", 400)
	call(valid+`{}`, first.Token, first.Identity.CSRFToken, origin, "application/json", 400)
	call(body(strings.Repeat("x", 5000), newPassword), first.Token, first.Identity.CSRFToken, origin, "application/json", 413)
	call(body("wrong-current-password", newPassword), first.Token, first.Identity.CSRFToken, origin, "application/json", 403)
	for _, invalid := range []string{"short", strings.Repeat(" ", 12), strings.Repeat("á", 129), oldPassword} {
		call(body(oldPassword, invalid), first.Token, first.Identity.CSRFToken, origin, "application/json", 422)
	}
	// Sem auditoria, senha e todas as sessões permanecem intactas.
	if _, err = owner.Exec(ctx, "REVOKE INSERT ON audit_events FROM flagpilot_app"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = owner.Exec(context.Background(), "GRANT INSERT ON audit_events TO flagpilot_app") }()
	call(valid, first.Token, first.Identity.CSRFToken, origin, "application/json", 503)
	for _, token := range []string{first.Token, second.Token} {
		if _, err = service.Find(ctx, token); err != nil {
			t.Fatal("falha revogou sessão")
		}
	}
	var encoded string
	if err = owner.QueryRow(ctx, "SELECT password_hash FROM users WHERE id=$1", userID).Scan(&encoded); err != nil || !auth.VerifyPassword(oldPassword, encoded) {
		t.Fatal("falha modificou senha")
	}
	if _, err = owner.Exec(ctx, "GRANT INSERT ON audit_events TO flagpilot_app"); err != nil {
		t.Fatal(err)
	}
	out := call(valid, first.Token, first.Identity.CSRFToken, origin, "application/json", 204)
	if len(out.Result().Cookies()) != 1 || out.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("cookie não removido")
	}
	for _, token := range []string{first.Token, second.Token} {
		if _, err = service.Find(ctx, token); !errors.Is(err, auth.ErrSession) {
			t.Fatal("sessão antiga aceita")
		}
	}
	if _, err = service.Login(ctx, email, oldPassword, ""); !errors.Is(err, auth.ErrCredentials) {
		t.Fatal("senha antiga aceita")
	}
	current, err := service.Login(ctx, email, newPassword, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = service.ChangePassword(ctx, second.Token, newPassword, "other-password-long"); !errors.Is(err, auth.ErrSession) {
		t.Fatal("sessão revogada alterou senha")
	}
	// Logout mantém um lock na sessão enquanto insere auditoria (FK do usuário).
	// O lock de senha não deve bloquear esse KEY SHARE, formando um deadlock.
	logoutSession, err := service.Login(ctx, email, newPassword, "")
	if err != nil {
		t.Fatal(err)
	}
	blocker, err := owner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	sum := sha256.Sum256([]byte(logoutSession.Token))
	if _, err = blocker.Exec(ctx, "SELECT id FROM sessions WHERE token_hash=$1 FOR UPDATE", sum[:]); err != nil {
		t.Fatal(err)
	}
	pending := make(chan error, 1)
	go func() {
		pending <- service.ChangePassword(ctx, logoutSession.Token, newPassword, "logout-race-password")
	}()
	waiting := false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		// pg_stat_activity é cacheado por transação: observar novamente o backend.
		if _, err = blocker.Exec(ctx, "SELECT pg_stat_clear_snapshot()"); err != nil {
			t.Fatal(err)
		}
		if err = blocker.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE '%SELECT s.user_id FROM sessions%')").Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("troca não aguardou sessão bloqueada")
	}
	if _, err = blocker.Exec(ctx, "INSERT INTO audit_events(id,organization_id,actor_id,actor_label,action,resource_type,resource_id) VALUES(gen_random_uuid(),$1,$2,'user','auth.logout','user',$2)", org, userID); err != nil {
		t.Fatal("auditoria de logout bloqueada pelo lock de senha:", err)
	}
	if _, err = blocker.Exec(ctx, "UPDATE sessions SET revoked_at=now() WHERE token_hash=$1", sum[:]); err != nil {
		t.Fatal(err)
	}
	if err = blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-pending; !errors.Is(err, auth.ErrSession) {
		t.Fatal("sessão revogada durante hash alterou senha")
	}
	// Duas alterações simultâneas: só uma vence; a outra observa sessão/hash antigo.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, next := range []string{"concurrent-password-one", "concurrent-password-two"} {
		wg.Add(1)
		go func(next string) {
			defer wg.Done()
			results <- service.ChangePassword(ctx, current.Token, newPassword, next)
		}(next)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, auth.ErrSession) && !errors.Is(err, auth.ErrCredentials) {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatal("alterações concorrentes não serializadas")
	}
	var events string
	if err = owner.QueryRow(ctx, "SELECT jsonb_agg(to_jsonb(a))::text FROM audit_events a WHERE actor_id=$1 AND action='auth.password_changed'", userID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(events, "auth.password_changed") {
		t.Fatal("auditoria ausente")
	}
	for _, secret := range []string{oldPassword, newPassword, first.Token, second.Token, first.Identity.CSRFToken, encoded} {
		if strings.Contains(events, secret) || strings.Contains(logs.String(), secret) {
			t.Fatal("segredo em log/auditoria")
		}
	}
	// Limites de senha são independentes dos contadores de login.
	h = httpapi.New(slog.New(slog.NewJSONHandler(&logs, nil)), store, httpapi.WithAuth(service, []string{origin}, false))
	var finalPassword string
	for _, next := range []string{"concurrent-password-one", "concurrent-password-two"} {
		if session, e := service.Login(ctx, email, next, ""); e == nil {
			current = session
			finalPassword = next
			break
		}
	}
	if finalPassword == "" {
		t.Fatal("senha concorrente não funciona")
	}
	for range 10 {
		call(body("wrong-current-password", newPassword), current.Token, current.Identity.CSRFToken, origin, "application/json", 403)
	}
	limited := call(body(finalPassword, newPassword), current.Token, current.Identity.CSRFToken, origin, "application/json", 429)
	if limited.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After ausente")
	}
}
