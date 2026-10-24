BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE TABLE organizations (
 id uuid PRIMARY KEY,
 name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
 slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z][a-z0-9-]{1,62}$'),
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE users (
 id uuid PRIMARY KEY,
 email text NOT NULL CHECK (char_length(email) BETWEEN 3 AND 254 AND email = btrim(email)),
 password_hash text NOT NULL CHECK (password_hash <> ''),
 disabled_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));
CREATE TABLE memberships (
 organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 role text NOT NULL CHECK (role IN ('Owner','Admin','Editor','Viewer')),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY (organization_id,user_id)
);
CREATE INDEX memberships_user_id_idx ON memberships(user_id);
CREATE TABLE sessions (
 id uuid PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
 expires_at timestamptz NOT NULL,
 revoked_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK (expires_at > created_at)
);
CREATE INDEX sessions_user_id_idx ON sessions(user_id);
CREATE INDEX sessions_expires_at_idx ON sessions(expires_at);
GRANT SELECT, INSERT, UPDATE, DELETE ON organizations, users, memberships, sessions TO flagpilot_app;
GRANT SELECT ON schema_migrations TO flagpilot_app;
COMMIT;
