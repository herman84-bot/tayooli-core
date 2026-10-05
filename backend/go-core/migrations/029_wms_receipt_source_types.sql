-- 029_wms_receipt_source_types.sql: Support Production output and Inter-Warehouse Transfer receipts in WMS inbound.
-- Adds 'PRODUCTION' to location_type enum and adds receipt_type, from_name, from_warehouse_id, source_ref to stock_receipts.
-- ==================
-- UP
-- ==================

-- 1. Add PRODUCTION to location_type enum if not already present (outside transaction)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'location_type' AND e.enumlabel = 'PRODUCTION'
    ) THEN
        ALTER TYPE location_type ADD VALUE 'PRODUCTION';
    END IF;
END $$;

BEGIN;

-- 2. Add source classification columns to stock_receipts
ALTER TABLE stock_receipts
    ADD COLUMN IF NOT EXISTS receipt_type VARCHAR(50) NOT NULL DEFAULT 'PRODUCTION',
    ADD COLUMN IF NOT EXISTS from_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS from_warehouse_id UUID,
    ADD COLUMN IF NOT EXISTS source_ref VARCHAR(255),
    ADD COLUMN IF NOT EXISTS transfer_id UUID;

-- Backfill from_name and source_ref for existing records
UPDATE stock_receipts
SET from_name = COALESCE(from_name, supplier_name, 'Pemasok'),
    source_ref = COALESCE(source_ref, supplier_ref)
WHERE from_name IS NULL;

-- Relax supplier_name NOT NULL constraint so non-vendor receipts don't require dummy supplier
ALTER TABLE stock_receipts ALTER COLUMN supplier_name DROP NOT NULL;

-- Validate receipt_type values
ALTER TABLE stock_receipts DROP CONSTRAINT IF EXISTS chk_stock_receipts_type;
ALTER TABLE stock_receipts ADD CONSTRAINT chk_stock_receipts_type
    CHECK (receipt_type IN ('PRODUCTION', 'TRANSFER', 'VENDOR'));

-- Add foreign key constraint for from_warehouse_id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'stock_receipts_from_warehouse_fkey'
    ) THEN
        ALTER TABLE stock_receipts
        ADD CONSTRAINT stock_receipts_from_warehouse_fkey
        FOREIGN KEY (from_warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE SET NULL;
    END IF;
END $$;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_stock_receipts_tenant_type ON stock_receipts(tenant_id, receipt_type);
CREATE INDEX IF NOT EXISTS idx_stock_receipts_tenant_from_wh ON stock_receipts(tenant_id, from_warehouse_id) WHERE from_warehouse_id IS NOT NULL;

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;
DROP INDEX IF EXISTS idx_stock_receipts_tenant_from_wh;
DROP INDEX IF EXISTS idx_stock_receipts_tenant_type;
ALTER TABLE stock_receipts DROP CONSTRAINT IF EXISTS stock_receipts_from_warehouse_fkey;
ALTER TABLE stock_receipts DROP CONSTRAINT IF EXISTS chk_stock_receipts_type;
ALTER TABLE stock_receipts DROP COLUMN IF EXISTS transfer_id;
ALTER TABLE stock_receipts DROP COLUMN IF EXISTS source_ref;
ALTER TABLE stock_receipts DROP COLUMN IF EXISTS from_warehouse_id;
ALTER TABLE stock_receipts DROP COLUMN IF EXISTS from_name;
ALTER TABLE stock_receipts DROP COLUMN IF EXISTS receipt_type;
COMMIT;
