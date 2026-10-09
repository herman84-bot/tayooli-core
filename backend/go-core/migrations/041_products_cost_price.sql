-- 041_products_cost_price.sql
-- M6: products carry a cost price so stock movements (e.g. marketplace sales)
-- record a real unit_cost instead of 0. Additive; existing rows default to 0.
BEGIN;
ALTER TABLE products ADD COLUMN IF NOT EXISTS cost_price NUMERIC(20, 4) NOT NULL DEFAULT 0;
ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_products_cost_price_nonneg;
ALTER TABLE products ADD CONSTRAINT chk_products_cost_price_nonneg CHECK (cost_price >= 0);
COMMIT;

-- DOWN
BEGIN;
ALTER TABLE products DROP CONSTRAINT IF EXISTS chk_products_cost_price_nonneg;
ALTER TABLE products DROP COLUMN IF EXISTS cost_price;
COMMIT;
