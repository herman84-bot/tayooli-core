-- 023_wms_marketplace_sales_import.sql: Omnichannel marketplace sales order import,
-- batch ingestion sessions, channel order idempotency, and SKU mapping resolution
-- with Row-Level Security (RLS).
-- ==================
-- UP
-- ==================
BEGIN;

-- ============================================================================
-- 0. PREREQUISITE CONSTRAINTS (Ensure composite FK compatibility)
-- ============================================================================

-- Ensure sales_orders table has composite unique constraint (id, tenant_id)
-- to support composite foreign key references from marketplace_orders.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_schema = current_schema() AND table_name = 'sales_orders'
    ) THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON c.conrelid = t.oid
            WHERE t.relname = 'sales_orders' AND c.conname = 'unique_sales_orders_id_tenant'
        ) THEN
            ALTER TABLE sales_orders ADD CONSTRAINT unique_sales_orders_id_tenant UNIQUE (id, tenant_id);
        END IF;
    END IF;
END $$;

-- ============================================================================
-- 1. MARKETPLACE IMPORT BATCHES (IMPORT SESSION TRACKING & STATS)
-- ============================================================================

CREATE TABLE IF NOT EXISTS marketplace_import_batches (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    batch_number      VARCHAR(100) NOT NULL,
    channel           VARCHAR(50) NOT NULL CHECK (channel IN ('SHOPEE', 'TOKOPEDIA', 'TIKTOK', 'LAZADA', 'BLIBLI', 'OTHER')),
    warehouse_id      UUID NOT NULL,
    file_name         VARCHAR(255) NOT NULL,
    total_orders      INT NOT NULL DEFAULT 0,
    processed_orders  INT NOT NULL DEFAULT 0,
    failed_orders     INT NOT NULL DEFAULT 0,
    unmapped_skus     INT NOT NULL DEFAULT 0,
    status            VARCHAR(50) NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    uploaded_by       UUID NOT NULL REFERENCES users(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, batch_number),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 2. MARKETPLACE ORDERS (NORMALIZED CANONICAL OMNICHANNEL ORDERS)
-- ============================================================================

CREATE TABLE IF NOT EXISTS marketplace_orders (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    batch_id           UUID,
    warehouse_id       UUID NOT NULL,
    channel            VARCHAR(50) NOT NULL CHECK (channel IN ('SHOPEE', 'TOKOPEDIA', 'TIKTOK', 'LAZADA', 'BLIBLI', 'OTHER')),
    external_order_id  VARCHAR(150) NOT NULL,
    order_date         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    customer_name      VARCHAR(255),
    customer_phone     VARCHAR(50),
    shipping_address   TEXT,
    courier            VARCHAR(100),
    tracking_number    VARCHAR(100),
    total_amount       NUMERIC(20, 4) NOT NULL DEFAULT 0,
    shipping_fee       NUMERIC(20, 4) NOT NULL DEFAULT 0,
    marketplace_fee    NUMERIC(20, 4) NOT NULL DEFAULT 0,
    net_amount         NUMERIC(20, 4) NOT NULL DEFAULT 0,
    status             VARCHAR(50) NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'UNMAPPED_SKU', 'STOCK_INSUFFICIENT')),
    sales_order_id     UUID,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (batch_id, tenant_id) REFERENCES marketplace_import_batches(id, tenant_id) ON DELETE SET NULL,
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (sales_order_id, tenant_id) REFERENCES sales_orders(id, tenant_id) ON DELETE SET NULL,
    UNIQUE (tenant_id, channel, external_order_id),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 3. MARKETPLACE ORDER ITEMS (LINE ITEMS & SKU MAPPING RESOLUTION)
-- ============================================================================

