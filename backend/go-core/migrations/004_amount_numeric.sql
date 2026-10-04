-- 004_amount_numeric.sql
-- ==================
-- UP
-- ==================
-- Reason: invoices.amount was stored as DOUBLE PRECISION (float64 in Go).
-- IEEE 754 binary floating-point cannot represent all decimal fractions exactly,
-- so values like 0.1 or 1234567.89 accumulate rounding errors across arithmetic
-- operations. The 3-way match engine (PO amount vs. GR amount vs. invoice amount)
-- compares these values for equality; even a single ULP of drift causes a false
-- mismatch, triggering incorrect manual-review escalations on valid invoices.
-- NUMERIC(20,4) stores values as exact decimal with up to 16 integer digits and
-- 4 decimal places, eliminating all rounding error at the storage layer.
BEGIN;

ALTER TABLE invoices
    ALTER COLUMN amount TYPE NUMERIC(20,4)
    USING amount::NUMERIC(20,4);

COMMIT;

-- ==================
-- DOWN
-- ==================
-- WARNING: reverting to DOUBLE PRECISION re-introduces float64 rounding errors
-- and will break 3-way match precision. Only run this rollback in a controlled
-- environment and re-validate all open invoices afterward.
BEGIN;

ALTER TABLE invoices
    ALTER COLUMN amount TYPE DOUBLE PRECISION
    USING amount::DOUBLE PRECISION;

COMMIT;
