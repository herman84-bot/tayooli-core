# PRD: Marketplace Sales Order Import & Channel SKU Mapping (Omnichannel O2C)

**Status:** Approved / Architecture Specification  
**Version:** 1.0.0  
**Author:** CTO Office  
**Target:** Tayooli ERP Engine (WMS + POS Expansion)  
**Date:** 2026-09-09  

---

## 1. Problem Statement
Bisnis distribusi dan ritel modern yang menggunakan Tayooli ERP menjual produk melalui berbagai kanal marketplace online (Shopee, Tokopedia, TikTok Shop, Lazada). Saat ini:
1. Tidak ada modul impor otomatis untuk batch penjualan marketplace (staf harus memasukkan pesanan satu per satu secara manual).
2. Kode SKU di marketplace seringkali berbeda dengan SKU internal gudang (misalnya: di Shopee `ABC-KOPISUSU-240ML` vs di gudang `MAS000123`, atau paket bundling 1 Dus = 24 Pcs).
3. Tidak ada penanganan idempotensi impor: file ekspor pesanan marketplace yang diunggah ulang berisiko menyebabkan *double counting* stok dan *duplicate order*.
4. Stok penjualan e-commerce tidak terpotong secara otomatis dari gudang fisik/rak online.

**Yang harus dibangun:**
1. Modul Impor Pesanan Penjualan Multi-Channel (CSV / Excel format Shopee, Tokopedia, TikTok Shop, Lazada, dan Generic CSV).
2. Mesin Resolusi SKU & Multiplier Otomatis dengan penanganan *Unmapped SKU* (1-click link ke internal product).
3. Batch Import Session dengan pelacakan idempotensi (`tenant_id`, `channel`, `external_order_id`).
4. Otomatisasi pembentukan Sales Order dan pemotongan stok WMS dari gudang e-commerce ke lokasi virtual `@CUSTOMER`.

---

## 2. Benchmark Repositori Acuan (OpenOMS, ERPNext, Odoo)

| Repositori | Pola Desain (*Pattern*) yang Diadopsi | Implementasi di Tayooli ERP |
|---|---|---|
| **OpenOMS** | *Order Ingestion & Channel Normalization* | Penyeragaman berbagai format pesanan eksternal menjadi entitas kanonikal `MarketplaceOrder` dan `MarketplaceOrderItem`. |
| **ERPNext (Data Import)** | *Idempotency & Batch Session Tracker* | Tabel `marketplace_import_batches` untuk melacak status batch (`PENDING`, `PROCESSING`, `COMPLETED`, `FAILED`), jumlah baris sukses, gagal, dan SKU belum terpetakan. Constraint `UNIQUE(tenant_id, channel, external_order_id)` mencegah duplikasi pesanan. |
| **Odoo (E-commerce Connector)** | *Double-Entry Stock Picking to @CUSTOMER* | Saat pesanan marketplace disetujui / diproses, stok fisik dipotong dari gudang/rak online langsung ke `@CUSTOMER` via `StockMovement`. |
| **Dolibarr (Product Aliasing)** | *Frictionless SKU Auto-Mapping* | Jika ditemukan SKU marketplace yang belum terdaftar di `product_sku_mappings`, sistem memfasilitasi pemetaan instan (link ke internal product) dan me-reproses baris pesanan tanpa perlu re-upload file. |

---

## 3. Scope (In-Scope & Out-of-Scope)

### In-Scope:
- Tabel database: `marketplace_import_batches`, `marketplace_orders`, `marketplace_order_items`.
- Parser file tabular (CSV / TSV / JSON) untuk template Shopee, Tokopedia, TikTok Shop, Lazada, dan Generic CSV.
- Deduplikasi otomatis pesanan berdasarkan nomor pesanan marketplace (`external_order_id`).
- SKU Resolver: `external_sku` $\to$ `product_sku_mappings` $\to$ `products.sku` dengan faktor pengali (`multiplier`).
- Alur pemotongan stok otomatis dari gudang yang dipilih.
- UI Frontend Next.js 15: Drag-and-drop file uploader, batch import log, modal resolusi unmapped SKU, dan tabel preview pesanan terimpor.

### Out-of-Scope (Fase Lanjutan):
- Integrasi real-time Open API langsung (OAuth2 Webhooks ke server Shopee Open Platform/TikTok Partner API) — memerlukan akun developer tersertifikasi marketplace.

---

## 4. Technical Requirements (Go Chi + PostgreSQL 15 + Next.js 15)

### Backend (Go Clean Architecture):
- Pure `database/sql` dengan parameterized queries (`$1, $2`).
- Dual-layer tenant isolation (`WHERE tenant_id = $n` dan `SET LOCAL app.current_tenant_id`).
- Transactional atomicity: Satu pesanan diproses dalam satu transaksi database lengkap bersama baris item dan mutasi stok.

