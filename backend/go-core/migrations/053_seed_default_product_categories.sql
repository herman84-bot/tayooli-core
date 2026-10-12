-- 053_seed_default_product_categories.sql
-- Seed standard default product categories for all existing tenants.
-- Default categories: Makanan, Minuman, Kemasan, Bahan Baku, Alat Tulis.

-- ==================
-- UP
-- ==================
INSERT INTO product_categories (id, tenant_id, name, created_at)
SELECT gen_random_uuid(), t.id, c.name, NOW()
FROM tenants t
CROSS JOIN (
    VALUES
        ('Makanan'),
        ('Minuman'),
        ('Kemasan'),
        ('Bahan Baku'),
        ('Alat Tulis')
) AS c(name)
WHERE NOT EXISTS (
    SELECT 1 FROM product_categories pc
    WHERE pc.tenant_id = t.id AND lower(pc.name) = lower(c.name)
);

-- ==================
-- DOWN
-- ==================
-- Tidak menghapus kategori default secara otomatis karena mungkin sudah digunakan produk.
