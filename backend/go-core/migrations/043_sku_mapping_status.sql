-- 043_sku_mapping_status.sql
-- M2/M10: Add status column to product_sku_mappings to track mapping approval lifecycle.
BEGIN;

ALTER TABLE product_sku_mappings
    ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'APPROVED';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_product_sku_mappings_status'
    ) THEN
        ALTER TABLE product_sku_mappings
            ADD CONSTRAINT chk_product_sku_mappings_status
            CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED'));
    END IF;
END $$;

COMMIT;

-- DOWN
BEGIN;
ALTER TABLE product_sku_mappings DROP CONSTRAINT IF EXISTS chk_product_sku_mappings_status;
ALTER TABLE product_sku_mappings DROP COLUMN IF EXISTS status;
COMMIT;
