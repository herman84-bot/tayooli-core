-- 033_wms_batches_staging_putaway.sql — Sprint 1 (PRD master §6, ADR-014, KO-1, CR-01, CR-03, PDF-06).
-- Adopted patterns:
--   sentry-wms §1.1/§1.2: Staging bins are not pickable; inbound is 2-step Receive -> Staging, Putaway -> rack.
--   OCA §1.3: FEFO needs expiry per batch (stock_batches.expiry_date).
--   sentry-wms §3.2: audit hash chain (audit_logs.details + prev/current hash written by repo).
-- ==================
-- UP
-- ==================

-- 1. New location types (ALTER TYPE ADD VALUE must run outside a transaction block)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
                   WHERE t.typname = 'location_type' AND e.enumlabel = 'STAGING_INBOUND') THEN
        ALTER TYPE location_type ADD VALUE 'STAGING_INBOUND';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
                   WHERE t.typname = 'location_type' AND e.enumlabel = 'STAGING_OUTBOUND') THEN
        ALTER TYPE location_type ADD VALUE 'STAGING_OUTBOUND';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid = e.enumtypid
                   WHERE t.typname = 'location_type' AND e.enumlabel = 'QUARANTINE') THEN
        ALTER TYPE location_type ADD VALUE 'QUARANTINE';
    END IF;
END $$;

BEGIN;

-- 2. Batches
CREATE TABLE IF NOT EXISTS stock_batches (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id        UUID NOT NULL,
    batch_number      VARCHAR(100) NOT NULL,
    expiry_date       DATE,
    source_receipt_id UUID,
    status            VARCHAR(20) NOT NULL DEFAULT 'RELEASED',
    is_legacy         BOOLEAN NOT NULL DEFAULT FALSE,
    created_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    CONSTRAINT chk_stock_batches_status CHECK (status IN ('RELEASED', 'ON_HOLD')),
    CONSTRAINT chk_stock_batches_number CHECK (length(btrim(batch_number)) > 0),
    UNIQUE (tenant_id, product_id, batch_number),
    UNIQUE (id, tenant_id)
);
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'stock_batches_source_receipt_fkey') THEN
        ALTER TABLE stock_batches ADD CONSTRAINT stock_batches_source_receipt_fkey
            FOREIGN KEY (source_receipt_id) REFERENCES stock_receipts(id) ON DELETE SET NULL;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_stock_batches_tenant_product_expiry ON stock_batches(tenant_id, product_id, expiry_date);
CREATE INDEX IF NOT EXISTS idx_stock_batches_receipt ON stock_batches(tenant_id, source_receipt_id) WHERE source_receipt_id IS NOT NULL;

-- 3. Ledger carries batch (Invariant 1). Backfill LEGACY batch per (tenant, product) for pre-existing rows.
ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS batch_id UUID;

INSERT INTO stock_batches (tenant_id, product_id, batch_number, status, is_legacy)
SELECT DISTINCT m.tenant_id, m.product_id, 'LEGACY', 'RELEASED', TRUE
FROM stock_movements m
WHERE m.batch_id IS NULL
ON CONFLICT (tenant_id, product_id, batch_number) DO NOTHING;

UPDATE stock_movements m
SET batch_id = b.id
FROM stock_batches b
WHERE m.batch_id IS NULL
  AND b.tenant_id = m.tenant_id AND b.product_id = m.product_id AND b.batch_number = 'LEGACY';

ALTER TABLE stock_movements ALTER COLUMN batch_id SET NOT NULL;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'stock_movements_batch_fkey') THEN
        ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_batch_fkey
            FOREIGN KEY (batch_id, tenant_id) REFERENCES stock_batches(id, tenant_id) ON DELETE RESTRICT;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_stock_movements_batch ON stock_movements(tenant_id, batch_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_ref ON stock_movements(tenant_id, reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_dest_prod ON stock_movements(tenant_id, dest_location_id, product_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_src_prod ON stock_movements(tenant_id, source_location_id, product_id);

-- 4. Receipt lines carry batch/expiry; one product may appear once per batch.
ALTER TABLE stock_receipt_items
    ADD COLUMN IF NOT EXISTS batch_number VARCHAR(100),
    ADD COLUMN IF NOT EXISTS expiry_date DATE,
    ADD COLUMN IF NOT EXISTS batch_id UUID;
ALTER TABLE stock_receipt_items DROP CONSTRAINT IF EXISTS stock_receipt_items_receipt_id_product_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_receipt_items_receipt_product_batch
    ON stock_receipt_items(receipt_id, product_id, COALESCE(batch_number, ''));

-- 5. Release approval (PDF-06)
ALTER TABLE stock_receipts
    ADD COLUMN IF NOT EXISTS released_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS released_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS wms_settings (
    tenant_id                UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    require_release_approval BOOLEAN NOT NULL DEFAULT FALSE,
    updated_by               UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Product categories (CR-01) and default rack per warehouse (CR-03)
CREATE TABLE IF NOT EXISTS product_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL CHECK (length(btrim(name)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, tenant_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_product_categories_name ON product_categories(tenant_id, lower(name));

ALTER TABLE products ADD COLUMN IF NOT EXISTS category_id UUID;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'products_category_fkey') THEN
        ALTER TABLE products ADD CONSTRAINT products_category_fkey
            FOREIGN KEY (category_id) REFERENCES product_categories(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS product_default_locations (
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id   UUID NOT NULL,
    warehouse_id UUID NOT NULL,
    location_id  UUID NOT NULL,
    updated_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, product_id, warehouse_id),
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE CASCADE
);

-- 7. Audit details column (repo already writes it; was missing -> every audit insert failed)
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS details JSONB;
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(tenant_id, entity_type, entity_id);

-- 8. RLS (same policy style as 028)
DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['stock_batches', 'wms_settings', 'product_categories', 'product_default_locations'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', t || '_tenant_isolation', t);
        EXECUTE format('CREATE POLICY %I ON %I USING (tenant_id = NULLIF(current_setting(''app.current_tenant_id'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant_id'', true), '''')::uuid)', t || '_tenant_isolation', t);
    END LOOP;
END $$;

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_category_fkey;
ALTER TABLE products DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS product_default_locations;
DROP TABLE IF EXISTS product_categories;
DROP TABLE IF EXISTS wms_settings;
ALTER TABLE stock_receipts DROP COLUMN IF EXISTS released_at, DROP COLUMN IF EXISTS released_by;
DROP INDEX IF EXISTS uq_stock_receipt_items_receipt_product_batch;
ALTER TABLE stock_receipt_items DROP COLUMN IF EXISTS batch_id, DROP COLUMN IF EXISTS expiry_date, DROP COLUMN IF EXISTS batch_number;
ALTER TABLE stock_movements DROP CONSTRAINT IF EXISTS stock_movements_batch_fkey;
ALTER TABLE stock_movements DROP COLUMN IF EXISTS batch_id;
DROP TABLE IF EXISTS stock_batches;
COMMIT;
