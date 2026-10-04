-- Security Definer functions for auth to bypass RLS
-- These functions allow the application to query users without tenant context

CREATE OR REPLACE FUNCTION get_user_by_email(p_email TEXT)
RETURNS TABLE(id UUID, tenant_id UUID, email VARCHAR, password_hash VARCHAR, role VARCHAR, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    RETURN QUERY SELECT u.id, u.tenant_id, u.email, u.password_hash, u.role, u.created_at, u.updated_at
    FROM users u WHERE u.email = p_email LIMIT 1;
END;
$$;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tayooli') THEN
        GRANT EXECUTE ON FUNCTION get_user_by_email(TEXT) TO tayooli;
    END IF;
END $$;

CREATE OR REPLACE FUNCTION get_user_by_id(p_id UUID, p_tenant_id UUID)
RETURNS TABLE(id UUID, tenant_id UUID, email VARCHAR, password_hash VARCHAR, role VARCHAR, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    RETURN QUERY SELECT u.id, u.tenant_id, u.email, u.password_hash, u.role, u.created_at, u.updated_at
    FROM users u WHERE u.id = p_id AND u.tenant_id = p_tenant_id LIMIT 1;
END;
$$;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tayooli') THEN
        GRANT EXECUTE ON FUNCTION get_user_by_id(UUID, UUID) TO tayooli;
    END IF;
END $$;

-- Approval workflows table
CREATE TABLE IF NOT EXISTS approval_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    steps JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
DROP POLICY IF EXISTS tenant_isolation ON approval_workflows;
CREATE POLICY tenant_isolation ON approval_workflows FOR ALL TO public USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
ALTER TABLE approval_workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE approval_workflows FORCE ROW LEVEL SECURITY;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tayooli') THEN
        GRANT ALL ON approval_workflows TO tayooli;
    END IF;
END $$;
