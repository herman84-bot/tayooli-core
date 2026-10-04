-- 009_vendor_management.sql: Vendor master and rating tables for vendor management module.
-- vendors holds per-tenant vendor records; vendor_ratings holds per-user ratings (1-5).
-- avg_rating and rating_count on vendors are maintained automatically via trigger.
-- ==================
-- UP
-- ==================
BEGIN;

-- 1. Create vendors table
CREATE TABLE IF NOT EXISTS vendors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    email           VARCHAR(255),
    phone           VARCHAR(50),
    address         TEXT,
    bank_account    VARCHAR(100),
    bank_name       VARCHAR(100),
    tax_id          VARCHAR(50),          -- NPWP for Indonesian vendors
    status          VARCHAR(50) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'inactive', 'suspended')),
    avg_rating      NUMERIC(3,2) NOT NULL DEFAULT 0,   -- cached average, updated by trigger
    rating_count    INTEGER NOT NULL DEFAULT 0,         -- cached count, updated by trigger
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite unique enables FK from child tables to (id, tenant_id),
    -- preventing cross-tenant references at the schema level.
    UNIQUE (id, tenant_id)
);

-- 2. Create vendor_ratings table
CREATE TABLE IF NOT EXISTS vendor_ratings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    vendor_id       UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    rated_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    rating          INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One rating per user per vendor (enforced per tenant via RLS)
    UNIQUE (tenant_id, vendor_id, rated_by)
);

-- 3. Indexes for vendors
CREATE INDEX IF NOT EXISTS idx_vendors_tenant_id
    ON vendors(tenant_id);

CREATE INDEX IF NOT EXISTS idx_vendors_tenant_status
    ON vendors(tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_vendors_tenant_name
    ON vendors(tenant_id, name);

-- 4. Indexes for vendor_ratings
CREATE INDEX IF NOT EXISTS idx_vendor_ratings_tenant_id
    ON vendor_ratings(tenant_id);

CREATE INDEX IF NOT EXISTS idx_vendor_ratings_tenant_vendor
    ON vendor_ratings(tenant_id, vendor_id);

-- 5. Rating aggregation trigger: keeps vendors.avg_rating and rating_count in sync
--    on INSERT, UPDATE, and DELETE of vendor_ratings rows.

CREATE OR REPLACE FUNCTION update_vendor_rating_aggregates()
RETURNS TRIGGER AS $$
DECLARE
    v_vendor_id UUID;
BEGIN
    -- Determine which vendor_id to recalculate.
    IF (TG_OP = 'DELETE') THEN
        v_vendor_id := OLD.vendor_id;
    ELSE
        v_vendor_id := NEW.vendor_id;
    END IF;

    UPDATE vendors
    SET avg_rating   = COALESCE(sub.avg, 0),
        rating_count = COALESCE(sub.cnt, 0),
        updated_at   = NOW()
    FROM (
        SELECT vendor_id,
               ROUND(AVG(rating)::numeric, 2) AS avg,
               COUNT(*)::int                   AS cnt
        FROM vendor_ratings
        WHERE vendor_id = v_vendor_id
        GROUP BY vendor_id
    ) sub
    WHERE vendors.id = sub.vendor_id
      AND vendors.id = v_vendor_id;

    -- If no ratings remain, reset to zero (the subquery returns no rows,
    -- so the UPDATE above doesn't match — handle explicitly).
    IF NOT FOUND THEN
        UPDATE vendors
        SET avg_rating   = 0,
            rating_count = 0,
            updated_at   = NOW()
        WHERE id = v_vendor_id;
    END IF;

    RETURN NULL;  -- AFTER trigger, return value is ignored
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_vendor_rating_aggregate
    AFTER INSERT OR UPDATE OR DELETE ON vendor_ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_vendor_rating_aggregates();

-- 6. RLS: enable and force row-level security on both tables
ALTER TABLE vendors        ENABLE ROW LEVEL SECURITY;
ALTER TABLE vendors        FORCE ROW LEVEL SECURITY;
ALTER TABLE vendor_ratings ENABLE ROW LEVEL SECURITY;
ALTER TABLE vendor_ratings FORCE ROW LEVEL SECURITY;

-- 7. RLS policies — idempotent (DROP IF EXISTS before CREATE)
--    Uses same NULLIF pattern as 006/008 for safe session-variable handling.

-- vendors policies
DROP POLICY IF EXISTS vendors_tenant_isolation_select ON vendors;
CREATE POLICY vendors_tenant_isolation_select ON vendors
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS vendors_tenant_isolation_insert ON vendors;
CREATE POLICY vendors_tenant_isolation_insert ON vendors
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS vendors_tenant_isolation_update ON vendors;
CREATE POLICY vendors_tenant_isolation_update ON vendors
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS vendors_tenant_isolation_delete ON vendors;
CREATE POLICY vendors_tenant_isolation_delete ON vendors
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- vendor_ratings policies
DROP POLICY IF EXISTS vendor_ratings_tenant_isolation_select ON vendor_ratings;
CREATE POLICY vendor_ratings_tenant_isolation_select ON vendor_ratings
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS vendor_ratings_tenant_isolation_insert ON vendor_ratings;
CREATE POLICY vendor_ratings_tenant_isolation_insert ON vendor_ratings
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS vendor_ratings_tenant_isolation_update ON vendor_ratings;
CREATE POLICY vendor_ratings_tenant_isolation_update ON vendor_ratings
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS vendor_ratings_tenant_isolation_delete ON vendor_ratings;
CREATE POLICY vendor_ratings_tenant_isolation_delete ON vendor_ratings
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

DROP POLICY IF EXISTS vendor_ratings_tenant_isolation_delete ON vendor_ratings;
DROP POLICY IF EXISTS vendor_ratings_tenant_isolation_update ON vendor_ratings;
DROP POLICY IF EXISTS vendor_ratings_tenant_isolation_insert ON vendor_ratings;
DROP POLICY IF EXISTS vendor_ratings_tenant_isolation_select ON vendor_ratings;

DROP POLICY IF EXISTS vendors_tenant_isolation_delete ON vendors;
DROP POLICY IF EXISTS vendors_tenant_isolation_update ON vendors;
DROP POLICY IF EXISTS vendors_tenant_isolation_insert ON vendors;
DROP POLICY IF EXISTS vendors_tenant_isolation_select ON vendors;

DROP TRIGGER IF EXISTS trg_vendor_rating_aggregate ON vendor_ratings;
DROP FUNCTION IF EXISTS update_vendor_rating_aggregates();

DROP INDEX IF EXISTS idx_vendor_ratings_tenant_vendor;
DROP INDEX IF EXISTS idx_vendor_ratings_tenant_id;
DROP INDEX IF EXISTS idx_vendors_tenant_name;
DROP INDEX IF EXISTS idx_vendors_tenant_status;
DROP INDEX IF EXISTS idx_vendors_tenant_id;

DROP TABLE IF EXISTS vendor_ratings;
DROP TABLE IF EXISTS vendors;

COMMIT;
