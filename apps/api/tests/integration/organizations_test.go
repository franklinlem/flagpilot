//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	httpapi "github.com/franklinlem/flagpilot/apps/api/internal/http"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/organizations"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
	"github.com/jackc/pgx/v5"
)

func exerciseOrganizations(t *testing.T, ctx context.Context, store *postgres.Store, owner *pgx.Conn) {
	session, err := auth.New(store.Pool, 12*time.Hour, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	service := organizations.New(store.Pool, 3*time.Second)
	password := "tenant-test-password-123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	a, b := identifiers.New(), identifiers.New()
	aid, bid := identifiers.String(a), identifiers.String(b)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := owner.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO users (id,email,password_hash) VALUES ($1,'tenant-a@example.invalid',$3),($2,'tenant-b@example.invalid',$3)", a, b, hash)
	orgA, err := service.Create(ctx, aid, "Tenant A", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	orgB, err := service.Create(ctx, bid, "Tenant B", "tenant-b")
	if err != nil {
		t.Fatal(err)
	}
	c, err := session.CreateUser(ctx, "tenant-viewer@example.invalid", password, orgA.ID)
	if err != nil {
		t.Fatal(err)
	}
	checkErr := func(got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Fatalf("want %v got %v", want, got)
		}
	}
	checkList := func(actor, org string, want error) {
		t.Helper()
		_, err := service.Members(ctx, actor, org, organizations.PageRequest{Limit: 25})
		if want == nil {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			checkErr(err, want)
		}
	}
	list, err := service.List(ctx, bid, organizations.PageRequest{Limit: 25})
	if err != nil || len(list.Items) != 1 || list.Items[0].ID != orgB.ID {
		t.Fatal("listagem cruzou tenants")
	}

	// Nenhum diretório global: B é ativo, mas não compartilha escopo com A.
	candidates, err := service.Candidates(ctx, aid, orgA.ID, organizations.PageRequest{Limit: 25})
	if err != nil || len(candidates.Items) != 0 {
		t.Fatal("candidatos expuseram usuário externo ou membro atual", err)
	}
	_, err = service.Candidates(ctx, bid, orgA.ID, organizations.PageRequest{Limit: 25})
	checkErr(err, organizations.ErrNotFound)
	checkList(bid, orgA.ID, organizations.ErrNotFound)
	_, err = service.Add(ctx, bid, orgA.ID, bid, "Owner")
	checkErr(err, organizations.ErrNotFound)
	_, err = service.Change(ctx, bid, orgA.ID, aid, "Viewer")
	checkErr(err, organizations.ErrNotFound)
	checkErr(service.Remove(ctx, bid, orgA.ID, aid), organizations.ErrNotFound)
	for _, role := range []string{"Viewer", "Editor", "Admin"} {
		if _, err = service.Change(ctx, aid, orgA.ID, c, role); err != nil {
			t.Fatal(err)
		}
		checkList(c, orgA.ID, nil)
		_, err = service.Candidates(ctx, c, orgA.ID, organizations.PageRequest{Limit: 25})
		checkErr(err, organizations.ErrForbidden)

		_, err = service.Add(ctx, c, orgA.ID, bid, "Viewer")
		checkErr(err, organizations.ErrForbidden)
		_, err = service.Change(ctx, c, orgA.ID, aid, "Viewer")
		checkErr(err, organizations.ErrForbidden)
		checkErr(service.Remove(ctx, c, orgA.ID, aid), organizations.ErrForbidden)
	}
	_, err = service.Change(ctx, aid, orgA.ID, aid, "Viewer")
	checkErr(err, organizations.ErrLastOwner)
	checkErr(service.Remove(ctx, aid, orgA.ID, aid), organizations.ErrLastOwner)
	_, err = service.Add(ctx, aid, orgA.ID, c, "Viewer")
	checkErr(err, organizations.ErrConflict)
	_, err = service.Add(ctx, aid, orgA.ID, bid, "Root")
	checkErr(err, organizations.ErrValidation)
	if _, err = service.Add(ctx, aid, orgA.ID, bid, "Viewer"); err != nil {
		t.Fatal(err)
	}

	// Seleção entre organizações conhecidas; paginação e exclusão de desativados.
	selection, err := service.Create(ctx, aid, "Seleção", "selection-org")
	if err != nil {
		t.Fatal(err)
	}
	candidatePage, err := service.Candidates(ctx, aid, selection.ID, organizations.PageRequest{Limit: 1})
	if err != nil || len(candidatePage.Items) != 1 || candidatePage.NextCursor == nil {
		t.Fatal("paginação de candidatos", err)
	}
	candidateNext, err := service.Candidates(ctx, aid, selection.ID, organizations.PageRequest{Limit: 1, Cursor: *candidatePage.NextCursor})
	if err != nil || len(candidateNext.Items) != 1 || candidateNext.Items[0].UserID == candidatePage.Items[0].UserID || candidateNext.NextCursor != nil {
		t.Fatal("candidatos repetidos/omitidos", err)
	}
	_, err = service.Candidates(ctx, aid, orgA.ID, organizations.PageRequest{Limit: 1, Cursor: *candidatePage.NextCursor})
	checkErr(err, organizations.ErrValidation)
	_, err = service.Members(ctx, aid, selection.ID, organizations.PageRequest{Limit: 1, Cursor: *candidatePage.NextCursor})
	checkErr(err, organizations.ErrValidation)
	exec("UPDATE users SET disabled_at=now() WHERE id=$1", c)
	candidates, err = service.Candidates(ctx, aid, selection.ID, organizations.PageRequest{Limit: 25})
	if err != nil || len(candidates.Items) != 1 || candidates.Items[0].UserID != bid {
		t.Fatal("candidato desativado", err)
	}
	exec("UPDATE users SET disabled_at=NULL WHERE id=$1", c)
	// Remoção da única associação não impede selecionar o antigo membro pelo email.
	checkErr(service.Remove(ctx, aid, orgA.ID, c), nil)
	candidates, err = service.Candidates(ctx, aid, orgA.ID, organizations.PageRequest{Limit: 25})
	if err != nil || len(candidates.Items) != 1 || candidates.Items[0].UserID != c {
		t.Fatal("antigo membro ausente", err)
	}
	if _, err = service.Add(ctx, aid, orgA.ID, c, "Admin"); err != nil {
		t.Fatal(err)
	}
	// Cursor validado por coleção, organização e usuário; página estável sem duplicatas.
	first, err := service.Members(ctx, aid, orgA.ID, organizations.PageRequest{Limit: 1})
	if err != nil || first.NextCursor == nil {
		t.Fatal("paginação ausente")
	}
	_, err = service.Members(ctx, bid, orgB.ID, organizations.PageRequest{Limit: 1, Cursor: *first.NextCursor})
	checkErr(err, organizations.ErrValidation)
	seen := map[string]bool{}
	cursor := ""
	for {
		page, err := service.Members(ctx, aid, orgA.ID, organizations.PageRequest{Limit: 1, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range page.Items {
			if seen[m.UserID] {
				t.Fatal("membro duplicado")
			}
			seen[m.UserID] = true
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if len(seen) != 3 {
		t.Fatal("membros omitidos")
	}
	// Falhas de auditoria revertem toda mutação e provisionamento.
	exec("REVOKE INSERT ON audit_events FROM flagpilot_app")
	_, err = service.Change(ctx, aid, orgA.ID, c, "Viewer")
	checkErr(err, organizations.ErrUnavailable)
	checkErr(service.Remove(ctx, aid, orgA.ID, c), organizations.ErrUnavailable)
	_, err = service.Create(ctx, aid, "Rollback", "rollback-org")
	checkErr(err, organizations.ErrUnavailable)
	_, err = session.CreateUser(ctx, "rollback-user@example.invalid", password, orgA.ID)
	checkErr(err, auth.ErrUnavailable)
	exec("GRANT INSERT ON audit_events TO flagpilot_app")
	var role string
	var count int
	if err = owner.QueryRow(ctx, "SELECT role FROM memberships WHERE organization_id=$1 AND user_id=$2", orgA.ID, c).Scan(&role); err != nil || role != "Admin" {
		t.Fatal("mutação parcial")
	}
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM organizations WHERE slug='rollback-org'").Scan(&count); err != nil || count != 0 {
		t.Fatal("organização parcial")
	}
	if err = owner.QueryRow(ctx, "SELECT count(*) FROM users WHERE email='rollback-user@example.invalid'").Scan(&count); err != nil || count != 0 {
		t.Fatal("usuário parcial")
	}
	// Leitura do papel acontece DEPOIS do lock, inclusive se aguardou revogação.
	if _, err = service.Change(ctx, aid, orgA.ID, c, "Owner"); err != nil {
		t.Fatal(err)
	}
	tx, err := owner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "SELECT id FROM organizations WHERE id=$1 FOR NO KEY UPDATE", orgA.ID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := service.Change(ctx, c, orgA.ID, bid, "Editor"); result <- err }()
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
	if _, err = tx.Exec(ctx, "UPDATE memberships SET role='Viewer' WHERE organization_id=$1 AND user_id=$2", orgA.ID, c); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	checkErr(<-result, organizations.ErrForbidden)
	// Duas auto-despromoções concorrentes não eliminam o último Owner.
	race, err := service.Create(ctx, aid, "Race", "race-org")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Add(ctx, aid, race.ID, bid, "Owner"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, actor := range []string{aid, bid} {
		wg.Add(1)
		go func(actor string) {
			defer wg.Done()
			_, err := service.Change(ctx, actor, race.ID, actor, "Viewer")
			results <- err
		}(actor)
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, organizations.ErrLastOwner) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("último Owner perdido")
	}
	// Só um Owner ativo, mesmo quando há outro Owner desativado.
	if _, err = service.Change(ctx, aid, orgA.ID, bid, "Owner"); err != nil {
		t.Fatal(err)
	}
	exec("UPDATE users SET disabled_at=now() WHERE id=$1", bid)
	checkErr(service.Remove(ctx, aid, orgA.ID, aid), organizations.ErrLastOwner)
	exec("UPDATE users SET disabled_at=NULL WHERE id=$1", bid)
	// Revogar A não encerra sessões nem altera o acesso de C a B.
	if _, err = service.Add(ctx, bid, orgB.ID, c, "Viewer"); err != nil {
		t.Fatal(err)
	}
	cs, err := session.Login(ctx, "tenant-viewer@example.invalid", password, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Remove(ctx, aid, orgA.ID, c); err != nil {
		t.Fatal(err)
	}
	if _, err = session.Find(ctx, cs.Token); err != nil {
		t.Fatal("sessão global revogada por remoção local")
	}
	checkList(c, orgA.ID, organizations.ErrNotFound)
	checkList(c, orgB.ID, nil)
	// HTTP realista: sessão, CSRF e isolamento não dependem dos controles da UI.
	handler := httpapi.New(slog.New(slog.NewJSONHandler(io.Discard, nil)), store, httpapi.WithAuth(session, []string{"http://127.0.0.1:5173"}, false), httpapi.WithOrganizations(session, service, []string{"http://127.0.0.1:5173"}, false))
	as, err := session.Login(ctx, "tenant-a@example.invalid", password, "")
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path string, body any, token, csrf string, want int) {
		t.Helper()
		data, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://127.0.0.1:5173")
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "flagpilot_session", Value: token})
		}
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, r)
		if res.Code != want {
			t.Fatalf("%s %s want %d got %d %s", method, path, want, res.Code, res.Body.String())
		}
		if want == 404 && strings.Contains(res.Body.String(), "tenant-a@example.invalid") {
			t.Fatal("erro expôs tenant")
		}
	}
	call("GET", "/api/v1/organizations", nil, "", "", 401)
	call("GET", "/api/v1/organizations/"+orgA.ID+"/members", nil, cs.Token, "", 404)
	call("GET", "/api/v1/organizations/"+orgB.ID+"/members", nil, cs.Token, "", 200)

	call("GET", "/api/v1/organizations/"+orgA.ID+"/member-candidates", nil, "", "", 401)
	call("GET", "/api/v1/organizations/"+orgA.ID+"/member-candidates", nil, cs.Token, "", 404)
	call("GET", "/api/v1/organizations/"+orgB.ID+"/member-candidates", nil, cs.Token, "", 403)
	call("GET", "/api/v1/organizations/"+orgA.ID+"/member-candidates", nil, as.Token, "", 200)
	call("GET", "/api/v1/organizations/"+orgA.ID+"/member-candidates?limit=101", nil, as.Token, "", 422)
	call("GET", "/api/v1/organizations?limit=101", nil, as.Token, "", 422)
	call("POST", "/api/v1/organizations", map[string]string{"name": "HTTP", "slug": "http-org"}, as.Token, "", 403)
	call("POST", "/api/v1/organizations", map[string]string{"name": "HTTP", "slug": "http-org"}, as.Token, as.Identity.CSRFToken, 201)
	call("PATCH", "/api/v1/organizations/"+orgB.ID+"/members/"+c, map[string]string{"role": "Owner"}, cs.Token, cs.Identity.CSRFToken, 403)
	call("DELETE", "/api/v1/organizations/"+orgA.ID+"/members/"+aid, nil, cs.Token, cs.Identity.CSRFToken, 404)
	var events string
	if err = owner.QueryRow(ctx, "SELECT jsonb_agg(to_jsonb(audit_events))::text FROM audit_events WHERE organization_id=$1", orgA.ID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"organization.created", "user.provisioned", "member.added", "member.role_changed", "member.removed"} {
		if !strings.Contains(events, action) {
			t.Fatal("mutação sem auditoria")
		}
	}
	if strings.Contains(events, password) || strings.Contains(events, cs.Token) {
		t.Fatal("segredo na auditoria")
	}
}