### Frontend (Next.js 15 App Router & ZenSpace UI):
- UI Impor responsif (`app/(app)/wms/marketplace/page.tsx`).
- File dropzone dengan validasi MIME (`text/csv`, `application/vnd.ms-excel`, `application/json`).
- Visual progress dan status badge (Green = Mapped & Processed, Yellow = Unmapped SKU, Red = Duplicate / Error).

---

## 5. Schema Changes (Migration 023)

1. `marketplace_import_batches`:
   - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
   - `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE`
   - `batch_number VARCHAR(100) NOT NULL`
   - `channel VARCHAR(50) NOT NULL` (SHOPEE, TOKOPEDIA, TIKTOK, LAZADA, OTHER)
   - `warehouse_id UUID NOT NULL REFERENCES warehouses(id)`
   - `file_name VARCHAR(255) NOT NULL`
   - `total_orders INT NOT NULL DEFAULT 0`
   - `processed_orders INT NOT NULL DEFAULT 0`
   - `failed_orders INT NOT NULL DEFAULT 0`
   - `unmapped_skus INT NOT NULL DEFAULT 0`
   - `status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED'`
   - `uploaded_by UUID NOT NULL REFERENCES users(id)`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
   - `UNIQUE (tenant_id, batch_number)`

2. `marketplace_orders`:
   - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
   - `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE`
   - `batch_id UUID REFERENCES marketplace_import_batches(id) ON DELETE SET NULL`
   - `warehouse_id UUID NOT NULL REFERENCES warehouses(id)`
   - `channel VARCHAR(50) NOT NULL`
   - `external_order_id VARCHAR(150) NOT NULL`
   - `order_date TIMESTAMPTZ NOT NULL DEFAULT NOW()`
   - `customer_name VARCHAR(255)`
   - `customer_phone VARCHAR(50)`
   - `shipping_address TEXT`
   - `courier VARCHAR(100)`
   - `tracking_number VARCHAR(100)`
   - `total_amount NUMERIC(20,4) NOT NULL DEFAULT 0`
   - `shipping_fee NUMERIC(20,4) NOT NULL DEFAULT 0`
   - `marketplace_fee NUMERIC(20,4) NOT NULL DEFAULT 0`
   - `net_amount NUMERIC(20,4) NOT NULL DEFAULT 0`
   - `status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED'`
   - `sales_order_id UUID REFERENCES sales_orders(id)`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
   - `UNIQUE (tenant_id, channel, external_order_id)`

3. `marketplace_order_items`:
   - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
   - `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE`
   - `order_id UUID NOT NULL REFERENCES marketplace_orders(id) ON DELETE CASCADE`
   - `external_sku VARCHAR(150) NOT NULL`
   - `product_id UUID REFERENCES products(id)`
   - `item_name VARCHAR(255) NOT NULL`
   - `quantity NUMERIC(20,4) NOT NULL CHECK (quantity > 0)`
   - `unit_price NUMERIC(20,4) NOT NULL DEFAULT 0`
   - `subtotal NUMERIC(20,4) NOT NULL DEFAULT 0`
   - `is_mapped BOOLEAN NOT NULL DEFAULT FALSE`

---

## 6. Acceptance Criteria (Testable & Numbered)

- **AC-1 (Idempotent Import):** *Given* file pesanan Shopee dengan order ID `240909SHP001`, *When* diunggah untuk kedua kalinya dalam channel yang sama, *Then* sistem mengabaikan atau mencatat sebagai duplikat tanpa menduplikasi data pesanan maupun mutasi stok.
- **AC-2 (SKU Resolution & Multiplier):** *Given* mapping `external_sku: "KOPISUSU-DUS"` terpetakan ke internal `product_id: MAS000123` dengan `multiplier: 24`, *When* pesanan sebanyak 2 Dus diimpor, *Then* stok internal yang dipotong di WMS adalah $2 \times 24 = 48\text{ Pcs}$.
- **AC-3 (Unmapped SKU Alert & In-Place Linking):** *Given* pesanan dengan external SKU yang belum pernah didaftarkan, *When* diimpor, *Then* sistem mencatat status `is_mapped = false`, menambah counter `unmapped_skus`, dan menyediakan endpoint resolusi SKU instan.
- **AC-4 (WMS Stock Deduction):** *Given* pesanan marketplace yang valid dan terpetakan penuh, *When* batch diproses, *Then* sistem secara otomatis menerbitkan `StockMovement` dari lokasi rak gudang yang dipilih ke `@CUSTOMER` dengan `reference_type = 'MARKETPLACE'`.

---

## 7. Security & Guardrails

1. **Multi-Tenancy Isolation:** Semua tabel menerapkan `ENABLE ROW LEVEL SECURITY` dan `FORCE ROW LEVEL SECURITY`.
2. **Payload Protection:** Batas upload file dibatasi `http.MaxBytesReader` 5MB untuk mencegah serangan DoS memori.
3. **No Negative Stock:** Jika stok gudang tidak mencukupi saat proses potong stok, pesanan ditandai `STOCK_INSUFFICIENT` tanpa membatalkan seluruh batch.
