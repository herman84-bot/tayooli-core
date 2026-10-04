-- 012_e2e_test_users.sql
-- E2E test users + demo vendor for the Playwright suite (frontend/e2e).
--
-- All users below share the password: password123
-- bcrypt hash: $2b$10$QwKrWvf4JTGvRjoqoR993ecO1bHhLPGlC3DA.r7iE3joi/ovc.rW2
--
-- Upserts are idempotent and safe to apply in any environment (the CI
-- migration loop applies every file in this directory).

INSERT INTO tenants (id, name, plan)
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'PT Test Indonesia', 'enterprise')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES
('550e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440000', 'admin@test.com',      '$2b$10$QwKrWvf4JTGvRjoqoR993ecO1bHhLPGlC3DA.r7iE3joi/ovc.rW2', 'admin'),
('550e8400-e29b-41d4-a716-446655440101', '550e8400-e29b-41d4-a716-446655440000', 'accountant@test.com', '$2b$10$QwKrWvf4JTGvRjoqoR993ecO1bHhLPGlC3DA.r7iE3joi/ovc.rW2', 'accountant'),
('550e8400-e29b-41d4-a716-446655440102', '550e8400-e29b-41d4-a716-446655440000', 'approver@test.com',    '$2b$10$QwKrWvf4JTGvRjoqoR993ecO1bHhLPGlC3DA.r7iE3joi/ovc.rW2', 'approver')
ON CONFLICT (id) DO UPDATE
  SET password_hash = EXCLUDED.password_hash,
      role          = EXCLUDED.role,
      tenant_id     = EXCLUDED.tenant_id;

-- Demo vendor used by the invoice-creation flow in E2E specs
-- (referenced by UUID from /dashboard/invoices/new).
INSERT INTO vendors (id, tenant_id, name, status)
VALUES ('550e8400-e29b-41d4-a716-446655440200', '550e8400-e29b-41d4-a716-446655440000', 'Test Vendor Inc.', 'active')
ON CONFLICT (id) DO NOTHING;
