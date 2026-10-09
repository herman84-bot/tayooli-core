-- 045_delivery_orders_dispatched_at.sql
-- delivery_orders never had a dispatched_at column, but two code paths use it:
--   * GET /wms/kpi (OrderToDispatchAvgHours) reads it   -> 500 on production
--   * DispatchShippingManifest writes it                -> manifest dispatch fails
-- Additive and idempotent; no data rewrite.

-- ==================
-- UP
-- ==================
ALTER TABLE delivery_orders ADD COLUMN IF NOT EXISTS dispatched_at TIMESTAMPTZ NULL;

-- ==================
-- DOWN
-- ==================
-- ALTER TABLE delivery_orders DROP COLUMN IF EXISTS dispatched_at;
