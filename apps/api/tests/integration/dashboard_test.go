//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/audit"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/flags"
	httpapi "github.com/franklinlem/flagpilot/apps/api/internal/http"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/organizations"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
	"github.com/jackc/pgx/v5"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func exerciseDashboard(t *testing.T, ctx context.Context, store *postgres.Store, owner *pgx.Conn) {
	check := func(got, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Fatalf("want %v got %v", want, got)
		}
	}
	exec := func(sql string, args ...any) { t.Helper(); _, err := owner.Exec(ctx, sql, args...); check(err, nil) }
	session, err := auth.New(store.Pool, 12*time.Hour, 3*time.Second)
	check(err, nil)
	os := organizations.New(store.Pool, 3*time.Second)
	ps := projects.New(store.Pool, 3*time.Second)
	fs := flags.New(store.Pool, 3*time.Second)
	as := audit.New(store.Pool, 3*time.Second)
	actor, outside := identifiers.String(identifiers.New()), identifiers.String(identifiers.New())
	password := "dashboard-fixture-password-123"
	hash, err := auth.HashPassword(password)
	check(err, nil)
	exec("INSERT INTO users(id,email,password_hash) VALUES($1,'dashboard@example.invalid',$3),($2,'dashboard-outside@example.invalid',$3)", actor, outside, hash)
	org, err := os.Create(ctx, actor, "Dashboard", "dashboard")
	check(err, nil)
	foreign, err := os.Create(ctx, outside, "Outside dashboard", "outside-dashboard")
	check(err, nil)
	project, err := ps.Create(ctx, actor, org.ID, "Dashboard project", "dashboard-project")
	check(err, nil)
	for _, key := range []string{"new_checkout", "new_search", "new_docs", "other_flag"} {
		flag, err := fs.Create(ctx, actor, project.ID, key, "")
		check(err, nil)
		if key == "new_docs" {
			archive := true
			_, err = fs.Change(ctx, actor, project.ID, key, flags.FlagETag(flag), nil, &archive, true)
			check(err, nil)
		}
	}
	first, err := fs.List(ctx, actor, project.ID, pagination.PageRequest{Limit: 1}, flags.Filter{Query: " NEW_ ", Status: "active"})
	check(err, nil)
	if len(first.Items) != 1 || first.NextCursor == nil {
		t.Fatal("busca/paginação ausente")
	}
	second, err := fs.List(ctx, actor, project.ID, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor}, flags.Filter{Query: "new_", Status: "active"})
	check(err, nil)
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID || second.NextCursor != nil {
		t.Fatal("filtro depois da paginação ou item repetido")
	}
	_, err = fs.List(ctx, actor, project.ID, pagination.PageRequest{Limit: 1, Cursor: *first.NextCursor}, flags.Filter{Query: "new_", Status: "archived"})
	check(err, flags.ErrValidation)
	archived, err := fs.List(ctx, actor, project.ID, pagination.PageRequest{Limit: 25}, flags.Filter{Status: "archived"})
	check(err, nil)
	if len(archived.Items) != 1 || archived.Items[0].Key != "new_docs" {
		t.Fatal("estado arquivado incorreto")
	}
	_, err = fs.List(ctx, outside, project.ID, pagination.PageRequest{Limit: 25})
	check(err, flags.ErrNotFound)
	// Mesmo timestamp: desempate UUID decrescente e nenhum evento duplicado.
	for range 5 {
		exec("INSERT INTO audit_events(id,organization_id,actor_label,action,resource_type,resource_id,after_redacted,created_at) VALUES($1,$2,'local-operator','test.event','flag',$3,$4,'2001-01-01T00:00:00Z')", identifiers.New(), org.ID, identifiers.New(), `{"key":"checkout","secret":"forbidden-audit-value","salt":"forbidden-audit-value","description":{"token":"forbidden-audit-value"}}`)
	}
	filter := audit.Filter{Action: "test.event", ResourceType: "flag"}
	seen := map[string]bool{}
	cursor := ""
	last := ""
	total := 0
	for {
		page, err := as.List(ctx, actor, org.ID, pagination.PageRequest{Limit: 2, Cursor: cursor}, filter)
		check(err, nil)
		for _, item := range page.Items {
			if seen[item.ID] || (last != "" && item.ID >= last) {
				t.Fatal("ordem/página incorreta")
			}
			last = item.ID
			seen[item.ID] = true
			total++
			data, _ := json.Marshal(item)
			if strings.Contains(string(data), "forbidden") {
				t.Fatal("JSON sensível exposto")
			}
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if total != 5 {
		t.Fatal("histórico incompleto")
	}
	_, err = as.List(ctx, actor, org.ID, pagination.PageRequest{Limit: 2, Cursor: cursor}, audit.Filter{})
	check(err, audit.ErrValidation)
	_, err = as.List(ctx, outside, org.ID, pagination.PageRequest{Limit: 2}, filter)
	check(err, audit.ErrNotFound)
	_, err = as.List(ctx, actor, foreign.ID, pagination.PageRequest{Limit: 2}, filter)
	check(err, audit.ErrNotFound)
	_, err = as.List(ctx, actor, org.ID, pagination.PageRequest{Limit: 2}, audit.Filter{Action: "bad action"})
	check(err, audit.ErrValidation)
	var viewer string
	for i, role := range []string{"Owner", "Admin", "Editor", "Viewer"} {
		user, err := session.CreateUser(ctx, "dashboard-"+strings.ToLower(role)+"@example.invalid", password, org.ID)
		check(err, nil)
		_, err = os.Change(ctx, actor, org.ID, user, role)
		check(err, nil)
		page, err := as.List(ctx, user, org.ID, pagination.PageRequest{Limit: 1}, filter)
		check(err, nil)
		if len(page.Items) != 1 {
			t.Fatal("papel não consultou histórico")
		}
		_, err = as.List(ctx, user, org.ID, pagination.PageRequest{Limit: 2, Cursor: cursor}, filter)
		check(err, audit.ErrValidation)
		if i == 3 {
			viewer = user
		}
	}
	login, err := session.Login(ctx, "dashboard-viewer@example.invalid", password, "")
	check(err, nil)
	handler := httpapi.New(slog.New(slog.NewJSONHandler(io.Discard, nil)), store, httpapi.WithAudit(session, as, nil, false), httpapi.WithFlags(session, fs, nil, false))
	call := func(method, path, cookie string, want int) {
		t.Helper()
		r := httptest.NewRequest(method, path, nil)
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: "flagpilot_session", Value: cookie})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s want %d got %d", path, want, w.Code)
		}
	}
	path := "/api/v1/organizations/" + org.ID + "/audit-events"
	call("GET", path, "", 401)
	call("GET", path, login.Token, 200)
	call("POST", path, login.Token, 405)
	call("GET", "/api/v1/organizations/"+foreign.ID+"/audit-events", login.Token, 404)
	for _, query := range []string{"?limit=101", "?action=bad%20action", "?action=a&action=b", "?resourceType=a&resourceType=b", "?cursor=invalid"} {
		call("GET", path+query, login.Token, 422)
	}
	for _, query := range []string{"?status=enabled", "?q=%25", "?q=a&q=b", "?status=all&status=active"} {
		call("GET", "/api/v1/projects/"+project.ID+"/flags"+query, login.Token, 422)
	}
	// Um leitor esperando remoção deve revalidar a associação após o lock.
	tx, err := owner.Begin(ctx)
	check(err, nil)
	_, err = tx.Exec(ctx, "SELECT id FROM organizations WHERE id=$1 FOR NO KEY UPDATE", org.ID)
	check(err, nil)
	_, err = tx.Exec(ctx, "DELETE FROM memberships WHERE organization_id=$1 AND user_id=$2", org.ID, viewer)
	check(err, nil)
	done := make(chan error, 1)
	go func() { _, err := as.List(ctx, viewer, org.ID, pagination.PageRequest{Limit: 2}, filter); done <- err }()
	select {
	case <-done:
		t.Fatal("leitura não respeitou lock organizacional")
	case <-time.After(50 * time.Millisecond):
	}
	check(tx.Commit(ctx), nil)
	check(<-done, audit.ErrNotFound)
	call("GET", path, login.Token, 404)
	exec("UPDATE users SET disabled_at=now() WHERE id=$1", viewer)
	call("GET", path, login.Token, 401)
}
