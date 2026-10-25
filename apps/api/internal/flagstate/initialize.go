// Package flagstate inicializa configurações sem dependência entre os domínios.
package flagstate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

func Insert(ctx context.Context, qt *q.Queries, project, flag, environment pgtype.UUID) error {
	var salt [32]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return err
	}
	return qt.InsertFlagConfig(ctx, q.InsertFlagConfigParams{ID: identifiers.New(), ProjectID: project, FlagID: flag, EnvironmentID: environment, Salt: hex.EncodeToString(salt[:])})
}

// O chamador mantém o lock do projeto e a transação de criação do ambiente.
func Initialize(ctx context.Context, qt *q.Queries, project, environment pgtype.UUID) error {
	ids, err := qt.ActiveFlagIDs(ctx, project)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = Insert(ctx, qt, project, id, environment); err != nil {
			return err
		}
	}
	return nil
}
