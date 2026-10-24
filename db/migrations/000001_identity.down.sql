-- DESTRUTIVA: somente para banco descartável. Produção exige recuperação planejada.
BEGIN;
SET LOCAL lock_timeout = '5s';
DROP TABLE sessions;
DROP TABLE memberships;
DROP TABLE users;
DROP TABLE organizations;
COMMIT;
