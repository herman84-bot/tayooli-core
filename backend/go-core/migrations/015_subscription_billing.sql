-- Migration: 015_subscription_billing.sql
-- Description: Subscription billing system for Tayooli ERP SaaS model

-- 1. Plan limits (configuration per plan tier)
CREATE TABLE IF NOT EXISTS plan_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan VARCHAR(50) NOT NULL UNIQUE,
    max_users INTEGER NOT NULL DEFAULT -1,
    max_vendors INTEGER NOT NULL DEFAULT -1,
    max_invoices_per_month INTEGER NOT NULL DEFAULT -1,
    max_ocr_per_month INTEGER NOT NULL DEFAULT -1,
    multi_entity BOOLEAN NOT NULL DEFAULT FALSE,
    api_access BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Tenant subscriptions
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Plan & status
    plan VARCHAR(50) NOT NULL DEFAULT 'trial',
    status VARCHAR(30) NOT NULL DEFAULT 'trialing',
    
    -- Trial
    trial_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trial_ends_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '14 days'),
    
    -- Billing cycle
    billing_period VARCHAR(10) NOT NULL DEFAULT 'monthly',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    
    -- Payment
    payment_provider VARCHAR(32) DEFAULT 'pakasir',
    
    -- Cancellation
    cancel_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id)
);

-- 3. Subscription invoices (tagihan langganan)
CREATE TABLE IF NOT EXISTS subscription_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES tenant_subscriptions(id),
    
    invoice_number VARCHAR(50) NOT NULL UNIQUE,
    amount INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    
    payment_link TEXT,
    payment_provider VARCHAR(32) DEFAULT 'pakasir',
    payment_reference VARCHAR(255),
    
    due_date TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Seed plan limits
INSERT INTO plan_limits (plan, max_users, max_vendors, max_invoices_per_month, max_ocr_per_month, multi_entity, api_access) VALUES
('trial',       3,   50,  100,  20,  FALSE, FALSE),
('starter',     3,   50,  100,  20,  FALSE, FALSE),
('bisnis',     10,   -1,   -1,  -1,  FALSE, TRUE),
('enterprise',  -1,  -1,   -1,  -1,  TRUE,  TRUE)
ON CONFLICT (plan) DO NOTHING;

-- 5. Create trial subscription for existing test users
-- This ensures existing demo users have a subscription record
INSERT INTO tenant_subscriptions (tenant_id, plan, status, trial_started_at, trial_ends_at, billing_period)
SELECT 
    id AS tenant_id,
    'trial' AS plan,
    'active' AS status,
    NOW() - INTERVAL '7 days' AS trial_started_at,
    NOW() + INTERVAL '7 days' AS trial_ends_at,
    'monthly' AS billing_period
FROM tenants
WHERE id NOT IN (SELECT tenant_id FROM tenant_subscriptions)
ON CONFLICT (tenant_id) DO NOTHING;

-- 6. Indexes for performance
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant_id ON tenant_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_status ON tenant_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscription_invoices_tenant_id ON subscription_invoices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_subscription_invoices_subscription_id ON subscription_invoices(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscription_invoices_status ON subscription_invoices(status);

-- 7. Row Level Security (RLS) policies
ALTER TABLE tenant_subscriptions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON tenant_subscriptions
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID);

ALTER TABLE subscription_invoices ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON subscription_invoices
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID);

