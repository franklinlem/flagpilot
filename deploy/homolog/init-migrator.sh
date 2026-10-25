#!/bin/sh
set -eu
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --set=migrator_password="$MIGRATOR_DB_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE flagpilot_migrator LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD %L', :'migrator_password') \gexec
ALTER DATABASE flagpilot OWNER TO flagpilot_migrator;
ALTER SCHEMA public OWNER TO flagpilot_migrator;
SQL
