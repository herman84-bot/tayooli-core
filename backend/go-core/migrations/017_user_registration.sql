-- Migration: 017_user_registration.sql
-- Description: User registration, email verification, workspace setup

-- 1. Add registration fields to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS full_name VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_expires_at TIMESTAMPTZ;

-- 2. Index for verification token lookup
CREATE INDEX IF NOT EXISTS idx_users_verification_token ON users(verification_token) WHERE verification_token IS NOT NULL;

-- 3. Tenants table already exists, but ensure company_name column
-- (tenants.name is used as company name)

-- 4. Workspace setup tracking
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS setup_completed_at TIMESTAMPTZ;
