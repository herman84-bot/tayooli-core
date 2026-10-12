-- 051_audit_logs_chain_index.sql
-- WriteAuditTx reads the tenant's latest audit row (ORDER BY created_at DESC,
-- id DESC LIMIT 1) on every audited write. Only (tenant_id) was indexed, so
-- Postgres sorted every audit row of the tenant per write.

-- ==================
-- UP
-- ==================
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created
    ON audit_logs (tenant_id, created_at DESC, id DESC);

-- ==================
-- DOWN
-- ==================
-- DROP INDEX IF EXISTS idx_audit_logs_tenant_created;
