-- 012_products_inventory.sql: Product master and inventory tables for the
-- products & inventory module.
-- products holds per-tenant product records; inventory holds per-warehouse
-- stock rows linked to products. UNIQUE (id, tenant_id) on products enables
-- the composite FK from inventory, preventing cross-tenant references.
-- ==================
-- UP
-- ==================
BEGIN;

-- 1. Create products table
CREATE TABLE IF NOT EXISTS products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    sku             VARCHAR(100) NOT NULL,
    price           NUMERIC(20,4) NOT NULL DEFAULT 0 CHECK (price >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One SKU per tenant — composite unique also blocks cross-tenant dupes
    UNIQUE (tenant_id, sku),

    -- Composite unique enables FK from inventory to (id, tenant_id),
    -- preventing cross-tenant references at the schema level.
    UNIQUE (id, tenant_id)
);

-- 2. Create inventory table
CREATE TABLE IF NOT EXISTS inventory (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id         UUID NOT NULL,
    -- Composite FK: guarantees the referenced product belongs to the same tenant
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE CASCADE,
    quantity           NUMERIC(20,4) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    warehouse_location VARCHAR(255),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Indexes for products
CREATE INDEX IF NOT EXISTS idx_products_tenant_id
    ON products(tenant_id);

CREATE INDEX IF NOT EXISTS idx_products_tenant_name
    ON products(tenant_id, name);

-- 4. Indexes for inventory
CREATE INDEX IF NOT EXISTS idx_inventory_tenant_id
    ON inventory(tenant_id);

CREATE INDEX IF NOT EXISTS idx_inventory_tenant_product
    ON inventory(tenant_id, product_id);

-- 5. RLS: enable and force row-level security on both tables
ALTER TABLE products  ENABLE ROW LEVEL SECURITY;
ALTER TABLE products  FORCE ROW LEVEL SECURITY;
ALTER TABLE inventory ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory FORCE ROW LEVEL SECURITY;

-- 6. RLS policies — idempotent (DROP IF EXISTS before CREATE)
--    Uses same NULLIF pattern as 006/008/009 for safe session-variable handling.

-- products policies
DROP POLICY IF EXISTS products_tenant_isolation_select ON products;
CREATE POLICY products_tenant_isolation_select ON products
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS products_tenant_isolation_insert ON products;
CREATE POLICY products_tenant_isolation_insert ON products
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS products_tenant_isolation_update ON products;
CREATE POLICY products_tenant_isolation_update ON products
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS products_tenant_isolation_delete ON products;
CREATE POLICY products_tenant_isolation_delete ON products
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- inventory policies
DROP POLICY IF EXISTS inventory_tenant_isolation_select ON inventory;
CREATE POLICY inventory_tenant_isolation_select ON inventory
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS inventory_tenant_isolation_insert ON inventory;
CREATE POLICY inventory_tenant_isolation_insert ON inventory
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS inventory_tenant_isolation_update ON inventory;
CREATE POLICY inventory_tenant_isolation_update ON inventory
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS inventory_tenant_isolation_delete ON inventory;
CREATE POLICY inventory_tenant_isolation_delete ON inventory
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

DROP POLICY IF EXISTS inventory_tenant_isolation_delete ON inventory;
DROP POLICY IF EXISTS inventory_tenant_isolation_update ON inventory;
DROP POLICY IF EXISTS inventory_tenant_isolation_insert ON inventory;
DROP POLICY IF EXISTS inventory_tenant_isolation_select ON inventory;

DROP POLICY IF EXISTS products_tenant_isolation_delete ON products;
DROP POLICY IF EXISTS products_tenant_isolation_update ON products;
DROP POLICY IF EXISTS products_tenant_isolation_insert ON products;
DROP POLICY IF EXISTS products_tenant_isolation_select ON products;

DROP INDEX IF EXISTS idx_inventory_tenant_product;
DROP INDEX IF EXISTS idx_inventory_tenant_id;
DROP INDEX IF EXISTS idx_products_tenant_name;
DROP INDEX IF EXISTS idx_products_tenant_id;

DROP TABLE IF EXISTS inventory;
DROP TABLE IF EXISTS products;

COMMIT;
