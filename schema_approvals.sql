CREATE TABLE approval_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    steps JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE approval_workflows ENABLE ROW LEVEL SECURITY;
CREATE POLICY approval_workflows_tenant_isolation_policy ON approval_workflows
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE TABLE approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL REFERENCES approval_workflows(id),
    target_type VARCHAR(255) NOT NULL,
    target_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    current_step_index INT NOT NULL DEFAULT 0,
    requested_by UUID NOT NULL,
    approved_by VARCHAR(255),
    approved_at TIMESTAMP WITH TIME ZONE,
    rejected_by VARCHAR(255),
    rejected_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE approval_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY approval_requests_tenant_isolation_policy ON approval_requests
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
