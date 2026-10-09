-- 038_wms_fraud_controls.sql
-- WMS Fraud Controls for Inbound GR, Outbound DO, Opname, Scrap, and Putaway.
BEGIN;

-- 1. Add ordered_qty to stock_receipt_items (F5)
ALTER TABLE stock_receipt_items ADD COLUMN IF NOT EXISTS ordered_qty NUMERIC(20, 4);
UPDATE stock_receipt_items SET ordered_qty = expected_qty WHERE ordered_qty IS NULL AND expected_qty IS NOT NULL;
ALTER TABLE stock_receipt_items DROP CONSTRAINT IF EXISTS chk_stock_receipt_items_ordered_qty;
ALTER TABLE stock_receipt_items ADD CONSTRAINT chk_stock_receipt_items_ordered_qty 
    CHECK (ordered_qty IS NULL OR ordered_qty >= 0);

-- 2. Update stock_opnames status check constraint to include PENDING_APPROVAL (F3)
ALTER TABLE stock_opnames DROP CONSTRAINT IF EXISTS stock_opnames_status_check;
ALTER TABLE stock_opnames ADD CONSTRAINT stock_opnames_status_check 
    CHECK (status IN ('DRAFT', 'IN_PROGRESS', 'PENDING_APPROVAL', 'COMPLETED', 'CANCELLED'));

-- 3. Add approved_by to stock_scraps for high-quantity approvals (F3)
ALTER TABLE stock_scraps ADD COLUMN IF NOT EXISTS approved_by UUID REFERENCES users(id) ON DELETE SET NULL;

COMMIT;

-- DOWN
BEGIN;
ALTER TABLE stock_scraps DROP COLUMN IF EXISTS approved_by;
ALTER TABLE stock_opnames DROP CONSTRAINT IF EXISTS stock_opnames_status_check;
ALTER TABLE stock_opnames ADD CONSTRAINT stock_opnames_status_check 
    CHECK (status IN ('DRAFT', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED'));
ALTER TABLE stock_receipt_items DROP CONSTRAINT IF EXISTS chk_stock_receipt_items_ordered_qty;
ALTER TABLE stock_receipt_items DROP COLUMN IF EXISTS ordered_qty;
COMMIT;
