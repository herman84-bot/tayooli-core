-- 034_wms_qc_inspections_quarantine.sql: Sprint 2 (PRD master section 6, ADR-014 Invariant 3, PDF-02).
-- Adopted patterns:
--   OCA/wms quality flow: received goods -> QC (pass / quarantine) before they become storable.
--   sentry-wms 1.1: quarantine bins are never pickable (separate location type QUARANTINE).
--   ADR-014 Invariant 3: a quarantine movement needs a qc_inspections row plus a BAK document
--   with driver signature; inbound goods never go straight to SCRAP without QC.
-- ==================
-- UP
-- ==================
BEGIN;

CREATE TABLE IF NOT EXISTS qc_inspections (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    receipt_id        UUID NOT NULL,
    warehouse_id      UUID NOT NULL,
    inspection_mode   VARCHAR(20) NOT NULL DEFAULT 'FULL',
    sample_qty        NUMERIC(20, 4),
    gross_cartons     INTEGER NOT NULL DEFAULT 0,
    status            VARCHAR(20) NOT NULL,
    total_checked_qty NUMERIC(20, 4) NOT NULL DEFAULT 0,
    total_passed_qty  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    total_damaged_qty NUMERIC(20, 4) NOT NULL DEFAULT 0,
    shortage_qty      NUMERIC(20, 4) NOT NULL DEFAULT 0,
    overage_qty       NUMERIC(20, 4) NOT NULL DEFAULT 0,
    bak_number        VARCHAR(100),
    bak_notes         TEXT,
    driver_name       VARCHAR(100),
    driver_signed     BOOLEAN NOT NULL DEFAULT FALSE,
    notes             TEXT,
    inspector_id      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (receipt_id, tenant_id) REFERENCES stock_receipts(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    CONSTRAINT chk_qc_mode CHECK (inspection_mode IN ('FULL', 'SAMPLING')),
    CONSTRAINT chk_qc_status CHECK (status IN ('QC_PASSED', 'QUARANTINED', 'QC_REJECTED')),
    CONSTRAINT chk_qc_sample CHECK (inspection_mode = 'FULL' OR (sample_qty IS NOT NULL AND sample_qty > 0)),
    CONSTRAINT chk_qc_cartons CHECK (gross_cartons >= 0),
    CONSTRAINT chk_qc_totals CHECK (total_checked_qty >= 0 AND total_passed_qty >= 0 AND total_damaged_qty >= 0
                                    AND shortage_qty >= 0 AND overage_qty >= 0),
    -- Invariant 3: damaged goods require a BAK number and a signed driver.
    CONSTRAINT chk_qc_bak CHECK (total_damaged_qty = 0
                                 OR (bak_number IS NOT NULL AND driver_signed AND length(btrim(COALESCE(driver_name, ''))) > 0)),
    UNIQUE (tenant_id, receipt_id),
    UNIQUE (tenant_id, bak_number),
    UNIQUE (id, tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_qc_inspections_wh ON qc_inspections(tenant_id, warehouse_id, created_at DESC);

CREATE TABLE IF NOT EXISTS qc_inspection_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    inspection_id UUID NOT NULL,
    product_id    UUID NOT NULL,
    batch_id      UUID NOT NULL,
    staged_qty    NUMERIC(20, 4) NOT NULL,
    checked_qty   NUMERIC(20, 4) NOT NULL,
    passed_qty    NUMERIC(20, 4) NOT NULL,
    damaged_qty   NUMERIC(20, 4) NOT NULL DEFAULT 0,
    shortage_qty  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    overage_qty   NUMERIC(20, 4) NOT NULL DEFAULT 0,
    damage_reason TEXT,
    FOREIGN KEY (inspection_id, tenant_id) REFERENCES qc_inspections(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (batch_id, tenant_id) REFERENCES stock_batches(id, tenant_id) ON DELETE RESTRICT,
    CONSTRAINT chk_qc_item_qty CHECK (staged_qty >= 0 AND checked_qty >= 0 AND damaged_qty >= 0
                                      AND damaged_qty <= checked_qty AND passed_qty = checked_qty - damaged_qty),
    CONSTRAINT chk_qc_item_reason CHECK (damaged_qty = 0 OR length(btrim(COALESCE(damage_reason, ''))) > 0),
    UNIQUE (inspection_id, batch_id)
);

ALTER TABLE qc_inspections ENABLE ROW LEVEL SECURITY;
ALTER TABLE qc_inspections FORCE ROW LEVEL SECURITY;
ALTER TABLE qc_inspection_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE qc_inspection_items FORCE ROW LEVEL SECURITY;
DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['qc_inspections', 'qc_inspection_items'] LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I_tenant_isolation ON %I', t, t);
        EXECUTE format('CREATE POLICY %I_tenant_isolation ON %I FOR ALL
            USING (tenant_id = NULLIF(current_setting(''app.current_tenant_id'', true), '''')::uuid)
            WITH CHECK (tenant_id = NULLIF(current_setting(''app.current_tenant_id'', true), '''')::uuid)', t, t);
    END LOOP;
END $$;

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;
DROP TABLE IF EXISTS qc_inspection_items;
DROP TABLE IF EXISTS qc_inspections;
COMMIT;
