-- 047_revoked_tokens.sql
-- Server-side session revocation. Logout previously only cleared the browser
-- cookie, so a copied JWT stayed valid until `exp` (1h). TenantMiddleware now
-- rejects any token whose SHA-256 hash is listed here.
-- Stores only the hash (never the raw JWT). Not tenant data: the lookup runs
-- before a tenant is known, so no RLS policy is attached (existing RLS untouched).
-- Rows past expires_at are useless and are pruned on each logout.

-- ==================
-- UP
-- ==================
CREATE TABLE IF NOT EXISTS revoked_tokens (
    token_hash  CHAR(64)    PRIMARY KEY,
    user_id     UUID        NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens (expires_at);

-- ==================
-- DOWN
-- ==================
-- DROP TABLE IF EXISTS revoked_tokens;
