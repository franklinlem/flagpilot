package evaluation

import (
	"context"
	"github.com/franklinlem/flagpilot/apps/api/internal/pagination"
	"github.com/franklinlem/flagpilot/apps/api/internal/projects"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/franklinlem/flagpilot/apps/api/internal/tokens"
	"github.com/jackc/pgx/v5/pgxpool"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrValidation = pagination.ErrValidation
var ErrUnavailable = projects.ErrUnavailable
var ErrUnauthorized = tokens.ErrUnauthorized

// Context é usado somente em memória. Bucketing será implementado em 4B.
type Context struct {
	BucketingKey *string `json:"bucketingKey"`
}
type Request struct {
	FlagKey string   `json:"flagKey"`
	Context *Context `json:"context,omitempty"`
}
type Result struct {
	FlagKey     string `json:"flagKey"`
	Value       bool   `json:"value"`
	Reason      string `json:"reason"`
	Environment string `json:"environment"`
	Revision    int64  `json:"revision"`
}
type Batch struct {
	Results  []Result `json:"results"`
	Revision int64    `json:"revision"`
}
type Service struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func New(pool *pgxpool.Pool, timeout time.Duration) *Service {
	return &Service{pool: pool, timeout: timeout}
}

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,79}$`)

func valid(request Request) bool {
	if !keyPattern.MatchString(request.FlagKey) {
		return false
	}
	if request.Context != nil && request.Context.BucketingKey != nil {
		key := *request.Context.BucketingKey
		return utf8.ValidString(key) && len(key) >= 1 && len(key) <= 256 && !strings.ContainsRune(key, 0)
	}
	return true
}

// Decide é determinístico e falha fechado para percentuais ainda não suportados.
func Decide(found, enabled bool, rollout int32) (bool, string, error) {
	if !found {
		return false, "NOT_FOUND", nil
	}
	if !enabled || rollout == 0 {
		return false, "DISABLED", nil
	}
	if rollout == 10000 {
		return true, "ENABLED", nil
	}
	return false, "", ErrUnavailable
}
func (s *Service) Evaluate(ctx context.Context, secret string, requests []Request) (Batch, error) {
	out := Batch{Results: []Result{}}
	if len(requests) < 1 || len(requests) > 100 {
		return out, ErrValidation
	}
	keys := make([]string, len(requests))
	for i, request := range requests {
		if !valid(request) {
			return out, ErrValidation
		}
		keys[i] = request.FlagKey
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := q.New(tx)
	credential, err := tokens.Authorize(ctx, qt, secret)
	if err != nil {
		return out, err
	}
	env, err := qt.ReadFlagEnvironment(ctx, credential.EnvironmentID)
	if err != nil {
		return out, ErrUnavailable
	}
	rows, err := qt.EvaluationConfigs(ctx, q.EvaluationConfigsParams{EnvironmentID: env.ID, Column2: keys})
	if err != nil {
		return out, ErrUnavailable
	}
	configs := make(map[string]q.EvaluationConfigsRow, len(rows))
	for _, row := range rows {
		configs[row.Key] = row
	}
	for _, request := range requests {
		row, found := configs[request.FlagKey]
		value, reason, err := Decide(found, row.Enabled, row.RolloutBps)
		if err != nil {
			return Batch{}, err
		}
		out.Results = append(out.Results, Result{FlagKey: request.FlagKey, Value: value, Reason: reason, Environment: env.Slug, Revision: env.Revision})
	}
	out.Revision = env.Revision
	if err = tx.Commit(ctx); err != nil {
		return Batch{}, ErrUnavailable
	}
	return out, nil
}
