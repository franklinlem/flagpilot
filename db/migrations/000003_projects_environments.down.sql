-- DESTRUTIVA: somente para banco descartável; remove projetos e ambientes.
BEGIN;
SET LOCAL lock_timeout = '5s';
DROP TABLE environments;
DROP TABLE projects;
COMMIT;
