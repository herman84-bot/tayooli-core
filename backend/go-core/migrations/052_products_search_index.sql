-- 052_products_search_index.sql
-- Server-side product search (GET /products?q=) uses name/sku ILIKE '%q%'.
-- A btree cannot serve a leading-wildcard LIKE; pg_trgm GIN indexes can
-- (https://www.postgresql.org/docs/15/pgtrgm.html#PGTRGM-INDEX).
-- Paging order (created_at DESC, id DESC) gets its own partial index.
--
-- pg_trgm is a trusted extension since PostgreSQL 13, so a database owner can
-- create it. If the role still lacks permission, skip the trigram indexes:
-- search stays correct, only slower on very large catalogs.

-- ==================
-- UP
-- ==================
CREATE INDEX IF NOT EXISTS idx_products_tenant_created_active
    ON products (tenant_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;

DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_trgm;
    CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL;
    CREATE INDEX IF NOT EXISTS idx_products_sku_trgm  ON products USING gin (sku  gin_trgm_ops) WHERE deleted_at IS NULL;
EXCEPTION WHEN insufficient_privilege OR undefined_file THEN
    RAISE NOTICE 'pg_trgm unavailable (%), product search runs without trigram index', SQLERRM;
END $$;

-- ==================
-- DOWN
-- ==================
-- DROP INDEX IF EXISTS idx_products_sku_trgm;
-- DROP INDEX IF EXISTS idx_products_name_trgm;
-- DROP INDEX IF EXISTS idx_products_tenant_created_active;
