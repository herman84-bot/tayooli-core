-- 032_delivery_orders_optional_sales_order.sql: Allow Surat Jalan without a Sales Order.
--
-- delivery_orders.sales_order_id was NOT NULL REFERENCES sales_orders(id), but the
-- 13-module app has no Sales Order screen, so the "Buat Surat Jalan" form could never
-- supply a valid id and every create failed with HTTP 400. A direct Surat Jalan
-- (no SO) is now allowed; when an id IS supplied the foreign key still enforces
-- that it exists. ALTER ... DROP NOT NULL is idempotent.
-- ==================
-- UP
-- ==================

ALTER TABLE delivery_orders ALTER COLUMN sales_order_id DROP NOT NULL;

-- ==================
-- DOWN
-- ==================
-- Only possible once no direct Surat Jalan rows remain:
--   ALTER TABLE delivery_orders ALTER COLUMN sales_order_id SET NOT NULL;
