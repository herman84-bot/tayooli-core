-- 013_seed_approval_users.sql: Seed accountant and approver test users for the
-- approval workflow E2E tests (approval-flow.spec.ts).
--
-- Credentials (password: password123 — bcrypt cost 10, $2a$ prefix; the same
-- scheme/cost used by the existing admin seed and verified via
-- bcrypt.CompareHashAndPassword in the login flow):
--   accountant@test.com / password123  (role: accountant)
--   approver@test.com   / password123  (role: approver)
--
-- Both users belong to the seed tenant '550e8400-e29b-41d4-a716-446655440000'
-- (PT Test Indonesia) so role-gated endpoints (POST /invoices, /approvals/*)
-- and the approval workflow remain tenant-isolated.
--
-- Idempotent: safe to re-run. No RLS policy changes — the tenant_isolation
-- policy on users already covers these rows via tenant_id.
--
-- NOTE: RLS on `users`/`tenants` filters by the `app.current_tenant_id` session
-- var, so the migration sets it for the seed tenant before INSERTing (the same
-- mechanism the API uses via TenantMiddleware). Run as the `tayooli` role:
--
--   psql -U tayooli -d tayooli -f 013_seed_approval_users.sql
-- ==================
-- UP
-- ==================
BEGIN;

SET LOCAL app.current_tenant_id = '550e8400-e29b-41d4-a716-446655440000';

INSERT INTO tenants (id, name, plan)
VALUES ('550e8400-e29b-41d4-a716-446655440000', 'PT Test Indonesia', 'enterprise')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, tenant_id, email, password_hash, role)
VALUES
    ('550e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440000', 'accountant@test.com', '$2a$10$Zq0rS90ys0BtyhAz.Bf5X.U6GgvxkxD9lm9XjgtIr0VhluP82sgya', 'accountant'),
    ('550e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440000', 'approver@test.com',   '$2a$10$Zq0rS90ys0BtyhAz.Bf5X.U6GgvxkxD9lm9XjgtIr0VhluP82sgya', 'approver')
-- CICD NOTE: keep SET LOCAL above the INSERTs (RLS filters by this session var).
ON CONFLICT (email) DO UPDATE
    SET password_hash = EXCLUDED.password_hash,
        role          = EXCLUDED.role;

COMMIT;

-- ==================
-- DOWN
-- ==================
BEGIN;

SET LOCAL app.current_tenant_id = '550e8400-e29b-41d4-a716-446655440000';

DELETE FROM users WHERE email IN ('accountant@test.com', 'approver@test.com');

COMMIT;
