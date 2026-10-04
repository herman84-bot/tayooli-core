-- Migration 019: SECURITY DEFINER functions for password reset
-- The forgot-password / reset-password endpoints run WITHOUT tenant context
-- (public auth routes), so plain UPDATE/SELECT on `users` is silently
-- filtered by RLS. These functions run as the table owner and bypass RLS,
-- following the same pattern as get_user_by_email / get_user_by_id (011).

CREATE OR REPLACE FUNCTION set_password_reset_token(p_user_id UUID, p_token TEXT, p_expires_at TIMESTAMPTZ)
RETURNS VOID
LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    UPDATE users
       SET password_reset_token = p_token,
           password_reset_expires_at = p_expires_at,
           updated_at = NOW()
     WHERE id = p_user_id;
END;
$$;

CREATE OR REPLACE FUNCTION get_user_by_password_reset_token(p_token TEXT)
RETURNS TABLE(id UUID, tenant_id UUID, email VARCHAR, password_hash VARCHAR, role VARCHAR, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    RETURN QUERY SELECT u.id, u.tenant_id, u.email, u.password_hash, u.role, u.created_at, u.updated_at
    FROM users u
    WHERE u.password_reset_token = p_token
      AND u.password_reset_expires_at > NOW()
    LIMIT 1;
END;
$$;

CREATE OR REPLACE FUNCTION update_user_password(p_user_id UUID, p_password_hash TEXT)
RETURNS VOID
LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    UPDATE users
       SET password_hash = p_password_hash,
           password_reset_token = NULL,
           password_reset_expires_at = NULL,
           updated_at = NOW()
     WHERE id = p_user_id;
END;
$$;
