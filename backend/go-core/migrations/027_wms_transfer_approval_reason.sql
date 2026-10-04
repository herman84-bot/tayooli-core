-- 027_wms_transfer_approval_reason.sql: Adds rejection_reason column to stock_transfers
-- to support the explicit DRAFT -> PENDING_APPROVAL -> APPROVED/REJECTED approval flow.
-- ==================
-- UP
-- ==================
BEGIN;

ALTER TABLE stock_transfers
    ADD COLUMN IF NOT EXISTS rejection_reason TEXT;

COMMIT;

-- ==================
-- DOWN
-- ==================
-- BEGIN;
-- ALTER TABLE stock_transfers DROP COLUMN IF EXISTS rejection_reason;
-- COMMIT;
