-- Telegram QRIS Payment Orders
CREATE TABLE IF NOT EXISTS telegram_payment_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_code VARCHAR(32) UNIQUE NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_email VARCHAR(255) NOT NULL,
    plan VARCHAR(32) NOT NULL,
    amount INTEGER NOT NULL,
    period VARCHAR(16) NOT NULL DEFAULT 'monthly',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    proof_file_id VARCHAR(255),
    approved_by VARCHAR(255),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telegram_payment_orders_tenant ON telegram_payment_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_telegram_payment_orders_status ON telegram_payment_orders(status);
CREATE INDEX IF NOT EXISTS idx_telegram_payment_orders_code ON telegram_payment_orders(order_code);

-- Row Level Security (RLS) policy
ALTER TABLE telegram_payment_orders ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON telegram_payment_orders
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID);

COMMENT ON TABLE telegram_payment_orders IS 'Tracks QRIS payments initiated via Telegram bot';
