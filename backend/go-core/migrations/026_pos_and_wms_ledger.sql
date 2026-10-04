-- 026_pos_and_wms_ledger.sql: POS checkout persistence, POS order items, and sales invoice safety.
-- Enforces per-tenant isolation via PostgreSQL Row-Level Security (RLS).
-- ==================
-- UP
-- ==================
BEGIN;

-- 1. Ensure sales_invoices table exists
CREATE TABLE IF NOT EXISTS sales_invoices (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sales_order_id UUID REFERENCES sales_orders(id) ON DELETE SET NULL,
    invoice_number VARCHAR(100) NOT NULL,
    amount        NUMERIC(15, 2) NOT NULL DEFAULT 0,
    status        VARCHAR(50) NOT NULL DEFAULT 'UNPAID',
    due_date      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_sales_invoice UNIQUE (tenant_id, invoice_number)
);

ALTER TABLE sales_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_invoices FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'sales_invoices' AND policyname = 'sales_invoices_tenant_isolation'
    ) THEN
        CREATE POLICY sales_invoices_tenant_isolation ON sales_invoices
            FOR ALL
            USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID);
    END IF;
END $$;

-- 2. POS Orders (Transaction Master)
CREATE TABLE IF NOT EXISTS pos_orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_number    VARCHAR(100) NOT NULL,
    customer_id     UUID REFERENCES customers(id) ON DELETE SET NULL,
    warehouse_id    UUID REFERENCES warehouses(id) ON DELETE SET NULL,
    subtotal        NUMERIC(15, 2) NOT NULL DEFAULT 0,
    tax_amount      NUMERIC(15, 2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(15, 2) NOT NULL DEFAULT 0,
    total_amount    NUMERIC(15, 2) NOT NULL DEFAULT 0,
    payment_method  VARCHAR(50) NOT NULL DEFAULT 'CASH',
    payment_amount  NUMERIC(15, 2) NOT NULL DEFAULT 0,
    change_amount   NUMERIC(15, 2) NOT NULL DEFAULT 0,
    sale_mode       VARCHAR(50) NOT NULL DEFAULT 'DIRECT',
    status          VARCHAR(50) NOT NULL DEFAULT 'COMPLETED',
    sales_order_id  UUID REFERENCES sales_orders(id) ON DELETE SET NULL,
    sales_invoice_id UUID REFERENCES sales_invoices(id) ON DELETE SET NULL,
    cashier_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    metadata        JSONB DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, order_number)
);

CREATE INDEX IF NOT EXISTS idx_pos_orders_tenant_created
    ON pos_orders(tenant_id, created_at DESC);

ALTER TABLE pos_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE pos_orders FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'pos_orders' AND policyname = 'pos_orders_tenant_isolation'
    ) THEN
        CREATE POLICY pos_orders_tenant_isolation ON pos_orders
            FOR ALL
            USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID);
    END IF;
END $$;

-- 3. POS Order Items (Line Items)
CREATE TABLE IF NOT EXISTS pos_order_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    pos_order_id    UUID NOT NULL REFERENCES pos_orders(id) ON DELETE CASCADE,
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name    VARCHAR(255) NOT NULL,
    sku             VARCHAR(100) NOT NULL,
    quantity        NUMERIC(15, 4) NOT NULL,
    price           NUMERIC(15, 2) NOT NULL,
    discount        NUMERIC(15, 2) NOT NULL DEFAULT 0,
    subtotal        NUMERIC(15, 2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pos_order_items_tenant_order
    ON pos_order_items(tenant_id, pos_order_id);

ALTER TABLE pos_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE pos_order_items FORCE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'pos_order_items' AND policyname = 'pos_order_items_tenant_isolation'
    ) THEN
        CREATE POLICY pos_order_items_tenant_isolation ON pos_order_items
            FOR ALL
            USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID)
            WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID);
    END IF;
END $$;

COMMIT;
