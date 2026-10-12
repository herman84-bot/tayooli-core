-- 054_repair_pick_wave_tables.sql
-- Repair: production recorded 035_wms_outbound_waves_and_manifests.sql as applied
-- but pick_waves / picking_tasks / picking_task_items were never created
-- (live error: pq: relation "pick_waves" does not exist, 42P01).
-- Cause: 035 runs inside BEGIN..COMMIT; a single "already exists" clash rolled the
-- whole file back, and migrations.Run treats "already exists" as success and
-- records the file anyway. This file re-creates the outbound wave tables with
-- fully idempotent DDL so it is a no-op on databases where 035 succeeded.
-- Pattern: OCA wave release (docs/references/oca-wms-spec.md §1.3), ADR-014.

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

CREATE INDEX IF NOT EXISTS idx_pick_waves_route ON pick_waves(tenant_id, order_type, expedition_name);
CREATE INDEX IF NOT EXISTS idx_pick_waves_tenant_status ON pick_waves(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_pick_waves_tenant_wh ON pick_waves(tenant_id, warehouse_id);

ALTER TABLE pick_waves ENABLE ROW LEVEL SECURITY;
ALTER TABLE pick_waves FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS pick_waves_tenant_isolation ON pick_waves;
CREATE POLICY pick_waves_tenant_isolation ON pick_waves
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

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

-- picking_tasks may pre-exist without wave_id if it was created by another path.
ALTER TABLE picking_tasks ADD COLUMN IF NOT EXISTS wave_id UUID REFERENCES pick_waves(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_picking_tasks_wave ON picking_tasks(tenant_id, wave_id);
CREATE INDEX IF NOT EXISTS idx_picking_tasks_tenant_status ON picking_tasks(tenant_id, status);

ALTER TABLE picking_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE picking_tasks FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS picking_tasks_tenant_isolation ON picking_tasks;
CREATE POLICY picking_tasks_tenant_isolation ON picking_tasks
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

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
CREATE INDEX IF NOT EXISTS idx_pti_task ON picking_task_items(tenant_id, task_id, shelf_order);

ALTER TABLE picking_task_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE picking_task_items FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS picking_task_items_tenant_isolation ON picking_task_items;
CREATE POLICY picking_task_items_tenant_isolation ON picking_task_items
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- Down: intentionally a no-op. These tables are owned by 035; dropping them
-- here would destroy live wave/picking data.
