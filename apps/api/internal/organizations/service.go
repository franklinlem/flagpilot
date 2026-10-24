package organizations

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound    = errors.New("recurso não encontrado")
	ErrForbidden   = errors.New("operação não permitida para seu papel")
	ErrValidation  = errors.New("dados ou paginação inválidos")
	ErrConflict    = errors.New("recurso já existe ou slug indisponível")
	ErrLastOwner   = errors.New("a organização precisa manter ao menos um Proprietário ativo")
	ErrUnavailable = errors.New("serviço indisponível")
)

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role"`
}
type Member struct {
	OrganizationID string `json:"organizationId"`
	UserID         string `json:"userId"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	Disabled       bool   `json:"disabled"`
}
type Service struct {
	pool    *pgxpool.Pool
	query   *q.Queries
	timeout time.Duration
}

func New(pool *pgxpool.Pool, timeout time.Duration) *Service {
	return &Service{pool: pool, query: q.New(pool), timeout: timeout}
}
func storageError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrConflict
	}
	return ErrUnavailable
}

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

func (s *Service) Create(ctx context.Context, actor, name, slug string) (Organization, error) {
	id, err := identifiers.Parse(actor)
	name = strings.TrimSpace(name)
	if err != nil || !utf8.ValidString(name) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 200 || !slugPattern.MatchString(slug) {
		return Organization{}, ErrValidation
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Organization{}, ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := s.query.WithTx(tx)
	if _, err = qt.ActiveUser(ctx, id); err != nil {
		return Organization{}, storageError(err)
	}
	org, err := qt.InsertOrganization(ctx, q.InsertOrganizationParams{ID: identifiers.New(), Name: name, Slug: slug})
	if err != nil {
		return Organization{}, storageError(err)
	}
	if err = qt.InsertOwner(ctx, q.InsertOwnerParams{OrganizationID: org.ID, UserID: id}); err != nil {
		return Organization{}, storageError(err)
	}
	after, _ := json.Marshal(map[string]string{"name": name, "slug": slug, "role": "Owner"})
	if err = qt.InsertAuditEvent(ctx, q.InsertAuditEventParams{ID: identifiers.New(), OrganizationID: org.ID, ActorID: id, ActorLabel: "user", Action: "organization.created", ResourceType: "organization", ResourceID: org.ID, AfterRedacted: after}); err != nil {
		return Organization{}, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return Organization{}, ErrUnavailable
	}
	return Organization{ID: identifiers.String(org.ID), Name: org.Name, Slug: org.Slug, Role: "Owner"}, nil
}
func (s *Service) List(ctx context.Context, actor string, page PageRequest) (Page[Organization], error) {
	out := Page[Organization]{Items: []Organization{}}
	actorID, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	scope := identifiers.String(actorID)
	after, err := page.after("organizations", scope)
	if err != nil {
		return out, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	rows, err := s.query.ListActorOrganizations(ctx, q.ListActorOrganizationsParams{ActorID: actorID, AfterID: after, PageLimit: int32(page.Limit + 1)})
	if err != nil {
		return out, ErrUnavailable
	}
	if len(rows) > page.Limit {
		out.NextCursor = nextCursor("organizations", scope, rows[page.Limit-1].ID)
		rows = rows[:page.Limit]
	}
	for _, row := range rows {
		out.Items = append(out.Items, Organization{ID: identifiers.String(row.ID), Name: row.Name, Slug: row.Slug, Role: row.Role})
	}
	return out, nil
}

// O lock precede a releitura da associação: uma consulta que aguardou outra
// transação nunca usa o papel calculado no snapshot anterior à espera.
func (s *Service) scope(ctx context.Context, qt *q.Queries, org, actor pgtype.UUID, write bool) (q.Organization, string, error) {
	var row q.Organization
	var err error
	if write {
		row, err = qt.WriteScopedOrganization(ctx, q.WriteScopedOrganizationParams{ID: org, UserID: actor})
	} else {
		row, err = qt.ReadScopedOrganization(ctx, q.ReadScopedOrganizationParams{ID: org, UserID: actor})
	}
	if err != nil {
		return row, "", storageError(err)
	}
	role, err := qt.ActorOrganizationRole(ctx, q.ActorOrganizationRoleParams{OrganizationID: org, UserID: actor})
	if err != nil {
		return row, "", storageError(err)
	}
	permission := auth.ReadMembers
	if write {
		permission = auth.ManageMembers
	}
	if !auth.Can(role, permission) {
		return row, "", ErrForbidden
	}
	return row, role, nil
}
func (s *Service) Get(ctx context.Context, actor, organization string) (Organization, error) {
	var out Organization
	user, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	org, err := identifiers.Parse(organization)
	if err != nil {
		return out, ErrValidation
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	row, role, err := s.scope(ctx, s.query.WithTx(tx), org, user, false)
	if err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, ErrUnavailable
	}
	return Organization{ID: identifiers.String(row.ID), Name: row.Name, Slug: row.Slug, Role: role}, nil
}
func (s *Service) Members(ctx context.Context, actor, organization string, page PageRequest) (Page[Member], error) {
	out := Page[Member]{Items: []Member{}}
	actorID, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	org, err := identifiers.Parse(organization)
	if err != nil {
		return out, ErrValidation
	}
	scope := identifiers.String(org) + ":" + identifiers.String(actorID)
	after, err := page.after("members", scope)
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
	qt := s.query.WithTx(tx)
	if _, _, err = s.scope(ctx, qt, org, actorID, false); err != nil {
		return out, err
	}
	rows, err := qt.ListOrganizationMembers(ctx, q.ListOrganizationMembersParams{OrganizationID: org, AfterID: after, PageLimit: int32(page.Limit + 1)})
	if err != nil {
		return out, ErrUnavailable
	}
	if len(rows) > page.Limit {
		out.NextCursor = nextCursor("members", scope, rows[page.Limit-1].UserID)
		rows = rows[:page.Limit]
	}
	for _, row := range rows {
		out.Items = append(out.Items, Member{OrganizationID: identifiers.String(org), UserID: identifiers.String(row.UserID), Email: row.Email, Role: row.Role, Disabled: row.Disabled})
	}
	if err = tx.Commit(ctx); err != nil {
		return out, ErrUnavailable
	}
	return out, nil
}

// Add/Change/Remove compartilham a mesma fronteira transacional e autorização.
func (s *Service) Add(ctx context.Context, actor, org, user, role string) (Member, error) {
	return s.mutate(ctx, actor, org, user, role, "member.added")
}
func (s *Service) Change(ctx context.Context, actor, org, user, role string) (Member, error) {
	return s.mutate(ctx, actor, org, user, role, "member.role_changed")
}
func (s *Service) Remove(ctx context.Context, actor, org, user string) error {
	_, err := s.mutate(ctx, actor, org, user, "", "member.removed")
	return err
}
func (s *Service) mutate(ctx context.Context, actor, organization, user, role, action string) (Member, error) {
	var out Member
	actorID, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	org, err := identifiers.Parse(organization)
	if err != nil {
		return out, ErrValidation
	}
	userID, err := identifiers.Parse(user)
	if err != nil {
		return out, ErrValidation
	}
	if action != "member.removed" && !auth.ValidRole(role) {
		return out, ErrValidation
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := s.query.WithTx(tx)
	if _, _, err = s.scope(ctx, qt, org, actorID, true); err != nil {
		return out, err
	}
	var before []byte
	if action == "member.added" {
		target, err := qt.ActiveUser(ctx, userID)
		if err != nil {
			return out, storageError(err)
		}
		out = Member{OrganizationID: identifiers.String(org), UserID: identifiers.String(userID), Email: target.Email, Role: role}
		err = qt.AddOrganizationMember(ctx, q.AddOrganizationMemberParams{OrganizationID: org, UserID: userID, Role: role})
		if err != nil {
			return out, storageError(err)
		}
	} else {
		target, err := qt.GetOrganizationMember(ctx, q.GetOrganizationMemberParams{OrganizationID: org, UserID: userID})
		if err != nil {
			return out, storageError(err)
		}
		out = Member{OrganizationID: identifiers.String(org), UserID: identifiers.String(userID), Email: target.Email, Role: role, Disabled: target.Disabled}
		before, _ = json.Marshal(map[string]string{"userId": out.UserID, "role": target.Role})
		if target.Role == "Owner" && !target.Disabled && role != "Owner" {
			count, err := qt.CountActiveOwners(ctx, org)
			if err != nil {
				return out, ErrUnavailable
			}
			if count <= 1 {
				return out, ErrLastOwner
			}
		}
		if action == "member.removed" {
			err = qt.RemoveOrganizationMember(ctx, q.RemoveOrganizationMemberParams{OrganizationID: org, UserID: userID})
		} else {
			err = qt.ChangeMemberRole(ctx, q.ChangeMemberRoleParams{OrganizationID: org, UserID: userID, Role: role})
		}
		if err != nil {
			return out, storageError(err)
		}
	}
	var after []byte
	if action != "member.removed" {
		after, _ = json.Marshal(map[string]string{"userId": out.UserID, "role": role})
	}
	if err = qt.InsertAuditEvent(ctx, q.InsertAuditEventParams{ID: identifiers.New(), OrganizationID: org, ActorID: actorID, ActorLabel: "user", Action: action, ResourceType: "membership", ResourceID: userID, BeforeRedacted: before, AfterRedacted: after}); err != nil {
		return out, ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return out, ErrUnavailable
	}
	return out, nil
}
