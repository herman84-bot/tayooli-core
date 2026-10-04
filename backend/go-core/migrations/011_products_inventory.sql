-- 011_products_inventory.sql
-- Products & Inventory tables (multi-tenant ERP catalog).
--
-- SECURITY NOTE: these tables intentionally have NO Row-Level Security.
-- The product/inventory repositories use pgxpool.Pool (which has no
-- setTenantLocally helper), so tenant isolation is enforced at the
-- application layer with explicit WHERE tenant_id = $1 clauses. Do not enable
-- RLS here without first migrating those repos to database/sql + RLS (see the
-- NOTE comments at the top of product_repo.go / inventory_repo.go).

CREATE TABLE IF NOT EXISTS products (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    sku         VARCHAR(100) NOT NULL,
    price       NUMERIC(20,4) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inventory (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id         UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity           NUMERIC(20,4) NOT NULL DEFAULT 0,
    warehouse_location VARCHAR(100) NOT NULL DEFAULT 'Main',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_tenant ON products (tenant_id);
CREATE INDEX IF NOT EXISTS idx_inventory_tenant ON inventory (tenant_id);
CREATE INDEX IF NOT EXISTS idx_inventory_product ON inventory (product_id);
