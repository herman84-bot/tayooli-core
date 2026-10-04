# PRD: Warehouse Management System (WMS) & Point of Sale (POS) Customization

**Status:** Proposed / Architecture Specification  
**Version:** 1.0.0  
**Author:** CTO Office  
**Target:** Tayooli ERP Engine Expansion  
**Date:** 2026-09-06  

---

## 1. Executive Summary & Problem Statement

### 1.1 Background
Tayooli ERP saat ini berfungsi sebagai sistem **Finance & Accounts Payable (AP)** dengan fokus pada otomasi invoice vendor (OCR Tesseract), 3-way matching (PO vs GR vs Invoice untuk validasi keuangan), pembayaran (Pakasir/Midtrans), dan akuntansi (CoA + Jurnal).

Namun, untuk kebutuhan operasional fisik distribusi dan ritel modern, sistem saat ini memiliki batasan fundamental:
1. Tidak ada entitas multi-gudang (hanya ada kolom string bebas `warehouse_location` pada tabel `inventory`).
2. Tidak ada pencatatan mutasi fisik riil: `goods_receipts` dan `sales_orders` belum memiliki line items produk dan belum memotong/menambah kuantitas stok barang.
3. Tidak ada penelusuran lokasi mikroskopis (nomor rak, bin, atau pallet/LPN).
4. Tidak ada mekanisme transfer antar-gudang berstatus *in-transit*.
5. Pengguna staf gudang dapat melihat data semua gudang karena isolasi keamanan saat ini baru di tingkat *tenant*, belum di tingkat *warehouse*.
6. Belum ada dukungan pemindaian barcode fisik (baik scanner USB maupun kamera HP).
7. Belum ada antarmuka kasir cepat (POS) dan integrasi pesanan omnichannel (Marketplace/Konsinyasi).

### 1.2 Objective
Membangun modul **WMS (Warehouse Management System) & POS (Point of Sale)** terpadu di atas arsitektur Go (Chi, Clean Architecture, `database/sql` murni) dan Next.js 15 (Tailwind, ZenSpace UI), dengan merujuk pada prinsip arsitektur teruji dari repositori ERP & WMS terkemuka di dunia.

---

## 2. Benchmark Arsitektur Open-Source & Landasan Teori

Sistem ini dirancang tidak dari nol secara sporadis, melainkan mengadopsi pola desain (*design patterns*) terbukti dari repositori berikut:

| Referensi Repositori | Komponen yang Diadopsi | Justifikasi Teknis |
|---|---|---|
| **Odoo (`stock` & `stock_account`)** | *Double-Entry Inventory & Virtual Locations* | Stok tidak pernah "hilang" atau "muncul dari ketiadaan". Setiap mutasi adalah perpindahan dari `source_location_id` ke `dest_location_id`. Termasuk lokasi virtual: Vendor (pembelian), Customer (penjualan), Inventory Loss (opname), Scrap (rusak), dan Transit. Menghilangkan bug selisih stok (*race condition*). |
| **ERPNext (Frappe)** | *Immutable Stock Ledger Entry (SLE)* | Seluruh mutasi barang dicatat ke tabel buku besar (*append-only*). Kuantitas di tabel agregat `inventory` adalah cerminan terindeks dari saldo akhir di SLE. Audit trail 100% tahan uji forensik akuntansi. |
| **OCA Stock Logistics Warehouse** | *Packaging, Pallet (LPN), & Strict Bin Location* | Pemodelan hierarki fisik: `Warehouse` $\to$ `Zone` $\to$ `Rack/Aisle` $\to$ `Shelf/Bin` $\to$ `Pallet/LPN`. Memungkinkan pemantauan kapasitas dan putaway barang terarah. |
| **OCA POS & Odoo POS** | *POS Session & Cash Control Lifecycle* | Siklus kasir terisolasi: `Open Session` (hitung kas awal) $\to$ `Transact` (scan & input cepat) $\to$ `Closing Control` (rekonsiliasi fisik vs sistem) $\to$ `Post Journal`. Menjamin kasir toko tertib finansial. |
| **MT-WMS & Sentry WMS** | *WMS State Machines & Barcode Verification* | Alur validasi fisik wajib 2-fase scan: Scan Lokasi Rak $\to$ Scan Barcode Barang. Mencegah salah letak barang (*misplacement*) dan salah ambil (*mis-picking*). |
| **Dolibarr** | *Multi-Barcode & Product Aliasing* | Satu entitas barang (*master item*) dapat memiliki banyak varian kode batang (EAN-8, EAN-13, Code 128) dan alias SKU per rekanan bisnis. |
| **OpenOMS** | *Channel SKU Mapping & Allocation Engine* | Pemetaan otomatis kode SKU Marketplace (Shopee/Tokopedia/TikTok) ke SKU internal ERP, serta pemisahan perlakuan order Jual Putus vs Konsinyasi. |

