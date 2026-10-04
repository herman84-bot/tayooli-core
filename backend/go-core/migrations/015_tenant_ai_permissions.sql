-- Migration: 015_tenant_ai_permissions.sql
-- Description: AI Permissions, Autonomy Levels, and Guardrails for Tayooli Copilot

-- 1. Create table tenant_ai_permissions
CREATE TABLE IF NOT EXISTS tenant_ai_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    autonomy_level VARCHAR(20) NOT NULL DEFAULT 'assisted' CHECK (autonomy_level IN ('advisory', 'assisted', 'autopilot')),
    allowed_scopes JSONB NOT NULL DEFAULT '["workspace.read", "workspace.profile_write", "workspace.master_write"]'::jsonb,
    emergency_stop BOOLEAN NOT NULL DEFAULT false,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_ai_permissions UNIQUE (tenant_id)
);

-- 2. Indexes for performance
CREATE INDEX IF NOT EXISTS idx_tenant_ai_permissions_tenant_id ON tenant_ai_permissions(tenant_id);

-- 3. Enable & Force Row-Level Security (RLS)
ALTER TABLE tenant_ai_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_ai_permissions FORCE ROW LEVEL SECURITY;

-- 4. Multi-tenant RLS Policies for tenant_ai_permissions
DROP POLICY IF EXISTS tenant_ai_permissions_isolation ON tenant_ai_permissions;
CREATE POLICY tenant_ai_permissions_isolation ON tenant_ai_permissions
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 5. Add audit log metadata for AI Agent actions
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS actor_type VARCHAR(50) DEFAULT 'user';
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS prompt_snippet TEXT;
