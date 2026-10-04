-- 008_payment_orders.sql: Payment orders table linking approved invoices to payment execution.
-- Status flow: draft -> pending_approval -> approved -> paid | rejected
-- ==================
-- UP
-- ==================
BEGIN;

-- 1. Create payment_orders table
CREATE TABLE IF NOT EXISTS payment_orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id      UUID NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    amount          NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    currency        VARCHAR(3) NOT NULL DEFAULT 'IDR',
    payment_method  VARCHAR(50) NOT NULL DEFAULT 'bank_transfer'
                        CHECK (payment_method IN (
                            'bank_transfer', 'virtual_account', 'credit_card',
                            'e_wallet', 'check', 'cash'
                        )),
    reference_number VARCHAR(100),
    status          VARCHAR(50) NOT NULL DEFAULT 'draft'
                        CHECK (status IN (
                            'draft', 'pending_approval', 'approved',
                            'processing', 'paid', 'rejected', 'cancelled'
                        )),
    notes           TEXT,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    approved_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite unique enables FK from future child tables to (id, tenant_id),
    -- preventing cross-tenant references at the schema level.
    UNIQUE (id, tenant_id)
);

-- 2. Partial unique index: only one active (non-rejected, non-cancelled) payment order per invoice.
--    This enforces the business rule without a CHECK constraint, which cannot reference other rows.
CREATE UNIQUE INDEX IF NOT EXISTS idx_po_one_active_per_invoice
    ON payment_orders(invoice_id)
    WHERE status NOT IN ('rejected', 'cancelled');

-- 3. Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_payment_orders_tenant_id
    ON payment_orders(tenant_id);

CREATE INDEX IF NOT EXISTS idx_payment_orders_tenant_status
    ON payment_orders(tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_payment_orders_tenant_invoice_id
    ON payment_orders(tenant_id, invoice_id);

CREATE INDEX IF NOT EXISTS idx_payment_orders_tenant_paid_at
    ON payment_orders(tenant_id, paid_at)
    WHERE paid_at IS NOT NULL;

-- 4. Add paid_at column to invoices for quick payment-date lookups in reporting
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_invoices_tenant_paid_at
    ON invoices(tenant_id, paid_at)
    WHERE paid_at IS NOT NULL;

-- 5. RLS: enable and force row-level security
ALTER TABLE payment_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_orders FORCE ROW LEVEL SECURITY;

-- 6. RLS policies — idempotent (DROP IF EXISTS before CREATE)
--    Uses same NULLIF pattern as 006 for safe session-variable handling.
DROP POLICY IF EXISTS payment_orders_tenant_isolation_select ON payment_orders;
CREATE POLICY payment_orders_tenant_isolation_select ON payment_orders
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS payment_orders_tenant_isolation_insert ON payment_orders;
CREATE POLICY payment_orders_tenant_isolation_insert ON payment_orders
    FOR INSERT WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS payment_orders_tenant_isolation_update ON payment_orders;
CREATE POLICY payment_orders_tenant_isolation_update ON payment_orders
    FOR UPDATE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS payment_orders_tenant_isolation_delete ON payment_orders;
CREATE POLICY payment_orders_tenant_isolation_delete ON payment_orders
    FOR DELETE USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

DROP POLICY IF EXISTS payment_orders_tenant_isolation_delete ON payment_orders;
DROP POLICY IF EXISTS payment_orders_tenant_isolation_update ON payment_orders;
DROP POLICY IF EXISTS payment_orders_tenant_isolation_insert ON payment_orders;
DROP POLICY IF EXISTS payment_orders_tenant_isolation_select ON payment_orders;

DROP INDEX IF EXISTS idx_invoices_tenant_paid_at;
ALTER TABLE invoices DROP COLUMN IF EXISTS paid_at;

DROP INDEX IF EXISTS idx_payment_orders_tenant_paid_at;
DROP INDEX IF EXISTS idx_payment_orders_tenant_invoice_id;
DROP INDEX IF EXISTS idx_payment_orders_tenant_status;
DROP INDEX IF EXISTS idx_payment_orders_tenant_id;
DROP INDEX IF EXISTS idx_po_one_active_per_invoice;

DROP TABLE IF EXISTS payment_orders;

COMMIT;
