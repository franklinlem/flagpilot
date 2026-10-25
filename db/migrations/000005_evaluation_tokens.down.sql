-- DESTRUTIVA: somente para banco descartável; remove credenciais de avaliação.
BEGIN;
SET LOCAL lock_timeout = '5s';
DROP TABLE api_tokens;
COMMIT;
