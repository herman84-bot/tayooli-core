-- 031_wms_transfer_cancelled_status.sql: Allow DRAFT stock transfers to be cancelled.
--
-- stock_transfers.status is the PostgreSQL ENUM transfer_status, not a VARCHAR,
-- so writing 'CANCELLED' fails with an invalid-value error (surfacing as HTTP 500)
-- until the label exists in the enum. This adds it idempotently.
--
-- Cancelling is a soft delete: the row is kept as CANCELLED so the transfer
-- history stays auditable. A draft never moved stock, so nothing is reversed.
-- ==================
-- UP
-- ==================

-- 1. Add CANCELLED to the enum. Kept outside an explicit transaction block
--    because ALTER TYPE ... ADD VALUE cannot run inside a transaction in
--    PostgreSQL < 12, matching migration 029's approach.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'transfer_status' AND e.enumlabel = 'CANCELLED'
    ) THEN
        ALTER TYPE transfer_status ADD VALUE 'CANCELLED';
    END IF;
END $$;

-- ==================
-- DOWN
-- ==================
-- PostgreSQL cannot remove an enum label. To roll back, first move any cancelled
-- transfers to REJECTED, then recreate the type:
--
--   UPDATE stock_transfers SET status = 'REJECTED' WHERE status = 'CANCELLED';
--   ALTER TABLE stock_transfers ALTER COLUMN status DROP DEFAULT;
--   ALTER TABLE stock_transfers ALTER COLUMN status TYPE VARCHAR(20)
--       USING status::text;
--   DROP TYPE transfer_status;
--   CREATE TYPE transfer_status AS ENUM ('DRAFT', 'PENDING_APPROVAL', 'APPROVED',
--       'DISPATCHED', 'IN_TRANSIT', 'RECEIVED', 'REJECTED');
--   ALTER TABLE stock_transfers ALTER COLUMN status TYPE transfer_status
--       USING status::transfer_status;
--   ALTER TABLE stock_transfers ALTER COLUMN status SET DEFAULT 'DRAFT';