---

## 3. Scope & Phased Roadmap

Pengembangan dibagi ke dalam 3 Fase berurutan (*sequential dependencies*):

```
┌────────────────────────────────────────────────────────────────────────┐
│ FASE 1 (PRD INI - FONDASI DATA, WMS CORE & SKU ENGINE)                │
│ - Master Warehouses & Hierarchy (Zone -> Rack -> Bin -> Pallet/LPN)     │
│ - Warehouse-Scoped Permission & RLS (Regional Manager vs Warehouse)   │
│ - SKU Multi-Barcode & Customer/Channel Mapping                         │
│ - Double-Entry Stock Movement Ledger (Append-only)                     │
│ - Stock Opname, Stock Adjustment, & Damaged/Scrap Quarantine           │
└────────────────────────────────────┬───────────────────────────────────┘
                                     │
                                     ▼
┌────────────────────────────────────────────────────────────────────────┐
│ FASE 2 (LOGISTIK OPERASIONAL & WORKFLOW TRANSAKSI)                     │
│ - Refactor Line Items (PO Items, GR Items, SO Items, SI Items)         │
│ - Inbound Receiving Workflow (PO -> Goods Receipt -> Putaway)          │
│ - Inter-Warehouse Transfer dengan Status IN-TRANSIT                    │
│ - Outbound Picking, Packing, & Penerbitan SURAT JALAN (Delivery Order) │
│ - Line of Approval multi-tier pada transaksi WMS                       │
└────────────────────────────────────┬───────────────────────────────────┘
                                     │
                                     ▼
┌────────────────────────────────────────────────────────────────────────┐
│ FASE 3 (OMNICHANNEL, HARDWARE & POINT OF SALE)                         │
│ - Mobile WebRTC Barcode Scanner (Kamera HP) & USB Wedge Listener       │
│ - Point of Sale (POS) Interface: Kasir Cepat, POS Session, Struk Cetak │
│ - Retail Modes: Jual Putus vs Konsinyasi (Consignment settlement)      │
│ - Marketplace Order Settlement Import (Excel/CSV Parser Tokopedia dll) │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Domain Model & Business Rules

### 4.1 Master Gudang & Hierarki Lokasi Fisik (Odoo + OCA Model)
1. **Entitas Gudang (`warehouses`):**
   - Setiap gudang terikat pada `tenant_id`.
   - Memiliki atribut: `code` (unik per tenant), `name`, `address`, `regional_id`, `is_active`.
2. **Hierarki Lokasi (`warehouse_locations`):**
   - Model hierarki pohon (*self-referencing parent*):
     - `VIEW` (Folder induk lokasi gudang)
     - `ZONE` (misal: Area Dingin, Area Kering, Staging Dock)
     - `RACK` (misal: Rak A01, Rak B02)
     - `BIN` (Level rak spesifik: A01-01-02)
     - `PALLET` / `LPN` (Nomor palet fisik terikat ke rak/bin: PLT-9901)
3. **Lokasi Virtual Bawaan Sistem (System Virtual Locations):**
   - Tiap tenant otomatis memiliki lokasi virtual terisolasi:
     - `@VENDOR`: Sumber inbound dari supplier.
     - `@CUSTOMER`: Tujuan outbound penjualan.
     - `@TRANSIT`: Lokasi mengambang saat transfer barang antar-gudang sedang dalam perjalanan.
     - `@INVENTORY_LOSS`: Akun perantara selisih stock opname/adjustment.
     - `@SCRAP`: Lokasi karantina barang rusak/afkir.

### 4.2 Multi-Warehouse Permission & Authorization (ERPNext + Odoo Model)
1. **User Scoping:**
   - Staf gudang biasa (`role = 'warehouse'`) diikat ke satu atau beberapa gudang tertentu via tabel relasi `user_warehouses`.
   - Staf gudang **DILARANG KERAS** melihat stok, daftar rak, surat jalan, dan mutasi barang gudang lain yang tidak ditugaskan kepadanya.
2. **Regional Manager (`role = 'regional_manager'`):**
   - Memiliki cakupan wilayah (`regional_id`).
   - Berhak memantau stok, menyetujui mutasi, dan melihat transaksi **seluruh gudang** di bawah regional yang dikelolanya.
3. **Admin / Owner / Auditor:**
   - Memiliki hak akses penuh (*cross-warehouse*) dalam cakupan `tenant_id` mereka.
4. **Implementasi Keamanan:**
   - Dilindungi di lapis Database PostgreSQL via Session Variable `app.current_user_warehouse_ids` dan verifikasi klaim JWT di backend Go middleware.

### 4.3 Barcode Engine & Multi-SKU Mapping (Dolibarr + OpenOMS Model)
1. **Fleksibilitas Barcode:**
   - Mendukung variasi panjang karakter (EAN-8 [8 digit], EAN-13 [13 digit], Code-128 [alfanumerik 1-50 char], QR Code).
   - Satu produk (*Master Item*) dapat memiliki lebih dari satu barcode fisik (misal: kemasan satuan vs kemasan karton/dus).
2. **SKU Aliasing (Mapping Mitra & Marketplace):**
   - Model `product_sku_mappings`:
     - `product_id`: ID produk internal Tayooli ERP (misal SKU Internal: `MAS000123`).
     - `mapping_type`: `CUSTOMER`, `VENDOR`, `MARKETPLACE`.
     - `party_id`: ID pelanggan (misal: Carrefour) atau ID Channel (misal: Shopee Official Store).
     - `external_sku`: Kode SKU versi pihak luar (misal: `1234000BCD`).
     - `multiplier`: Faktor konversi kuantitas (misal: jika SKU Carrefour berbentuk 1 Karton = 24 Pcs internal).
3. **Resolusi Pencarian (Search Resolver Strategy):**
   - Saat kasir atau scanner memindai kode input:
     - Langkah 1: Cocokkan langsung ke `products.sku`.
     - Langkah 2: Cocokkan ke tabel `product_barcodes.barcode`.
     - Langkah 3: Cocokkan ke tabel `product_sku_mappings.external_sku`.
     - Return: Master `product_id` dan kuantitas unit ekuivalen.

### 4.4 Double-Entry Stock Movement Ledger (Odoo + ERPNext SLE Model)
1. **Prinsip Immutability (Buku Besar Mutasi Tak Terubah):**
   - Tabel `stock_movements` bersifat *append-only*. Baris yang sudah berstatus `DONE` **tidak boleh di-UPDATE atau di-DELETE**.
   - Jika terjadi pembatalan, sistem wajib menerbitkan baris pembalik (*reversal movement*).
2. **Struktur Gerakan:**
   - Setiap mutasi wajib mendefinisikan:
     - `source_location_id`
     - `destination_location_id`
     - `product_id`
     - `quantity` (selalu positif)
     - `reference_doc_type` (`GOODS_RECEIPT`, `DELIVERY_ORDER`, `TRANSFER`, `ADJUSTMENT`, `OPNAME`, `SCRAP`)
     - `reference_doc_id`
3. **Integritas Saldo:**
   - Saldo aktual di lokasi fisik dihitung dari agregasi gerakan masuk dikurangi gerakan keluar.
   - Constraint database ketat: Saldo stok di lokasi internal fisik **tidak boleh bernilai negatif** (`CHECK (quantity >= 0)`).

### 4.5 Inter-Warehouse Stock Transfer & In-Transit Lifecycle
State Machine transfer antar-gudang mengadopsi alur Sentry WMS:

```
[DRAFT]
   │
   ▼ (Submit Request)
