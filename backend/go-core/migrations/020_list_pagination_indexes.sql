-- Migration 020: composite indexes to serve paginated list queries.
-- Pattern: WHERE tenant_id = $1 ORDER BY <ts> DESC LIMIT $2 OFFSET $3
-- A leading (tenant_id, ts DESC) index lets Postgres stream rows already
-- ordered instead of sorting the tenant's full result set on every request.
-- Additive & idempotent: no schema/behavior change.

-- invoices: list + paged list order by created_at
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_created_at
    ON invoices(tenant_id, created_at DESC);

-- purchase_orders: list + paged list order by created_at
CREATE INDEX IF NOT EXISTS idx_po_tenant_created_at
    ON purchase_orders(tenant_id, created_at DESC);

-- goods_receipts: list + paged list order by received_at (not created_at)
CREATE INDEX IF NOT EXISTS idx_gr_tenant_received_at
    ON goods_receipts(tenant_id, received_at DESC);

-- approval_requests: list + paged list order by created_at
CREATE INDEX IF NOT EXISTS idx_approval_requests_tenant_created_at
    ON approval_requests(tenant_id, created_at DESC);
