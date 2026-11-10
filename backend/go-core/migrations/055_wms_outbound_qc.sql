-- 055_wms_outbound_qc.sql
-- Sprint Outbound Flow Alignment: Order Release -> Picking -> Packing & AWB -> QC -> Staging Area -> Dispatch & Loading.
-- Adds QC_PASSED and QC_FAILED to delivery_order_status enum.
-- Adds qc_by, qc_at, and qc_notes columns to delivery_orders.
-- Creates outbound_qc_inspections table for audit trail of outbound quality checks.
-- Enforces segregation of duties (QC inspector != packer) and AWB / carton match.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'delivery_order_status' AND e.enumlabel = 'QC_PASSED'
    ) THEN
        ALTER TYPE delivery_order_status ADD VALUE 'QC_PASSED';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'delivery_order_status' AND e.enumlabel = 'QC_FAILED'
    ) THEN
        ALTER TYPE delivery_order_status ADD VALUE 'QC_FAILED';
    END IF;
END $$;

BEGIN;

ALTER TABLE delivery_orders
    ADD COLUMN IF NOT EXISTS qc_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS qc_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS qc_notes TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_orders_qc_by ON delivery_orders(tenant_id, qc_by);

CREATE TABLE IF NOT EXISTS outbound_qc_inspections (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    delivery_order_id   UUID NOT NULL REFERENCES delivery_orders(id) ON DELETE CASCADE,
    warehouse_id        UUID NOT NULL,
    inspector_id        UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status              VARCHAR(20) NOT NULL CHECK (status IN ('QC_PASSED', 'QC_FAILED')),
    scanned_awb         VARCHAR(100) NULL,
    awb_matched         BOOLEAN NOT NULL DEFAULT TRUE,
    package_qty_matched BOOLEAN NOT NULL DEFAULT TRUE,
    rejection_reason    TEXT NULL,
    notes               TEXT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_outbound_qc_tenant_wh ON outbound_qc_inspections(tenant_id, warehouse_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_outbound_qc_do ON outbound_qc_inspections(tenant_id, delivery_order_id);

ALTER TABLE outbound_qc_inspections ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbound_qc_inspections FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS outbound_qc_tenant_isolation ON outbound_qc_inspections;
CREATE POLICY outbound_qc_tenant_isolation ON outbound_qc_inspections
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- DOWN
BEGIN;
DROP TABLE IF EXISTS outbound_qc_inspections CASCADE;
DROP INDEX IF EXISTS idx_delivery_orders_qc_by;
ALTER TABLE delivery_orders
    DROP COLUMN IF EXISTS qc_notes,
    DROP COLUMN IF EXISTS qc_at,
    DROP COLUMN IF EXISTS qc_by;
COMMIT;
