-- 021_wms_multi_warehouse_locations.sql: Multi-warehouse management, location hierarchy,
-- SKU barcodes & omnichannel mappings, immutable stock movement ledger, inter-warehouse
-- transfers, and delivery orders (Surat Jalan) with Row-Level Security (RLS).
-- ==================
-- UP
-- ==================
BEGIN;

-- ============================================================================
-- 0. CUSTOM ENUM TYPES (Idempotent creation)
-- ============================================================================

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'location_type') THEN
        CREATE TYPE location_type AS ENUM ('INTERNAL', 'VENDOR', 'CUSTOMER', 'TRANSIT', 'LOSS', 'SCRAP');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'sku_mapping_type') THEN
        CREATE TYPE sku_mapping_type AS ENUM ('CUSTOMER', 'MARKETPLACE', 'VENDOR');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'stock_movement_status') THEN
        CREATE TYPE stock_movement_status AS ENUM ('PENDING', 'DONE', 'CANCELLED');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'transfer_status') THEN
        CREATE TYPE transfer_status AS ENUM ('DRAFT', 'PENDING_APPROVAL', 'APPROVED', 'DISPATCHED', 'IN_TRANSIT', 'RECEIVED', 'REJECTED');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'delivery_order_status') THEN
        CREATE TYPE delivery_order_status AS ENUM ('DRAFT', 'CONFIRMED', 'PICKED', 'PACKED', 'SHIPPED', 'DELIVERED', 'RETURNED', 'CANCELLED');
    END IF;
END $$;

-- ============================================================================
-- 0.1 PREREQUISITE TABLES (Guarantees fresh standalone migration consistency)
-- ============================================================================

CREATE TABLE IF NOT EXISTS customers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    email       VARCHAR(255),
    phone       VARCHAR(50),
    address     TEXT,
    status      VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE customers FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS customers_tenant_isolation_select ON customers;
CREATE POLICY customers_tenant_isolation_select ON customers
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS customers_tenant_isolation_insert ON customers;
CREATE POLICY customers_tenant_isolation_insert ON customers
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS customers_tenant_isolation_update ON customers;
CREATE POLICY customers_tenant_isolation_update ON customers
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS customers_tenant_isolation_delete ON customers;
CREATE POLICY customers_tenant_isolation_delete ON customers
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE TABLE IF NOT EXISTS sales_orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id   UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    order_number  VARCHAR(100) NOT NULL,
    total_amount  NUMERIC(15, 2) NOT NULL DEFAULT 0,
    status        VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'CONFIRMED', 'SHIPPED', 'CANCELLED')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_sales_order UNIQUE (tenant_id, order_number)
);

ALTER TABLE sales_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_orders FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS sales_orders_tenant_isolation_select ON sales_orders;
CREATE POLICY sales_orders_tenant_isolation_select ON sales_orders
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS sales_orders_tenant_isolation_insert ON sales_orders;
CREATE POLICY sales_orders_tenant_isolation_insert ON sales_orders
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS sales_orders_tenant_isolation_update ON sales_orders;
CREATE POLICY sales_orders_tenant_isolation_update ON sales_orders
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS sales_orders_tenant_isolation_delete ON sales_orders;
CREATE POLICY sales_orders_tenant_isolation_delete ON sales_orders
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- ============================================================================
-- 1. MASTER REGIONALS & WAREHOUSES
-- ============================================================================

