-- Migration 005: add DB-level check constraint on invoices.amount
-- Provides a last-resort defence in case application-layer validation is
-- bypassed.  The constraint mirrors the NUMERIC(20,4) ceiling enforced in the
-- handler and usecase layers (9999999999999999.9999 = 16 integer digits +
-- 4 decimal places).
--
-- UP
BEGIN;
ALTER TABLE invoices
    ADD CONSTRAINT invoices_amount_positive
    CHECK (amount > 0 AND amount <= 9999999999999999.9999);
COMMIT;

-- DOWN
-- BEGIN;
-- ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_amount_positive;
-- COMMIT;
