-- 037_wms_docks_and_lpns.sql
-- Sprint 5 Master PRD WMS: Inbound Dock Scheduling, Queue & Pallet LPN Containerization
-- ADR-014 Invariant 1 & Master PRD §3.2, §4.5, §5

BEGIN;

-- 1. Inbound Docks Table
CREATE TABLE IF NOT EXISTS inbound_docks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id  UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    dock_code     VARCHAR(50) NOT NULL,
    dock_name     VARCHAR(100) NOT NULL,
    dock_type     VARCHAR(50) NOT NULL DEFAULT 'INBOUND' CHECK (dock_type IN ('INBOUND', 'OUTBOUND', 'CROSS_DOCK')),
    max_tonnage   NUMERIC(10, 2) NOT NULL DEFAULT 10.00 CHECK (max_tonnage >= 0),
    status        VARCHAR(50) NOT NULL DEFAULT 'AVAILABLE' CHECK (status IN ('AVAILABLE', 'OCCUPIED', 'MAINTENANCE')),
    notes         TEXT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, warehouse_id, dock_code)
);

CREATE INDEX IF NOT EXISTS idx_inbound_docks_wh ON inbound_docks(tenant_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inbound_docks_status ON inbound_docks(tenant_id, status);

ALTER TABLE inbound_docks ENABLE ROW LEVEL SECURITY;
ALTER TABLE inbound_docks FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS inbound_docks_tenant_isolation ON inbound_docks;
CREATE POLICY inbound_docks_tenant_isolation ON inbound_docks
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 2. Dock Appointments Table
CREATE TABLE IF NOT EXISTS dock_appointments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id        UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    dock_id             UUID NULL REFERENCES inbound_docks(id) ON DELETE SET NULL,
    appointment_number  VARCHAR(100) NOT NULL,
    vendor_name         VARCHAR(150) NOT NULL,
    vehicle_plate       VARCHAR(50) NOT NULL,
    driver_name         VARCHAR(100) NOT NULL,
    driver_phone        VARCHAR(50) NULL,
    po_reference        VARCHAR(100) NULL,
    estimated_arrival   TIMESTAMPTZ NOT NULL,
    actual_arrival      TIMESTAMPTZ NULL,
    start_unloading_at  TIMESTAMPTZ NULL,
    completed_at        TIMESTAMPTZ NULL,
    status              VARCHAR(50) NOT NULL DEFAULT 'SCHEDULED' 
                        CHECK (status IN ('SCHEDULED', 'ARRIVED', 'UNLOADING', 'COMPLETED', 'CANCELLED')),
    notes               TEXT NULL,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, appointment_number)
);

CREATE INDEX IF NOT EXISTS idx_dock_app_wh_status ON dock_appointments(tenant_id, warehouse_id, status);
CREATE INDEX IF NOT EXISTS idx_dock_app_eta ON dock_appointments(tenant_id, estimated_arrival);

ALTER TABLE dock_appointments ENABLE ROW LEVEL SECURITY;
ALTER TABLE dock_appointments FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS dock_appointments_tenant_isolation ON dock_appointments;
CREATE POLICY dock_appointments_tenant_isolation ON dock_appointments
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 3. Stock LPNs Table (License Plate Number / Pallet Container)
CREATE TABLE IF NOT EXISTS stock_lpns (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id  UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    lpn_code      VARCHAR(100) NOT NULL,
    location_id   UUID NOT NULL REFERENCES warehouse_locations(id) ON DELETE RESTRICT,
    pallet_type   VARCHAR(50) NOT NULL DEFAULT 'WOODEN' CHECK (pallet_type IN ('WOODEN', 'PLASTIC', 'METAL', 'CAGE')),
    status        VARCHAR(50) NOT NULL DEFAULT 'STAGED' CHECK (status IN ('STAGED', 'STORED', 'PICKED', 'SHIPPED', 'DECOMMISSIONED')),
    max_weight_kg NUMERIC(10, 2) NOT NULL DEFAULT 1000.00 CHECK (max_weight_kg >= 0),
    total_weight_kg NUMERIC(10, 2) NOT NULL DEFAULT 0.00 CHECK (total_weight_kg >= 0),
    notes         TEXT NULL,
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, warehouse_id, lpn_code)
);

CREATE INDEX IF NOT EXISTS idx_stock_lpns_wh_loc ON stock_lpns(tenant_id, warehouse_id, location_id);
CREATE INDEX IF NOT EXISTS idx_stock_lpns_status ON stock_lpns(tenant_id, status);

ALTER TABLE stock_lpns ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_lpns FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS stock_lpns_tenant_isolation ON stock_lpns;
CREATE POLICY stock_lpns_tenant_isolation ON stock_lpns
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 4. Stock LPN Items Table
CREATE TABLE IF NOT EXISTS stock_lpn_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    lpn_id      UUID NOT NULL REFERENCES stock_lpns(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    batch_id    UUID NOT NULL REFERENCES stock_batches(id) ON DELETE RESTRICT,
    quantity    NUMERIC(14, 4) NOT NULL CHECK (quantity > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, lpn_id, batch_id)
);

CREATE INDEX IF NOT EXISTS idx_stock_lpn_items_lpn ON stock_lpn_items(tenant_id, lpn_id);

ALTER TABLE stock_lpn_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_lpn_items FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS stock_lpn_items_tenant_isolation ON stock_lpn_items;
CREATE POLICY stock_lpn_items_tenant_isolation ON stock_lpn_items
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- DOWN
BEGIN;
DROP TABLE IF EXISTS stock_lpn_items CASCADE;
DROP TABLE IF EXISTS stock_lpns CASCADE;
DROP TABLE IF EXISTS dock_appointments CASCADE;
DROP TABLE IF EXISTS inbound_docks CASCADE;
COMMIT;
