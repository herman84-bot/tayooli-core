-- 040_marketplace_source_location_multiplier.sql
-- M5: marketplace orders deduct from an explicit rack chosen at import time
--     (no more @DEFAULT / first-rack fallback).
-- M3: the SKU multiplier is snapshotted per line when the line is mapped, so
--     editing a mapping later never changes how much an existing order deducts.
-- Additive only: both columns are nullable; legacy rows keep NULL.
BEGIN;

ALTER TABLE marketplace_orders ADD COLUMN IF NOT EXISTS source_location_id UUID;

ALTER TABLE marketplace_order_items ADD COLUMN IF NOT EXISTS multiplier NUMERIC(20, 4);
ALTER TABLE marketplace_order_items DROP CONSTRAINT IF EXISTS chk_marketplace_order_items_multiplier;
ALTER TABLE marketplace_order_items ADD CONSTRAINT chk_marketplace_order_items_multiplier
    CHECK (multiplier IS NULL OR multiplier > 0);

COMMIT;

-- DOWN
BEGIN;
ALTER TABLE marketplace_order_items DROP CONSTRAINT IF EXISTS chk_marketplace_order_items_multiplier;
ALTER TABLE marketplace_order_items DROP COLUMN IF EXISTS multiplier;
ALTER TABLE marketplace_orders DROP COLUMN IF EXISTS source_location_id;
COMMIT;
