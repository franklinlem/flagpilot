package flags

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/flagstate"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound     = projects.ErrNotFound
	ErrForbidden    = projects.ErrForbidden
	ErrValidation   = pagination.ErrValidation
	ErrUnavailable  = projects.ErrUnavailable
	ErrConflict     = errors.New("chave já utilizada ou flag arquivada")
	ErrPrecondition = errors.New("os dados foram alterados; recarregue antes de salvar")
	ErrRequired     = errors.New("revisão obrigatória; consulte o recurso antes de salvar")
	ErrProduction   = errors.New("confirme explicitamente o impacto em produção")
)

type Flag struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectId"`
	Key         string     `json:"key"`
	Description string     `json:"description"`
	ArchivedAt  *time.Time `json:"archivedAt"`
	Version     int64      `json:"version"`
}
type Config struct {
	FlagID        string `json:"flagId"`
	FlagKey       string `json:"flagKey"`
	EnvironmentID string `json:"environmentId"`
	Enabled       bool   `json:"enabled"`
	RolloutBps    int32  `json:"rolloutBps"`
	Version       int64  `json:"version"`
	Revision      int64  `json:"revision"`
}
type Snapshot struct {
	Items       []Config             `json:"items"`
	NextCursor  *string              `json:"nextCursor"`
	Environment projects.Environment `json:"environment"`
	ProjectName string               `json:"projectName"`
	Role        string               `json:"role"`
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

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,79}$`)

func validDescription(value string) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= 2000 && !strings.ContainsRune(value, 0)
}
func parse(actor, id string) (pgtype.UUID, pgtype.UUID, error) {
	user, err := identifiers.Parse(actor)
	if err != nil {
		return user, pgtype.UUID{}, ErrValidation
	}
	target, err := identifiers.Parse(id)
	if err != nil {
		return user, target, ErrValidation
	}
	return user, target, nil
}
func toFlag(row q.Flag) Flag {
	var archived *time.Time
	if row.ArchivedAt.Valid {
		value := row.ArchivedAt.Time
		archived = &value
	}
	return Flag{identifiers.String(row.ID), identifiers.String(row.ProjectID), row.Key, row.Description, archived, row.Version}
}
func FlagETag(flag Flag) string { return fmt.Sprintf(`"flag:%s:%d"`, flag.ID, flag.Version) }
func EnvironmentETag(id string, revision int64, role string) string {
	return fmt.Sprintf(`"env:%s:%d:%s"`, id, revision, role)
}
func precondition(header, expected string) error {
	if header == "" {
		return ErrRequired
	}
	if len(header) > 256 {
		return ErrValidation
	}
	if header != expected {
		return ErrPrecondition
	}
	return nil
}
func audit(ctx context.Context, qt *q.Queries, org, user, id pgtype.UUID, action, kind string, before, after any) error {
	var b, a []byte
	var err error
	if before != nil {
		b, err = json.Marshal(before)
		if err != nil {
			return ErrUnavailable
		}
	}
	if after != nil {
		a, err = json.Marshal(after)
		if err != nil {
			return ErrUnavailable
		}
	}
	if err = qt.InsertAuditEvent(ctx, q.InsertAuditEventParams{ID: identifiers.New(), OrganizationID: org, ActorID: user, ActorLabel: "user", Action: action, ResourceType: kind, ResourceID: id, BeforeRedacted: b, AfterRedacted: a}); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (s *Service) Create(ctx context.Context, actor, project, key, description string) (Flag, error) {
	var out Flag
	user, id, err := parse(actor, project)
	if err != nil {
		return out, err
	}
	if !keyPattern.MatchString(key) || !validDescription(description) {
		return out, ErrValidation
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		p, _, err := projects.Authorize(ctx, qt, user, id, auth.ManageFlags, true)
		if err != nil {
			return err
		}
		envs, err := qt.LockFlagEnvironments(ctx, id)
		if err != nil {
			return ErrUnavailable
		}
		row, err := qt.InsertFlag(ctx, q.InsertFlagParams{ID: identifiers.New(), ProjectID: id, Key: key, Description: description})
		if err != nil {
			return storageError(err)
		}
		out = toFlag(row)
		for _, env := range envs {
			if err = flagstate.Insert(ctx, qt, id, row.ID, env.ID); err != nil {
				return storageError(err)
			}
			if _, err = qt.BumpEnvironment(ctx, env.ID); err != nil {
				return ErrUnavailable
			}
		}
		return audit(ctx, qt, p.OrganizationID, user, row.ID, "flag.created", "flag", nil, out)
	})
	return out, err
}
func (s *Service) Get(ctx context.Context, actor, project, key string) (Flag, error) {
	var out Flag
	user, id, err := parse(actor, project)
	if err != nil {
		return out, err
	}
	if !keyPattern.MatchString(key) {
		return out, ErrValidation
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		if _, _, err := projects.Authorize(ctx, qt, user, id, auth.ReadFlags, false); err != nil {
			return err
		}
		row, err := qt.FlagByKey(ctx, q.FlagByKeyParams{ProjectID: id, Key: key})
		if err != nil {
			return storageError(err)
		}
		out = toFlag(row)
		return nil
	})
	return out, err
}
func (s *Service) List(ctx context.Context, actor, project string, page pagination.PageRequest) (pagination.Page[Flag], error) {
	out := pagination.Page[Flag]{Items: []Flag{}}
	user, id, err := parse(actor, project)
	if err != nil {
		return out, err
	}
	scope := identifiers.String(id) + ":" + identifiers.String(user)
	after, err := page.After("flags", scope)
	if err != nil {
		return out, err
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		if _, _, err := projects.Authorize(ctx, qt, user, id, auth.ReadFlags, false); err != nil {
			return err
		}
		rows, err := qt.ListFlags(ctx, q.ListFlagsParams{ProjectID: id, AfterID: after, PageLimit: int32(page.Limit + 1)})
		if err != nil {
			return ErrUnavailable
		}
		if len(rows) > page.Limit {
			out.NextCursor = pagination.NextCursor("flags", scope, rows[page.Limit-1].ID)
			rows = rows[:page.Limit]
		}
		for _, row := range rows {
			out.Items = append(out.Items, toFlag(row))
		}
		return nil
	})
	return out, err
}
func (s *Service) Change(ctx context.Context, actor, project, key, match string, description *string, archive *bool, confirmProduction bool) (Flag, error) {
	var out Flag
	user, id, err := parse(actor, project)
	if err != nil {
		return out, err
	}
	if !keyPattern.MatchString(key) || (description == nil && archive == nil) || (description != nil && !validDescription(*description)) || (archive != nil && !*archive) {
		return out, ErrValidation
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		p, _, err := projects.Authorize(ctx, qt, user, id, auth.ManageFlags, true)
		if err != nil {
			return err
		}
		// Arquivar muda o snapshot de todos os ambientes; a ordem UUID evita ciclos.
		var envs []q.Environment
		if archive != nil {
			envs, err = qt.LockFlagEnvironments(ctx, id)
			if err != nil {
				return ErrUnavailable
			}
		}
		row, err := qt.FlagByKey(ctx, q.FlagByKeyParams{ProjectID: id, Key: key})
		if err != nil {
			return storageError(err)
		}
		before := toFlag(row)
		if err = precondition(match, FlagETag(before)); err != nil {
			return err
		}
		if row.ArchivedAt.Valid {
			return ErrConflict
		}
		if archive != nil {
			for _, env := range envs {
				if env.IsProduction && !confirmProduction {
					return ErrProduction
				}
			}
		}
		text := row.Description
		if description != nil {
			text = *description
		}
		row, err = qt.ChangeFlag(ctx, q.ChangeFlagParams{ProjectID: id, Key: key, Description: text, Column4: archive != nil})
		if err != nil {
			return storageError(err)
		}
		for _, env := range envs {
			if _, err = qt.BumpEnvironment(ctx, env.ID); err != nil {
				return ErrUnavailable
			}
		}
		out = toFlag(row)
		action := "flag.updated"
		if archive != nil {
			action = "flag.archived"
		}
		return audit(ctx, qt, p.OrganizationID, user, row.ID, action, "flag", before, out)
	})
	return out, err
}
