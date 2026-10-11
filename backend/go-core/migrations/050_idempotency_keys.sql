-- 050_idempotency_keys.sql
-- Client-supplied Idempotency-Key for POS checkout (and future write
-- endpoints). Prevents a retried request (lost response after commit, double
-- tap, reconnect after a power cut) from writing a second sale and deducting
-- stock twice. PRIMARY KEY (tenant_id, key) makes the reservation atomic.

-- ==================
-- UP
-- ==================
CREATE TABLE IF NOT EXISTS idempotency_keys (
    tenant_id     UUID        NOT NULL,
    key           TEXT        NOT NULL,
    request_hash  CHAR(64)    NOT NULL,
    status_code   INT         NULL,
    response_body BYTEA       NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ NULL,
    PRIMARY KEY (tenant_id, key)
);
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_created_at ON idempotency_keys (created_at);

ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE idempotency_keys FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'idempotency_keys' AND policyname = 'idempotency_keys_tenant_isolation'
    ) THEN
        CREATE POLICY idempotency_keys_tenant_isolation ON idempotency_keys
            FOR ALL
            USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tayooli') THEN
        GRANT ALL ON idempotency_keys TO tayooli;
    END IF;
END $$;

-- ==================
-- DOWN
-- ==================
-- DROP TABLE IF EXISTS idempotency_keys;
