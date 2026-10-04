-- 013_approval_requests.sql
-- approval_requests table (the approvals module was previously only wired in
-- the CI inline bootstrap SQL, never in the migration set — this closes that
-- gap so the feature works from migrations alone).
--
-- RLS is FORCE-enabled like approval_workflows (011): every read/write must
-- run inside a transaction that sets app.current_tenant_id (the repository
-- layer does this via setTenantLocally).

CREATE TABLE IF NOT EXISTS approval_requests (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    workflow_id       UUID, -- nullable: requests can exist without an assigned workflow
    target_type       VARCHAR(50) NOT NULL,
    target_id         VARCHAR(255) NOT NULL,
    status            VARCHAR(50) NOT NULL DEFAULT 'pending',
    current_step_index INTEGER NOT NULL DEFAULT 0,
    requested_by      UUID,
    approved_by       UUID,
    approved_at       TIMESTAMPTZ,
    rejected_by       UUID,
    rejected_at       TIMESTAMPTZ,
    rejection_reason  TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_approval_requests_tenant ON approval_requests (tenant_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_target ON approval_requests (target_type, target_id);

DROP POLICY IF EXISTS tenant_isolation ON approval_requests;
CREATE POLICY tenant_isolation ON approval_requests
    FOR ALL TO public
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE approval_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE approval_requests FORCE ROW LEVEL SECURITY;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tayooli') THEN
        GRANT ALL ON approval_requests TO tayooli;
    END IF;
END $$;
