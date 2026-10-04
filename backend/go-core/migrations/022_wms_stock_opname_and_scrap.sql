-- 022_wms_stock_opname_and_scrap.sql: Physical inventory counting (stock opname)
-- and damaged goods / scrap quarantine management with Row-Level Security (RLS).
-- ==================
-- UP
-- ==================
BEGIN;

-- ============================================================================
-- 1. STOCK OPNAMES (PHYSICAL INVENTORY COUNTING HEADERS)
-- ============================================================================

CREATE TABLE IF NOT EXISTS stock_opnames (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id   UUID NOT NULL,
    opname_number  VARCHAR(100) NOT NULL,
    status         VARCHAR(50) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED')),
    conducted_by   UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, opname_number),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 2. STOCK OPNAME ITEMS (PHYSICAL COUNT LINE ITEMS & DISCREPANCIES)
-- ============================================================================

CREATE TABLE IF NOT EXISTS stock_opname_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opname_id       UUID NOT NULL,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id      UUID NOT NULL,
    location_id     UUID NOT NULL,
    system_qty      NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (system_qty >= 0),
    physical_qty    NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (physical_qty >= 0),
    discrepancy_qty NUMERIC(20, 4) NOT NULL DEFAULT 0,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (opname_id, tenant_id) REFERENCES stock_opnames(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 3. STOCK SCRAPS (DAMAGED GOODS & SCRAP QUARANTINE LEDGER)
-- ============================================================================

CREATE TABLE IF NOT EXISTS stock_scraps (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    scrap_number       VARCHAR(100) NOT NULL,
    warehouse_id       UUID NOT NULL,
    product_id         UUID NOT NULL,
    source_location_id UUID NOT NULL,
    scrap_location_id  UUID NOT NULL,
    quantity           NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    reason             TEXT NOT NULL,
    reported_by        UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (source_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (scrap_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    CONSTRAINT chk_scrap_distinct_locations CHECK (source_location_id <> scrap_location_id),
    UNIQUE (tenant_id, scrap_number),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 4. PERFORMANCE & FOREIGN KEY INDEXES
-- ============================================================================

-- stock_opnames indexes
CREATE INDEX IF NOT EXISTS idx_stock_opnames_tenant_id ON stock_opnames(tenant_id);
CREATE INDEX IF NOT EXISTS idx_stock_opnames_tenant_status ON stock_opnames(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_stock_opnames_tenant_warehouse ON stock_opnames(tenant_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_stock_opnames_tenant_conducted_by ON stock_opnames(tenant_id, conducted_by);
CREATE INDEX IF NOT EXISTS idx_stock_opnames_tenant_approved_by ON stock_opnames(tenant_id, approved_by);
CREATE INDEX IF NOT EXISTS idx_stock_opnames_tenant_created ON stock_opnames(tenant_id, created_at DESC);

-- stock_opname_items indexes
CREATE INDEX IF NOT EXISTS idx_stock_opname_items_tenant_id ON stock_opname_items(tenant_id);
CREATE INDEX IF NOT EXISTS idx_stock_opname_items_tenant_opname ON stock_opname_items(tenant_id, opname_id);
CREATE INDEX IF NOT EXISTS idx_stock_opname_items_tenant_product ON stock_opname_items(tenant_id, product_id);
CREATE INDEX IF NOT EXISTS idx_stock_opname_items_tenant_location ON stock_opname_items(tenant_id, location_id);

-- stock_scraps indexes
CREATE INDEX IF NOT EXISTS idx_stock_scraps_tenant_id ON stock_scraps(tenant_id);
CREATE INDEX IF NOT EXISTS idx_stock_scraps_tenant_warehouse ON stock_scraps(tenant_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_stock_scraps_tenant_product ON stock_scraps(tenant_id, product_id);
CREATE INDEX IF NOT EXISTS idx_stock_scraps_tenant_source_loc ON stock_scraps(tenant_id, source_location_id);
CREATE INDEX IF NOT EXISTS idx_stock_scraps_tenant_scrap_loc ON stock_scraps(tenant_id, scrap_location_id);
CREATE INDEX IF NOT EXISTS idx_stock_scraps_tenant_reported_by ON stock_scraps(tenant_id, reported_by);
CREATE INDEX IF NOT EXISTS idx_stock_scraps_tenant_created ON stock_scraps(tenant_id, created_at DESC);

-- ============================================================================
-- 5. ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================================

-- 5.1 stock_opnames
ALTER TABLE stock_opnames ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_opnames FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stock_opnames_tenant_isolation_select ON stock_opnames;
CREATE POLICY stock_opnames_tenant_isolation_select ON stock_opnames
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_opnames_tenant_isolation_insert ON stock_opnames;
CREATE POLICY stock_opnames_tenant_isolation_insert ON stock_opnames
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_opnames_tenant_isolation_update ON stock_opnames;
CREATE POLICY stock_opnames_tenant_isolation_update ON stock_opnames
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_opnames_tenant_isolation_delete ON stock_opnames;
CREATE POLICY stock_opnames_tenant_isolation_delete ON stock_opnames
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 5.2 stock_opname_items
ALTER TABLE stock_opname_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_opname_items FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stock_opname_items_tenant_isolation_select ON stock_opname_items;
CREATE POLICY stock_opname_items_tenant_isolation_select ON stock_opname_items
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_opname_items_tenant_isolation_insert ON stock_opname_items;
CREATE POLICY stock_opname_items_tenant_isolation_insert ON stock_opname_items
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_opname_items_tenant_isolation_update ON stock_opname_items;
CREATE POLICY stock_opname_items_tenant_isolation_update ON stock_opname_items
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_opname_items_tenant_isolation_delete ON stock_opname_items;
CREATE POLICY stock_opname_items_tenant_isolation_delete ON stock_opname_items
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 5.3 stock_scraps
ALTER TABLE stock_scraps ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_scraps FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stock_scraps_tenant_isolation_select ON stock_scraps;
CREATE POLICY stock_scraps_tenant_isolation_select ON stock_scraps
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_scraps_tenant_isolation_insert ON stock_scraps;
CREATE POLICY stock_scraps_tenant_isolation_insert ON stock_scraps
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_scraps_tenant_isolation_update ON stock_scraps;
CREATE POLICY stock_scraps_tenant_isolation_update ON stock_scraps
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_scraps_tenant_isolation_delete ON stock_scraps;
CREATE POLICY stock_scraps_tenant_isolation_delete ON stock_scraps
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

-- 1. Drop RLS Policies
DROP POLICY IF EXISTS stock_scraps_tenant_isolation_delete ON stock_scraps;
DROP POLICY IF EXISTS stock_scraps_tenant_isolation_update ON stock_scraps;
DROP POLICY IF EXISTS stock_scraps_tenant_isolation_insert ON stock_scraps;
DROP POLICY IF EXISTS stock_scraps_tenant_isolation_select ON stock_scraps;

DROP POLICY IF EXISTS stock_opname_items_tenant_isolation_delete ON stock_opname_items;
DROP POLICY IF EXISTS stock_opname_items_tenant_isolation_update ON stock_opname_items;
DROP POLICY IF EXISTS stock_opname_items_tenant_isolation_insert ON stock_opname_items;
DROP POLICY IF EXISTS stock_opname_items_tenant_isolation_select ON stock_opname_items;

DROP POLICY IF EXISTS stock_opnames_tenant_isolation_delete ON stock_opnames;
DROP POLICY IF EXISTS stock_opnames_tenant_isolation_update ON stock_opnames;
DROP POLICY IF EXISTS stock_opnames_tenant_isolation_insert ON stock_opnames;
DROP POLICY IF EXISTS stock_opnames_tenant_isolation_select ON stock_opnames;

-- 2. Drop Indexes
DROP INDEX IF EXISTS idx_stock_scraps_tenant_created;
DROP INDEX IF EXISTS idx_stock_scraps_tenant_reported_by;
DROP INDEX IF EXISTS idx_stock_scraps_tenant_scrap_loc;
DROP INDEX IF EXISTS idx_stock_scraps_tenant_source_loc;
DROP INDEX IF EXISTS idx_stock_scraps_tenant_product;
DROP INDEX IF EXISTS idx_stock_scraps_tenant_warehouse;
DROP INDEX IF EXISTS idx_stock_scraps_tenant_id;

DROP INDEX IF EXISTS idx_stock_opname_items_tenant_location;
DROP INDEX IF EXISTS idx_stock_opname_items_tenant_product;
DROP INDEX IF EXISTS idx_stock_opname_items_tenant_opname;
DROP INDEX IF EXISTS idx_stock_opname_items_tenant_id;

DROP INDEX IF EXISTS idx_stock_opnames_tenant_created;
DROP INDEX IF EXISTS idx_stock_opnames_tenant_approved_by;
DROP INDEX IF EXISTS idx_stock_opnames_tenant_conducted_by;
DROP INDEX IF EXISTS idx_stock_opnames_tenant_warehouse;
DROP INDEX IF EXISTS idx_stock_opnames_tenant_status;
DROP INDEX IF EXISTS idx_stock_opnames_tenant_id;

-- 3. Drop Tables (in reverse dependency order)
DROP TABLE IF EXISTS stock_scraps;
DROP TABLE IF EXISTS stock_opname_items;
DROP TABLE IF EXISTS stock_opnames;

COMMIT;
