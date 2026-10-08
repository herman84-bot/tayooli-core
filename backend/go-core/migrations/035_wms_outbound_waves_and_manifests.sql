-- 035_wms_outbound_waves_and_manifests.sql
-- Sprint 3 Master PRD WMS: Outbound Wave Picking, Pack Station, Thermal AWB, Customer link & Actors
-- ADR-014 Invariant 1 & Sentry-WMS §1.1/§1.2/§1.3, OCA §1.3

BEGIN;

-- 1. Tambah kolom customer, pelaku (audit actor), kemasan, dan tipe order pada delivery_orders
ALTER TABLE delivery_orders
    ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS confirmed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS packed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS dispatched_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS package_weight_kg NUMERIC(10, 3) NULL CHECK (package_weight_kg IS NULL OR package_weight_kg >= 0),
    ADD COLUMN IF NOT EXISTS package_length_cm NUMERIC(10, 2) NULL CHECK (package_length_cm IS NULL OR package_length_cm >= 0),
    ADD COLUMN IF NOT EXISTS package_width_cm  NUMERIC(10, 2) NULL CHECK (package_width_cm IS NULL OR package_width_cm >= 0),
    ADD COLUMN IF NOT EXISTS package_height_cm NUMERIC(10, 2) NULL CHECK (package_height_cm IS NULL OR package_height_cm >= 0),
    ADD COLUMN IF NOT EXISTS packaging_type    VARCHAR(50) NULL,
    ADD COLUMN IF NOT EXISTS order_type        VARCHAR(50) NOT NULL DEFAULT 'DIRECT_DO';

CREATE INDEX IF NOT EXISTS idx_delivery_orders_customer ON delivery_orders(tenant_id, customer_id);
CREATE INDEX IF NOT EXISTS idx_delivery_orders_order_type ON delivery_orders(tenant_id, order_type);

-- 2. Tambah kolom batch_id (KO-1), is_free_item (PDF-01 bonus item), dan packed_qty (BE-07 pack station) pada delivery_order_items
ALTER TABLE delivery_order_items
    ADD COLUMN IF NOT EXISTS batch_id UUID REFERENCES stock_batches(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS is_free_item BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS packed_qty NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (packed_qty >= 0);

CREATE INDEX IF NOT EXISTS idx_doi_batch ON delivery_order_items(tenant_id, batch_id);

-- 3. Tabel pick_waves: gelombang pelepasan pesanan (Wave Release) per tipe order / rute (OCA §1.3, PDF-05)
CREATE TABLE IF NOT EXISTS pick_waves (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id       UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    wave_number        VARCHAR(100) NOT NULL,
    order_type         VARCHAR(50) NOT NULL DEFAULT 'DIRECT_DO'
                       CHECK (order_type IN ('DIRECT_DO', 'SALES_ORDER', 'MARKETPLACE', 'TRANSFER')),
    expedition_name    VARCHAR(100) NULL,
    route_zone         VARCHAR(100) NULL,
    status             VARCHAR(50) NOT NULL DEFAULT 'OPEN'
                       CHECK (status IN ('OPEN', 'RELEASED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED')),
    picker_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at         TIMESTAMPTZ NULL,
    completed_at       TIMESTAMPTZ NULL,
    notes              TEXT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, wave_number)
);

ALTER TABLE pick_waves ADD COLUMN IF NOT EXISTS expedition_name VARCHAR(100);
ALTER TABLE pick_waves ADD COLUMN IF NOT EXISTS route_zone VARCHAR(100);
CREATE INDEX IF NOT EXISTS idx_pick_waves_route ON pick_waves(tenant_id, order_type, expedition_name);

ALTER TABLE pick_waves ENABLE ROW LEVEL SECURITY;
ALTER TABLE pick_waves FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS pick_waves_tenant_isolation ON pick_waves;
CREATE POLICY pick_waves_tenant_isolation ON pick_waves
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_setting', true), '')::uuid OR tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE INDEX IF NOT EXISTS idx_pick_waves_tenant_status ON pick_waves(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_pick_waves_tenant_wh ON pick_waves(tenant_id, warehouse_id);

