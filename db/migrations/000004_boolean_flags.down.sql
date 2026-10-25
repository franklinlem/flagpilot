-- DESTRUTIVA: somente para banco descartável.
BEGIN;
SET LOCAL lock_timeout = '5s';
REVOKE UPDATE (revision) ON environments FROM flagpilot_app;
DROP TABLE flag_configs;
DROP TABLE flags;
COMMIT;
