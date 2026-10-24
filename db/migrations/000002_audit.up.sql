BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE TABLE audit_events (
 id uuid PRIMARY KEY,
 organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
 actor_id uuid REFERENCES users(id) ON DELETE RESTRICT,
 actor_label text NOT NULL,
 action text NOT NULL,
 resource_type text NOT NULL,
 resource_id uuid NOT NULL,
 before_redacted jsonb,
 after_redacted jsonb,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_organization_created_idx ON audit_events(organization_id, created_at DESC, id DESC);
GRANT SELECT, INSERT ON audit_events TO flagpilot_app;
COMMIT;
