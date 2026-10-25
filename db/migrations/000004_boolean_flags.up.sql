BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE TABLE flags (
 id uuid PRIMARY KEY,
 project_id uuid NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
 key text NOT NULL CHECK (key ~ '^[a-z][a-z0-9_.-]{1,79}$'),
 description text NOT NULL DEFAULT '' CHECK (char_length(description)<=2000),
 archived_at timestamptz,
 version bigint NOT NULL DEFAULT 1 CHECK (version>=1),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (project_id,key), UNIQUE (id,project_id)
);
CREATE TABLE flag_configs (
 id uuid PRIMARY KEY,
 project_id uuid NOT NULL,
 flag_id uuid NOT NULL,
 environment_id uuid NOT NULL,
 enabled boolean NOT NULL DEFAULT false,
 rollout_bps integer NOT NULL DEFAULT 10000 CHECK (rollout_bps BETWEEN 0 AND 10000),
 salt text NOT NULL CHECK (salt ~ '^[a-f0-9]{64}$'),
 version bigint NOT NULL DEFAULT 1 CHECK (version>=1),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE (flag_id,environment_id),
 FOREIGN KEY (flag_id,project_id) REFERENCES flags(id,project_id) ON DELETE RESTRICT,
 FOREIGN KEY (environment_id,project_id) REFERENCES environments(id,project_id) ON DELETE RESTRICT
);
CREATE INDEX flag_configs_environment_flag_idx ON flag_configs(environment_id,flag_id);
GRANT SELECT, INSERT ON flags, flag_configs TO flagpilot_app;
GRANT UPDATE (description,archived_at,version) ON flags TO flagpilot_app;
GRANT UPDATE (enabled,rollout_bps,version,updated_at) ON flag_configs TO flagpilot_app;
GRANT UPDATE (revision) ON environments TO flagpilot_app;
COMMIT;
