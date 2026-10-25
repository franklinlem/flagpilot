package postgres

import (
	"context"
	"errors"
	"time"

	queries "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

const SchemaVersion int64 = 5

type Store struct {
	Pool    *pgxpool.Pool
	query   *queries.Queries
	timeout time.Duration
}

func New(ctx context.Context, databaseURL string, timeout time.Duration) (*Store, error) {
	c, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("configuração de banco inválida")
	}
	c.MaxConns = 5
	c.MinConns = 0
	c.ConnConfig.ConnectTimeout = timeout
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, errors.New("falha ao criar pool de banco")
	}
	return &Store{Pool: pool, query: queries.New(pool), timeout: timeout}, nil
}

// Ready verifica conectividade e versão, sem revelar erro interno na resposta HTTP.
func (s *Store) Ready(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	schema, err := s.query.SchemaStatus(ctx)
	if err != nil {
		return errors.New("banco indisponível ou schema ausente")
	}
	if schema.Dirty || schema.Version != SchemaVersion {
		return errors.New("schema incompatível")
	}
	return nil
}
func (s *Store) Close() { s.Pool.Close() }
