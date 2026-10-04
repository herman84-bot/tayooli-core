CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- Asset, Liability, Equity, Revenue, Expense
    balance NUMERIC(15, 4) DEFAULT 0.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, code)
);

ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS accounts_tenant_policy ON accounts;
CREATE POLICY accounts_tenant_policy ON accounts
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE TABLE IF NOT EXISTS journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    transaction_date DATE NOT NULL,
    description TEXT,
    reference_id VARCHAR(255),
    debit_amount NUMERIC(15, 4) DEFAULT 0.0000,
    credit_amount NUMERIC(15, 4) DEFAULT 0.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE journal_entries ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS journal_entries_tenant_policy ON journal_entries;
CREATE POLICY journal_entries_tenant_policy ON journal_entries
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE journal_entries ADD COLUMN IF NOT EXISTS reference_id VARCHAR(255);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_journal_entries_reference_account'
    ) THEN
        ALTER TABLE journal_entries ADD CONSTRAINT uq_journal_entries_reference_account UNIQUE(tenant_id, reference_id, account_id);
    END IF;
END $$;

-- Seed Chart of Accounts for test tenant
INSERT INTO accounts (id, tenant_id, code, name, type, balance) VALUES 
(gen_random_uuid(), '550e8400-e29b-41d4-a716-446655440000', '1001', 'Cash', 'Asset', 0.00),
(gen_random_uuid(), '550e8400-e29b-41d4-a716-446655440000', '2001', 'Accounts Payable', 'Liability', 0.00),
(gen_random_uuid(), '550e8400-e29b-41d4-a716-446655440000', '1101', 'Accounts Receivable', 'Asset', 0.00),
(gen_random_uuid(), '550e8400-e29b-41d4-a716-446655440000', '4001', 'Sales Revenue', 'Revenue', 0.00)
ON CONFLICT (tenant_id, code) DO NOTHING;
