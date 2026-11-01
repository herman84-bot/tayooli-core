-- 055_seed_demo_tenant_wms_data.sql
-- Demo data for the PUBLIC DEMO TENANT ONLY (550e8400-e29b-41d4-a716-446655440000,
-- the tenant of admin@test.com). Real client tenants are never touched: every
-- row below is selected FROM tenants WHERE id = <demo tenant>, so nothing is
-- inserted when the demo tenant does not exist.
--
-- Purpose: client demos and post-deploy smoke tests (POS checkout, FEFO,
-- batch expiry control panel). Contents:
--   1 warehouse "Gudang Demo Jakarta" (DEMO-JKT) with racks RAK-A1, RAK-A2, RAK-B1
--   8 products (SKU prefix DEMO-) with stock per batch in the WMS ledger
--   batches already expired, critical (<= 7 days) and soon-expiring, so the
--   "Kontrol Expired per Batch" panel and the expired-stock block can be shown
--   2 customers for Surat Jalan demos
-- Expiry dates are relative to the day this migration runs (CURRENT_DATE), so
-- they drift with time: a "critical" batch becomes "expired" a few days later.
--
-- Safe to re-run: products/batches/locations are looked up by their natural
-- keys and inserted with ON CONFLICT / NOT EXISTS guards. Runs in one
-- transaction with the tenant context set, because these tables use
-- FORCE ROW LEVEL SECURITY (policies read app.current_tenant_id).

-- ==================
-- UP
-- ==================
BEGIN;

SET LOCAL app.current_tenant_id = '550e8400-e29b-41d4-a716-446655440000';

-- Warehouse
INSERT INTO warehouses (tenant_id, code, name)
SELECT t.id, 'DEMO-JKT', 'Gudang Demo Jakarta'
FROM tenants t
WHERE t.id = '550e8400-e29b-41d4-a716-446655440000'
  AND NOT EXISTS (SELECT 1 FROM warehouses w WHERE w.tenant_id = t.id AND w.code = 'DEMO-JKT');

-- Racks (INTERNAL locations)
INSERT INTO warehouse_locations (tenant_id, warehouse_id, code, name, type)
SELECT w.tenant_id, w.id, r.code, r.name, 'INTERNAL'
FROM warehouses w
CROSS JOIN (VALUES ('RAK-A1', 'Rak A1 - Dairy'), ('RAK-A2', 'Rak A2 - Roti & Snack'), ('RAK-B1', 'Rak B1 - Sembako')) AS r(code, name)
WHERE w.tenant_id = '550e8400-e29b-41d4-a716-446655440000' AND w.code = 'DEMO-JKT'
ON CONFLICT (tenant_id, warehouse_id, code) DO NOTHING;

-- Supplier system location (same shape as WMSRepo.GetOrCreateSystemLocation: code @VENDOR, no warehouse)
INSERT INTO warehouse_locations (tenant_id, warehouse_id, code, name, type)
SELECT t.id, NULL, '@VENDOR', 'VENDOR', 'VENDOR'
FROM tenants t
WHERE t.id = '550e8400-e29b-41d4-a716-446655440000'
  AND NOT EXISTS (SELECT 1 FROM warehouse_locations l WHERE l.tenant_id = t.id AND l.type = 'VENDOR' AND l.warehouse_id IS NULL);

-- Products
INSERT INTO products (tenant_id, name, sku, price, cost_price)
SELECT t.id, p.name, p.sku, p.price, p.cost
FROM tenants t
CROSS JOIN (VALUES
    ('Susu UHT Full Cream 1L',  'DEMO-SUSU-UHT-1L', 18500, 15000),
    ('Yogurt Stroberi 200ml',   'DEMO-YOGURT-200',   9500,  7000),
    ('Roti Tawar Kupas',        'DEMO-ROTI-TAWAR',  16000, 12500),
    ('Beras Premium 5kg',       'DEMO-BERAS-5KG',   78000, 70000),
    ('Minyak Goreng 2L',        'DEMO-MINYAK-2L',   36000, 32000),
    ('Gula Pasir 1kg',          'DEMO-GULA-1KG',    17500, 15500),
    ('Kopi Bubuk 200g',         'DEMO-KOPI-200',    28000, 23000),
    ('Air Mineral 600ml',       'DEMO-AIR-600',      4000,  2800)
) AS p(name, sku, price, cost)
WHERE t.id = '550e8400-e29b-41d4-a716-446655440000'
  -- SKU is unique only among active products (partial index, soft delete)
  AND NOT EXISTS (SELECT 1 FROM products x WHERE x.tenant_id = t.id AND x.sku = p.sku AND x.deleted_at IS NULL);