-- 4. Tabel picking_tasks: dokumen instruksi pengambilan barang terurut rak
CREATE TABLE IF NOT EXISTS picking_tasks (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    delivery_order_id  UUID NOT NULL REFERENCES delivery_orders(id) ON DELETE CASCADE,
    wave_id            UUID REFERENCES pick_waves(id) ON DELETE SET NULL,
    task_number        VARCHAR(100) NOT NULL,
    status             VARCHAR(50) NOT NULL DEFAULT 'PENDING'
                       CHECK (status IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'SHORTAGE', 'CANCELLED')),
    picker_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    started_at         TIMESTAMPTZ NULL,
    completed_at       TIMESTAMPTZ NULL,
    notes              TEXT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, task_number),
    UNIQUE (tenant_id, delivery_order_id)
);

ALTER TABLE picking_tasks ADD COLUMN IF NOT EXISTS wave_id UUID REFERENCES pick_waves(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_picking_tasks_wave ON picking_tasks(tenant_id, wave_id);

ALTER TABLE picking_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE picking_tasks FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS picking_tasks_tenant_isolation ON picking_tasks;
CREATE POLICY picking_tasks_tenant_isolation ON picking_tasks
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_setting', true), '')::uuid OR tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE INDEX IF NOT EXISTS idx_picking_tasks_tenant_status ON picking_tasks(tenant_id, status);

-- 4. Tabel picking_task_items: rincian baris pick per batch & lokasi rak terurut (shelf_order)
CREATE TABLE IF NOT EXISTS picking_task_items (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    task_id            UUID NOT NULL REFERENCES picking_tasks(id) ON DELETE CASCADE,
    product_id         UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    batch_id           UUID NOT NULL REFERENCES stock_batches(id) ON DELETE RESTRICT,
    source_location_id UUID NOT NULL REFERENCES warehouse_locations(id) ON DELETE RESTRICT,
    requested_qty      NUMERIC(20, 4) NOT NULL CHECK (requested_qty > 0),
    picked_qty         NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (picked_qty >= 0),
    damaged_qty        NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (damaged_qty >= 0),
    status             VARCHAR(50) NOT NULL DEFAULT 'PENDING'
                       CHECK (status IN ('PENDING', 'PICKED', 'SHORTAGE', 'DAMAGED')),
    shelf_order        INT NOT NULL DEFAULT 0,
    is_free_item       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE picking_task_items ADD COLUMN IF NOT EXISTS is_free_item BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE picking_task_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE picking_task_items FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS picking_task_items_tenant_isolation ON picking_task_items;
CREATE POLICY picking_task_items_tenant_isolation ON picking_task_items
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_setting', true), '')::uuid OR tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE INDEX IF NOT EXISTS idx_pti_task ON picking_task_items(tenant_id, task_id, shelf_order);

COMMIT;

-- DOWN
BEGIN;
DROP TABLE IF EXISTS picking_task_items CASCADE;
DROP TABLE IF EXISTS picking_tasks CASCADE;
DROP TABLE IF EXISTS pick_waves CASCADE;
ALTER TABLE delivery_order_items DROP COLUMN IF EXISTS packed_qty;
ALTER TABLE delivery_order_items DROP COLUMN IF EXISTS is_free_item;
ALTER TABLE delivery_order_items DROP COLUMN IF EXISTS batch_id;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS order_type;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS packaging_type;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS package_height_cm;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS package_width_cm;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS package_length_cm;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS package_weight_kg;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS dispatched_by;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS packed_by;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS confirmed_by;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS created_by;
ALTER TABLE delivery_orders DROP COLUMN IF EXISTS customer_id;
COMMIT;
