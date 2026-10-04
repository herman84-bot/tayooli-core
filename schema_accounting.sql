CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- Asset, Liability, Equity, Revenue, Expense
    balance NUMERIC(15, 4) DEFAULT 0.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, code)
);

ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;
CREATE POLICY accounts_tenant_policy ON accounts
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE TABLE IF NOT EXISTS journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL REFERENCES accounts(id),
    transaction_date DATE NOT NULL,
    description TEXT,
    debit_amount NUMERIC(15, 4) DEFAULT 0.0000,
    credit_amount NUMERIC(15, 4) DEFAULT 0.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE journal_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY journal_entries_tenant_policy ON journal_entries
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
