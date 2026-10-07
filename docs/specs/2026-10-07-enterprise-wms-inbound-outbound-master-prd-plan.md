# MASTER PRD & TECHNICAL IMPLEMENTATION PLAN
## Enterprise WMS Inbound & Outbound End-to-End Lifecycle

- **Document ID:** `SPEC-WMS-2026-10-07-ENTERPRISE-IN-OUT`
- **Status:** APPROVED MASTER SPECIFICATION
- **Version:** 1.0.0
- **Author:** CTO Office & WMS Architecture Guild
- **Source Documents:** 
  - `ALUR PROSES INBOUND.pdf` (SOP Operasional Gudang - Inbound Management, ISO 9001:2015 Compliant)
  - `ALUR PROSES OUTBOUND.pdf` (SOP Operasional Gudang - Outbound Management, ISO 9001:2015 Compliant)
- **Target Repository:** `tayooli-core`
- **Last Updated:** 2026-10-07

---

## DAFTAR ISI
1. [Latar Belakang & Pernyataan Masalah](#1-latar-belakang--pernyataan-masalah)
2. [Mekanisme Anti-Lupa & Anti-Halusinasi (AI Governance)](#2-mekanisme-anti-lupa--anti-halusinasi-ai-governance)
3. [PRD: Spesifikasi Kebutuhan Produk & Operasional (Bisnis)](#3-prd-spesifikasi-kebutuhan-produk--operasional-bisnis)
   - 3.1 [Alur Inbound 6 Tahap](#31-alur-inbound-6-tahap)
   - 3.2 [Alur Outbound 6 Tahap](#32-alur-outbound-6-tahap)
   - 3.3 [Matriks Peran, Tanggung Jawab & SLA](#33-matriks-peran-tanggung-jawab--sla)
   - 3.4 [Review Tambahan Klien (RIVIEW TAYOLI.xlsx)](#34-review-tambahan-klien-riview-tayolixlsx-diterima-2026-10-07)
4. [Tech Spec: Spesifikasi Teknis & Arsitektur (Engineering)](#4-tech-spec-spesifikasi-teknis--arsitektur-engineering)
   - 4.1 [Skema Database PostgreSQL & Migrasi](#41-skema-database-postgresql--migrasi)
   - 4.2 [Mesin Status (State Machines) & Validasi Transisi](#42-mesin-status-state-machines--validasi-transisi)
   - 4.3 [Backend Go (Hexagonal Clean Architecture & API Endpoints)](#43-backend-go-hexagonal-clean-architecture--api-endpoints)
   - 4.4 [Frontend Next.js 15 UI/UX & PDA Scanner Engine](#44-frontend-nextjs-15-uiux--pda-scanner-engine)
   - 4.5 [Integrasi Perangkat Keras (Thermal Printer & Handheld PDA)](#45-integrasi-perangkat-keras-thermal-printer--handheld-pda)
5. [Roadmap Eksekusi Bertahap (Sprint 1 s/d Sprint 5)](#5-roadmap-eksekusi-bertahap-sprint-1-sd-sprint-5)
6. [Living Execution Checklist (Tracking Kemajuan Real-Time)](#6-living-execution-checklist-tracking-kemajuan-real-time)

---

## 1. Latar Belakang & Pernyataan Masalah

### 1.1 Kondisi Saat Ini (As-Is)
Aplikasi `tayooli-core` saat ini memiliki fondasi multi-gudang dan buku besar mutasi stok atomik (*double-entry stock ledger*) yang tangguh, namun alur fisiknya masih bertipe **SME Inventory Tracker (Tier-3 WMS)**:
- **Inbound:** Pengguna mengisi formulir modal sederhana (`DRAFT`) lalu klik konfirmasi (`POSTED`). Stok langsung bertambah ke satu rak definitif. Tidak ada pelacakan kedatangan truk, area staging, nomor lot/batch, tanggal kedaluwarsa, maupun tugas putaway terpisah.
- **Outbound:** Surat Jalan dibuat (`DRAFT`) lalu langsung dikirim (`SHIPPED`), memotong stok rak internal langsung ke `@CUSTOMER`. Tidak ada picking list, meja kemas QC pemindaian 100%, cetak resi termal AWB, maupun konsolidasi manifest serah terima kurir.
- **Scanner Mobile:** Halaman `/wms/scanner` masih berupa prototipe kamera WebRTC dengan logika simulasi *state* lokal browser tanpa integrasi API mutasi stok backend.

### 1.2 Kondisi yang Ditargetkan (To-Be)
Klien membutuhkan **Enterprise 3PL / Industrial Logistics WMS (Tier-1/Tier-2 WMS)** yang patuh standar ISO 9001:2015 dengan 6 tahapan Inbound dan 6 tahapan Outbound yang disiplin, terdokumentasi, dan terverifikasi barcode pada setiap perpindahan barang fisik.

---

## 2. Mekanisme Anti-Lupa & Anti-Halusinasi (AI Governance)

Agar pengerjaan ini **tuntas dari A sampai Z, tidak terputus, dan tidak hilang arah** saat pergantian sesi AI atau kompresi konteks, diterapkan 6 lapis pengamanan (*guardrails*):

1. **Single Source of Truth Terpatri di Git (Dokumen Ini):**
   - Dokumen ini adalah satu-satunya referensi sah arsitektur WMS Inbound & Outbound.
   - Setiap AI atau developer yang melanjutkan pekerjaan **WAJIB membaca dokumen ini terlebih dahulu** sebelum menulis kode.
2. **Living Execution Checklist Terbuka (Bab 6):**
   - Setiap subtugas memiliki checkbox `[ ]` / `[x]`.
   - Tugas hanya boleh dicentang `[x]` jika telah disertai bukti nyata: **Nomor Commit Git**, **Hasil Pengujian Otomatis (Pass)**, dan **Verifikasi URL / Endpoint**.
3. **Persisted Goal System:**
   - Sesi agen diikat dengan persistent goal di environment kerja DeepSeek Harness (`create_goal`).
4. **Architecture Decision Record (ADR-014):**
   - Dokumen invariant arsitektur di `docs/adr/014-wms-enterprise-inbound-outbound-lifecycle.md` yang mengunci aturan bahwa:
     - Buku besar `stock_movements` tetap atomik dan tidak boleh diubah (append-only ledger).
     - Seluruh mutasi rak wajib divalidasi dengan tenant isolation RLS PostgreSQL.
     - Mode offline PDA wajib menerapkan idempotency key berbasis UUID.
5. **Sinkronisasi Knowledge Graph Otomatis (`graphify`):**
   - Setiap penambahan tabel atau endpoint baru, file `graphify-out/graph.json` wajib diperbarui sehingga pemetaan simbol dan relasi antar-file selalu akurat 100%.
6. **Gerbang Pengujian Otomatis (CI/CD Deployment Gates):**
   - Tidak ada kode yang di-deploy ke Zeabur produksi (`https://tayooli.my.id`) tanpa melewati:
     - `go test ./...` & `go vet` (Backend Go)
     - `npm run test` & `npx tsc --noEmit` (Frontend Next.js)
     - Pengujian E2E Playwright pada alur fisik.

---

## 3. PRD: Spesifikasi Kebutuhan Produk & Operasional (Bisnis)

### 3.1 Alur Inbound 6 Tahap
Standar operasional penerimaan barang dari kedatangan armada hingga siap jual:

```
[1. Kedatangan & Docking] ──> [2. Unloading] ──> [3. Pengecekan & QC]
       (Gate / ASN)              (Staging Area)     (Batch, Exp, BAK)
                                                            │
                                                            ▼
[6. Stock Release]       <── [5. Putaway]   <── [4. System Entry & LPN]
   (Ready for Picking)        (Bin Allocation)       (GRN & Barcode Palet)
```

1. **Tahap 1: Kedatangan & Docking (Pre-Receiving)**
   - *Aktivitas:* Sopir melapor di pos gerbang membawa Surat Jalan & PO/RO. Petugas gate memverifikasi jadwal kedatangan terhadap ASN gudang. Sistem menetapkan nomor dermaga (*Inbound Bay / Dock Slot*) yang kosong.
   - *Keluaran:* Slot Unloading Aktif & Tiket Masuk Truk. SLA: $\le 15$ menit.
2. **Tahap 2: Pembongkaran (Unloading)**
   - *Aktivitas:* Muatan dibongkar ke area transit (*Receiving Staging Area*). Petugas tally memeriksa kondisi pembungkus luar (kemasan basah/penyok) dan menghitung total dus/palet gelondongan (*Gross Count*).
   - *Keluaran:* Dokumen Bukti Bongkar Muat Fisik (Tally Sheet). SLA: 30–45 menit.
3. **Tahap 3: Pengecekan Fisik & Kontrol Kualitas (QC)**
   - *Aktivitas:* Penghitungan detail per SKU menggunakan barcode scanner. Wajib menginput Nomor Batch/Lot dan Tanggal Kedaluwarsa (*Exp Date*). Petugas QC menentukan kelayakan: *Pass*, *Reject*, atau *Quarantine*.
   - *Penanganan Selisih:*
     - Barang Rusak: Dipisahkan ke *Quarantine Area*, foto kondisi fisik, buat Berita Acara Kerusakan (BAK), status sistem *Blocked Stock*.
     - Selisih Jumlah (Kurang/Lebih): Sistem menerbitkan *Discrepancy Report*, konfirmasi tanda tangan sopir sebelum truk pergi.
   - *Keluaran:* Form Hasil QC & Lampiran BAK (jika ada). SLA: $\le 45$ menit.
4. **Tahap 4: System Entry & Labeling**
   - *Aktivitas:* Input data GRN ke sistem WMS sesuai PO/RO. Cetak label stiker *License Plate Number* (LPN) untuk palet dan barcode SKU unit. Sistem menerbitkan daftar tugas penataan (*Putaway Task List*).
   - *Keluaran:* Good Receipt Note (GRN) sah & Label LPN terpasang. SLA: $\le 15$ menit.
5. **Tahap 5: Putaway (Penyimpanan Rak)**
   - *Aktivitas:* Operator MHE/Reach Truck membawa barang dari Staging ke rak. Operator memindai barcode LPN pada palet & memindai barcode rak tujuan (*Bin Location*). Sistem memvalidasi kesesuaian lokasi secara real-time.
   - *Strategi Penataan:*
     - *Velocity Mapping:* Fast-moving di lorong utama/bawah, slow-moving di atas/belakang.
     - *Rotasi FEFO:* Expired terdekat diletakkan di posisi paling mudah dijangkau.
     - *Weight Balancing:* Barang berat di ground level, barang ringan di level atas.
   - *Keluaran:* Terkonfirmasi *Putaway Complete*. SLA: $\le 30$ menit.
6. **Tahap 6: Dispatch & Stock Release**
   - *Aktivitas:* Sistem mengubah status stok dari *Unallocated/In-Transit* menjadi *Available Stock*. Dokumen PO/RO otomatis ditutup. Stok langsung siap dipesan oleh sistem penjualan/outbound.
   - *Keluaran:* Stok Siap Jual (*Ready Stock*). SLA: Real-time.

---

### 3.2 Alur Outbound 6 Tahap
Standar operasional pengeluaran pesanan dari pemrosesan hingga armada berangkat:

```
[1. Order Release] ──> [2. Picking] ──> [3. Pengecekan & QC]
    (Wave / SO)        (Shortest Path)     (100% Rescan Meja Kemas)
                                                    │
                                                    ▼
[6. Dispatch & Loading] <── [5. Staging] <── [4. Packing & AWB]
    (Loading Scan / DO)      (Rute / Kurir)      (Resi Thermal & Slip)
```

1. **Tahap 1: Pemrosesan Pesanan (Order Release & Wave Planning)**
   - *Aktivitas:* Pesanan penjualan (*Sales Order / Permintaan Kirim*) dikonsolidasikan. Sistem membagi pesanan ke dalam gelombang (*Wave Release*) berdasarkan jenis barang, kurir ekspedisi, atau rute pengiriman.
   - *Keluaran:* Dokumen *Picking List* dengan alokasi rak FEFO & penugasan tugas ke PDA picker. SLA: $\le 15$ menit.
2. **Tahap 2: Pengambilan Barang (Picking)**
   - *Aktivitas:* Petugas picker mengikuti alur rute terpendek di lorong gudang (*shortest path routing*). Memindai barcode lokasi rak dan barcode SKU barang (disiplin FIFO/FEFO). Mengumpulkan item ke troli dan memindahkannya ke area meja packing QC.
   - *Penanganan Selisih:* Jika barang di rak rusak, alihkan ke *Holding Area* dan sistem mengalokasikan picking ulang dari rak cadangan; jika stok rak fisik kosong, terbitkan *Shortage Ticket*.
   - *Keluaran:* Barang terambil di troli & status stok dialokasikan (*Allocated*). SLA: 30–45 menit.
3. **Tahap 3: Pengecekan & QC (Meja Packing)**
   - *Aktivitas:* Seluruh item di troli dipindai ulang satu per satu (*100% SKU scan*) di meja packing untuk mencocokkan fisik terhadap Surat Jalan. Pemeriksaan fisik kemasan dan tanggal kedaluwarsa.
   - *Keluaran:* Checklist QC Passed & Verifikasi 100% Cocok. SLA: $\le 20$ menit.
4. **Tahap 4: Packing & Labeling (Kemasan & Resi)**
   - *Aktivitas:* Pengemasan standar (kardus, bubble wrap, lakban segel, palet). Memasukkan lembar *Packing Slip* ke dalam kardus. Mencetak dan menempelkan stiker label resi pengiriman (*Airway Bill / AWB*) termal 100x150 mm pada kardus bagian luar.
   - *Keluaran:* Paket tersegel & label AWB terpasang. SLA: $\le 20$ menit.
5. **Tahap 5: Staging Area & Konsolidasi**
   - *Aktivitas:* Paket dipindahkan ke area *Outbound Staging*. Disusun mengelompok sesuai rute tujuan, kota, atau nama ekspedisi penjemput. Verifikasi total koli gabungan.
   - *Keluaran:* Dokumen Manifest Serah Terima Ekspedisi (*Manifest Staging Sheet*). SLA: $\le 15$ menit.
6. **Tahap 6: Dispatch & Loading (Pemuatan Armada)**
   - *Aktivitas:* Pemindaian barcode resi saat paket dinaikkan ke bak truk (*Loading Scan*). Penataan muatan dengan kepadatan LIFO (*Last In, First Out*). Sopir dan supervisor menandatangani Surat Jalan (DO). Sistem memotong stok resmi menjadi *Goods Issue / Dispatched*.
   - *Keluaran:* Surat Jalan Sah bertanda tangan & Status WMS Dispatched. SLA: $\le 30$ menit.

---

### 3.3 Matriks Peran, Tanggung Jawab & SLA

| No | Tahapan | Penanggung Jawab | Dokumen / Sistem Acuan | SLA Waktu | KPI Keberhasilan |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **I-1** | Pre-Receiving & Dock | Gate Clerk / Admin Inbound | Surat Jalan, ASN, Booking Schedule | 15 Menit | Ketepatan jadwal armada & no delay gerbang |
| **I-2** | Unloading Staging | Tim Tally & Checker | Physical Delivery Sheet, Tally Form | 30–45 Menit | Zero damage saat bongkar, koli akurat |
| **I-3** | Inbound QC & Batch | Inbound Checker & Tim QC | PO/RO, Form Inspeksi QC, Dokumen BAK | 45 Menit | Akurasi SKU 100%, deteksi dini barang expired |
| **I-4** | GRN Entry & LPN | Admin Inbound WMS | WMS Good Receipt, Label LPN Barcode | 15 Menit | Pencetakan label LPN tepat & data PO sinkron |
| **I-5** | Putaway Storage | Operator MHE / Forklift | Putaway Tasklist (PDA / Handheld) | 30 Menit | Penempatan rak 100% tepat (Zero Misplacement) |
| **I-6** | Stock Release | Supervisor Warehouse | WMS Inventory Master, Release Approval | Real-Time | Stok langsung Available untuk penjualan |
| **O-1** | Order Release | Admin Outbound / Planner | Sales Order (SO), WMS Release Note | 15 Menit | Ketepatan pembentukan Wave & alokasi picker |
| **O-2** | Guided Picking | Picker / Operator MHE | Picking List, PDA Handheld Scanner | 30–45 Menit | Zero mispick, disiplin scan rak & SKU FEFO |
| **O-3** | Quality Check Pack | Inspector QC / Checker | QC Checklist, SO Verification Screen | 20 Menit | Akurasi 100% barang pesanan sebelum dipacking |
| **O-4** | Packing & AWB | Packer / Outbound Staff | Packing Slip, Thermal Label AWB | 20 Menit | Kemasan aman standar kurir, barcode terbaca |
| **O-5** | Staging Konsolidasi | Staging Leader / Marshal | Manifest Staging Sheet, Rute Buffer | 15 Menit | Pengelompokan tepat 100% per ekspedisi/tujuan |
| **O-6** | Loading & Dispatch | Dispatcher / Loader | Surat Jalan (DO), Bill of Lading, Loading Scan | 30 Menit | Pemuatan tepat waktu, data ledger Goods Issue |

---

### 3.4 Review Tambahan Klien (`RIVIEW TAYOLI.xlsx`, diterima 2026-10-07)

Sumber: `RIVIEW TAYOLI.xlsx` (Sheet1, B2:F11), berisi 5 poin. Setiap poin punya 1 screenshot di kolom "PCT". Screenshot **belum dianalisis** karena model yang dipakai saat dokumen ini ditulis tidak bisa membaca gambar. Sebelum mengerjakan item CR, buka screenshot itu (`xl/media/image*.jpg`; urutan per baris: CR-01=image2, CR-02=image3, CR-03=image4, CR-04=image5, CR-05=image1) untuk memastikan layar mana yang dimaksud.

| ID | Permintaan Klien (verbatim) | Kondisi Kode Saat Ini (diverifikasi) | Keputusan & Penempatan |
| :--- | :--- | :--- | :--- |
| **CR-01** | "untuk nama produk bisa ditambahkan jenis/kategori" | Tabel `products` (`012_products_inventory.sql`) hanya punya `name, description, sku, price`. Tidak ada kolom/tabel kategori di migrasi maupun domain Go. | Tambah master `product_categories` (per tenant) + `products.category_id` nullable. Field kategori di form Produk, filter & kolom di tabel Produk, ikut ekspor. Juga dipakai untuk ABC/putaway (§1.2 `oca-wms-spec.md`). **Sprint 1.** |
| **CR-02** | "bisa ditambahkan untuk customer baru, outbound & proses putaway" | Tabel `customers` + endpoint `GET/POST /api/v1/customers` sudah ada (`main.go:591-595`), halaman `app/(app)/customers/page.tsx` ada tapi **bukan** bagian 13 modul menu. `delivery_orders` **tidak punya `customer_id`** (hanya `recipient_name`). Putaway belum ada (scanner masih mock). | (a) Quick-add customer baru langsung dari form Surat Jalan (inline modal, pakai endpoint yang sudah ada) + `delivery_orders.customer_id` nullable. (b) Outbound & putaway sudah tercakup roadmap ini (Sprint 1 putaway, Sprint 3 outbound). Tidak menambah modul menu baru (aturan 13 modul). **Sprint 1 (putaway) & Sprint 3 (customer di DO).** |
| **CR-03** | "proses barang masuk bisa menyesuaikan dengan rak yg telah ditentukan di awal" | Tidak ada kolom lokasi default per produk (`default_location`/`preferred_location` tidak ditemukan). Form inbound memilih satu rak tujuan manual. | Tambah `products.default_location_id` (rak tetap/"home bin" per gudang; tabel `product_default_locations(product_id, warehouse_id, location_id)` karena multi-gudang). Saat putaway, rak ini menjadi **saran pertama** (`suggested_location_id`), sesuai pola *suggested bin* `sentry-wms-spec.md` §2. Operator tetap scan konfirmasi; bila beda rak, wajib alasan. **Sprint 1.** |
| **CR-04** | "untuk dasboard mungkin tampilkan jumlah barang keluar atau topten produk yg sering keluar atau toko yg paling banyak order" | `GET /dashboard/summary` (`domain/dashboard.go`) punya statistik POS, WMS (SKU, unit, lokasi, `today_movements`, low stock) dan `TopVendors`, tetapi **tidak ada** total barang keluar, top-10 produk keluar, maupun top customer/toko. | Tambah ke `WMSDashStats`: `outbound_qty_today/month`, `top_outbound_products[10]` (agregasi `stock_movements` dengan tujuan `@CUSTOMER`, termasuk POS & DO), `top_customers[10]` (jumlah DO/order per customer/toko, termasuk toko marketplace). Filter periode 7/30/90 hari. **Sprint 3** (butuh `customer_id` dari CR-02). |
| **CR-05** | "untuk setiap transaksi atau approval bisa dilihat atau terdetect siapa yg melakukan proses transaksi tersebut atau approval tersebut" | Tabel `audit_logs` ada (hash-chain) tapi hanya dipakai usecase payment/vendor/approval, **tidak** dipakai usecase WMS. `stock_movements.executed_by` ada; `stock_receipts.created_by/posted_by` ada; `stock_transfers.requested_by/approved_by` ada; `stock_opnames.approved_by` ada; **`delivery_orders` tidak punya `created_by`/`dispatched_by`**. UI belum menampilkan pelaku. | (a) Lengkapi kolom pelaku di semua dokumen WMS (`delivery_orders.created_by, confirmed_by, packed_by, dispatched_by`; tabel baru sprint ini wajib punya kolom pelaku per transisi status). (b) Setiap transisi status WMS menulis `audit_logs`. (c) UI: kolom "Dibuat oleh / Disetujui oleh" di tabel + panel "Riwayat Aktivitas" (timeline siapa, apa, kapan) di detail dokumen. **Lintas sprint, wajib mulai Sprint 1** (invariant: tidak ada transisi status tanpa user_id). |

**Catatan aturan 13 modul:** semua CR di atas ditempatkan di modul yang sudah ada (Products, Barang Masuk, Surat Jalan, Dashboard). Tidak ada menu baru yang ditambahkan.

---

## 4. Tech Spec: Spesifikasi Teknis & Arsitektur (Engineering)

### 4.1 Skema Database PostgreSQL & Migrasi

#### Migrasi SQL 1: `033_wms_batches_and_staging_locations.sql`
Menyediakan fondasi Batch/Lot, Expire Date, dan tipe lokasi Staging Inbound/Outbound.
```sql
-- 1. Tipe Lokasi Tambahan
ALTER TYPE location_type ADD VALUE IF NOT EXISTS 'STAGING_INBOUND';
ALTER TYPE location_type ADD VALUE IF NOT EXISTS 'STAGING_OUTBOUND';
ALTER TYPE location_type ADD VALUE IF NOT EXISTS 'QUARANTINE';

-- 2. Tabel Master Batch / Lot Produk
CREATE TABLE IF NOT EXISTS stock_batches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id    UUID NOT NULL REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    batch_number  VARCHAR(100) NOT NULL,
    mfg_date      DATE,
    expiry_date   DATE NOT NULL,
    status        VARCHAR(50) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'EXPIRED', 'QUARANTINED', 'DEPLETED')),
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, product_id, batch_number),
    UNIQUE (id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_stock_batches_lookup ON stock_batches(tenant_id, product_id, expiry_date ASC);

-- 3. Hubungkan batch_id ke Buku Besar dan Transaksi
ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS batch_id UUID REFERENCES stock_batches(id);
ALTER TABLE stock_receipt_items ADD COLUMN IF NOT EXISTS batch_id UUID REFERENCES stock_batches(id);
ALTER TABLE stock_receipt_items ADD COLUMN IF NOT EXISTS batch_number VARCHAR(100);
ALTER TABLE stock_receipt_items ADD COLUMN IF NOT EXISTS expiry_date DATE;
ALTER TABLE stock_receipt_items ADD COLUMN IF NOT EXISTS mfg_date DATE;

ALTER TABLE delivery_order_items ADD COLUMN IF NOT EXISTS batch_id UUID REFERENCES stock_batches(id);

-- RLS Policies
ALTER TABLE stock_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_batches FORCE ROW LEVEL SECURITY;
CREATE POLICY stock_batches_tenant_isolation ON stock_batches
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
```

#### Migrasi SQL 2: `034_wms_putaway_and_qc_inspections.sql`
Menyediakan alur tugas Putaway dan dokumen Berita Acara Kerusakan (BAK).
```sql
-- 1. Tabel QC Inspections & Berita Acara Kerusakan
CREATE TABLE IF NOT EXISTS qc_inspections (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    receipt_id     UUID NOT NULL REFERENCES stock_receipts(id, tenant_id) ON DELETE CASCADE,
    inspector_id   UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    gross_cartons  INTEGER NOT NULL DEFAULT 0,
    damaged_count  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    shortage_count NUMERIC(20, 4) NOT NULL DEFAULT 0,
    overage_count  NUMERIC(20, 4) NOT NULL DEFAULT 0,
    bak_number     VARCHAR(100),
    bak_notes      TEXT,
    bak_photo_urls TEXT[],
    driver_signed  BOOLEAN NOT NULL DEFAULT FALSE,
    driver_name    VARCHAR(100),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, tenant_id)
);

-- 2. Tabel Tugas Putaway
CREATE TABLE IF NOT EXISTS putaway_tasks (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    receipt_id            UUID NOT NULL REFERENCES stock_receipts(id, tenant_id) ON DELETE CASCADE,
    product_id            UUID NOT NULL REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    batch_id              UUID REFERENCES stock_batches(id),
    staging_location_id   UUID NOT NULL REFERENCES warehouse_locations(id, tenant_id),
    suggested_location_id UUID NOT NULL REFERENCES warehouse_locations(id, tenant_id),
    confirmed_location_id UUID REFERENCES warehouse_locations(id, tenant_id),
    quantity              NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    status                VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED')),
    assigned_to           UUID REFERENCES users(id) ON DELETE SET NULL,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, tenant_id)
);
```

#### Migrasi SQL 3: `035_wms_outbound_waves_and_manifests.sql`
Menyediakan Picking List, Gelombang Pesanan (Wave), Meja Kemas QC, dan Manifest Ekspedisi.
```sql
-- 1. Tabel Wave & Picking List
CREATE TABLE IF NOT EXISTS pick_waves (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    wave_number    VARCHAR(100) NOT NULL,
    warehouse_id   UUID NOT NULL REFERENCES warehouses(id, tenant_id),
    courier_filter VARCHAR(100),
    status         VARCHAR(50) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED')),
    created_by     UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, wave_number),
    UNIQUE (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS picking_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    wave_id         UUID REFERENCES pick_waves(id, tenant_id) ON DELETE SET NULL,
    do_id           UUID NOT NULL REFERENCES delivery_orders(id, tenant_id) ON DELETE CASCADE,
    product_id      UUID NOT NULL REFERENCES products(id, tenant_id),
    batch_id        UUID REFERENCES stock_batches(id),
    source_loc_id   UUID NOT NULL REFERENCES warehouse_locations(id, tenant_id),
    requested_qty   NUMERIC(20, 4) NOT NULL,
    picked_qty      NUMERIC(20, 4) NOT NULL DEFAULT 0,
    status          VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PICKED', 'SHORTAGE')),
    picker_id       UUID REFERENCES users(id),
    picked_at       TIMESTAMPTZ,
    UNIQUE (id, tenant_id)
);

-- 2. Manifest Pengiriman Ekspedisi (Gabungan DO)
CREATE TABLE IF NOT EXISTS shipping_manifests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    manifest_number VARCHAR(100) NOT NULL,
    warehouse_id    UUID NOT NULL REFERENCES warehouses(id, tenant_id),
    expedition_name VARCHAR(100) NOT NULL,
    driver_name     VARCHAR(100) NOT NULL,
    vehicle_plate   VARCHAR(50) NOT NULL,
    total_packages  INTEGER NOT NULL DEFAULT 0,
    status          VARCHAR(50) NOT NULL DEFAULT 'STAGED' CHECK (status IN ('STAGED', 'LOADED', 'DISPATCHED')),
    driver_signature_svg TEXT,
    dispatched_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, manifest_number),
    UNIQUE (id, tenant_id)
);

ALTER TABLE delivery_orders ADD COLUMN IF NOT EXISTS manifest_id UUID REFERENCES shipping_manifests(id);
ALTER TABLE delivery_orders ADD COLUMN IF NOT EXISTS package_weight_kg NUMERIC(10, 2);
ALTER TABLE delivery_orders ADD COLUMN IF NOT EXISTS package_box_type VARCHAR(50);
```

---

### 4.2 Mesin Status (State Machines) & Validasi Transisi

#### Inbound Lifecycle State Machine
```
[DRAFT] 
   │
   ├─► [CANCELLED]
   │
   ▼
[QC_PENDING] (Tiba di Staging Inbound, dihitung koli)
   │
   ▼
[QC_PASSED / BAK_RECORDED] (Inspeksi fisik, catat batch & exp date)
   │
   ▼
[PUTAWAY_ACTIVE] (Label LPN dicetak, tugas penataan terbit)
   │
   ▼
[POSTED] (Stok dikonfirmasi di rak definitif, Ready for Sale)
```

#### Outbound Lifecycle State Machine
```
[DRAFT] 
   │
   ▼
[CONFIRMED] (Pesanan disetujui, stok dialokasikan / FEFO reservation)
   │
   ▼
[PICKING] (Petugas menyusuri lorong rak & scan barcode rak + SKU)
   │
   ▼
[PICKED] (Seluruh item terkumpul di troli & dibawa ke meja packing)
   │
   ▼
[PACKING_QC] (100% SKU scan ulang di meja kemas, dimasukkan dus)
   │
   ▼
[PACKED] (Packing slip dimasukkan, resi termal AWB tertempel)
   │
   ▼
[STAGED] (Paket ditaruh di Outbound Staging Area per ekspedisi)
   │
   ▼
[SHIPPED] (Loading scan ke bak truk, tanda tangan DO/Manifest, mutasi stok final)
```

---

### 4.3 Backend Go (Hexagonal Clean Architecture & API Endpoints)

Endpoint baru yang disediakan di layer REST API (`backend/go-core/internal/handler/`):

| Method | Endpoint | Fungsi & Deskripsi |
| :--- | :--- | :--- |
| `GET` / `POST` | `/api/v1/wms/batches` | Daftar & pembuatan nomor Batch/Lot dengan masa kedaluwarsa. |
| `POST` | `/api/v1/wms/receipts/{id}/qc` | Submit form hasil QC, Blind Count, dan foto Berita Acara Kerusakan (BAK). |
| `GET` | `/api/v1/wms/putaway/tasks` | Mengambil daftar tugas penataan rak aktif per gudang. |
| `POST` | `/api/v1/wms/putaway/tasks/{id}/confirm` | Konfirmasi pemindahan fisik dari Staging ke Rak melalui pemindaian barcode. |
| `POST` | `/api/v1/wms/outbound/waves` | Membuat kelompok gelombang picking (*Wave Release*) per ekspedisi. |
| `GET` | `/api/v1/wms/outbound/picking/tasks` | Mengambil instruksi picking terurut rute terpendek untuk PDA. |
| `POST` | `/api/v1/wms/outbound/picking/scan` | Validasi scan barcode rak dan produk saat pengambilan barang. |
| `POST` | `/api/v1/wms/outbound/pack-station/verify` | Pemindaian verifikasi 100% di meja packing sebelum cetak AWB. |
| `POST` | `/api/v1/wms/outbound/manifests` | Membuat lembar manifest serah terima ekspedisi menggabungkan Surat Jalan. |
| `POST` | `/api/v1/wms/outbound/loading/scan` | Pemindaian barcode resi saat paket dimuat ke dalam armada truk. |

---

### 4.4 Frontend Next.js 15 UI/UX & PDA Scanner Engine

Submodul dan halaman baru di `app/(app)/wms/`:
1. `/wms/inbound/qc` &mdash; **Layar Meja QC Inbound:**
   - Input Blind Count per SKU.
   - Datepicker tanggal kedaluwarsa & nomor batch otomatis/manual.
   - Unggah foto bukti barang cacat & cetak dokumen formal Berita Acara Kerusakan (BAK).
2. `/wms/outbound/pack-station` &mdash; **Layar Meja Packing QC:**
   - Dirancang khusus untuk monitor stasiun pengepakan.
   - Operator cukup memindai nomor Surat Jalan, lalu memindai setiap fisik barang. Kotak indikator berubah hijau saat kuantitas 100% cocok.
   - Peringatan suara (*audio beeper*) keras jika ada barang salah atau berlebih (*mispick alert*).
3. `/wms/outbound/manifests` &mdash; **Konsolidasi & Manifest Ekspedisi:**
   - Mengelompokkan paket Surat Jalan berdasarkan kurir (JNE, J&T, SiCepat, Truk Internal).
   - Kanvas tanda tangan digital (*e-signature*) sopir kurir penjemput.
   - Cetak lembar Manifest Staging Sheet ukuran A4.
4. Perombakan Total `/wms/scanner` &mdash; **PDA Task-Driven Engine:**
   - Memiliki 3 tab tugas operasional:
     - **Mode Putaway:** Memandu operator memindahkan barang dari Staging ke Bin.
     - **Mode Picking:** Menampilkan urutan rak terpendek untuk mengambil barang pesanan.
     - **Mode Loading:** Memindai paket masuk ke bak truk.
   - Dukungan tombol besar (ergonomis untuk sarung tangan kerja) dan responsivitas Webview Android $\le 30$ ms.

---

### 4.5 Integrasi Perangkat Keras (Thermal Printer & Handheld PDA)

1. **Format Cetak Stiker Termal Barcode (100 x 150 mm):**
   - **Label Palet LPN:** Memuat Barcode Code128 / QR Code nomor LPN, Nama Gudang, Tanggal Masuk, dan daftar ringkas SKU di atas palet.
   - **Label Resi Pengiriman (AWB Ekspedisi):** Standar 3PL logistik (KOP Ekspedisi, Barcode No. Resi, Alamat Tujuan, Rute Pengiriman, Berat Paket, dan Daftar Isi Barang Ringkas).
2. **Kesesuaian Printer:**
   - Mendukung pencetakan langsung dari browser menggunakan driver printer thermal (Zebra, Xprinter, TSC, Honeywell) dengan media ukuran $100\text{mm} \times 150\text{mm}$ (4" x 6") tanpa margin potong.

---

## 5. Roadmap Eksekusi Bertahap (Sprint 1 s/d Sprint 5)

```
┌────────────────────────────────────────────────────────────────────────┐
│                        ROADMAP EKSEKUSI WMS                            │
├──────────┬─────────────────────────────────────┬───────────────────────┤
│ Sprint 1 │ Fondasi Batch/Exp Date & Staging    │ 2 – 3 Hari Kerja      │
├──────────┼─────────────────────────────────────┼───────────────────────┤
│ Sprint 2 │ QC Inbound, Karantina & Dokumen BAK │ 2 Hari Kerja          │
├──────────┼─────────────────────────────────────┼───────────────────────┤
│ Sprint 3 │ Picking FEFO, Meja Packing & AWB    │ 3 Hari Kerja          │
├──────────┼─────────────────────────────────────┼───────────────────────┤
│ Sprint 4 │ Manifest Ekspedisi & Loading Scan   │ 2 Hari Kerja          │
├──────────┼─────────────────────────────────────┼───────────────────────┤
│ Sprint 5 │ Docking Scheduling & Palet LPN      │ 3 – 4 Hari Kerja      │
└──────────┴─────────────────────────────────────┴───────────────────────┘
```

---

## 6. Living Execution Checklist (Tracking Kemajuan Real-Time)

> 💡 **Instruksi untuk AI / Developer:**  
> Dilarang menandai checkbox menjadi `[x]` sebelum kode di-commit ke Git dan diuji melalui verifikasi otomatis yang berhasil. Sertakan nomor Commit SHA pada setiap poin yang selesai.

### SPRINT 1: Fondasi Batch, Expiry Date & 2-Step Inbound (Staging &rarr; Putaway)
- [ ] **DB-01:** Buat migrasi SQL `033_wms_batches_and_staging_locations.sql` (tabel `stock_batches`, tipe `STAGING_INBOUND`, dan kolom `batch_id`).
- [ ] **BE-01:** Implementasikan entitas domain `Batch` dan repository PostgreSQL di Go backend.
- [ ] **BE-02:** Perbarui usecase penerimaan barang agar mencatat `batch_id` dan memasukkan barang pertama kali ke lokasi `STAGING_INBOUND`.
- [ ] **BE-03:** Buat usecase dan endpoint `Putaway` untuk memindahkan stok dari Staging ke Rak definitif.
- [ ] **FE-01:** Tambahkan input Nomor Batch & Tanggal Kedaluwarsa pada modal form Barang Masuk (`/wms/inbound`).
- [ ] **FE-02:** Buat modal cetak label stiker barcode SKU + Batch + Exp Date dari detail penerimaan.
- [ ] **FE-03:** Buat layar panduan Putaway sederhana di antarmuka web/mobile.
- [ ] **CR-01a (DB/BE):** Tabel `product_categories` + `products.category_id` (nullable, RLS), CRUD kategori, kategori ikut di response produk & ekspor.
- [ ] **CR-01b (FE):** Field kategori di form Produk, kolom + filter kategori di tabel Produk.
- [ ] **CR-03a (DB/BE):** Tabel `product_default_locations(product_id, warehouse_id, location_id)`; putaway memakai rak default sebagai saran pertama.
- [ ] **CR-03b (FE):** Atur rak default di detail Produk; layar Putaway menampilkan rak default dan meminta alasan bila operator memilih rak lain.
- [ ] **CR-05a (Invariant):** Setiap transisi status dokumen WMS baru (receipt, putaway) menyimpan user pelaku dan menulis `audit_logs`. Test Go gagal jika transisi tanpa user_id.
- [ ] **TEST-01:** Tulis pengujian unit Go untuk validasi batch expiry date dan mutasi staging-to-rack.
- [ ] **DEPLOY-01:** Push commit, pastikan GitHub Actions CI lulus, dan verifikasi deploy di Zeabur.

### SPRINT 2: Kontrol Mutu (QC Inbound), Karantina & Dokumen Kerusakan (BAK)
- [ ] **DB-02:** Buat migrasi SQL `034_wms_putaway_and_qc_inspections.sql` (tabel `qc_inspections` & `putaway_tasks`).
- [ ] **BE-04:** Endpoint submit QC Inbound dengan perhitungan selisih (*shortage/overage*) dan barang rusak.
- [ ] **BE-05:** Mutasi barang rusak otomatis dialokasikan ke lokasi `@QUARANTINE` bukan langsung `@SCRAP`.
- [ ] **FE-04:** Halaman khusus `/wms/inbound/qc` untuk pemeriksaan mutu per SKU (Blind Count & Expire Date).
- [ ] **FE-05:** Komponen cetak dokumen resmi Berita Acara Kerusakan (BAK) berformat A4 dengan bukti foto cacat.
- [ ] **TEST-02:** Pengujian Go untuk isolasi stok karantina dan pembuatan dokumen BAK.
- [ ] **DEPLOY-02:** Push commit, verifikasi CI dan produksi Zeabur.

### SPRINT 3: Alur Outbound Inti (Picking FEFO, Meja Kemas QC 100% Scan & Resi Termal AWB)
- [ ] **DB-03:** Buat migrasi SQL `035_wms_outbound_waves_and_manifests.sql` (tabel `picking_tasks` & penambahan kolom kemasan).
- [ ] **BE-06:** Logic alokasi rak FEFO: Surat Jalan otomatis memilih batch dengan masa kedaluwarsa terdekat.
- [ ] **BE-07:** Endpoint verifikasi meja kemas (*packing station*): memvalidasi barcode SKU pesanan secara interaktif.
- [ ] **FE-06:** Format cetak dokumen *Picking List* terurut lokasi rak untuk petugas gudang.
- [ ] **FE-07:** Halaman stasiun meja kemas `/wms/outbound/pack-station` dengan antarmuka pencocokan scan 100% & audio beeper.
- [ ] **FE-08:** Generator cetak label stiker resi termal pengiriman AWB ukuran 100x150 mm.
- [ ] **CR-02a (DB/BE):** `delivery_orders.customer_id` (nullable, FK `customers`) + kolom pelaku `created_by, confirmed_by, packed_by, dispatched_by`.
- [ ] **CR-02b (FE):** Pilih customer di form Surat Jalan + quick-add customer baru via modal (pakai `POST /api/v1/customers` yang sudah ada).
- [ ] **CR-04a (BE):** Tambah `outbound_qty_today/month`, `top_outbound_products[10]`, `top_customers[10]` di `GET /dashboard/summary` dengan filter periode 7/30/90 hari.
- [ ] **CR-04b (FE):** Kartu "Barang Keluar", tabel Top 10 Produk Keluar, dan Top 10 Toko/Customer di Dashboard.
- [ ] **CR-05b (FE):** Kolom "Dibuat oleh / Disetujui oleh" di tabel Barang Masuk, Surat Jalan, Transfer, Opname, Scrap + panel "Riwayat Aktivitas" di detail dokumen.
- [ ] **TEST-03:** Pengujian otomatis alokasi FEFO dan pencegahan salah kirim barang di meja kemas.
- [ ] **DEPLOY-03:** Push commit, verifikasi CI dan produksi Zeabur.

### SPRINT 4: Konsolidasi Staging, Manifest Ekspedisi & Loading Scan Truk
- [ ] **DB-04:** Tabel `shipping_manifests` dan relasi ke multi-DO.
- [ ] **BE-08:** Endpoint penerbitan manifest kurir dan endpoint validasi pemuatan armada (*Loading Scan*).
- [ ] **FE-09:** Halaman konsolidasi manifest `/wms/outbound/manifests` dengan kanvas tanda tangan sopir kurir.
- [ ] **FE-10:** Mode pemindaian loading truk sebelum status Surat Jalan final berubah menjadi `SHIPPED`.
- [ ] **TEST-04:** Pengujian mutasi ledger Goods Issue saat manifest ditutup.
- [ ] **DEPLOY-04:** Push commit, verifikasi CI dan produksi Zeabur.

### SPRINT 5: Fitur Lanjutan Enterprise (Dock Scheduling & Kontainerisasi LPN)
- [ ] **DB-05:** Tabel master dermaga `inbound_docks`, jadwal armada ASN, dan kontainer palet `stock_lpns`.
- [ ] **BE-09:** Endpoint penjadwalan dock & generator penomoran LPN palet dinamis.
- [ ] **FE-11:** Papan jadwal dermaga `/wms/inbound/dock-board` untuk antrean truk di gerbang.
- [ ] **FE-12:** Mode PDA interaktif penuh untuk operator MHE / Forklift.
- [ ] **TEST-05:** Pengujian menyeluruh end-to-end simulasi 100 transaksi barang masuk dan keluar.
- [ ] **DEPLOY-05:** Verifikasi final seluruh alur di domain resmi `https://tayooli.my.id`.

---
*Dokumen ini merupakan properti arsitektur resmi Tayooli ERP Core. Seluruh modifikasi wajib melalui peninjauan arsitektur tim CTO.*
