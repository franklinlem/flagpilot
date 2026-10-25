-- name: ProjectScopeOrganization :one
SELECT o.id FROM organizations o JOIN memberships m ON m.organization_id=o.id
WHERE o.id=$1 AND m.user_id=$2 FOR SHARE OF o;

-- name: ScopedProject :one
SELECT p.* FROM projects p JOIN memberships m ON m.organization_id=p.organization_id
WHERE p.id=$1 AND m.user_id=$2;

-- name: LockProject :one
SELECT id FROM projects WHERE id=$1 FOR NO KEY UPDATE;

-- name: InsertProject :one
INSERT INTO projects (id,organization_id,name,slug) VALUES ($1,$2,$3,$4) RETURNING *;

-- name: ListProjects :many
SELECT * FROM projects WHERE organization_id=sqlc.arg(organization_id)
AND (sqlc.narg(after_id)::uuid IS NULL OR id>sqlc.narg(after_id)::uuid)
ORDER BY id LIMIT sqlc.arg(page_limit);

-- name: InsertEnvironment :one
INSERT INTO environments (id,project_id,name,slug,is_production) VALUES ($1,$2,$3,$4,$5) RETURNING *;

-- name: ListEnvironments :many
SELECT * FROM environments WHERE project_id=sqlc.arg(project_id)
AND (sqlc.narg(after_id)::uuid IS NULL OR id>sqlc.narg(after_id)::uuid)
ORDER BY id LIMIT sqlc.arg(page_limit);
