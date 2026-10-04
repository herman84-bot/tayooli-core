-- Insert test tenant
INSERT INTO tenants (id, name, plan) VALUES 
('550e8400-e29b-41d4-a716-446655440000', 'PT Test Indonesia', 'enterprise');

-- Insert test user (password: test123 - this is just a hash for testing)
INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES 
('550e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440000', 'admin@test.com', '$2b$12$LQv3c1yqBo9SkvXS8QvZ.eN8Kz9ZQvZ.eN8Kz9ZQvZ.eN8Kz9ZQv', 'admin');

-- Insert test invoice
INSERT INTO invoices (id, tenant_id, vendor_id, invoice_number, amount, currency, status, ai_confidence_score, due_date) VALUES 
('550e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440000', 'VENDOR001', 'INV-2024-001', 1500000.00, 'IDR', 'draft', 0.95, '2024-12-31');
