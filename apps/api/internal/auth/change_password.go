package auth

import (
	"context"
	"errors"

	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5"
)

// ChangePassword exige sessão e senha atuais; revoga inclusive a sessão chamadora.
// Hashes são calculados fora da transação e revalidados sob o mesmo lock do login/reset.
func (s *Service) ChangePassword(ctx context.Context, token, currentPassword, newPassword string) error {
	if !validToken(token) {
		return ErrSession
	}
	if currentPassword == "" || len(currentPassword) > 256 || !ValidatePassword(newPassword) || currentPassword == newPassword {
		return ErrValidation
	}
	select {
	case s.hashes <- struct{}{}:
		defer func() { <-s.hashes }()
	default:
		return ErrBusy
	}
	dbctx, cancel := context.WithTimeout(ctx, s.dbTimeout)
	session, err := s.query.FindSession(dbctx, tokenHash(token))
	cancel()
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSession
	}
	if err != nil {
		return ErrUnavailable
	}
	dbctx, cancel = context.WithTimeout(ctx, s.dbTimeout)
	user, err := s.query.FindLoginUser(dbctx, session.Email)
	cancel()
	if err != nil {
		return ErrUnavailable
	}
	if user.DisabledAt.Valid || !VerifyPassword(currentPassword, user.PasswordHash) {
		return ErrCredentials
	}
	encoded, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	dbctx, cancel = context.WithTimeout(ctx, s.dbTimeout)
	defer cancel()
	tx, err := s.pool.Begin(dbctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := s.query.WithTx(tx)
	current, err := qt.LockLoginUser(dbctx, user.Email)
	if err != nil {
		return ErrUnavailable
	}
	if current.DisabledAt.Valid || current.PasswordHash != user.PasswordHash {
		return ErrSession
	}
	activeUser, err := qt.LockActiveSession(dbctx, tokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSession
	}
	if err != nil {
		return ErrUnavailable
	}
	if activeUser != user.ID {
		return ErrSession
	}
	if err = qt.UpdatePassword(dbctx, q.UpdatePasswordParams{ID: user.ID, PasswordHash: encoded}); err != nil {
		return ErrUnavailable
	}
	if err = qt.RevokeUserSessions(dbctx, user.ID); err != nil {
		return ErrUnavailable
	}
	members, err := qt.ListUserMemberships(dbctx, user.ID)
	if err != nil {
		return ErrUnavailable
	}
	if err = s.auditMemberships(dbctx, qt, user.ID, members, "auth.password_changed", "user"); err != nil {
		return err
	}
	if err = tx.Commit(dbctx); err != nil {
		return ErrUnavailable
	}
	return nil
}