CREATE TABLE IF NOT EXISTS regionals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code        VARCHAR(50) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code),
    UNIQUE (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS warehouses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    regional_id  UUID,
    code         VARCHAR(50) NOT NULL,
    name         VARCHAR(255) NOT NULL,
    address      TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (regional_id, tenant_id) REFERENCES regionals(id, tenant_id) ON DELETE SET NULL,
    UNIQUE (tenant_id, code),
    UNIQUE (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS user_warehouses (
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    warehouse_id  UUID NOT NULL,
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    assigned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, warehouse_id),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE CASCADE
);

-- ============================================================================
-- 2. WAREHOUSE LOCATIONS (ZONE, RACK, BIN, PALLET & VIRTUAL)
-- ============================================================================

CREATE TABLE IF NOT EXISTS warehouse_locations (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id   UUID,
    parent_id      UUID,
    code           VARCHAR(100) NOT NULL,
    barcode        VARCHAR(100),
    name           VARCHAR(255) NOT NULL,
    type           location_type NOT NULL DEFAULT 'INTERNAL',
    is_pallet      BOOLEAN NOT NULL DEFAULT FALSE,
    pallet_number  VARCHAR(100),
    max_capacity   NUMERIC(15, 2),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE CASCADE,
    UNIQUE (tenant_id, warehouse_id, code),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 3. SKU MULTI-BARCODE & PRODUCT CHANNEL MAPPINGS
-- ============================================================================

CREATE TABLE IF NOT EXISTS product_barcodes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id        UUID NOT NULL,
    barcode           VARCHAR(100) NOT NULL,
    barcode_symbology VARCHAR(50) DEFAULT 'CODE128',
    uom_name          VARCHAR(50) DEFAULT 'PCS',
    multiplier        NUMERIC(10, 4) NOT NULL DEFAULT 1.0 CHECK (multiplier > 0),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE CASCADE,
    UNIQUE (tenant_id, barcode),
    UNIQUE (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS product_sku_mappings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id     UUID NOT NULL,
    mapping_type   sku_mapping_type NOT NULL,
    channel_name   VARCHAR(100) NOT NULL,
    external_sku   VARCHAR(100) NOT NULL,
    external_name  VARCHAR(255),
    multiplier     NUMERIC(10, 4) NOT NULL DEFAULT 1.0 CHECK (multiplier > 0),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE CASCADE,
    UNIQUE (tenant_id, channel_name, external_sku),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 4. STOCK MOVEMENTS (IMMUTABLE LEDGER)
-- ============================================================================

CREATE TABLE IF NOT EXISTS stock_movements (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    movement_number    VARCHAR(100) NOT NULL,
    product_id         UUID NOT NULL,
    source_location_id UUID NOT NULL,
    dest_location_id   UUID NOT NULL,
    quantity           NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    unit_cost          NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (unit_cost >= 0),
    status             stock_movement_status NOT NULL DEFAULT 'DONE',
    reference_type     VARCHAR(50) NOT NULL,
    reference_id       UUID NOT NULL,
    executed_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (source_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (dest_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, movement_number),
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 5. STOCK TRANSFERS & ITEMS
-- ============================================================================

CREATE TABLE IF NOT EXISTS stock_transfers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transfer_number     VARCHAR(100) NOT NULL,
    from_warehouse_id   UUID NOT NULL,
    to_warehouse_id     UUID NOT NULL,
    status              transfer_status NOT NULL DEFAULT 'DRAFT',
    requested_by        UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    vehicle_plate       VARCHAR(50),
    driver_name         VARCHAR(100),
    dispatched_at       TIMESTAMPTZ,
    received_at         TIMESTAMPTZ,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (from_warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (to_warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, transfer_number),
    UNIQUE (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS stock_transfer_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transfer_id         UUID NOT NULL,
    product_id          UUID NOT NULL,
    requested_qty       NUMERIC(20, 4) NOT NULL CHECK (requested_qty > 0),
    sent_qty            NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (sent_qty >= 0),
    received_qty        NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (received_qty >= 0),
    source_location_id  UUID,
    dest_location_id    UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (transfer_id, tenant_id) REFERENCES stock_transfers(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (source_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE SET NULL,
    FOREIGN KEY (dest_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE SET NULL,
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 6. DELIVERY ORDERS (SURAT JALAN) & ITEMS
-- ============================================================================

CREATE TABLE IF NOT EXISTS delivery_orders (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sales_order_id      UUID NOT NULL REFERENCES sales_orders(id) ON DELETE RESTRICT,
    warehouse_id        UUID NOT NULL,
    do_number           VARCHAR(100) NOT NULL,
    status              delivery_order_status NOT NULL DEFAULT 'DRAFT',
    expedition_name     VARCHAR(100),
    tracking_number     VARCHAR(100),
    driver_name         VARCHAR(100),
    vehicle_plate       VARCHAR(50),
    recipient_name      VARCHAR(100),
    received_date       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, do_number),
    UNIQUE (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS delivery_order_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    delivery_order_id   UUID NOT NULL,
    product_id          UUID NOT NULL,
    quantity            NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    location_id         UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (delivery_order_id, tenant_id) REFERENCES delivery_orders(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (id, tenant_id)
);

-- ============================================================================
-- 7. INDEXES
-- ============================================================================

-- regionals
CREATE INDEX IF NOT EXISTS idx_regionals_tenant_id ON regionals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_regionals_tenant_name ON regionals(tenant_id, name);

-- warehouses
CREATE INDEX IF NOT EXISTS idx_warehouses_tenant_id ON warehouses(tenant_id);
CREATE INDEX IF NOT EXISTS idx_warehouses_tenant_regional ON warehouses(tenant_id, regional_id);
CREATE INDEX IF NOT EXISTS idx_warehouses_tenant_is_active ON warehouses(tenant_id, is_active);

-- user_warehouses
CREATE INDEX IF NOT EXISTS idx_user_warehouses_tenant_user ON user_warehouses(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_user_warehouses_tenant_wh ON user_warehouses(tenant_id, warehouse_id);

-- warehouse_locations
CREATE INDEX IF NOT EXISTS idx_wh_locations_tenant_id ON warehouse_locations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wh_locations_tenant_wh ON warehouse_locations(tenant_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_wh_locations_tenant_parent ON warehouse_locations(tenant_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_wh_locations_tenant_barcode ON warehouse_locations(tenant_id, barcode) WHERE barcode IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_wh_locations_tenant_type ON warehouse_locations(tenant_id, type);

-- product_barcodes
CREATE INDEX IF NOT EXISTS idx_product_barcodes_tenant_id ON product_barcodes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_product_barcodes_tenant_prod ON product_barcodes(tenant_id, product_id);
CREATE INDEX IF NOT EXISTS idx_product_barcodes_tenant_barcode ON product_barcodes(tenant_id, barcode);

-- product_sku_mappings
CREATE INDEX IF NOT EXISTS idx_product_sku_map_tenant_id ON product_sku_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_product_sku_map_tenant_prod ON product_sku_mappings(tenant_id, product_id);
CREATE INDEX IF NOT EXISTS idx_product_sku_map_lookup ON product_sku_mappings(tenant_id, external_sku);
CREATE INDEX IF NOT EXISTS idx_product_sku_map_channel ON product_sku_mappings(tenant_id, channel_name);

-- stock_movements
CREATE INDEX IF NOT EXISTS idx_stock_movements_tenant_id ON stock_movements(tenant_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_tenant_prod ON stock_movements(tenant_id, product_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_tenant_src ON stock_movements(tenant_id, source_location_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_tenant_dst ON stock_movements(tenant_id, dest_location_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_tenant_ref ON stock_movements(tenant_id, reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_tenant_created ON stock_movements(tenant_id, created_at DESC);

-- stock_transfers
CREATE INDEX IF NOT EXISTS idx_stock_transfers_tenant_id ON stock_transfers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_tenant_status ON stock_transfers(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_tenant_from_wh ON stock_transfers(tenant_id, from_warehouse_id);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_tenant_to_wh ON stock_transfers(tenant_id, to_warehouse_id);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_tenant_created ON stock_transfers(tenant_id, created_at DESC);

-- stock_transfer_items
CREATE INDEX IF NOT EXISTS idx_transfer_items_tenant_transfer ON stock_transfer_items(tenant_id, transfer_id);
CREATE INDEX IF NOT EXISTS idx_transfer_items_tenant_prod ON stock_transfer_items(tenant_id, product_id);

-- delivery_orders
CREATE INDEX IF NOT EXISTS idx_delivery_orders_tenant_id ON delivery_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_delivery_orders_tenant_status ON delivery_orders(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_delivery_orders_tenant_wh ON delivery_orders(tenant_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_delivery_orders_tenant_so ON delivery_orders(tenant_id, sales_order_id);
CREATE INDEX IF NOT EXISTS idx_delivery_orders_tenant_created ON delivery_orders(tenant_id, created_at DESC);

-- delivery_order_items
CREATE INDEX IF NOT EXISTS idx_do_items_tenant_do ON delivery_order_items(tenant_id, delivery_order_id);
CREATE INDEX IF NOT EXISTS idx_do_items_tenant_prod ON delivery_order_items(tenant_id, product_id);
CREATE INDEX IF NOT EXISTS idx_do_items_tenant_loc ON delivery_order_items(tenant_id, location_id);

-- ============================================================================
-- 8. ROW-LEVEL SECURITY (RLS) POLICIES
-- ============================================================================

-- 8.1 regionals
ALTER TABLE regionals ENABLE ROW LEVEL SECURITY;
ALTER TABLE regionals FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS regionals_tenant_isolation_select ON regionals;
CREATE POLICY regionals_tenant_isolation_select ON regionals
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS regionals_tenant_isolation_insert ON regionals;
CREATE POLICY regionals_tenant_isolation_insert ON regionals
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS regionals_tenant_isolation_update ON regionals;
CREATE POLICY regionals_tenant_isolation_update ON regionals
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS regionals_tenant_isolation_delete ON regionals;
CREATE POLICY regionals_tenant_isolation_delete ON regionals
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.2 warehouses
ALTER TABLE warehouses ENABLE ROW LEVEL SECURITY;
ALTER TABLE warehouses FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS warehouses_tenant_isolation_select ON warehouses;
CREATE POLICY warehouses_tenant_isolation_select ON warehouses
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS warehouses_tenant_isolation_insert ON warehouses;
CREATE POLICY warehouses_tenant_isolation_insert ON warehouses
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS warehouses_tenant_isolation_update ON warehouses;
CREATE POLICY warehouses_tenant_isolation_update ON warehouses
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS warehouses_tenant_isolation_delete ON warehouses;
CREATE POLICY warehouses_tenant_isolation_delete ON warehouses
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.3 user_warehouses
ALTER TABLE user_warehouses ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_warehouses FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS user_warehouses_tenant_isolation_select ON user_warehouses;
CREATE POLICY user_warehouses_tenant_isolation_select ON user_warehouses
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS user_warehouses_tenant_isolation_insert ON user_warehouses;
CREATE POLICY user_warehouses_tenant_isolation_insert ON user_warehouses
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS user_warehouses_tenant_isolation_update ON user_warehouses;
CREATE POLICY user_warehouses_tenant_isolation_update ON user_warehouses
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS user_warehouses_tenant_isolation_delete ON user_warehouses;
CREATE POLICY user_warehouses_tenant_isolation_delete ON user_warehouses
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.4 warehouse_locations
ALTER TABLE warehouse_locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE warehouse_locations FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS warehouse_locations_tenant_isolation_select ON warehouse_locations;
CREATE POLICY warehouse_locations_tenant_isolation_select ON warehouse_locations
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS warehouse_locations_tenant_isolation_insert ON warehouse_locations;
CREATE POLICY warehouse_locations_tenant_isolation_insert ON warehouse_locations
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS warehouse_locations_tenant_isolation_update ON warehouse_locations;
CREATE POLICY warehouse_locations_tenant_isolation_update ON warehouse_locations
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS warehouse_locations_tenant_isolation_delete ON warehouse_locations;
CREATE POLICY warehouse_locations_tenant_isolation_delete ON warehouse_locations
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.5 product_barcodes
ALTER TABLE product_barcodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_barcodes FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS product_barcodes_tenant_isolation_select ON product_barcodes;
CREATE POLICY product_barcodes_tenant_isolation_select ON product_barcodes
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS product_barcodes_tenant_isolation_insert ON product_barcodes;
CREATE POLICY product_barcodes_tenant_isolation_insert ON product_barcodes
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS product_barcodes_tenant_isolation_update ON product_barcodes;
CREATE POLICY product_barcodes_tenant_isolation_update ON product_barcodes
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS product_barcodes_tenant_isolation_delete ON product_barcodes;
CREATE POLICY product_barcodes_tenant_isolation_delete ON product_barcodes
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.6 product_sku_mappings
ALTER TABLE product_sku_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_sku_mappings FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS product_sku_mappings_tenant_isolation_select ON product_sku_mappings;
CREATE POLICY product_sku_mappings_tenant_isolation_select ON product_sku_mappings
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS product_sku_mappings_tenant_isolation_insert ON product_sku_mappings;
CREATE POLICY product_sku_mappings_tenant_isolation_insert ON product_sku_mappings
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS product_sku_mappings_tenant_isolation_update ON product_sku_mappings;
CREATE POLICY product_sku_mappings_tenant_isolation_update ON product_sku_mappings
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS product_sku_mappings_tenant_isolation_delete ON product_sku_mappings;
CREATE POLICY product_sku_mappings_tenant_isolation_delete ON product_sku_mappings
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.7 stock_movements
ALTER TABLE stock_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_movements FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stock_movements_tenant_isolation_select ON stock_movements;
CREATE POLICY stock_movements_tenant_isolation_select ON stock_movements
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_movements_tenant_isolation_insert ON stock_movements;
CREATE POLICY stock_movements_tenant_isolation_insert ON stock_movements
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_movements_tenant_isolation_update ON stock_movements;
CREATE POLICY stock_movements_tenant_isolation_update ON stock_movements
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_movements_tenant_isolation_delete ON stock_movements;
CREATE POLICY stock_movements_tenant_isolation_delete ON stock_movements
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.8 stock_transfers
ALTER TABLE stock_transfers ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_transfers FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stock_transfers_tenant_isolation_select ON stock_transfers;
CREATE POLICY stock_transfers_tenant_isolation_select ON stock_transfers
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_transfers_tenant_isolation_insert ON stock_transfers;
CREATE POLICY stock_transfers_tenant_isolation_insert ON stock_transfers
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_transfers_tenant_isolation_update ON stock_transfers;
CREATE POLICY stock_transfers_tenant_isolation_update ON stock_transfers
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_transfers_tenant_isolation_delete ON stock_transfers;
CREATE POLICY stock_transfers_tenant_isolation_delete ON stock_transfers
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.9 stock_transfer_items
ALTER TABLE stock_transfer_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_transfer_items FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS stock_transfer_items_tenant_isolation_select ON stock_transfer_items;
CREATE POLICY stock_transfer_items_tenant_isolation_select ON stock_transfer_items
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_transfer_items_tenant_isolation_insert ON stock_transfer_items;
CREATE POLICY stock_transfer_items_tenant_isolation_insert ON stock_transfer_items
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_transfer_items_tenant_isolation_update ON stock_transfer_items;
CREATE POLICY stock_transfer_items_tenant_isolation_update ON stock_transfer_items
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS stock_transfer_items_tenant_isolation_delete ON stock_transfer_items;
CREATE POLICY stock_transfer_items_tenant_isolation_delete ON stock_transfer_items
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.10 delivery_orders
ALTER TABLE delivery_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE delivery_orders FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS delivery_orders_tenant_isolation_select ON delivery_orders;
CREATE POLICY delivery_orders_tenant_isolation_select ON delivery_orders
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS delivery_orders_tenant_isolation_insert ON delivery_orders;
CREATE POLICY delivery_orders_tenant_isolation_insert ON delivery_orders
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS delivery_orders_tenant_isolation_update ON delivery_orders;
CREATE POLICY delivery_orders_tenant_isolation_update ON delivery_orders
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS delivery_orders_tenant_isolation_delete ON delivery_orders;
CREATE POLICY delivery_orders_tenant_isolation_delete ON delivery_orders
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 8.11 delivery_order_items
ALTER TABLE delivery_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE delivery_order_items FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS delivery_order_items_tenant_isolation_select ON delivery_order_items;
CREATE POLICY delivery_order_items_tenant_isolation_select ON delivery_order_items
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS delivery_order_items_tenant_isolation_insert ON delivery_order_items;
CREATE POLICY delivery_order_items_tenant_isolation_insert ON delivery_order_items
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS delivery_order_items_tenant_isolation_update ON delivery_order_items;
CREATE POLICY delivery_order_items_tenant_isolation_update ON delivery_order_items
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS delivery_order_items_tenant_isolation_delete ON delivery_order_items;
CREATE POLICY delivery_order_items_tenant_isolation_delete ON delivery_order_items
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

-- 1. Drop RLS Policies
DROP POLICY IF EXISTS delivery_order_items_tenant_isolation_delete ON delivery_order_items;
DROP POLICY IF EXISTS delivery_order_items_tenant_isolation_update ON delivery_order_items;
DROP POLICY IF EXISTS delivery_order_items_tenant_isolation_insert ON delivery_order_items;
DROP POLICY IF EXISTS delivery_order_items_tenant_isolation_select ON delivery_order_items;

DROP POLICY IF EXISTS delivery_orders_tenant_isolation_delete ON delivery_orders;
DROP POLICY IF EXISTS delivery_orders_tenant_isolation_update ON delivery_orders;
DROP POLICY IF EXISTS delivery_orders_tenant_isolation_insert ON delivery_orders;
DROP POLICY IF EXISTS delivery_orders_tenant_isolation_select ON delivery_orders;

DROP POLICY IF EXISTS stock_transfer_items_tenant_isolation_delete ON stock_transfer_items;
DROP POLICY IF EXISTS stock_transfer_items_tenant_isolation_update ON stock_transfer_items;
DROP POLICY IF EXISTS stock_transfer_items_tenant_isolation_insert ON stock_transfer_items;
DROP POLICY IF EXISTS stock_transfer_items_tenant_isolation_select ON stock_transfer_items;

DROP POLICY IF EXISTS stock_transfers_tenant_isolation_delete ON stock_transfers;
DROP POLICY IF EXISTS stock_transfers_tenant_isolation_update ON stock_transfers;
DROP POLICY IF EXISTS stock_transfers_tenant_isolation_insert ON stock_transfers;
DROP POLICY IF EXISTS stock_transfers_tenant_isolation_select ON stock_transfers;

DROP POLICY IF EXISTS stock_movements_tenant_isolation_delete ON stock_movements;
DROP POLICY IF EXISTS stock_movements_tenant_isolation_update ON stock_movements;
DROP POLICY IF EXISTS stock_movements_tenant_isolation_insert ON stock_movements;
DROP POLICY IF EXISTS stock_movements_tenant_isolation_select ON stock_movements;

DROP POLICY IF EXISTS product_sku_mappings_tenant_isolation_delete ON product_sku_mappings;
DROP POLICY IF EXISTS product_sku_mappings_tenant_isolation_update ON product_sku_mappings;
DROP POLICY IF EXISTS product_sku_mappings_tenant_isolation_insert ON product_sku_mappings;
DROP POLICY IF EXISTS product_sku_mappings_tenant_isolation_select ON product_sku_mappings;

DROP POLICY IF EXISTS product_barcodes_tenant_isolation_delete ON product_barcodes;
DROP POLICY IF EXISTS product_barcodes_tenant_isolation_update ON product_barcodes;
DROP POLICY IF EXISTS product_barcodes_tenant_isolation_insert ON product_barcodes;
DROP POLICY IF EXISTS product_barcodes_tenant_isolation_select ON product_barcodes;

DROP POLICY IF EXISTS warehouse_locations_tenant_isolation_delete ON warehouse_locations;
DROP POLICY IF EXISTS warehouse_locations_tenant_isolation_update ON warehouse_locations;
DROP POLICY IF EXISTS warehouse_locations_tenant_isolation_insert ON warehouse_locations;
DROP POLICY IF EXISTS warehouse_locations_tenant_isolation_select ON warehouse_locations;

DROP POLICY IF EXISTS user_warehouses_tenant_isolation_delete ON user_warehouses;
DROP POLICY IF EXISTS user_warehouses_tenant_isolation_update ON user_warehouses;
DROP POLICY IF EXISTS user_warehouses_tenant_isolation_insert ON user_warehouses;
DROP POLICY IF EXISTS user_warehouses_tenant_isolation_select ON user_warehouses;

DROP POLICY IF EXISTS warehouses_tenant_isolation_delete ON warehouses;
DROP POLICY IF EXISTS warehouses_tenant_isolation_update ON warehouses;
DROP POLICY IF EXISTS warehouses_tenant_isolation_insert ON warehouses;
DROP POLICY IF EXISTS warehouses_tenant_isolation_select ON warehouses;

DROP POLICY IF EXISTS regionals_tenant_isolation_delete ON regionals;
DROP POLICY IF EXISTS regionals_tenant_isolation_update ON regionals;
DROP POLICY IF EXISTS regionals_tenant_isolation_insert ON regionals;
DROP POLICY IF EXISTS regionals_tenant_isolation_select ON regionals;

-- 2. Drop Indexes
DROP INDEX IF EXISTS idx_do_items_tenant_loc;
DROP INDEX IF EXISTS idx_do_items_tenant_prod;
DROP INDEX IF EXISTS idx_do_items_tenant_do;

DROP INDEX IF EXISTS idx_delivery_orders_tenant_created;
DROP INDEX IF EXISTS idx_delivery_orders_tenant_so;
DROP INDEX IF EXISTS idx_delivery_orders_tenant_wh;
DROP INDEX IF EXISTS idx_delivery_orders_tenant_status;
DROP INDEX IF EXISTS idx_delivery_orders_tenant_id;

DROP INDEX IF EXISTS idx_transfer_items_tenant_prod;
DROP INDEX IF EXISTS idx_transfer_items_tenant_transfer;

DROP INDEX IF EXISTS idx_stock_transfers_tenant_created;
DROP INDEX IF EXISTS idx_stock_transfers_tenant_to_wh;
DROP INDEX IF EXISTS idx_stock_transfers_tenant_from_wh;
DROP INDEX IF EXISTS idx_stock_transfers_tenant_status;
DROP INDEX IF EXISTS idx_stock_transfers_tenant_id;

DROP INDEX IF EXISTS idx_stock_movements_tenant_created;
DROP INDEX IF EXISTS idx_stock_movements_tenant_ref;
DROP INDEX IF EXISTS idx_stock_movements_tenant_dst;
DROP INDEX IF EXISTS idx_stock_movements_tenant_src;
DROP INDEX IF EXISTS idx_stock_movements_tenant_prod;
DROP INDEX IF EXISTS idx_stock_movements_tenant_id;

DROP INDEX IF EXISTS idx_product_sku_map_channel;
DROP INDEX IF EXISTS idx_product_sku_map_lookup;
DROP INDEX IF EXISTS idx_product_sku_map_tenant_prod;
DROP INDEX IF EXISTS idx_product_sku_map_tenant_id;

DROP INDEX IF EXISTS idx_product_barcodes_tenant_barcode;
DROP INDEX IF EXISTS idx_product_barcodes_tenant_prod;
DROP INDEX IF EXISTS idx_product_barcodes_tenant_id;

DROP INDEX IF EXISTS idx_wh_locations_tenant_type;
DROP INDEX IF EXISTS idx_wh_locations_tenant_barcode;
DROP INDEX IF EXISTS idx_wh_locations_tenant_parent;
DROP INDEX IF EXISTS idx_wh_locations_tenant_wh;
DROP INDEX IF EXISTS idx_wh_locations_tenant_id;

DROP INDEX IF EXISTS idx_user_warehouses_tenant_wh;
DROP INDEX IF EXISTS idx_user_warehouses_tenant_user;

DROP INDEX IF EXISTS idx_warehouses_tenant_is_active;
DROP INDEX IF EXISTS idx_warehouses_tenant_regional;
DROP INDEX IF EXISTS idx_warehouses_tenant_id;

DROP INDEX IF EXISTS idx_regionals_tenant_name;
DROP INDEX IF EXISTS idx_regionals_tenant_id;

-- 3. Drop WMS Tables in reverse dependency order
DROP TABLE IF EXISTS delivery_order_items;
DROP TABLE IF EXISTS delivery_orders;
DROP TABLE IF EXISTS stock_transfer_items;
DROP TABLE IF EXISTS stock_transfers;
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS product_sku_mappings;
DROP TABLE IF EXISTS product_barcodes;
DROP TABLE IF EXISTS warehouse_locations;
DROP TABLE IF EXISTS user_warehouses;
DROP TABLE IF EXISTS warehouses;
DROP TABLE IF EXISTS regionals;

-- 4. Drop Custom ENUM Types
DROP TYPE IF EXISTS delivery_order_status;
DROP TYPE IF EXISTS transfer_status;
DROP TYPE IF EXISTS stock_movement_status;
DROP TYPE IF EXISTS sku_mapping_type;
DROP TYPE IF EXISTS location_type;

COMMIT;
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