-- Batches: (sku, batch_number, days until expiry, quantity, rack)
CREATE TEMP TABLE demo_seed_batches (sku TEXT, batch_number TEXT, days INT, qty NUMERIC, rack TEXT) ON COMMIT DROP;
INSERT INTO demo_seed_batches VALUES
    ('DEMO-SUSU-UHT-1L', 'DEMO-SUSU-A', -5,  24, 'RAK-A1'),  -- already expired: blocked from sales
    ('DEMO-SUSU-UHT-1L', 'DEMO-SUSU-B',  4,  48, 'RAK-A1'),  -- critical
    ('DEMO-SUSU-UHT-1L', 'DEMO-SUSU-C', 150, 120, 'RAK-A1'),
    ('DEMO-YOGURT-200',  'DEMO-YOG-A',  -2,  12, 'RAK-A1'),  -- already expired
    ('DEMO-YOGURT-200',  'DEMO-YOG-B',  20,  60, 'RAK-A1'),  -- soon
    ('DEMO-ROTI-TAWAR',  'DEMO-ROTI-A',  3,  30, 'RAK-A2'),  -- critical
    ('DEMO-ROTI-TAWAR',  'DEMO-ROTI-B', 10,  40, 'RAK-A2'),  -- soon
    ('DEMO-BERAS-5KG',   'DEMO-BERAS-A', 365, 200, 'RAK-B1'),
    ('DEMO-MINYAK-2L',   'DEMO-MINYAK-A', 300, 150, 'RAK-B1'),
    ('DEMO-GULA-1KG',    'DEMO-GULA-A', 400, 180, 'RAK-B1'),
    ('DEMO-KOPI-200',    'DEMO-KOPI-A', 200,  90, 'RAK-A2'),
    ('DEMO-AIR-600',     'DEMO-AIR-A',  250, 500, 'RAK-B1');

INSERT INTO stock_batches (tenant_id, product_id, batch_number, expiry_date, status)
SELECT p.tenant_id, p.id, s.batch_number, CURRENT_DATE + s.days, 'RELEASED'
FROM demo_seed_batches s
JOIN products p ON p.tenant_id = '550e8400-e29b-41d4-a716-446655440000' AND p.sku = s.sku AND p.deleted_at IS NULL
ON CONFLICT (tenant_id, product_id, batch_number) DO NOTHING;

-- Opening stock: one ledger movement per batch, supplier -> rack
INSERT INTO stock_movements (tenant_id, movement_number, product_id, source_location_id, dest_location_id,
                             quantity, unit_cost, status, reference_type, reference_id, batch_id)
SELECT p.tenant_id, 'DEMO-OPEN-' || s.batch_number, p.id, v.id, rack.id,
       s.qty, p.cost_price, 'DONE', 'ADJUSTMENT', b.id, b.id
FROM demo_seed_batches s
JOIN products p ON p.tenant_id = '550e8400-e29b-41d4-a716-446655440000' AND p.sku = s.sku AND p.deleted_at IS NULL
JOIN stock_batches b ON b.tenant_id = p.tenant_id AND b.product_id = p.id AND b.batch_number = s.batch_number
JOIN warehouses w ON w.tenant_id = p.tenant_id AND w.code = 'DEMO-JKT'
JOIN warehouse_locations rack ON rack.tenant_id = p.tenant_id AND rack.warehouse_id = w.id AND rack.code = s.rack
JOIN LATERAL (
    SELECT l.id FROM warehouse_locations l
    WHERE l.tenant_id = p.tenant_id AND l.type = 'VENDOR' AND l.warehouse_id IS NULL
    ORDER BY l.created_at, l.id LIMIT 1
) v ON true
ON CONFLICT (tenant_id, movement_number) DO NOTHING;

-- Customers for Surat Jalan demos
INSERT INTO customers (tenant_id, name, phone, address)
SELECT t.id, c.name, c.phone, c.address
FROM tenants t
CROSS JOIN (VALUES
    ('Toko Sumber Rejeki (Demo)', '081200000001', 'Jl. Pasar Baru No. 10, Jakarta'),
    ('Minimarket Berkah (Demo)',  '081200000002', 'Jl. Raya Bogor Km 21, Jakarta')
) AS c(name, phone, address)
WHERE t.id = '550e8400-e29b-41d4-a716-446655440000'
  AND NOT EXISTS (SELECT 1 FROM customers x WHERE x.tenant_id = t.id AND x.name = c.name);

COMMIT;

-- ==================
-- DOWN
-- ==================
-- Tidak dihapus otomatis: data demo bisa sudah dipakai transaksi uji.
