-- DESTRUTIVA: somente para bancos descartáveis; remove histórico de auditoria.
BEGIN;
SET LOCAL lock_timeout = '5s';
DROP TABLE audit_events;
COMMIT;
