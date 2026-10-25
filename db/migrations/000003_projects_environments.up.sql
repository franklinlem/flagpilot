BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE TABLE projects (
 id uuid PRIMARY KEY,
 organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
 name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200 AND name=btrim(name)),
 slug text NOT NULL CHECK (slug ~ '^[a-z][a-z0-9-]{1,62}$'),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (organization_id,slug)
);
CREATE TABLE environments (
 id uuid PRIMARY KEY,
 project_id uuid NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
 name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200 AND name=btrim(name)),
 slug text NOT NULL CHECK (slug ~ '^[a-z][a-z0-9-]{1,62}$'),
 is_production boolean NOT NULL,
 revision bigint NOT NULL DEFAULT 0 CHECK (revision>=0),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (project_id,slug),
 UNIQUE (id,project_id),
 CHECK (slug <> 'production' OR is_production)
);
GRANT SELECT, INSERT ON projects, environments TO flagpilot_app;
-- PostgreSQL exige UPDATE em ao menos uma coluna para SELECT FOR NO KEY UPDATE.
-- Não concede edição de nome/slug/classificação ao runtime.
GRANT UPDATE (id) ON projects TO flagpilot_app;
COMMIT;
