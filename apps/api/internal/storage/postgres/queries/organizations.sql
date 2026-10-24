-- name: ListActorOrganizations :many
SELECT o.id, o.name, o.slug, m.role FROM organizations o
JOIN memberships m ON m.organization_id=o.id JOIN users u ON u.id=m.user_id
WHERE m.user_id=sqlc.arg(actor_id) AND u.disabled_at IS NULL
AND (sqlc.narg(after_id)::uuid IS NULL OR o.id>sqlc.narg(after_id)::uuid)
ORDER BY o.id LIMIT sqlc.arg(page_limit);

-- name: ReadScopedOrganization :one
SELECT o.* FROM organizations o JOIN memberships m ON m.organization_id=o.id
WHERE o.id=$1 AND m.user_id=$2 FOR SHARE OF o;

-- name: WriteScopedOrganization :one
SELECT o.* FROM organizations o JOIN memberships m ON m.organization_id=o.id
WHERE o.id=$1 AND m.user_id=$2 FOR NO KEY UPDATE OF o;

-- name: LockProvisionOrganization :one
SELECT * FROM organizations WHERE id=$1 FOR NO KEY UPDATE;

-- name: ActorOrganizationRole :one
SELECT m.role FROM memberships m JOIN users u ON u.id=m.user_id
WHERE m.organization_id=$1 AND m.user_id=$2 AND u.disabled_at IS NULL;

-- name: ListOrganizationMembers :many
SELECT m.user_id, m.role, u.email, (u.disabled_at IS NOT NULL)::boolean AS disabled
FROM memberships m JOIN users u ON u.id=m.user_id
WHERE m.organization_id=sqlc.arg(organization_id)
AND (sqlc.narg(after_id)::uuid IS NULL OR m.user_id>sqlc.narg(after_id)::uuid)
ORDER BY m.user_id LIMIT sqlc.arg(page_limit);

-- name: GetOrganizationMember :one
SELECT m.user_id, m.role, u.email, (u.disabled_at IS NOT NULL)::boolean AS disabled
FROM memberships m JOIN users u ON u.id=m.user_id
WHERE m.organization_id=$1 AND m.user_id=$2;

-- name: ActiveUser :one
SELECT id,email FROM users WHERE id=$1 AND disabled_at IS NULL;

-- name: CountActiveOwners :one
SELECT count(*) FROM memberships m JOIN users u ON u.id=m.user_id
WHERE m.organization_id=$1 AND m.role='Owner' AND u.disabled_at IS NULL;

-- name: AddOrganizationMember :exec
INSERT INTO memberships (organization_id,user_id,role) VALUES ($1,$2,$3);

-- name: ChangeMemberRole :exec
UPDATE memberships SET role=$3 WHERE organization_id=$1 AND user_id=$2;

-- name: RemoveOrganizationMember :exec
DELETE FROM memberships WHERE organization_id=$1 AND user_id=$2;