CREATE TABLE IF NOT EXISTS marketplace_order_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_id      UUID NOT NULL,
    external_sku  VARCHAR(150) NOT NULL,
    product_id    UUID,
    item_name     VARCHAR(255) NOT NULL,
    quantity      NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    unit_price    NUMERIC(20, 4) NOT NULL DEFAULT 0,
    subtotal      NUMERIC(20, 4) NOT NULL DEFAULT 0,
    is_mapped     BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY (order_id, tenant_id) REFERENCES marketplace_orders(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 4. PERFORMANCE & FOREIGN KEY INDEXES
-- ============================================================================

-- 4.1 marketplace_import_batches indexes
CREATE INDEX IF NOT EXISTS idx_marketplace_import_batches_tenant_id 
    ON marketplace_import_batches(tenant_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_import_batches_tenant_warehouse 
    ON marketplace_import_batches(tenant_id, warehouse_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_import_batches_tenant_channel 
    ON marketplace_import_batches(tenant_id, channel);

CREATE INDEX IF NOT EXISTS idx_marketplace_import_batches_tenant_status 
    ON marketplace_import_batches(tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_marketplace_import_batches_tenant_uploaded_by 
    ON marketplace_import_batches(tenant_id, uploaded_by);

CREATE INDEX IF NOT EXISTS idx_marketplace_import_batches_tenant_created 
    ON marketplace_import_batches(tenant_id, created_at DESC);

-- 4.2 marketplace_orders indexes
CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_id 
    ON marketplace_orders(tenant_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_batch 
    ON marketplace_orders(tenant_id, batch_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_warehouse 
    ON marketplace_orders(tenant_id, warehouse_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_sales_order 
    ON marketplace_orders(tenant_id, sales_order_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_channel 
    ON marketplace_orders(tenant_id, channel);

CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_status 
    ON marketplace_orders(tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_ext_order 
    ON marketplace_orders(tenant_id, external_order_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_order_date 
    ON marketplace_orders(tenant_id, order_date DESC);

CREATE INDEX IF NOT EXISTS idx_marketplace_orders_tenant_created 
    ON marketplace_orders(tenant_id, created_at DESC);

-- 4.3 marketplace_order_items indexes
CREATE INDEX IF NOT EXISTS idx_marketplace_order_items_tenant_id 
    ON marketplace_order_items(tenant_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_order_items_tenant_order 
    ON marketplace_order_items(tenant_id, order_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_order_items_tenant_product 
    ON marketplace_order_items(tenant_id, product_id);

CREATE INDEX IF NOT EXISTS idx_marketplace_order_items_tenant_ext_sku 
    ON marketplace_order_items(tenant_id, external_sku);

CREATE INDEX IF NOT EXISTS idx_marketplace_order_items_tenant_mapped 
    ON marketplace_order_items(tenant_id, is_mapped);

-- ============================================================================
-- 5. ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================================

-- 5.1 marketplace_import_batches RLS
ALTER TABLE marketplace_import_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE marketplace_import_batches FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS marketplace_import_batches_tenant_isolation_select ON marketplace_import_batches;
CREATE POLICY marketplace_import_batches_tenant_isolation_select ON marketplace_import_batches
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_import_batches_tenant_isolation_insert ON marketplace_import_batches;
CREATE POLICY marketplace_import_batches_tenant_isolation_insert ON marketplace_import_batches
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_import_batches_tenant_isolation_update ON marketplace_import_batches;
CREATE POLICY marketplace_import_batches_tenant_isolation_update ON marketplace_import_batches
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_import_batches_tenant_isolation_delete ON marketplace_import_batches;
CREATE POLICY marketplace_import_batches_tenant_isolation_delete ON marketplace_import_batches
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 5.2 marketplace_orders RLS
ALTER TABLE marketplace_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE marketplace_orders FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS marketplace_orders_tenant_isolation_select ON marketplace_orders;
CREATE POLICY marketplace_orders_tenant_isolation_select ON marketplace_orders
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_orders_tenant_isolation_insert ON marketplace_orders;
CREATE POLICY marketplace_orders_tenant_isolation_insert ON marketplace_orders
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_orders_tenant_isolation_update ON marketplace_orders;
CREATE POLICY marketplace_orders_tenant_isolation_update ON marketplace_orders
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_orders_tenant_isolation_delete ON marketplace_orders;
CREATE POLICY marketplace_orders_tenant_isolation_delete ON marketplace_orders
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 5.3 marketplace_order_items RLS
ALTER TABLE marketplace_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE marketplace_order_items FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS marketplace_order_items_tenant_isolation_select ON marketplace_order_items;
CREATE POLICY marketplace_order_items_tenant_isolation_select ON marketplace_order_items
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_order_items_tenant_isolation_insert ON marketplace_order_items;
CREATE POLICY marketplace_order_items_tenant_isolation_insert ON marketplace_order_items
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_order_items_tenant_isolation_update ON marketplace_order_items;
CREATE POLICY marketplace_order_items_tenant_isolation_update ON marketplace_order_items
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS marketplace_order_items_tenant_isolation_delete ON marketplace_order_items;
CREATE POLICY marketplace_order_items_tenant_isolation_delete ON marketplace_order_items
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

-- 1. Drop RLS Policies
DROP POLICY IF EXISTS marketplace_order_items_tenant_isolation_delete ON marketplace_order_items;
DROP POLICY IF EXISTS marketplace_order_items_tenant_isolation_update ON marketplace_order_items;
DROP POLICY IF EXISTS marketplace_order_items_tenant_isolation_insert ON marketplace_order_items;
DROP POLICY IF EXISTS marketplace_order_items_tenant_isolation_select ON marketplace_order_items;

DROP POLICY IF EXISTS marketplace_orders_tenant_isolation_delete ON marketplace_orders;
DROP POLICY IF EXISTS marketplace_orders_tenant_isolation_update ON marketplace_orders;
DROP POLICY IF EXISTS marketplace_orders_tenant_isolation_insert ON marketplace_orders;
DROP POLICY IF EXISTS marketplace_orders_tenant_isolation_select ON marketplace_orders;

DROP POLICY IF EXISTS marketplace_import_batches_tenant_isolation_delete ON marketplace_import_batches;
DROP POLICY IF EXISTS marketplace_import_batches_tenant_isolation_update ON marketplace_import_batches;
DROP POLICY IF EXISTS marketplace_import_batches_tenant_isolation_insert ON marketplace_import_batches;
DROP POLICY IF EXISTS marketplace_import_batches_tenant_isolation_select ON marketplace_import_batches;

-- 2. Drop Indexes
DROP INDEX IF EXISTS idx_marketplace_order_items_tenant_mapped;
DROP INDEX IF EXISTS idx_marketplace_order_items_tenant_ext_sku;
DROP INDEX IF EXISTS idx_marketplace_order_items_tenant_product;
DROP INDEX IF EXISTS idx_marketplace_order_items_tenant_order;
DROP INDEX IF EXISTS idx_marketplace_order_items_tenant_id;

DROP INDEX IF EXISTS idx_marketplace_orders_tenant_created;
DROP INDEX IF EXISTS idx_marketplace_orders_tenant_order_date;
DROP INDEX IF EXISTS idx_marketplace_orders_tenant_ext_order;
DROP INDEX IF EXISTS idx_marketplace_orders_tenant_status;
DROP INDEX IF EXISTS idx_marketplace_orders_tenant_channel;
DROP INDEX IF EXISTS idx_marketplace_orders_tenant_sales_order;
DROP INDEX IF EXISTS idx_marketplace_orders_tenant_warehouse;
DROP INDEX IF EXISTS idx_marketplace_orders_tenant_batch;
DROP INDEX IF EXISTS idx_marketplace_orders_tenant_id;

DROP INDEX IF EXISTS idx_marketplace_import_batches_tenant_created;
DROP INDEX IF EXISTS idx_marketplace_import_batches_tenant_uploaded_by;
DROP INDEX IF EXISTS idx_marketplace_import_batches_tenant_status;
DROP INDEX IF EXISTS idx_marketplace_import_batches_tenant_channel;
DROP INDEX IF EXISTS idx_marketplace_import_batches_tenant_warehouse;
DROP INDEX IF EXISTS idx_marketplace_import_batches_tenant_id;

-- 3. Drop Tables (in reverse dependency order)
DROP TABLE IF EXISTS marketplace_order_items;
DROP TABLE IF EXISTS marketplace_orders;
DROP TABLE IF EXISTS marketplace_import_batches;

-- 4. Revert Prerequisite Constraints
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_schema = current_schema() AND table_name = 'sales_orders'
    ) THEN
        IF EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_class t ON c.conrelid = t.oid
            WHERE t.relname = 'sales_orders' AND c.conname = 'unique_sales_orders_id_tenant'
        ) THEN
            ALTER TABLE sales_orders DROP CONSTRAINT unique_sales_orders_id_tenant;
        END IF;
    END IF;
END $$;

COMMIT;