[PENDING_APPROVAL]
   │
   ├─► [REJECTED]
   ▼ (Approved by Source WH / Regional Mgr)
[READY_TO_DISPATCH]
   │
   ▼ (Picking & Loading onto Vehicle -> Outbound Movement: Source Loc -> @TRANSIT)
[IN_TRANSIT] ◄── Status aktif saat barang di jalan (Dapat dilacak plat armada & resi)
   │
   ▼ (Arrival at Target WH -> Physical Check)
[RECEIVED_PARTIAL] ──► (Discrepancy Investigation)
   │
   ▼ (Full Inbound Movement: @TRANSIT -> Target Loc)
[COMPLETED]
```

### 4.6 Stock Opname, Adjustment & Scrap/Quarantine (OCA Model)
1. **Stock Opname (Physical Count):**
   - Dokumen penghitungan stok berkala per gudang/rak.
   - Status: `DRAFT` $\to$ `COUNTING` (dapat membekukan pergerakan stok pada rak terkait) $\to$ `AUDITED` $\to$ `APPROVED` $\to$ `POSTED`.
   - Menghasilkan selisih: `discrepancy_qty = actual_qty - system_qty`.
   - Saat `POSTED`, otomatis menerbitkan `stock_movements`:
     - Jika surplus: `@INVENTORY_LOSS` $\to$ `Target Bin`.
     - Jika defisit: `Source Bin` $\to$ `@INVENTORY_LOSS`.
2. **Barang Rusak (Scrap & Quarantine):**
   - Barang yang rusak saat penerimaan inbound atau di gudang dipindahkan ke lokasi `@SCRAP` / `@QUARANTINE`.
   - Stok pada lokasi scrap dikeluarkan dari perhitungan *Available to Promise (ATP)* penjualan, sehingga kasir POS dan sales tidak bisa menjual barang cacat.

### 4.7 Surat Jalan (Delivery Order) & Integrasi Invoice
1. **Surat Jalan (DO):**
   - Diterbitkan saat pesanan penjualan (*Sales Order*) telah siap kirim dan barang telah selesai di-*packing*.
   - Mencatat identitas pengemudi, nomor polisi kendaraan, ekspedisi/kurir, daftar item fisik, dan tanda tangan serah terima.
   - Penerbitan DO memicu mutasi stok fisik keluar: `Warehouse Rack` $\to$ `@CUSTOMER`.
2. **Sales Invoice (Faktur Penjualan):**
   - Merujuk pada Surat Jalan yang telah berstatus `DELIVERED` atau `SHIPPED`.
   - Mengirim event ke modul Akuntansi untuk mencatat Piutang Usaha (AR) dan Pendapatan Penjualan.

---

## 5. Database Schema Design (PostgreSQL 15 RLS)

Semua tabel wajib menyertakan `tenant_id` dengan RLS diaktifkan (`FORCE ROW LEVEL SECURITY`) serta isolasi gudang.

```sql
-- ============================================================================
-- 1. MASTER GUDANG & WILAYAH REGIONAL
-- ============================================================================

