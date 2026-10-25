BEGIN;
SET LOCAL lock_timeout = '5s';
CREATE TABLE api_tokens (
 id uuid PRIMARY KEY,
 project_id uuid NOT NULL,
 environment_id uuid NOT NULL,
 name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200 AND name=btrim(name)),
 token_prefix text NOT NULL CHECK (token_prefix ~ '^fp_eval_[a-f0-9]{8}$'),
 token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash)=32),
 scope text NOT NULL CHECK (scope='evaluation'),
 expires_at timestamptz NOT NULL,
 revoked_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 created_by uuid REFERENCES users(id) ON DELETE SET NULL,
 FOREIGN KEY(environment_id,project_id) REFERENCES environments(id,project_id) ON DELETE RESTRICT,
 CHECK(expires_at>created_at)
);
CREATE INDEX api_tokens_environment_id_idx ON api_tokens(environment_id,id);
GRANT SELECT, INSERT ON api_tokens TO flagpilot_app;
GRANT UPDATE (revoked_at) ON api_tokens TO flagpilot_app;
COMMIT;
