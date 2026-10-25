-- name: AuditAnchor :one
SELECT created_at FROM audit_events WHERE organization_id=$1 AND id=$2;

-- name: ListAuditEvents :many
SELECT * FROM audit_events
WHERE organization_id=sqlc.arg(organization_id)
AND (sqlc.arg(action_filter)::text='' OR action=sqlc.arg(action_filter))
AND (sqlc.arg(resource_filter)::text='' OR resource_type=sqlc.arg(resource_filter))
AND (sqlc.narg(before_time)::timestamptz IS NULL OR (created_at,id)<(sqlc.narg(before_time)::timestamptz,sqlc.narg(before_id)::uuid))
ORDER BY created_at DESC,id DESC LIMIT sqlc.arg(page_limit);
