-- name: InsertFlag :one
INSERT INTO flags(id,project_id,key,description) VALUES($1,$2,$3,$4) RETURNING *;

-- name: FlagByKey :one
SELECT * FROM flags WHERE project_id=$1 AND key=$2;

-- name: ListFlags :many
SELECT * FROM flags WHERE project_id=sqlc.arg(project_id)
AND (sqlc.arg(catalog_status)::text='all' OR (sqlc.arg(catalog_status)='active' AND archived_at IS NULL) OR (sqlc.arg(catalog_status)='archived' AND archived_at IS NOT NULL))
AND strpos(key,sqlc.arg(search_key)::text)>0
AND (sqlc.narg(after_id)::uuid IS NULL OR id>sqlc.narg(after_id)::uuid)
ORDER BY id LIMIT sqlc.arg(page_limit);

-- name: ActiveFlagIDs :many
SELECT id FROM flags WHERE project_id=$1 AND archived_at IS NULL ORDER BY id;

-- name: InsertFlagConfig :exec
INSERT INTO flag_configs(id,project_id,flag_id,environment_id,salt) VALUES($1,$2,$3,$4,$5);

-- name: LockFlagEnvironments :many
SELECT * FROM environments WHERE project_id=$1 ORDER BY id FOR NO KEY UPDATE;

-- name: ScopedFlagEnvironment :one
SELECT e.* FROM environments e JOIN projects p ON p.id=e.project_id
JOIN memberships m ON m.organization_id=p.organization_id WHERE e.id=$1 AND m.user_id=$2;

-- name: ReadFlagEnvironment :one
SELECT * FROM environments WHERE id=$1 FOR SHARE;

-- name: WriteFlagEnvironment :one
SELECT * FROM environments WHERE id=$1 FOR NO KEY UPDATE;

-- name: BumpEnvironment :one
UPDATE environments SET revision=revision+1 WHERE id=$1 RETURNING revision;

-- name: ChangeFlag :one
UPDATE flags SET description=$3, archived_at=CASE WHEN $4::boolean THEN now() ELSE archived_at END,version=version+1
WHERE project_id=$1 AND key=$2 RETURNING *;

-- name: ChangeFlagConfig :exec
UPDATE flag_configs SET enabled=$3,rollout_bps=$4,version=version+1,updated_at=now() WHERE environment_id=$1 AND flag_id=$2;

-- name: ListFlagConfigs :many
SELECT c.*, f.key AS flag_key, f.description FROM flag_configs c JOIN flags f ON f.id=c.flag_id
WHERE c.environment_id=sqlc.arg(environment_id) AND f.archived_at IS NULL
AND (sqlc.narg(after_id)::uuid IS NULL OR c.flag_id>sqlc.narg(after_id)::uuid)
ORDER BY c.flag_id LIMIT sqlc.arg(page_limit);

-- name: FlagConfigByKey :one
SELECT c.*,f.key AS flag_key,f.description FROM flag_configs c JOIN flags f ON f.id=c.flag_id
WHERE c.environment_id=$1 AND f.key=$2 AND f.archived_at IS NULL;
