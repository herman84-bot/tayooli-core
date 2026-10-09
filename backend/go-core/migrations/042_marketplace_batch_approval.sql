-- 042_marketplace_batch_approval.sql
-- M7: marketplace imports no longer deduct stock on upload. A batch waits in
-- PENDING_APPROVAL until an owner/admin other than the uploader approves it
-- (segregation of duties); approval deducts and ends in DEDUCTED.
-- Additive: legacy statuses stay valid, new columns are nullable.
BEGIN;
ALTER TABLE marketplace_import_batches DROP CONSTRAINT IF EXISTS marketplace_import_batches_status_check;
ALTER TABLE marketplace_import_batches ADD CONSTRAINT marketplace_import_batches_status_check
    CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED',
                      'PENDING_APPROVAL', 'APPROVED', 'DEDUCTED', 'REJECTED'));
ALTER TABLE marketplace_import_batches ADD COLUMN IF NOT EXISTS approved_by UUID REFERENCES users(id);
ALTER TABLE marketplace_import_batches ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ;
COMMIT;

-- DOWN
BEGIN;
ALTER TABLE marketplace_import_batches DROP COLUMN IF EXISTS approved_at;
ALTER TABLE marketplace_import_batches DROP COLUMN IF EXISTS approved_by;
ALTER TABLE marketplace_import_batches DROP CONSTRAINT IF EXISTS marketplace_import_batches_status_check;
ALTER TABLE marketplace_import_batches ADD CONSTRAINT marketplace_import_batches_status_check
    CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED'));
COMMIT;