CREATE TABLE regionals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code        VARCHAR(50) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code)
);

CREATE TABLE warehouses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    regional_id  UUID REFERENCES regionals(id) ON DELETE SET NULL,
    code         VARCHAR(50) NOT NULL,
    name         VARCHAR(255) NOT NULL,
    address      TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code),
    UNIQUE (id, tenant_id)
);

CREATE TABLE user_warehouses (
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    warehouse_id  UUID NOT NULL,
    tenant_id     UUID NOT NULL,
    assigned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, warehouse_id),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE CASCADE
);

-- ============================================================================
-- 2. HIERARKI LOKASI GUDANG (ZONA, RAK, BIN, PALLET) & LOKASI VIRTUAL
-- ============================================================================

CREATE TYPE location_type AS ENUM ('INTERNAL', 'VENDOR', 'CUSTOMER', 'TRANSIT', 'LOSS', 'SCRAP');

CREATE TABLE warehouse_locations (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id   UUID REFERENCES warehouses(id) ON DELETE CASCADE, -- NULL untuk lokasi global transit/virtual
    parent_id      UUID REFERENCES warehouse_locations(id) ON DELETE CASCADE,
    code           VARCHAR(100) NOT NULL, -- Contoh: 'WH1-ZONEA-R01-B02'
    barcode        VARCHAR(100),          -- Barcode yang ditempel di fisik rak/pallet
    name           VARCHAR(255) NOT NULL,
    type           location_type NOT NULL DEFAULT 'INTERNAL',
    is_pallet      BOOLEAN NOT NULL DEFAULT FALSE,
    pallet_number  VARCHAR(100),
    max_capacity   NUMERIC(15, 2),        -- Kapasitas maksimum (opsional)
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, warehouse_id, code)
);

