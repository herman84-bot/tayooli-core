-- 044_products_soft_delete.sql: Add soft delete support to products table.
-- Products with deleted_at IS NOT NULL are excluded from List/Get operations.
-- Hard deletes are replaced with logical (soft) deletes for audit trail preservation.

-- ==================
-- UP
-- ==================
BEGIN;

ALTER TABLE products ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Drop strict table constraint to allow re-creating SKU after soft delete
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_tenant_id_sku_key;

-- Re-enforce tenant-scoped SKU uniqueness on active (non-deleted) products only
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_tenant_sku_active ON products(tenant_id, sku) WHERE deleted_at IS NULL;

-- Index for fast tenant filtering of active products
CREATE INDEX IF NOT EXISTS idx_products_tenant_not_deleted ON products(tenant_id) WHERE deleted_at IS NULL;

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

DROP INDEX IF EXISTS idx_products_tenant_not_deleted;
DROP INDEX IF EXISTS idx_products_tenant_sku_active;

ALTER TABLE products DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE products ADD CONSTRAINT products_tenant_id_sku_key UNIQUE (tenant_id, sku);

COMMIT;
