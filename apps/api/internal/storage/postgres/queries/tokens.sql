-- name: InsertEvaluationToken :one
INSERT INTO api_tokens(id,project_id,environment_id,name,token_prefix,token_hash,scope,expires_at,created_by)
VALUES($1,$2,$3,$4,$5,$6,'evaluation',$7,$8) RETURNING *;

-- name: ListEvaluationTokens :many
SELECT * FROM api_tokens WHERE environment_id=sqlc.arg(environment_id)
AND (sqlc.narg(after_id)::uuid IS NULL OR id>sqlc.narg(after_id)::uuid)
ORDER BY id LIMIT sqlc.arg(page_limit);

-- name: LockManagedToken :one
SELECT * FROM api_tokens WHERE environment_id=$1 AND id=$2 FOR NO KEY UPDATE;

-- name: RevokeEvaluationToken :one
UPDATE api_tokens SET revoked_at=clock_timestamp() WHERE id=$1 RETURNING *;

-- name: LockEvaluationToken :one
SELECT * FROM api_tokens WHERE token_hash=$1 AND scope='evaluation'
AND revoked_at IS NULL AND expires_at>clock_timestamp() FOR SHARE;

-- name: EvaluationTokenActive :one
SELECT id FROM api_tokens WHERE id=$1 AND scope='evaluation' AND revoked_at IS NULL AND expires_at>clock_timestamp();

-- name: EvaluationConfigs :many
SELECT f.key, c.enabled, c.rollout_bps FROM flag_configs c JOIN flags f ON f.id=c.flag_id
WHERE c.environment_id=$1 AND f.archived_at IS NULL AND f.key=ANY($2::text[]);
