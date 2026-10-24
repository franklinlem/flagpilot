-- Apenas para sqlc: esta tabela é criada e mantida por golang-migrate.
CREATE TABLE schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL);
