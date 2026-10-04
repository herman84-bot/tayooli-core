-- 025_copilot_audit.sql: Audit logging for Tayooli Copilot action proposals and executions.
-- Enforces per-tenant isolation via PostgreSQL Row-Level Security (RLS).
-- ==================
-- UP
-- ==================
BEGIN;

CREATE TABLE IF NOT EXISTS copilot_audit (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    tool_name       VARCHAR(100) NOT NULL,
    action_title    VARCHAR(255) NOT NULL,
    risk_level      VARCHAR(20) NOT NULL,
    status          VARCHAR(50) NOT NULL,
    diff            JSONB,
    payload         JSONB,
    error           TEXT,
    ip_address      VARCHAR(45),
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance and querying audit trails
CREATE INDEX IF NOT EXISTS idx_copilot_audit_tenant_created
    ON copilot_audit(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_copilot_audit_tool_name
    ON copilot_audit(tenant_id, tool_name);

-- Multi-tenant Row-Level Security (RLS)
ALTER TABLE copilot_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE copilot_audit FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'copilot_audit' AND policyname = 'copilot_audit_tenant_isolation'
    ) THEN
        CREATE POLICY copilot_audit_tenant_isolation ON copilot_audit
            FOR ALL
            USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID);
    END IF;
END $$;

COMMIT;
