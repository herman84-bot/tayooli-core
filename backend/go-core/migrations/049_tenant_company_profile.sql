-- 049_tenant_company_profile.sql
-- Add company profile columns to tenants table for official document branding (Surat Jalan, Invoice, BAK, Struk POS).
-- Idempotent: uses ADD COLUMN IF NOT EXISTS.

-- ==================
-- UP
-- ==================
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS address TEXT,
    ADD COLUMN IF NOT EXISTS phone VARCHAR(50),
    ADD COLUMN IF NOT EXISTS email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS tax_id VARCHAR(50),
    ADD COLUMN IF NOT EXISTS website VARCHAR(255),
    ADD COLUMN IF NOT EXISTS logo_url TEXT,
    ADD COLUMN IF NOT EXISTS tagline VARCHAR(255),
    ADD COLUMN IF NOT EXISTS division VARCHAR(100);

-- ==================
-- DOWN
-- ==================
-- ALTER TABLE tenants
--     DROP COLUMN IF EXISTS address,
--     DROP COLUMN IF EXISTS phone,
--     DROP COLUMN IF EXISTS email,
--     DROP COLUMN IF EXISTS tax_id,
--     DROP COLUMN IF EXISTS website,
--     DROP COLUMN IF EXISTS logo_url,
--     DROP COLUMN IF EXISTS tagline,
--     DROP COLUMN IF EXISTS division;