-- ============================================================================
-- 3. SKU MULTI-BARCODE & MARKETPLACE/CUSTOMER MAPPINGS
-- ============================================================================

CREATE TABLE product_barcodes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL,
    barcode     VARCHAR(100) NOT NULL,
    barcode_symbology VARCHAR(50) DEFAULT 'CODE128', -- 'EAN13', 'EAN8', 'CODE128', 'QR'
    uom_name    VARCHAR(50) DEFAULT 'PCS',           -- 'PCS', 'BOX', 'CTN'
    multiplier  NUMERIC(10, 4) NOT NULL DEFAULT 1.0, -- 1 BOX = 24 PCS
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE CASCADE,
    UNIQUE (tenant_id, barcode)
);

CREATE TYPE sku_mapping_type AS ENUM ('CUSTOMER', 'MARKETPLACE', 'VENDOR');

CREATE TABLE product_sku_mappings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id     UUID NOT NULL,
    mapping_type   sku_mapping_type NOT NULL,
    channel_name   VARCHAR(100) NOT NULL, -- Contoh: 'SHOPEE', 'TOKOPEDIA', 'CAREFOUR'
    external_sku   VARCHAR(100) NOT NULL,
    external_name  VARCHAR(255),
    multiplier     NUMERIC(10, 4) NOT NULL DEFAULT 1.0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE CASCADE,
    UNIQUE (tenant_id, channel_name, external_sku)
);

-- ============================================================================
-- 4. BUKU BESAR MUTASI STOK (IMMUTABLE STOCK LEDGER ENTRY)
-- ============================================================================

CREATE TYPE stock_movement_status AS ENUM ('PENDING', 'DONE', 'CANCELLED');

CREATE TABLE stock_movements (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    movement_number  VARCHAR(100) NOT NULL,
    product_id       UUID NOT NULL,
    source_location_id UUID NOT NULL REFERENCES warehouse_locations(id),
    dest_location_id   UUID NOT NULL REFERENCES warehouse_locations(id),
    quantity         NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    unit_cost        NUMERIC(20, 4) NOT NULL DEFAULT 0,
    status           stock_movement_status NOT NULL DEFAULT 'DONE',
    reference_type   VARCHAR(50) NOT NULL, -- 'PO_RECEIPT', 'DELIVERY_ORDER', 'TRANSFER', 'OPNAME', 'SCRAP'
    reference_id     UUID NOT NULL,
    executed_by      UUID REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, movement_number)
);

