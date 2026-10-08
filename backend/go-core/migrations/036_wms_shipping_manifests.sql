-- 036_wms_shipping_manifests.sql
-- Sprint 4 Master PRD WMS: Shipping Manifests, Loading Scan, Courier Multi-DO consolidation
-- ADR-014 Invariant 1 & Master PRD §3.2, §4.1

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'delivery_order_status' AND e.enumlabel = 'STAGED'
    ) THEN
        ALTER TYPE delivery_order_status ADD VALUE 'STAGED';
    END IF;
END $$;

BEGIN;

CREATE TABLE IF NOT EXISTS shipping_manifests (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id         UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    manifest_number      VARCHAR(100) NOT NULL,
    expedition_name      VARCHAR(100) NOT NULL,
    driver_name          VARCHAR(100) NOT NULL,
    vehicle_plate        VARCHAR(50) NOT NULL,
    driver_phone         VARCHAR(50) NULL,
    total_packages       INTEGER NOT NULL DEFAULT 0 CHECK (total_packages >= 0),
    total_weight_kg      NUMERIC(10, 3) NOT NULL DEFAULT 0 CHECK (total_weight_kg >= 0),
    status               VARCHAR(50) NOT NULL DEFAULT 'STAGED' 
                         CHECK (status IN ('STAGED', 'LOADED', 'DISPATCHED', 'CANCELLED')),
    driver_signature_svg TEXT NULL,
    notes                TEXT NULL,
    created_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    dispatched_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    dispatched_at        TIMESTAMPTZ NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, manifest_number)
);

CREATE INDEX IF NOT EXISTS idx_shipping_manifests_tenant_wh 
    ON shipping_manifests(tenant_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_shipping_manifests_status 
    ON shipping_manifests(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_shipping_manifests_expedition 
    ON shipping_manifests(tenant_id, expedition_name);

ALTER TABLE shipping_manifests ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipping_manifests FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS shipping_manifests_tenant_isolation ON shipping_manifests;
CREATE POLICY shipping_manifests_tenant_isolation ON shipping_manifests
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE delivery_orders
    ADD COLUMN IF NOT EXISTS manifest_id UUID REFERENCES shipping_manifests(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS loading_scanned_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS loading_scanned_by UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_orders_manifest ON delivery_orders(tenant_id, manifest_id);

COMMIT;

-- DOWN
BEGIN;
DROP INDEX IF EXISTS idx_delivery_orders_manifest;
ALTER TABLE delivery_orders
    DROP COLUMN IF EXISTS loading_scanned_by,
    DROP COLUMN IF EXISTS loading_scanned_at,
    DROP COLUMN IF EXISTS manifest_id;
DROP TABLE IF EXISTS shipping_manifests CASCADE;
COMMIT;
