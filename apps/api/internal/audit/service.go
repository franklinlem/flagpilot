// Package audit disponibiliza somente consulta de histórico com projeção explícita de campos.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/organizations"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"time"
)

var ErrValidation = pagination.ErrValidation
var ErrNotFound = organizations.ErrNotFound
var ErrUnavailable = organizations.ErrUnavailable
var ErrForbidden = organizations.ErrForbidden

type Event struct {
	ID             string                     `json:"id"`
	OrganizationID string                     `json:"organizationId"`
	ActorID        *string                    `json:"actorId"`
	ActorLabel     string                     `json:"actorLabel"`
	Action         string                     `json:"action"`
	ResourceType   string                     `json:"resourceType"`
	ResourceID     string                     `json:"resourceId"`
	Before         map[string]json.RawMessage `json:"before"`
	After          map[string]json.RawMessage `json:"after"`
	CreatedAt      time.Time                  `json:"createdAt"`
}
type Filter struct{ Action, ResourceType string }

var filterPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,79}$`)

func (f Filter) Valid() bool {
	return (f.Action == "" || filterPattern.MatchString(f.Action)) && (f.ResourceType == "" || filterPattern.MatchString(f.ResourceType))
}

type Service struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func New(pool *pgxpool.Pool, timeout time.Duration) *Service { return &Service{pool, timeout} }
func storageError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return ErrUnavailable
}
func (s *Service) List(ctx context.Context, actor, organization string, page pagination.PageRequest, filter Filter) (pagination.Page[Event], error) {
	out := pagination.Page[Event]{Items: []Event{}}
	user, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	org, err := identifiers.Parse(organization)
	if err != nil || !filter.Valid() {
		return out, ErrValidation
	}
	digest := sha256.Sum256([]byte(filter.Action + ":" + filter.ResourceType))
	scope := fmt.Sprintf("%s:%s:%x", identifiers.String(org), identifiers.String(user), digest)
	after, err := page.After("audit", scope)
	if err != nil {
		return out, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := q.New(tx)
	if _, err = qt.ReadScopedOrganization(ctx, q.ReadScopedOrganizationParams{ID: org, UserID: user}); err != nil {
		return out, storageError(err)
	}
	role, err := qt.ActorOrganizationRole(ctx, q.ActorOrganizationRoleParams{OrganizationID: org, UserID: user})
	if err != nil {
		return out, storageError(err)
	}
	if !auth.Can(role, auth.ReadAudit) {
		return out, ErrForbidden
	}
	var before pgtype.Timestamptz
	if after.Valid {
		before, err = qt.AuditAnchor(ctx, q.AuditAnchorParams{OrganizationID: org, ID: after})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return out, ErrValidation
			}
			return out, ErrUnavailable
		}
	}
	rows, err := qt.ListAuditEvents(ctx, q.ListAuditEventsParams{OrganizationID: org, ActionFilter: filter.Action, ResourceFilter: filter.ResourceType, BeforeTime: before, BeforeID: after, PageLimit: int32(page.Limit + 1)})
	if err != nil {
		return out, ErrUnavailable
	}
	if len(rows) > page.Limit {
		out.NextCursor = pagination.NextCursor("audit", scope, rows[page.Limit-1].ID)
		rows = rows[:page.Limit]
	}
	for _, row := range rows {
		var actorID *string
		if row.ActorID.Valid {
			value := identifiers.String(row.ActorID)
			actorID = &value
		}
		label := row.ActorLabel
		if label != "user" && label != "local-operator" {
			label = "unknown"
		}
		out.Items = append(out.Items, Event{ID: identifiers.String(row.ID), OrganizationID: identifiers.String(row.OrganizationID), ActorID: actorID, ActorLabel: label, Action: row.Action, ResourceType: row.ResourceType, ResourceID: identifiers.String(row.ResourceID), Before: redact(row.ResourceType, row.BeforeRedacted), After: redact(row.ResourceType, row.AfterRedacted), CreatedAt: row.CreatedAt.Time})
	}
	if tx.Commit(ctx) != nil {
		return pagination.Page[Event]{}, ErrUnavailable
	}
	return out, nil
}

// Não devolver JSON arbitrário de escritores atuais/futuros: somente campos escalares conhecidos.
var fields = map[string][]string{
	"organization": {"id", "name", "slug", "role", "userId"},
	"user":         {"id", "userId", "email", "role", "organizationId"},
	"membership":   {"organizationId", "userId", "email", "role", "disabled"},
	"project":      {"id", "organizationId", "name", "slug"},
	"environment":  {"id", "projectId", "name", "slug", "isProduction", "revision"},
	"flag":         {"id", "projectId", "key", "description", "archivedAt", "version"},
	"flag_config":  {"flagId", "flagKey", "environmentId", "enabled", "rolloutBps", "version", "revision"},
	"api_token":    {"id", "environmentId", "name", "prefix", "expiresAt", "revokedAt"},
}

func redact(kind string, data []byte) map[string]json.RawMessage {
	if len(data) == 0 || len(data) > 32768 || fields[kind] == nil {
		return nil
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil || raw == nil {
		return nil
	}
	result := map[string]json.RawMessage{}
	for _, name := range fields[kind] {
		if value, ok := raw[name]; ok {
			var scalar any
			if json.Unmarshal(value, &scalar) == nil {
				switch scalar.(type) {
				case string, float64, bool, nil:
					result[name] = value
				}
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
