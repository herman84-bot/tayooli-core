-- 007_invoice_match_columns.sql: Extends the invoices table with po_id and match_result columns to support 3-way match engine output.
-- ==================
-- UP
-- ==================
BEGIN;

ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS po_id        UUID REFERENCES purchase_orders(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS match_result VARCHAR(50)
        CHECK (match_result IN ('matched', 'amount_mismatch', 'qty_mismatch', 'no_po', 'no_gr', 'pending'));

-- Index for match engine: look up invoices by po_id + tenant
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_po_id
    ON invoices(tenant_id, po_id)
    WHERE po_id IS NOT NULL;

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;
DROP INDEX IF EXISTS idx_invoices_tenant_po_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS match_result;
ALTER TABLE invoices DROP COLUMN IF EXISTS po_id;
COMMIT;
