-- 003_add_ocr_fields.sql
-- ==================
-- UP
-- ==================
BEGIN;

-- 1. Add OCR pipeline columns to invoices
--    Both are nullable: populated only after Python OCR worker processes the document.
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS extracted_text    TEXT,
    ADD COLUMN IF NOT EXISTS ocr_processed_at TIMESTAMPTZ;

-- 2. Update status CHECK constraint if one exists on the invoices.status column.
--    001_init_schema.sql defines status as VARCHAR(50) with no CHECK constraint,
--    so this block will be a no-op on a clean environment.
--    Kept here to handle any environment where a constraint was added out-of-band.
DO $$
DECLARE
    v_constraint_name TEXT;
BEGIN
    SELECT c.conname
    INTO   v_constraint_name
    FROM   pg_constraint c
    JOIN   pg_class      t ON t.oid = c.conrelid
    WHERE  t.relname = 'invoices'
      AND  c.contype = 'c'
      AND  pg_get_constraintdef(c.oid) ILIKE '%status%'
    LIMIT 1;

    IF v_constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE invoices DROP CONSTRAINT %I', v_constraint_name);

        ALTER TABLE invoices
            ADD CONSTRAINT invoices_status_check CHECK (status IN (
                'draft',
                'pending',
                'approved',
                'rejected',
                'paid',
                'ai_processing',
                'ai_processed',
                'ai_failed'
            ));
    END IF;
END;
$$;

-- 3. Composite index on (tenant_id, status) for OCR pipeline status-filter queries.
--    001_init_schema.sql has idx_invoices_tenant_id and idx_invoices_status as two
--    separate single-column indexes; no composite exists yet.
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_status
    ON invoices(tenant_id, status);

-- 4. Partial index for OCR completion lookups: only rows that have been processed,
--    avoiding index bloat from the majority of rows where ocr_processed_at IS NULL.
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_ocr_processed_at
    ON invoices(tenant_id, ocr_processed_at)
    WHERE ocr_processed_at IS NOT NULL;

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

DROP INDEX IF EXISTS idx_invoices_tenant_ocr_processed_at;
DROP INDEX IF EXISTS idx_invoices_tenant_status;

-- Remove the ai_processing / ai_processed values from the CHECK constraint
-- if the UP block added one, restoring the pre-003 state (no constraint).
DO $$
DECLARE
    v_constraint_name TEXT;
BEGIN
    SELECT c.conname
    INTO   v_constraint_name
    FROM   pg_constraint c
    JOIN   pg_class      t ON t.oid = c.conrelid
    WHERE  t.relname = 'invoices'
      AND  c.contype = 'c'
      AND  pg_get_constraintdef(c.oid) ILIKE '%ai_processing%'
    LIMIT 1;

    IF v_constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE invoices DROP CONSTRAINT %I', v_constraint_name);
    END IF;
END;
$$;

ALTER TABLE invoices
    DROP COLUMN IF EXISTS ocr_processed_at,
    DROP COLUMN IF EXISTS extracted_text;

COMMIT;
