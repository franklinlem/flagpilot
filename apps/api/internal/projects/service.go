package projects

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/flagstate"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound    = errors.New("recurso não encontrado")
	ErrForbidden   = errors.New("operação não permitida para seu papel")
	ErrValidation  = pagination.ErrValidation
	ErrConflict    = errors.New("slug já utilizado neste escopo")
	ErrUnavailable = errors.New("serviço indisponível")
)

type Project struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Role           string `json:"role"`
}
type Environment struct {
	ID           string `json:"id"`
	ProjectID    string `json:"projectId"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	IsProduction bool   `json:"isProduction"`
	Revision     int64  `json:"revision"`
}
type Service struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func New(pool *pgxpool.Pool, timeout time.Duration) *Service {
	return &Service{pool: pool, timeout: timeout}
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

func validName(name, slug string) bool {
	return utf8.ValidString(name) && utf8.RuneCountInString(name) >= 1 && utf8.RuneCountInString(name) <= 200 && slugPattern.MatchString(slug)
}
func toProject(row q.Project, role string) Project {
	return Project{ID: identifiers.String(row.ID), OrganizationID: identifiers.String(row.OrganizationID), Name: row.Name, Slug: row.Slug, Role: role}
}
func toEnvironment(row q.Environment) Environment {
	return Environment{ID: identifiers.String(row.ID), ProjectID: identifiers.String(row.ProjectID), Name: row.Name, Slug: row.Slug, IsProduction: row.IsProduction, Revision: row.Revision}
}

// O lock do escopo precede a releitura do papel; sessão não é cache de autorização.
func scope(ctx context.Context, qt *q.Queries, org, user pgtype.UUID, permission auth.Permission) (string, error) {
	if _, err := qt.ProjectScopeOrganization(ctx, q.ProjectScopeOrganizationParams{ID: org, UserID: user}); err != nil {
		return "", storageError(err)
	}
	role, err := qt.ActorOrganizationRole(ctx, q.ActorOrganizationRoleParams{OrganizationID: org, UserID: user})
	if err != nil {
		return "", storageError(err)
	}
	if !auth.Can(role, permission) {
		return "", ErrForbidden
	}
	return role, nil
}

// Toda operação usa timeout e transação, mantendo a associação estável no escopo.
func (s *Service) transaction(ctx context.Context, run func(context.Context, *q.Queries) error) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err = run(ctx, q.New(tx)); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return ErrUnavailable
	}
	return nil
}
func audit(ctx context.Context, qt *q.Queries, org, user, id pgtype.UUID, kind string, value any) error {
	after, err := json.Marshal(value)
	if err != nil {
		return ErrUnavailable
	}
	err = qt.InsertAuditEvent(ctx, q.InsertAuditEventParams{ID: identifiers.New(), OrganizationID: org, ActorID: user, ActorLabel: "user", Action: kind + ".created", ResourceType: kind, ResourceID: id, AfterRedacted: after})
	if err != nil {
		return ErrUnavailable
	}
	return nil
}
func insertEnvironment(ctx context.Context, qt *q.Queries, org, user, project pgtype.UUID, name, slug string, production bool) (Environment, error) {
	row, err := qt.InsertEnvironment(ctx, q.InsertEnvironmentParams{ID: identifiers.New(), ProjectID: project, Name: name, Slug: slug, IsProduction: production})
	if err != nil {
		return Environment{}, storageError(err)
	}
	if err = flagstate.Initialize(ctx, qt, project, row.ID); err != nil {
		return Environment{}, ErrUnavailable
	}
	out := toEnvironment(row)
	if err = audit(ctx, qt, org, user, row.ID, "environment", out); err != nil {
		return Environment{}, err
	}
	return out, nil
}
func (s *Service) Create(ctx context.Context, actor, organization, name, slug string) (Project, error) {
	var out Project
	user, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	org, err := identifiers.Parse(organization)
	if err != nil {
		return out, ErrValidation
	}
	name = strings.TrimSpace(name)
	if !validName(name, slug) {
		return out, ErrValidation
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		role, err := scope(ctx, qt, org, user, auth.ManageProjects)
		if err != nil {
			return err
		}
		row, err := qt.InsertProject(ctx, q.InsertProjectParams{ID: identifiers.New(), OrganizationID: org, Name: name, Slug: slug})
		if err != nil {
			return storageError(err)
		}
		out = toProject(row, role)
		if err = audit(ctx, qt, org, user, row.ID, "project", map[string]string{"name": name, "slug": slug}); err != nil {
			return err
		}
		for _, preset := range []struct {
			name, slug string
			production bool
		}{{"Desenvolvimento", "development", false}, {"Homologação", "staging", false}, {"Produção", "production", true}} {
			if _, err = insertEnvironment(ctx, qt, org, user, row.ID, preset.name, preset.slug, preset.production); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

// Authorize exige Queries vinculadas à transação completa da operação.
// O lock de organização protege a autorização; lock=true serializa o catálogo no projeto.
func Authorize(ctx context.Context, qt *q.Queries, user, id pgtype.UUID, permission auth.Permission, lock bool) (q.Project, string, error) {
	row, err := qt.ScopedProject(ctx, q.ScopedProjectParams{ID: id, UserID: user})
	if err != nil {
		return row, "", storageError(err)
	}
	role, err := scope(ctx, qt, row.OrganizationID, user, permission)
	if err != nil {
		return row, "", err
	}
	// Preparar ordem organização → projeto para futuras criações de flags/configurações.
	if lock {
		if _, err = qt.LockProject(ctx, id); err != nil {
			return row, "", storageError(err)
		}
	}
	return row, role, nil
}
func (s *Service) Get(ctx context.Context, actor, project string) (Project, error) {
	var out Project
	user, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	id, err := identifiers.Parse(project)
	if err != nil {
		return out, ErrValidation
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		row, role, err := Authorize(ctx, qt, user, id, auth.ReadProjects, false)
		if err != nil {
			return err
		}
		out = toProject(row, role)
		return nil
	})
	return out, err
}
func (s *Service) List(ctx context.Context, actor, organization string, page pagination.PageRequest) (pagination.Page[Project], error) {
	out := pagination.Page[Project]{Items: []Project{}}
	user, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	org, err := identifiers.Parse(organization)
	if err != nil {
		return out, ErrValidation
	}
	cursorScope := identifiers.String(org) + ":" + identifiers.String(user)
	after, err := page.After("projects", cursorScope)
	if err != nil {
		return out, err
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		role, err := scope(ctx, qt, org, user, auth.ReadProjects)
		if err != nil {
			return err
		}
		rows, err := qt.ListProjects(ctx, q.ListProjectsParams{OrganizationID: org, AfterID: after, PageLimit: int32(page.Limit + 1)})
		if err != nil {
			return ErrUnavailable
		}
		if len(rows) > page.Limit {
			out.NextCursor = pagination.NextCursor("projects", cursorScope, rows[page.Limit-1].ID)
			rows = rows[:page.Limit]
		}
		for _, row := range rows {
			out.Items = append(out.Items, toProject(row, role))
		}
		return nil
	})
	return out, err
}
func (s *Service) Environments(ctx context.Context, actor, project string, page pagination.PageRequest) (pagination.Page[Environment], error) {
	out := pagination.Page[Environment]{Items: []Environment{}}
	user, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	id, err := identifiers.Parse(project)
	if err != nil {
		return out, ErrValidation
	}
	cursorScope := identifiers.String(id) + ":" + identifiers.String(user)
	after, err := page.After("environments", cursorScope)
	if err != nil {
		return out, err
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		if _, _, err := Authorize(ctx, qt, user, id, auth.ReadProjects, false); err != nil {
			return err
		}
		rows, err := qt.ListEnvironments(ctx, q.ListEnvironmentsParams{ProjectID: id, AfterID: after, PageLimit: int32(page.Limit + 1)})
		if err != nil {
			return ErrUnavailable
		}
		if len(rows) > page.Limit {
			out.NextCursor = pagination.NextCursor("environments", cursorScope, rows[page.Limit-1].ID)
			rows = rows[:page.Limit]
		}
		for _, row := range rows {
			out.Items = append(out.Items, toEnvironment(row))
		}
		return nil
	})
	return out, err
}
func (s *Service) CreateEnvironment(ctx context.Context, actor, project, name, slug string, production bool) (Environment, error) {
	var out Environment
	user, err := identifiers.Parse(actor)
	if err != nil {
		return out, ErrValidation
	}
	id, err := identifiers.Parse(project)
	if err != nil {
		return out, ErrValidation
	}
	name = strings.TrimSpace(name)
	if !validName(name, slug) || (slug == "production" && !production) {
		return out, ErrValidation
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		row, _, err := Authorize(ctx, qt, user, id, auth.ManageProjects, true)
		if err != nil {
			return err
		}
		out, err = insertEnvironment(ctx, qt, row.OrganizationID, user, id, name, slug, production)
		return err
	})
	return out, err
}
