package tokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrValidation   = pagination.ErrValidation
	ErrNotFound     = projects.ErrNotFound
	ErrForbidden    = projects.ErrForbidden
	ErrUnavailable  = projects.ErrUnavailable
	ErrUnauthorized = errors.New("token de avaliação inválido ou expirado")
)

type Token struct {
	ID            string     `json:"id"`
	EnvironmentID string     `json:"environmentId"`
	Name          string     `json:"name"`
	Prefix        string     `json:"prefix"`
	ExpiresAt     time.Time  `json:"expiresAt"`
	RevokedAt     *time.Time `json:"revokedAt"`
}
type Issued struct {
	Token  Token  `json:"token"`
	Secret string `json:"secret"`
}
type Page struct {
	Items       []Token              `json:"items"`
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
func scope(ctx context.Context, qt *q.Queries, user, environment pgtype.UUID) (q.Environment, q.Project, string, error) {
	env, err := qt.ScopedFlagEnvironment(ctx, q.ScopedFlagEnvironmentParams{ID: environment, UserID: user})
	if err != nil {
		return env, q.Project{}, "", storageError(err)
	}
	project, role, err := projects.Authorize(ctx, qt, user, env.ProjectID, auth.ManageTokens, false)
	return env, project, role, err
}
func parse(actor, environment string) (pgtype.UUID, pgtype.UUID, error) {
	user, err := identifiers.Parse(actor)
	if err != nil {
		return user, pgtype.UUID{}, ErrValidation
	}
	env, err := identifiers.Parse(environment)
	if err != nil {
		return user, env, ErrValidation
	}
	return user, env, nil
}
func toToken(row q.ApiToken) Token {
	var revoked *time.Time
	if row.RevokedAt.Valid {
		value := row.RevokedAt.Time
		revoked = &value
	}
	return Token{ID: identifiers.String(row.ID), EnvironmentID: identifiers.String(row.EnvironmentID), Name: row.Name, Prefix: row.TokenPrefix, ExpiresAt: row.ExpiresAt.Time, RevokedAt: revoked}
}
func audit(ctx context.Context, qt *q.Queries, org, user, id pgtype.UUID, action string, before, after any) error {
	var b, a []byte
	if before != nil {
		b, _ = json.Marshal(before)
	}
	if after != nil {
		a, _ = json.Marshal(after)
	}
	if err := qt.InsertAuditEvent(ctx, q.InsertAuditEventParams{ID: identifiers.New(), OrganizationID: org, ActorID: user, ActorLabel: "user", Action: action, ResourceType: "api_token", ResourceID: id, BeforeRedacted: b, AfterRedacted: a}); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (s *Service) Issue(ctx context.Context, actor, environment, name string, expires time.Time) (Issued, error) {
	var out Issued
	user, env, err := parse(actor, environment)
	if err != nil {
		return out, err
	}
	name = strings.TrimSpace(name)
	now := time.Now()
	if !utf8.ValidString(name) || strings.ContainsRune(name, 0) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 200 || !expires.After(now) || expires.After(now.Add(365*24*time.Hour)) {
		return out, ErrValidation
	}
	var raw [32]byte
	if _, err = rand.Read(raw[:]); err != nil {
		return out, ErrUnavailable
	}
	secret := "fp_eval_" + hex.EncodeToString(raw[:])
	digest := sha256.Sum256([]byte(secret))
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		e, project, _, err := scope(ctx, qt, user, env)
		if err != nil {
			return err
		}
		if !expires.After(time.Now()) {
			return ErrValidation
		}
		row, err := qt.InsertEvaluationToken(ctx, q.InsertEvaluationTokenParams{ID: identifiers.New(), ProjectID: e.ProjectID, EnvironmentID: env, Name: name, TokenPrefix: secret[:16], TokenHash: digest[:], ExpiresAt: pgtype.Timestamptz{Time: expires.UTC(), Valid: true}, CreatedBy: user})
		if err != nil {
			return ErrUnavailable
		}
		metadata := toToken(row)
		if err = audit(ctx, qt, project.OrganizationID, user, row.ID, "token.issued", nil, metadata); err != nil {
			return err
		}
		out = Issued{Token: metadata, Secret: secret}
		return nil
	})
	if err != nil {
		return Issued{}, err
	}
	return out, nil
}
func (s *Service) List(ctx context.Context, actor, environment string, page pagination.PageRequest) (Page, error) {
	out := Page{Items: []Token{}}
	user, env, err := parse(actor, environment)
	if err != nil {
		return out, err
	}
	binding := identifiers.String(env) + ":" + identifiers.String(user)
	after, err := page.After("tokens", binding)
	if err != nil {
		return out, err
	}
	err = s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		e, project, role, err := scope(ctx, qt, user, env)
		if err != nil {
			return err
		}
		rows, err := qt.ListEvaluationTokens(ctx, q.ListEvaluationTokensParams{EnvironmentID: env, AfterID: after, PageLimit: int32(page.Limit + 1)})
		if err != nil {
			return ErrUnavailable
		}
		if len(rows) > page.Limit {
			out.NextCursor = pagination.NextCursor("tokens", binding, rows[page.Limit-1].ID)
			rows = rows[:page.Limit]
		}
		for _, row := range rows {
			out.Items = append(out.Items, toToken(row))
		}
		out.Environment = projects.Environment{ID: identifiers.String(env), ProjectID: identifiers.String(e.ProjectID), Name: e.Name, Slug: e.Slug, IsProduction: e.IsProduction, Revision: e.Revision}
		out.ProjectName = project.Name
		out.Role = role
		return nil
	})
	return out, err
}
func (s *Service) Revoke(ctx context.Context, actor, environment, token string) error {
	user, env, err := parse(actor, environment)
	if err != nil {
		return err
	}
	id, err := identifiers.Parse(token)
	if err != nil {
		return ErrValidation
	}
	return s.transaction(ctx, func(ctx context.Context, qt *q.Queries) error {
		_, project, _, err := scope(ctx, qt, user, env)
		if err != nil {
			return err
		}
		row, err := qt.LockManagedToken(ctx, q.LockManagedTokenParams{EnvironmentID: env, ID: id})
		if err != nil {
			return storageError(err)
		}
		if row.RevokedAt.Valid {
			return nil
		}
		before := toToken(row)
		row, err = qt.RevokeEvaluationToken(ctx, id)
		if err != nil {
			return ErrUnavailable
		}
		return audit(ctx, qt, project.OrganizationID, user, id, "token.revoked", before, toToken(row))
	})
}

var secretPattern = regexp.MustCompile(`^fp_eval_[a-f0-9]{64}$`)

func ValidSecret(secret string) bool { return secretPattern.MatchString(secret) }

// Authorize exige a mesma transação da avaliação. SHARE mantém a autorização
// estável durante o snapshot; revogação aguardará somente operações já em voo.
func Authorize(ctx context.Context, qt *q.Queries, secret string) (q.ApiToken, error) {
	if !ValidSecret(secret) {
		return q.ApiToken{}, ErrUnauthorized
	}
	hash := sha256.Sum256([]byte(secret))
	row, err := qt.LockEvaluationToken(ctx, hash[:])
	if errors.Is(err, pgx.ErrNoRows) {
		return row, ErrUnauthorized
	}
	if err != nil {
		return row, ErrUnavailable
	}
	if _, err = qt.EvaluationTokenActive(ctx, row.ID); errors.Is(err, pgx.ErrNoRows) {
		return row, ErrUnauthorized
	} else if err != nil {
		return row, ErrUnavailable
	}
	return row, nil
}
