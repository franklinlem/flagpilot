//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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

func exerciseAuth(t *testing.T, ctx context.Context, store *postgres.Store, owner *pgx.Conn) {
	service, err := auth.New(store.Pool, 12*time.Hour, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var secret [16]byte
	if _, err = rand.Read(secret[:]); err != nil {
		t.Fatal(err)
	}
	password := "test-" + hex.EncodeToString(secret[:])
	email := "admin@example.invalid"
	execOwner := func(sql string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	// Auditoria participa da mesma transação do bootstrap.
	execOwner("REVOKE INSERT ON audit_events FROM flagpilot_app")
	if err = service.Bootstrap(ctx, email, password, "E2E Organization", "e2e"); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatalf("bootstrap sem auditoria: %v", err)
	}
	var count int
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil || count != 0 {
		t.Fatal("bootstrap parcial")
	}
	execOwner("GRANT INSERT ON audit_events TO flagpilot_app")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- service.Bootstrap(ctx, email, password, "E2E Organization", "e2e") }()
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, auth.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("bootstrap concorrente não protegido")
	}
	for _, sql := range []string{"UPDATE audit_events SET action='tamper'", "DELETE FROM audit_events"} {
		if _, err := store.Pool.Exec(ctx, sql); err == nil {
			t.Fatal("runtime alterou auditoria")
		}
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := httpapi.New(logger, store, httpapi.WithAuth(service, []string{"http://127.0.0.1:5173"}, false))
	call := func(h http.Handler, method, path, body, token, csrf, origin string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "127.0.0.1:12345"
		r.Header.Set("Content-Type", "application/json")
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "flagpilot_session", Value: token})
		}
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		res := httptest.NewRecorder()
		h.ServeHTTP(res, r)
		return res
	}
	origin := "http://127.0.0.1:5173"
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	check := func(res *httptest.ResponseRecorder, want int) {
		t.Helper()
		if res.Code != want {
			t.Fatalf("status want=%d got=%d body=%s", want, res.Code, res.Body.String())
		}
	}
	check(call(handler, "GET", "/api/v1/me", "", "", "", ""), 401)
	check(call(handler, "POST", "/api/v1/auth/register", "{}", "", "", origin), 404)
	check(call(handler, "POST", "/api/v1/auth/login", string(body), "", "", ""), 403)
	check(call(handler, "POST", "/api/v1/auth/login", string(body), "", "", "https://evil.invalid"), 403)
	for _, tc := range []struct {
		site, content string
		want          int
	}{{"cross-site", "application/json", 403}, {"same-origin", "text/plain", 415}} {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", tc.content)
		req.Header.Set("Sec-Fetch-Site", tc.site)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		check(res, tc.want)
	}

	check(call(handler, "POST", "/api/v1/auth/login", `{"extra":true}`, "", "", origin), 400)
	check(call(handler, "POST", "/api/v1/auth/login", string(body)+`{}`, "", "", origin), 400)
	check(call(handler, "POST", "/api/v1/auth/login", `{"password":"`+strings.Repeat("x", 5000)+`"}`, "", "", origin), 413)
	check(call(handler, "POST", "/api/v1/auth/login", `{"email":"bad","password":"value"}`, "", "", origin), 422)
	for _, who := range []string{email, "missing@example.invalid"} {
		b, _ := json.Marshal(map[string]string{"email": who, "password": "wrong"})
		check(call(handler, "POST", "/api/v1/auth/login", string(b), "", "", origin), 401)
	}
	login := call(handler, "POST", "/api/v1/auth/login", string(body), "", "", origin)
	check(login, 200)
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("cookie ausente")
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.MaxAge <= 0 {
		t.Fatal("atributos do cookie inválidos")
	}
	var me auth.Identity
	if err = json.Unmarshal(login.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.User.Email != email || len(me.Memberships) != 1 || me.Memberships[0].Role != "Owner" || me.Memberships[0].OrganizationName != "E2E Organization" {
		t.Fatal("identidade/escopo incorreto")
	}
	if strings.Contains(login.Body.String(), cookie.Value) || strings.Contains(login.Body.String(), password) {
		t.Fatal("segredo na resposta")
	}
	check(call(handler, "GET", "/api/v1/me", "", cookie.Value, "", ""), 200)
	duplicate := httptest.NewRequest("GET", "/api/v1/me", nil)
	duplicate.AddCookie(cookie)
	duplicate.AddCookie(cookie)
	duplicateResponse := httptest.NewRecorder()
	handler.ServeHTTP(duplicateResponse, duplicate)
	check(duplicateResponse, 401)

	check(call(handler, "POST", "/api/v1/auth/logout", "", cookie.Value, "", origin), 403)
	check(call(handler, "POST", "/api/v1/auth/logout", "", cookie.Value, strings.Repeat("a", 64), origin), 403)
	check(call(handler, "POST", "/api/v1/auth/logout", "", cookie.Value, me.CSRFToken, "https://evil.invalid"), 403)
	// Cookie rotacionado invalida sessão anterior e seu CSRF.
	second := call(handler, "POST", "/api/v1/auth/login", string(body), cookie.Value, "", origin)
	check(second, 200)
	token := second.Result().Cookies()[0].Value
	if token == cookie.Value {
		t.Fatal("sessão não rotacionada")
	}
	check(call(handler, "GET", "/api/v1/me", "", cookie.Value, "", ""), 401)
	if err = json.Unmarshal(second.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	var stored []byte
	sum := sha256.Sum256([]byte(token))
	if err = owner.QueryRow(ctx, "SELECT token_hash FROM sessions WHERE token_hash=$1", sum[:]).Scan(&stored); err != nil || !bytes.Equal(stored, sum[:]) {
		t.Fatal("token não armazenado como hash")
	}
	// Falha na auditoria reverte criação/revogação de sessão e recuperação de senha.
	execOwner("REVOKE INSERT ON audit_events FROM flagpilot_app")
	if _, err = service.Login(ctx, email, password, token); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("login sem auditoria")
	}
	if err = service.ResetPassword(ctx, email, "replacement-test-password"); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("reset sem auditoria")
	}
	if _, err = service.Find(ctx, token); err != nil {
		t.Fatal("falha de reset revogou sessão")
	}
	execOwner("GRANT INSERT ON audit_events TO flagpilot_app")
	check(call(handler, "GET", "/api/v1/me", "", token, "", ""), 200)
	out := call(handler, "POST", "/api/v1/auth/logout", "", token, me.CSRFToken, origin)
	check(out, 204)
	if out.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("cookie não removido")
	}
	check(call(handler, "GET", "/api/v1/me", "", token, "", ""), 401)
	a, err := service.Login(ctx, email, password, "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := service.Login(ctx, email, password, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = service.ResetPassword(ctx, email, "replacement-test-password"); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{a.Token, b.Token} {
		if _, err = service.Find(ctx, token); !errors.Is(err, auth.ErrSession) {
			t.Fatal("reset não revogou todas as sessões")
		}
	}
	if _, err = service.Login(ctx, email, password, ""); !errors.Is(err, auth.ErrCredentials) {
		t.Fatal("senha antiga aceita")
	}
	active, err := service.Login(ctx, email, "replacement-test-password", "")
	if err != nil {
		t.Fatal(err)
	}
	execOwner("UPDATE users SET disabled_at=now() WHERE email=$1", email)
	if _, err = service.Find(ctx, active.Token); !errors.Is(err, auth.ErrSession) {
		t.Fatal("conta desativada acessou sessão")
	}
	if _, err = service.Login(ctx, email, "replacement-test-password", ""); !errors.Is(err, auth.ErrCredentials) {
		t.Fatal("conta desativada entrou")
	}
	execOwner("UPDATE users SET disabled_at=NULL WHERE email=$1", email)
	execOwner("UPDATE sessions SET expires_at=created_at+interval '1 microsecond'")
	if _, err = service.Find(ctx, active.Token); !errors.Is(err, auth.ErrSession) {
		t.Fatal("sessão expirada aceita")
	}
	secure := httpapi.New(logger, store, httpapi.WithAuth(service, []string{origin}, true))
	secureBody, _ := json.Marshal(map[string]string{"email": email, "password": "replacement-test-password"})
	secureLogin := call(secure, "POST", "/api/v1/auth/login", string(secureBody), "", "", origin)
	check(secureLogin, 200)
	if !secureLogin.Result().Cookies()[0].Secure {
		t.Fatal("cookie produção sem Secure")
	}
	rateHandler := httpapi.New(logger, store, httpapi.WithAuth(service, []string{origin}, false))
	for range 10 {
		check(call(rateHandler, "POST", "/api/v1/auth/login", `{"email":"missing@example.invalid","password":"wrong"}`, "", "", origin), 401)
	}
	limited := call(rateHandler, "POST", "/api/v1/auth/login", string(secureBody), "", "", origin)
	check(limited, 429)
	if limited.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After ausente")
	}
	var events string
	if err = owner.QueryRow(ctx, "SELECT jsonb_agg(to_jsonb(audit_events))::text FROM audit_events").Scan(&events); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"installation.bootstrap", "auth.login", "auth.logout", "auth.password_reset"} {
		if !strings.Contains(events, action) {
			t.Fatal("ação não auditada")
		}
	}
	for _, secret := range []string{password, "replacement-test-password", token, cookie.Value, me.CSRFToken} {
		if strings.Contains(events, secret) || strings.Contains(logs.String(), secret) {
			t.Fatal("segredo em auditoria/log")
		}
	}
}
