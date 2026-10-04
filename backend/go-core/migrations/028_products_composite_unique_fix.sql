-- 028_products_composite_unique_fix.sql
-- Defensive fix: on deployments where 011_products_inventory.sql created the
-- "products" table BEFORE 012_products_inventory.sql's CREATE TABLE IF NOT
-- EXISTS (a no-op in that case), the UNIQUE(id, tenant_id) composite
-- constraint required by downstream composite foreign keys (021, 022, 023,
-- 026 WMS/ledger migrations) was never created. This migration adds it
-- idempotently regardless of which "011/012" variant ran first.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'products_id_tenant_id_key'
    ) THEN
        ALTER TABLE products ADD CONSTRAINT products_id_tenant_id_key UNIQUE (id, tenant_id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'products_tenant_id_sku_key'
    ) THEN
        ALTER TABLE products ADD CONSTRAINT products_tenant_id_sku_key UNIQUE (tenant_id, sku);
    END IF;
END $$;
