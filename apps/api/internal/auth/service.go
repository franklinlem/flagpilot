package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	q "github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCredentials = errors.New("email ou senha inválidos")
	ErrSession     = errors.New("sessão inválida ou expirada")
	ErrValidation  = errors.New("dados inválidos")
	ErrConflict    = errors.New("bootstrap indisponível: instalação já inicializada")
	ErrUnavailable = errors.New("serviço de autenticação indisponível")
	ErrBusy        = errors.New("tente novamente em instantes")
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}
type Membership struct {
	OrganizationID   string `json:"organizationId"`
	UserID           string `json:"userId"`
	Role             string `json:"role"`
	OrganizationName string `json:"organizationName"`
}
type Identity struct {
	User        User         `json:"user"`
	Memberships []Membership `json:"memberships"`
	CSRFToken   string       `json:"csrfToken"`
}
type Session struct {
	Identity  Identity
	Token     string
	ExpiresAt time.Time
}

type Service struct {
	pool      *pgxpool.Pool
	query     *q.Queries
	ttl       time.Duration
	dbTimeout time.Duration
	dummy     string
	hashes    chan struct{}
}

func New(pool *pgxpool.Pool, ttl, dbTimeout time.Duration) (*Service, error) {
	secret, err := randomToken()
	if err != nil {
		return nil, err
	}
	dummy, err := HashPassword(secret)
	if err != nil {
		return nil, err
	}
	return &Service{pool: pool, query: q.New(pool), ttl: ttl, dbTimeout: dbTimeout, dummy: dummy, hashes: make(chan struct{}, 2)}, nil
}
func NormalizeEmail(email string) (string, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) < 3 || len(email) > 254 {
		return "", false
	}
	a, err := mail.ParseAddress(email)
	return email, err == nil && a.Address == email
}
func tokenHash(token string) []byte { sum := sha256.Sum256([]byte(token)); return sum[:] }
func validToken(token string) bool {
	if len(token) != 64 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}
