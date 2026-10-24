package auth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5/pgconn"
)

// CreateUser é operação exclusiva do operador local, não é endpoint HTTP.
// Vincular o novo usuário como Viewer garante uma trilha de auditoria organizacional.
func (s *Service) CreateUser(ctx context.Context, email, password, organization string) (string, error) {
	email, ok := NormalizeEmail(email)
	org, err := identifiers.Parse(organization)
	if !ok || err != nil {
		return "", ErrValidation
	}
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, s.dbTimeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := s.query.WithTx(tx)
	if _, err = qt.LockProvisionOrganization(ctx, org); err != nil {
		return "", ErrValidation
	}
	user, err := qt.InsertUser(ctx, q.InsertUserParams{ID: identifiers.New(), Email: email, PasswordHash: hash})
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return "", errors.New("usuário já cadastrado; associe o ID existente pelo painel")
		}
		return "", ErrUnavailable
	}
	if err = qt.AddOrganizationMember(ctx, q.AddOrganizationMemberParams{OrganizationID: org, UserID: user.ID, Role: "Viewer"}); err != nil {
		return "", ErrUnavailable
	}
	after, _ := json.Marshal(map[string]string{"userId": identifiers.String(user.ID), "role": "Viewer"})
	if err = qt.InsertAuditEvent(ctx, q.InsertAuditEventParams{ID: identifiers.New(), OrganizationID: org, ActorLabel: "local-operator", Action: "user.provisioned", ResourceType: "user", ResourceID: user.ID, AfterRedacted: after}); err != nil {
		return "", ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return "", ErrUnavailable
	}
	return identifiers.String(user.ID), nil
}
