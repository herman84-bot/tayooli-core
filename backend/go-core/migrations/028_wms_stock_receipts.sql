-- 028_wms_stock_receipts.sql: WMS inbound goods receipt ("Barang Masuk").
-- Header + line items with Row-Level Security (RLS). Stock is NOT stored here;
-- posting a receipt writes DONE rows into stock_movements (the only stock ledger):
--   accepted_qty: @VENDOR -> dest_location_id
--   rejected_qty: @VENDOR -> @SCRAP
-- Named stock_receipt(s) to avoid collision with legacy P2P goods_receipts (006).
-- ==================
-- UP
-- ==================
BEGIN;

-- ============================================================================
-- 1. STOCK RECEIPTS (HEADER)
-- ============================================================================

CREATE TABLE IF NOT EXISTS stock_receipts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    receipt_number   VARCHAR(100) NOT NULL,
    warehouse_id     UUID NOT NULL,
    dest_location_id UUID NOT NULL,
    supplier_name    VARCHAR(255) NOT NULL CHECK (btrim(supplier_name) <> ''),
    supplier_ref     VARCHAR(255),
    notes            TEXT,
    status           VARCHAR(50) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'POSTED', 'CANCELLED')),
    created_by       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    posted_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    posted_at        TIMESTAMPTZ,
    cancelled_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    cancelled_at     TIMESTAMPTZ,
    cancel_reason    TEXT,
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (dest_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, receipt_number),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 2. STOCK RECEIPT ITEMS
-- ============================================================================

CREATE TABLE IF NOT EXISTS stock_receipt_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    receipt_id    UUID NOT NULL,
    product_id    UUID NOT NULL,
    expected_qty  NUMERIC(20, 4) CHECK (expected_qty IS NULL OR expected_qty >= 0),
    accepted_qty  NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (accepted_qty >= 0),
    rejected_qty  NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (rejected_qty >= 0),
    reject_reason TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (receipt_id, tenant_id) REFERENCES stock_receipts(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    CONSTRAINT chk_stock_receipt_items_qty_positive CHECK (accepted_qty + rejected_qty > 0),
    UNIQUE (receipt_id, product_id),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 3. INDEXES
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_stock_receipts_tenant_wh_status ON stock_receipts(tenant_id, warehouse_id, status);
CREATE INDEX IF NOT EXISTS idx_stock_receipts_tenant_created ON stock_receipts(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_stock_receipts_tenant_dest_loc ON stock_receipts(tenant_id, dest_location_id);
CREATE INDEX IF NOT EXISTS idx_stock_receipt_items_tenant_receipt ON stock_receipt_items(tenant_id, receipt_id);
CREATE INDEX IF NOT EXISTS idx_stock_receipt_items_tenant_product ON stock_receipt_items(tenant_id, product_id);

-- ============================================================================
-- 4. ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================================

-- 4.1 stock_receipts
ALTER TABLE stock_receipts ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_receipts FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stock_receipts_tenant_isolation_select ON stock_receipts;
CREATE POLICY stock_receipts_tenant_isolation_select ON stock_receipts
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_receipts_tenant_isolation_insert ON stock_receipts;
CREATE POLICY stock_receipts_tenant_isolation_insert ON stock_receipts
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_receipts_tenant_isolation_update ON stock_receipts;
CREATE POLICY stock_receipts_tenant_isolation_update ON stock_receipts
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_receipts_tenant_isolation_delete ON stock_receipts;
CREATE POLICY stock_receipts_tenant_isolation_delete ON stock_receipts
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 4.2 stock_receipt_items
ALTER TABLE stock_receipt_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_receipt_items FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stock_receipt_items_tenant_isolation_select ON stock_receipt_items;
CREATE POLICY stock_receipt_items_tenant_isolation_select ON stock_receipt_items
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_receipt_items_tenant_isolation_insert ON stock_receipt_items;
CREATE POLICY stock_receipt_items_tenant_isolation_insert ON stock_receipt_items
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_receipt_items_tenant_isolation_update ON stock_receipt_items;
CREATE POLICY stock_receipt_items_tenant_isolation_update ON stock_receipt_items
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_receipt_items_tenant_isolation_delete ON stock_receipt_items;
CREATE POLICY stock_receipt_items_tenant_isolation_delete ON stock_receipt_items
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

DROP POLICY IF EXISTS stock_receipt_items_tenant_isolation_delete ON stock_receipt_items;
DROP POLICY IF EXISTS stock_receipt_items_tenant_isolation_update ON stock_receipt_items;
DROP POLICY IF EXISTS stock_receipt_items_tenant_isolation_insert ON stock_receipt_items;
DROP POLICY IF EXISTS stock_receipt_items_tenant_isolation_select ON stock_receipt_items;

DROP POLICY IF EXISTS stock_receipts_tenant_isolation_delete ON stock_receipts;
DROP POLICY IF EXISTS stock_receipts_tenant_isolation_update ON stock_receipts;
DROP POLICY IF EXISTS stock_receipts_tenant_isolation_insert ON stock_receipts;
DROP POLICY IF EXISTS stock_receipts_tenant_isolation_select ON stock_receipts;

DROP INDEX IF EXISTS idx_stock_receipt_items_tenant_product;
DROP INDEX IF EXISTS idx_stock_receipt_items_tenant_receipt;
DROP INDEX IF EXISTS idx_stock_receipts_tenant_dest_loc;
DROP INDEX IF EXISTS idx_stock_receipts_tenant_created;
DROP INDEX IF EXISTS idx_stock_receipts_tenant_wh_status;

DROP TABLE IF EXISTS stock_receipt_items;
DROP TABLE IF EXISTS stock_receipts;

COMMIT;
