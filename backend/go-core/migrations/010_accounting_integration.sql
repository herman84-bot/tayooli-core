ALTER TABLE journal_entries ADD COLUMN reference_id VARCHAR(255);
ALTER TABLE journal_entries ADD CONSTRAINT uq_journal_entries_reference_account UNIQUE(tenant_id, reference_id, account_id);

-- Seed Chart of Accounts for test tenant
INSERT INTO accounts (id, tenant_id, code, name, type, balance) VALUES 
(gen_random_uuid(), '550e8400-e29b-41d4-a716-446655440000', '1001', 'Cash', 'Asset', 0.00),
(gen_random_uuid(), '550e8400-e29b-41d4-a716-446655440000', '2001', 'Accounts Payable', 'Liability', 0.00),
(gen_random_uuid(), '550e8400-e29b-41d4-a716-446655440000', '1101', 'Accounts Receivable', 'Asset', 0.00),
(gen_random_uuid(), '550e8400-e29b-41d4-a716-446655440000', '4001', 'Sales Revenue', 'Revenue', 0.00)
ON CONFLICT (tenant_id, code) DO NOTHING;

