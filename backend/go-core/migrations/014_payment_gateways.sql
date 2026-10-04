-- Migration: 014_payment_gateways.sql
-- Description: Multi-tenant payment gateway configuration and transaction logging with strict RLS (ADR-008).

-- 1. Create table tenant_payment_configs
CREATE TABLE IF NOT EXISTS tenant_payment_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    slug VARCHAR(128),
    api_key TEXT,
    server_key TEXT,
    client_key VARCHAR(255),
    is_production BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    settlement_bank_name VARCHAR(64),
    settlement_bank_account VARCHAR(64),
    settlement_holder_name VARCHAR(128),
    gateway_fee_percent NUMERIC(5,2) NOT NULL DEFAULT 0.70,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_provider UNIQUE (tenant_id, provider)
);

-- 2. Create table payment_transactions
CREATE TABLE IF NOT EXISTS payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    invoice_id VARCHAR(64) NOT NULL,
    order_id VARCHAR(64) NOT NULL UNIQUE,
    amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    method VARCHAR(32) NOT NULL,
    payment_link TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    is_demo BOOLEAN NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Indexes for performance
CREATE INDEX IF NOT EXISTS idx_tenant_payment_configs_tenant_id ON tenant_payment_configs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_tenant_id ON payment_transactions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_order_id ON payment_transactions(order_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_invoice_id ON payment_transactions(invoice_id);

-- 4. Enable & Force Row-Level Security (RLS)
ALTER TABLE tenant_payment_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_payment_configs FORCE ROW LEVEL SECURITY;

ALTER TABLE payment_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_transactions FORCE ROW LEVEL SECURITY;

-- 5. Multi-tenant RLS Policies for tenant_payment_configs
DROP POLICY IF EXISTS tenant_payment_configs_isolation ON tenant_payment_configs;
CREATE POLICY tenant_payment_configs_isolation ON tenant_payment_configs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- 6. Multi-tenant RLS Policies for payment_transactions
DROP POLICY IF EXISTS payment_transactions_isolation ON payment_transactions;
CREATE POLICY payment_transactions_isolation ON payment_transactions
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
