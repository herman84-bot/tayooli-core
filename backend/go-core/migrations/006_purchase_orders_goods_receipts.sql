-- 006_purchase_orders_goods_receipts.sql: Creates purchase_orders and goods_receipts tables for the 3-way match engine.
-- ==================
-- UP
-- ==================
BEGIN;

CREATE TABLE IF NOT EXISTS purchase_orders (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    vendor_id    VARCHAR(255) NOT NULL,
    po_number    VARCHAR(100) NOT NULL,
    amount       NUMERIC(20,4) NOT NULL CHECK (amount > 0),
    qty          INTEGER NOT NULL CHECK (qty > 0),
    currency     VARCHAR(10) NOT NULL DEFAULT 'IDR',
    status       VARCHAR(50) NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open', 'partially_received', 'received', 'closed', 'cancelled')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- composite unique enables FK from goods_receipts to (id, tenant_id)
    -- preventing cross-tenant GR→PO references at the schema level
    UNIQUE (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS goods_receipts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    po_id           UUID NOT NULL,
    FOREIGN KEY (po_id, tenant_id) REFERENCES purchase_orders(id, tenant_id) ON DELETE RESTRICT,
    vendor_id       VARCHAR(255) NOT NULL,
    received_qty    INTEGER NOT NULL CHECK (received_qty > 0),
    received_amount NUMERIC(20,4) NOT NULL CHECK (received_amount > 0),
    currency        VARCHAR(10) NOT NULL DEFAULT 'IDR',
    status          VARCHAR(50) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'accepted', 'rejected')),
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique PO number per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_po_tenant_po_number ON purchase_orders(tenant_id, po_number);

-- Fast lookup: GRs by PO
CREATE INDEX IF NOT EXISTS idx_gr_tenant_po_id ON goods_receipts(tenant_id, po_id);

-- RLS
ALTER TABLE purchase_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_orders FORCE ROW LEVEL SECURITY;
ALTER TABLE goods_receipts  ENABLE ROW LEVEL SECURITY;
ALTER TABLE goods_receipts  FORCE ROW LEVEL SECURITY;

-- PostgreSQL does not support CREATE POLICY IF NOT EXISTS.
-- DROP IF EXISTS before each CREATE makes this block idempotent.
DROP POLICY IF EXISTS po_tenant_isolation_select ON purchase_orders;
CREATE POLICY po_tenant_isolation_select ON purchase_orders
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS po_tenant_isolation_insert ON purchase_orders;
CREATE POLICY po_tenant_isolation_insert ON purchase_orders
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS po_tenant_isolation_update ON purchase_orders;
CREATE POLICY po_tenant_isolation_update ON purchase_orders
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS po_tenant_isolation_delete ON purchase_orders;
CREATE POLICY po_tenant_isolation_delete ON purchase_orders
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS gr_tenant_isolation_select ON goods_receipts;
CREATE POLICY gr_tenant_isolation_select ON goods_receipts
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS gr_tenant_isolation_insert ON goods_receipts;
CREATE POLICY gr_tenant_isolation_insert ON goods_receipts
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS gr_tenant_isolation_update ON goods_receipts;
CREATE POLICY gr_tenant_isolation_update ON goods_receipts
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS gr_tenant_isolation_delete ON goods_receipts;
CREATE POLICY gr_tenant_isolation_delete ON goods_receipts
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;
DROP POLICY IF EXISTS gr_tenant_isolation_delete ON goods_receipts;
DROP POLICY IF EXISTS gr_tenant_isolation_update ON goods_receipts;
DROP POLICY IF EXISTS gr_tenant_isolation_insert ON goods_receipts;
DROP POLICY IF EXISTS gr_tenant_isolation_select ON goods_receipts;
DROP POLICY IF EXISTS po_tenant_isolation_delete ON purchase_orders;
DROP POLICY IF EXISTS po_tenant_isolation_update ON purchase_orders;
DROP POLICY IF EXISTS po_tenant_isolation_insert ON purchase_orders;
DROP POLICY IF EXISTS po_tenant_isolation_select ON purchase_orders;
DROP TABLE IF EXISTS goods_receipts;
DROP TABLE IF EXISTS purchase_orders;
COMMIT;