func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", ErrUnavailable
	}
	return hex.EncodeToString(b[:]), nil
}
func csrf(token string) string {
	h := hmac.New(sha256.New, []byte(token))
	_, _ = h.Write([]byte("flagpilot/csrf/v1"))
	return hex.EncodeToString(h.Sum(nil))
}
func ValidCSRF(token, provided string) bool {
	return validToken(token) && hmac.Equal([]byte(csrf(token)), []byte(provided))
}
func newID() pgtype.UUID {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("random source unavailable")
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return pgtype.UUID{Bytes: b, Valid: true}
}
func idString(id pgtype.UUID) string {
	b := id.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s *Service) Find(ctx context.Context, token string) (Identity, error) {
	if !validToken(token) {
		return Identity{}, ErrSession
	}
	ctx, cancel := context.WithTimeout(ctx, s.dbTimeout)
	defer cancel()
	row, err := s.query.FindSession(ctx, tokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, ErrSession
	}
	if err != nil {
		return Identity{}, ErrUnavailable
	}
	members, err := s.query.ListUserMemberships(ctx, row.ID)
	if err != nil {
		return Identity{}, ErrUnavailable
	}
	return identity(row.ID, row.Email, row.CreatedAt.Time, members, token), nil
}
func identity(id pgtype.UUID, email string, created time.Time, members []q.ListUserMembershipsRow, token string) Identity {
	out := Identity{User: User{ID: idString(id), Email: email, CreatedAt: created.UTC()}, Memberships: make([]Membership, 0, len(members)), CSRFToken: csrf(token)}
	for _, m := range members {
		out.Memberships = append(out.Memberships, Membership{OrganizationID: idString(m.OrganizationID), UserID: idString(m.UserID), Role: m.Role, OrganizationName: m.OrganizationName})
	}
	return out
}
func (s *Service) Login(ctx context.Context, email, password, previous string) (Session, error) {
	email, ok := NormalizeEmail(email)
	if !ok || password == "" || len(password) > 256 {
		return Session{}, ErrValidation
	}
	select {
	case s.hashes <- struct{}{}:
		defer func() { <-s.hashes }()
	default:
		return Session{}, ErrBusy
	}
	dbctx, cancel := context.WithTimeout(ctx, s.dbTimeout)
	user, err := s.query.FindLoginUser(dbctx, email)
	cancel()
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrUnavailable
	}
	encoded := s.dummy
	if err == nil && !user.DisabledAt.Valid {
		encoded = user.PasswordHash
	}
	verified := VerifyPassword(password, encoded)
	if err != nil || user.DisabledAt.Valid || !verified {
		return Session{}, ErrCredentials
	}
	dbctx, cancel = context.WithTimeout(ctx, s.dbTimeout)
	defer cancel()
	tx, err := s.pool.Begin(dbctx)
	if err != nil {
		return Session{}, ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := s.query.WithTx(tx)
	current, err := qt.LockLoginUser(dbctx, email)
	if err != nil {
		return Session{}, ErrUnavailable
	}
	// Serializar com recuperação de senha: nenhuma sessão nasce com hash antigo.
	if current.DisabledAt.Valid || current.PasswordHash != user.PasswordHash {
		return Session{}, ErrCredentials
	}
	token, err := randomToken()
	if err != nil {
		return Session{}, err
	}
	expires := time.Now().UTC().Add(s.ttl)
	if err = qt.InsertSession(dbctx, q.InsertSessionParams{ID: newID(), UserID: user.ID, TokenHash: tokenHash(token), ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true}}); err != nil {
		return Session{}, ErrUnavailable
	}
	if validToken(previous) {
		if _, err = qt.RevokeSession(dbctx, tokenHash(previous)); err != nil {
			return Session{}, ErrUnavailable
		}
	}
	members, err := qt.ListUserMemberships(dbctx, user.ID)
	if err != nil {
		return Session{}, ErrUnavailable
	}
	if err = s.auditMemberships(dbctx, qt, user.ID, members, "auth.login", "user"); err != nil {
		return Session{}, err
	}
	if err = tx.Commit(dbctx); err != nil {
		return Session{}, ErrUnavailable
	}
	return Session{Identity: identity(user.ID, user.Email, user.CreatedAt.Time, members, token), Token: token, ExpiresAt: expires}, nil
}
func (s *Service) Logout(ctx context.Context, token string) error {
	if !validToken(token) {
		return ErrSession
	}
	ctx, cancel := context.WithTimeout(ctx, s.dbTimeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := s.query.WithTx(tx)
	user, err := qt.FindSession(ctx, tokenHash(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSession
	}
	if err != nil {
		return ErrUnavailable
	}
	rows, err := qt.RevokeSession(ctx, tokenHash(token))
	if err != nil {
		return ErrUnavailable
	}
	if rows == 0 {
		return ErrSession
	}
	members, err := qt.ListUserMemberships(ctx, user.ID)
	if err != nil {
		return ErrUnavailable
	}
	if err = s.auditMemberships(ctx, qt, user.ID, members, "auth.logout", "user"); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (s *Service) auditMemberships(ctx context.Context, qt *q.Queries, user pgtype.UUID, members []q.ListUserMembershipsRow, action, label string) error {
	for _, m := range members {
		if err := qt.InsertAuditEvent(ctx, q.InsertAuditEventParams{ID: newID(), OrganizationID: m.OrganizationID, ActorID: user, ActorLabel: label, Action: action, ResourceType: "user", ResourceID: user}); err != nil {
			return ErrUnavailable
		}
	}
	return nil
}

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

func (s *Service) Bootstrap(ctx context.Context, email, password, name, slug string) error {
	email, ok := NormalizeEmail(email)
	name = strings.TrimSpace(name)
	if !ok || !slugPattern.MatchString(slug) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 200 {
		return ErrValidation
	}
	encoded, err := HashPassword(password)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.dbTimeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := s.query.WithTx(tx)
	if err = qt.BootstrapLock(ctx); err != nil {
		return ErrUnavailable
	}
	empty, err := qt.BootstrapEmpty(ctx)
	if err != nil {
		return ErrUnavailable
	}
	if !empty.Bool {
		return ErrConflict
	}
	user, err := qt.InsertUser(ctx, q.InsertUserParams{ID: newID(), Email: email, PasswordHash: encoded})
	if err != nil {
		return ErrUnavailable
	}
	org, err := qt.InsertOrganization(ctx, q.InsertOrganizationParams{ID: newID(), Name: name, Slug: slug})
	if err != nil {
		return ErrUnavailable
	}
	if err = qt.InsertOwner(ctx, q.InsertOwnerParams{OrganizationID: org.ID, UserID: user.ID}); err != nil {
		return ErrUnavailable
	}
	after, _ := json.Marshal(map[string]string{"role": "Owner", "userId": idString(user.ID)})
	if err = qt.InsertAuditEvent(ctx, q.InsertAuditEventParams{ID: newID(), OrganizationID: org.ID, ActorID: user.ID, ActorLabel: "local-operator", Action: "installation.bootstrap", ResourceType: "organization", ResourceID: org.ID, AfterRedacted: after}); err != nil {
		return ErrUnavailable
	}
	if err = tx.Commit(ctx); err != nil {
		return ErrUnavailable
	}
	return nil
}
func (s *Service) ResetPassword(ctx context.Context, email, password string) error {
	email, ok := NormalizeEmail(email)
	if !ok {
		return ErrValidation
	}
	encoded, err := HashPassword(password)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.dbTimeout)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	qt := s.query.WithTx(tx)
	user, err := qt.LockLoginUser(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCredentials
	}
	if err != nil {
		return ErrUnavailable
	}
	if err = qt.UpdatePassword(ctx, q.UpdatePasswordParams{ID: user.ID, PasswordHash: encoded}); err != nil {
		return ErrUnavailable
	}
	if err = qt.RevokeUserSessions(ctx, user.ID); err != nil {
		return ErrUnavailable
	}
	members, err := qt.ListUserMemberships(ctx, user.ID)
	if err != nil {
		return ErrUnavailable
	}
	if err = s.auditMemberships(ctx, qt, user.ID, members, "auth.password_reset", "local-operator"); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return ErrUnavailable
	}
	return nil
}