-- ============================================================================
-- 5. INTER-WAREHOUSE STOCK TRANSFER
-- ============================================================================

CREATE TYPE transfer_status AS ENUM ('DRAFT', 'PENDING_APPROVAL', 'APPROVED', 'DISPATCHED', 'IN_TRANSIT', 'RECEIVED', 'REJECTED');

CREATE TABLE stock_transfers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    transfer_number     VARCHAR(100) NOT NULL,
    from_warehouse_id   UUID NOT NULL REFERENCES warehouses(id),
    to_warehouse_id     UUID NOT NULL REFERENCES warehouses(id),
    status              transfer_status NOT NULL DEFAULT 'DRAFT',
    requested_by        UUID NOT NULL REFERENCES users(id),
    approved_by         UUID REFERENCES users(id),
    vehicle_plate       VARCHAR(50),
    driver_name         VARCHAR(100),
    dispatched_at       TIMESTAMPTZ,
    received_at         TIMESTAMPTZ,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, transfer_number)
);

CREATE TABLE stock_transfer_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_id         UUID NOT NULL REFERENCES stock_transfers(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL,
    requested_qty       NUMERIC(20, 4) NOT NULL CHECK (requested_qty > 0),
    sent_qty            NUMERIC(20, 4) NOT NULL DEFAULT 0,
    received_qty        NUMERIC(20, 4) NOT NULL DEFAULT 0,
    source_location_id  UUID REFERENCES warehouse_locations(id),
    dest_location_id    UUID REFERENCES warehouse_locations(id)
);

-- ============================================================================
-- 6. DELIVERY ORDER (SURAT JALAN)
-- ============================================================================

CREATE TYPE delivery_order_status AS ENUM ('DRAFT', 'CONFIRMED', 'PICKED', 'PACKED', 'SHIPPED', 'DELIVERED', 'RETURNED', 'CANCELLED');

CREATE TABLE delivery_orders (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sales_order_id      UUID NOT NULL REFERENCES sales_orders(id),
    warehouse_id        UUID NOT NULL REFERENCES warehouses(id),
    do_number           VARCHAR(100) NOT NULL,
    status              delivery_order_status NOT NULL DEFAULT 'DRAFT',
    expedition_name     VARCHAR(100),
    tracking_number     VARCHAR(100),
    driver_name         VARCHAR(100),
    vehicle_plate       VARCHAR(50),
    recipient_name      VARCHAR(100),
    received_date       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, do_number)
);

