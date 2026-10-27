-- name: FindLoginUser :one
SELECT * FROM users WHERE lower(email) = lower($1);

-- name: LockLoginUser :one
SELECT * FROM users WHERE lower(email) = lower($1) FOR NO KEY UPDATE;

-- name: ListUserMemberships :many
SELECT m.organization_id, m.user_id, m.role, o.name AS organization_name
FROM memberships m JOIN organizations o ON o.id=m.organization_id
WHERE m.user_id=$1 ORDER BY m.organization_id;

-- name: InsertSession :exec
INSERT INTO sessions (id,user_id,token_hash,expires_at) VALUES ($1,$2,$3,$4);

-- name: FindSession :one
SELECT s.id AS session_id, s.expires_at, u.id, u.email, u.created_at
FROM sessions s JOIN users u ON u.id=s.user_id
WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.disabled_at IS NULL;

-- name: LockActiveSession :one
SELECT s.user_id FROM sessions s
WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now()
FOR UPDATE;

-- name: RevokeSession :execrows
UPDATE sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL;

-- name: RevokeUserSessions :exec
UPDATE sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL;

-- name: BootstrapLock :exec
SELECT pg_advisory_xact_lock(614912800);

-- name: BootstrapEmpty :one
SELECT NOT EXISTS(SELECT 1 FROM users) AND NOT EXISTS(SELECT 1 FROM organizations) AS empty;

-- name: InsertUser :one
INSERT INTO users (id,email,password_hash) VALUES ($1,$2,$3) RETURNING *;

-- name: InsertOrganization :one
INSERT INTO organizations (id,name,slug) VALUES ($1,$2,$3) RETURNING *;

-- name: InsertOwner :exec
INSERT INTO memberships (organization_id,user_id,role) VALUES ($1,$2,'Owner');

-- name: UpdatePassword :exec
UPDATE users SET password_hash=$2 WHERE id=$1;

-- name: InsertAuditEvent :exec
INSERT INTO audit_events (id,organization_id,actor_id,actor_label,action,resource_type,resource_id,before_redacted,after_redacted)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9);
