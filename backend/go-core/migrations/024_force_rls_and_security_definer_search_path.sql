-- Migration 024: FORCE ROW LEVEL SECURITY on tenants and users, and pin search_path on all SECURITY DEFINER functions.
-- Mitigates CWE-284 (table owner bypassing RLS) and CWE-426 (untrusted search_path).

ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

ALTER FUNCTION get_user_by_email(TEXT) SET search_path = public, pg_temp;
ALTER FUNCTION get_user_by_id(UUID, UUID) SET search_path = public, pg_temp;
ALTER FUNCTION set_password_reset_token(UUID, TEXT, TIMESTAMPTZ) SET search_path = public, pg_temp;
ALTER FUNCTION get_user_by_password_reset_token(TEXT) SET search_path = public, pg_temp;
ALTER FUNCTION update_user_password(UUID, TEXT) SET search_path = public, pg_temp;