CREATE TABLE delivery_order_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_order_id   UUID NOT NULL REFERENCES delivery_orders(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL,
    quantity            NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    location_id         UUID NOT NULL REFERENCES warehouse_locations(id)
);
```

---

## 6. Technical Requirements & Architecture Rules (Go + Next.js)

### 6.1 Backend (Go Chi Clean Architecture)
1. **Dilarang ORM:** Wajib menggunakan `database/sql` murni dengan parameterized queries.
2. **Transactional Consistency:**
   - Setiap transaksi mutasi stok (penerbitan Surat Jalan, penerimaan Transfer, Stock Opname) **wajib berada di dalam single database transaction (`tx`)**.
   - Sebelum mengeksekusi mutasi keluar, lakukan `SELECT quantity FROM stock_movements ... FOR UPDATE` atau kunci baris kuantitas untuk mencegah *concurrency race condition* (dua kasir/picker mengambil barang yang sama).
3. **Kafka Event-Driven Architecture:**
   - Event `stock.transferred` $\to$ Konsumer notifikasi ke gudang tujuan.
   - Event `delivery_order.shipped` $\to$ Pemicu otomatisasi pembuatan draft Sales Invoice dan jurnal akuntansi COGS / Persediaan Barang Dagang.

### 6.2 Frontend (Next.js 15 App Router & Mobile Barcode)
1. **ZenSpace Responsive Design:**
   - Tampilan WMS Mobile untuk staf gudang: Desain kartu berukuran sentuh minimal 48px, kontras tinggi di lingkungan gudang, indikator status visual (Hijau = Match, Merah = Mismatch).
2. **Barcode Scanner Handling:**
   - **Hardware Scanner (USB/Bluetooth):** Implementasi listener global `keydown` dengan detektor interval keystroke ($\le 30\text{ms}$ per karakter) diakhiri enter key untuk menangkap data scanner tanpa membutuhkan fokus manual pada input text.
   - **Mobile Camera Scanner:** Menggunakan `@zxing/browser` atau `html5-qrcode` yang terisolasi dalam hook `useBarcodeScanner()`.

---

## 7. Acceptance Criteria (Testable & Numbered)

### AC-1: Multi-Warehouse Scoping & Isolation
- **AC-1.1:** *Given* seorang user dengan role `warehouse` yang ditugaskan hanya ke Warehouse-A, *When* melakukan query `GET /api/v1/inventory` atau `GET /api/v1/warehouses/{id}`, *Then* sistem hanya mengembalikan data Warehouse-A dan menolak akses ke Warehouse-B dengan `403 Forbidden`.
- **AC-1.2:** *Given* user dengan role `regional_manager`, *When* melakukan query, *Then* sistem mengembalikan data seluruh gudang di bawah regional ID yang menjadi tanggung jawabnya.

### AC-2: SKU & Barcode Resolution
- **AC-2.1:** *Given* produk A memiliki SKU Internal `MAS000123` dan mapping pelanggan `1234000BCD`, *When* API `/api/v1/products/resolve?code=1234000BCD` dipanggil, *Then* sistem mengembalikan Master Product ID `MAS000123` beserta nama produk aslinya dalam waktu kurang dari 50ms.
- **AC-2.2:** *Given* barcode fisik dengan panjang 8 digit (EAN-8), 13 digit (EAN-13), dan alfanumerik 24 karakter, *When* discan, *Then* sistem berhasil mendeteksi dan menyelesaikan produk tanpa error validasi panjang karakter.

### AC-3: Inter-Warehouse In-Transit Lifecycle
- **AC-3.1:** *Given* transfer stok sebanyak 10 unit dari Gudang Jakarta ke Gudang Surabaya, *When* status berubah menjadi `DISPATCHED`, *Then* saldo fisik Gudang Jakarta berkurang 10 unit dan lokasi `@TRANSIT` bertambah 10 unit. Saldo Gudang Surabaya belum bertambah.
- **AC-3.2:** *When* Gudang Surabaya mengonfirmasi penerimaan (`RECEIVED`), *Then* stok di `@TRANSIT` berkurang 10 unit dan stok fisik Gudang Surabaya bertambah 10 unit.

### AC-4: Surat Jalan & Outbound Stock Deduction
- **AC-4.1:** *Given* Sales Order yang sudah dikonfirmasi, *When* Surat Jalan (Delivery Order) berstatus `SHIPPED` diterbitkan, *Then* sistem otomatis mengurangi stok pada nomor rak/bin yang dipilih dan mencatat entri baru di tabel `stock_movements`.
- **AC-4.2:** Surat Jalan dapat diekspor menjadi dokumen cetak PDF resmi dengan format standar yang mencantumkan nomor DO, daftar item, nomor rak/pallet, dan kolom tanda tangan pengirim/pengemudi/penerima.

---

## 8. Security & Guardrails

1. **Anti Cross-Tenant Leakage:** Setiap query wajib menyertakan `WHERE tenant_id = $1` dan dilindungi RLS di level basis data.
2. **Negative Stock Prevention:** Database constraint `CHECK (quantity >= 0)` aktif di semua level penampungan kuantitas. Transaksi yang menyebabkan stok negatif wajib di-abort dengan pesan error spesifik: `ERR_INSUFFICIENT_STOCK`.
3. **Audit Trail Immutability:** Semua mutasi stok mencatat `executed_by` (User UUID) dan `created_at` yang terhubung dengan hash audit log sistem Tayooli.
