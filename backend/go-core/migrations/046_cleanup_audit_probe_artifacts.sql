-- 046_cleanup_audit_probe_artifacts.sql
-- 1. Cancel stuck DRAFT delivery orders from previous probe/e2e tests so allocated stock is freed.
-- 2. Soft-delete probe/audit test products (AUD-*, SD-PROBE-*, PROBE-*) so product catalog stays clean.
-- 3. Deactivate probe/audit test warehouses (AUDIT-*, WA-*, WB-*).

-- ==================
-- UP
-- ==================

-- Cancel stuck DRAFT probe DOs to release allocated inventory back to available stock
UPDATE delivery_orders
SET status = 'CANCELLED', updated_at = NOW()
WHERE status = 'DRAFT' AND (do_number LIKE 'PROBE-%' OR do_number LIKE 'DO/E2E/%');

-- Soft-delete test probe products so they no longer appear in catalog/inventory views
UPDATE products
SET deleted_at = NOW()
WHERE deleted_at IS NULL AND (sku LIKE 'AUD-%' OR sku LIKE 'SD-PROBE-%' OR sku LIKE 'PROBE-%');

-- Deactivate test probe warehouses
UPDATE warehouses
SET is_active = false
WHERE name LIKE 'AUDIT-%' OR code LIKE 'WA-%' OR code LIKE 'WB-%';

-- ==================
-- DOWN
-- ==================
-- No-op: probe data cleanup does not need restoration.
