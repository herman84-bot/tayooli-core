-- 048_grant_revoked_tokens.sql
-- Ensure application role `tayooli` (if configured as non-owner) has full
-- permissions on revoked_tokens.
-- Idempotent: checks pg_roles before granting.

-- ==================
-- UP
-- ==================
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tayooli') THEN
        GRANT ALL ON revoked_tokens TO tayooli;
    END IF;
END $$;

-- ==================
-- DOWN
-- ==================
-- REVOKE ALL ON revoked_tokens FROM tayooli;
