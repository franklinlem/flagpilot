#!/bin/sh
set -eu
# Executado somente na criação do volume. Senha recebida por ambiente, sem log.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB"   --set=runtime_password="$APP_DB_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE flagpilot_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L', :'runtime_password') \gexec
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO flagpilot_app;
SQL
