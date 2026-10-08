# SPEC-WMS-SPRINT-5: DOCK SCHEDULING, INBOUND QUEUE & PALLET LPN CONTAINERIZATION

- **Dokumen ID:** `SPEC-WMS-2026-10-08-SPRINT-5-DOCKS-LPN`
- **Status:** DRAFT PROPOSED FOR REVIEW
- **Sprint:** Sprint 5 (WMS Advanced Enterprise - Inbound Dock Management & LPN Pallet Handling)
- **Dasar Arsitektur:** Master PRD §3.2, §4.5, §5, ADR-014 Invariant 1 (Double-Entry Atomic Ledger) & OCA/wms dock-management patterns
- **Target Repository:** `tayooli-core`

---

## 1. TUJUAN & RUANG LINGKUP

Mengimplementasikan fitur pergudangan tingkat lanjut (*Enterprise WMS*) untuk mengelola gerbang/dermaga bongkar muat armada (*Dock Scheduling & Inbound Queue*) serta kontainerisasi palet barang menggunakan *License Plate Number* (LPN).

### 1.1 Invariant Kunci (Non-Negotiable)
1. **Dock Exclusivity & Anti-Collision:** Satu dermaga aktif (`inbound_docks`) hanya boleh melayani 1 antrean armada (*appointment*) berstatus `OCCUPIED` / `UNLOADING` dalam satu rentang waktu.
2. **Atomic LPN Containerization:** 1 LPN mewakili satu unit palet fisik yang memuat 1 atau lebih lot batch produk. Pemindahan LPN ke lokasi rak otomatis memindahkan seluruh stok yang terikat di dalamnya dalam 1 transaksi buku besar double-entry (ADR-014).
3. **12 Modul Kepatuhan (KO-2):** Papan jadwal dermaga (*Dock Board*) diintegrasikan ke `/wms/arus-barang?mode=masuk` sebagai tab "Antrean Dermaga (Dock)". Mode LPN diintegrasikan ke `/wms/scanner` sebagai tab "PALLET (LPN)".
4. **Tenant Isolation:** Seluruh tabel baru (`inbound_docks`, `dock_appointments`, `stock_lpns`, `stock_lpn_items`) diproteksi PostgreSQL 15 Row-Level Security (RLS).

---

## 2. SKEMA DATABASE & MIGRASI (`037_wms_docks_and_lpns.sql`)

### 2.1 Tabel `inbound_docks`
```sql
CREATE TABLE IF NOT EXISTS inbound_docks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id  UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    dock_code     VARCHAR(50) NOT NULL,
    dock_name     VARCHAR(100) NOT NULL,
    dock_type     VARCHAR(50) NOT NULL DEFAULT 'INBOUND' CHECK (dock_type IN ('INBOUND', 'OUTBOUND', 'CROSS_DOCK')),
    max_tonnage   NUMERIC(10, 2) NOT NULL DEFAULT 10.00 CHECK (max_tonnage >= 0),
    status        VARCHAR(50) NOT NULL DEFAULT 'AVAILABLE' CHECK (status IN ('AVAILABLE', 'OCCUPIED', 'MAINTENANCE')),
    notes         TEXT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, warehouse_id, dock_code)
);

CREATE INDEX IF NOT EXISTS idx_inbound_docks_wh ON inbound_docks(tenant_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inbound_docks_status ON inbound_docks(tenant_id, status);

ALTER TABLE inbound_docks ENABLE ROW LEVEL SECURITY;
ALTER TABLE inbound_docks FORCE ROW LEVEL SECURITY;
CREATE POLICY inbound_docks_tenant_isolation ON inbound_docks
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
```

### 2.2 Tabel `dock_appointments`
```sql
CREATE TABLE IF NOT EXISTS dock_appointments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id        UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    dock_id             UUID NULL REFERENCES inbound_docks(id) ON DELETE SET NULL,
    appointment_number  VARCHAR(100) NOT NULL,
    vendor_name         VARCHAR(150) NOT NULL,
    vehicle_plate       VARCHAR(50) NOT NULL,
    driver_name         VARCHAR(100) NOT NULL,
    driver_phone        VARCHAR(50) NULL,
    po_reference        VARCHAR(100) NULL,
    estimated_arrival   TIMESTAMPTZ NOT NULL,
    actual_arrival      TIMESTAMPTZ NULL,
    start_unloading_at  TIMESTAMPTZ NULL,
    completed_at        TIMESTAMPTZ NULL,
    status              VARCHAR(50) NOT NULL DEFAULT 'SCHEDULED' 
                        CHECK (status IN ('SCHEDULED', 'ARRIVED', 'UNLOADING', 'COMPLETED', 'CANCELLED')),
    notes               TEXT NULL,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, appointment_number)
);

CREATE INDEX IF NOT EXISTS idx_dock_app_wh_status ON dock_appointments(tenant_id, warehouse_id, status);
CREATE INDEX IF NOT EXISTS idx_dock_app_eta ON dock_appointments(tenant_id, estimated_arrival);

ALTER TABLE dock_appointments ENABLE ROW LEVEL SECURITY;
ALTER TABLE dock_appointments FORCE ROW LEVEL SECURITY;
CREATE POLICY dock_appointments_tenant_isolation ON dock_appointments
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
```

### 2.3 Tabel `stock_lpns` & `stock_lpn_items` (Pallet License Plate Number)
```sql
CREATE TABLE IF NOT EXISTS stock_lpns (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id  UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    lpn_code      VARCHAR(100) NOT NULL,
    location_id   UUID NOT NULL REFERENCES warehouse_locations(id) ON DELETE RESTRICT,
    pallet_type   VARCHAR(50) NOT NULL DEFAULT 'WOODEN' CHECK (pallet_type IN ('WOODEN', 'PLASTIC', 'METAL', 'CAGE')),
    status        VARCHAR(50) NOT NULL DEFAULT 'STAGED' CHECK (status IN ('STAGED', 'STORED', 'PICKED', 'SHIPPED', 'DECOMMISSIONED')),
    max_weight_kg NUMERIC(10, 2) NOT NULL DEFAULT 1000.00 CHECK (max_weight_kg >= 0),
    total_weight_kg NUMERIC(10, 2) NOT NULL DEFAULT 0.00 CHECK (total_weight_kg >= 0),
    notes         TEXT NULL,
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, warehouse_id, lpn_code)
);

CREATE INDEX IF NOT EXISTS idx_stock_lpns_wh_loc ON stock_lpns(tenant_id, warehouse_id, location_id);
CREATE INDEX IF NOT EXISTS idx_stock_lpns_status ON stock_lpns(tenant_id, status);

ALTER TABLE stock_lpns ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_lpns FORCE ROW LEVEL SECURITY;
CREATE POLICY stock_lpns_tenant_isolation ON stock_lpns
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE TABLE IF NOT EXISTS stock_lpn_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    lpn_id      UUID NOT NULL REFERENCES stock_lpns(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    batch_id    UUID NOT NULL REFERENCES stock_batches(id) ON DELETE RESTRICT,
    quantity    NUMERIC(14, 4) NOT NULL CHECK (quantity > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, lpn_id, batch_id)
);

CREATE INDEX IF NOT EXISTS idx_stock_lpn_items_lpn ON stock_lpn_items(tenant_id, lpn_id);

ALTER TABLE stock_lpn_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_lpn_items FORCE ROW LEVEL SECURITY;
CREATE POLICY stock_lpn_items_tenant_isolation ON stock_lpn_items
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
```

---

## 3. ARSITEKTUR BACKEND GO (REST API & USECASE)

### 3.1 Domain Model (`internal/domain/wms_dock_lpn.go`)
- `type DockStatus string` (`AVAILABLE`, `OCCUPIED`, `MAINTENANCE`)
- `type AppointmentStatus string` (`SCHEDULED`, `ARRIVED`, `UNLOADING`, `COMPLETED`, `CANCELLED`)
- `type LPNStatus string` (`STAGED`, `STORED`, `PICKED`, `SHIPPED`, `DECOMMISSIONED`)
- `type InboundDock struct`
- `type DockAppointment struct`
- `type StockLPN struct` & `type StockLPNItem struct`
- `type StockLPNDetail struct` (Header LPN + Array Item Lot Produk)
- Request DTOs:
  - `CreateDockRequest`, `UpdateDockStatusRequest`
  - `CreateAppointmentRequest`, `UpdateAppointmentStatusRequest`, `AssignDockRequest`
  - `CreateLPNRequest`, `AddLPNItemRequest`, `MoveLPNRequest` (Putaway palet sekaligus)

### 3.2 REST API Endpoints
| Method | Endpoint | Fungsi |
|---|---|---|
| `GET` | `/api/v1/wms/docks` | Daftar dermaga gudang aktif beserta status keterisian |
| `POST` | `/api/v1/wms/docks` | Daftarkan dermaga bongkar muat baru |
| `PATCH` | `/api/v1/wms/docks/{id}/status` | Update status operasional dermaga |
| `GET` | `/api/v1/wms/dock-appointments` | Daftar antrean & jadwal kedatangan truk |
| `POST` | `/api/v1/wms/dock-appointments` | Daftarkan jadwal armada baru |
| `POST` | `/api/v1/wms/dock-appointments/{id}/assign-dock` | Alokasikan antrean ke dermaga tertentu |
| `POST` | `/api/v1/wms/dock-appointments/{id}/status` | Transisi status antrean (`ARRIVED`, `UNLOADING`, `COMPLETED`) |
| `GET` | `/api/v1/wms/lpns` | Daftar palet LPN aktif per gudang & lokasi rak |
| `POST` | `/api/v1/wms/lpns` | Buat kontainer palet LPN baru |
| `GET` | `/api/v1/wms/lpns/{id}` | Detail isi palet LPN (SKU, batch, kuantitas) |
| `POST` | `/api/v1/wms/lpns/{id}/items` | Tambah item lot batch ke dalam palet LPN |
| `POST` | `/api/v1/wms/lpns/{id}/move` | Putaway palet massal ke rak tujuan via forklift scan |

### 3.3 Logika Atomik Putaway LPN Massal (`MoveLPN`)
1. Kunci palet `stock_lpns` `FOR UPDATE`.
2. Validasi lokasi tujuan harus bertipe `INTERNAL` / `RACK` (bukan staging keluar atau scrap tanpa alasan).
3. Baca seluruh item di `stock_lpn_items`:
   - Untuk setiap item, buat pergerakan mutasi stok `PUTAWAY-LPN-...` dari lokasi asal ke lokasi tujuan rak dengan membawa `batch_id` yang valid.
   - Pindahkan saldo kuantitas stok per lokasi di tabel buku besar.
4. Perbarui `stock_lpns.location_id` ke lokasi rak tujuan baru dan ubah status menjadi `STORED`.
5. Catat audit log pemindahan palet LPN.
6. `tx.Commit()`.

---

## 4. ANTARMUKA PENGGUNA (FRONTEND NEXT.JS 15)

### 4.1 Submodul Dock Board di `/wms/arus-barang?mode=masuk`
- Tambah tab ke-3 di panel Barang Masuk: `Daftar Penerimaan (GRN)` | `Inspeksi QC & Karantina` | `Antrean Dermaga (Dock)`.
- **Kanban / Matrix Papan Dermaga (*Dock Board*):**
  - Kolom dermaga: Dock 1, Dock 2, Dock 3 dengan status visual (Hijau = Available, Biru = Unloading, Merah/Kuning = Occupied/Maintenance).
  - Kartu antrean: nomor appointment, vendor, plat nomor truk, nomor PO/SJ, estimasi kedatangan.
  - Tombol alokasi cepat: drag & drop atau tombol "Arahkan ke Dock" saat truk tiba.
- **Form Pendaftaran Antrean Truk:** Modal jadwal kedatangan PO / Surat Jalan vendor.

### 4.2 Submodul & Cetak Label Palet LPN
- Tab/Modal LPN di Arus Masuk:
  - Buat palet LPN saat penerimaan koli.
  - Cetak label stiker barcode palet A6 / 100x150 mm (`PrintLPNLabel.tsx`) untuk ditempel di plastik wrap palet.

### 4.3 Mode PDA Scanner: Putaway Palet LPN (`/wms/scanner`)
- Tambah tab mode ke-5 di scanner: `PALLET (LPN)`.
- Alur 2-Scan Operator Forklift/MHE:
  - Scan 1: Barcode Palet LPN (contoh: `LPN-20261008-0001`). Sistem memuat seluruh isi palet dan merekomendasikan rak tujuan.
  - Scan 2: Barcode Rak Tujuan (contoh: `RAK-A-01-02`).
  - Sistem mengeksekusi endpoint `POST /wms/lpns/{id}/move` dan membunyikan nada sukses Web Audio beeper. Seluruh isi palet berpindah seketika tanpa perlu scan 50 dus satu per satu.

---

## 5. RENCANA PENGUJIAN & VERIFIKASI

1. **Go Unit & Repo Tests:**
   - Validasi dock collision (menolak 2 appointment `UNLOADING` di dock yang sama).
   - Validasi pemindahan LPN massal (atomik, mutasi buku besar ke rak tujuan dengan `batch_id`).
   - Tenant isolation & RLS testing.
2. **Jest Frontend Tests:**
   - Render Dock Board dan interaksi alokasi armada.
   - Render cetak stiker barcode LPN (`PrintLPNLabel.tsx`).
   - Interaksi scanner PDA mode LPN.
3. **Audit AI Debt & Zeabur Deploy:**
   - Audit kegagalan rollback, error handling, dan zero/null edge cases via `ai-debt-detector`.
   - Push branch `main`, CI pass, verifikasi live produksi Zeabur.

---

## 6. DEFINITION OF DONE SPRINT 5
- Migrasi database `037_wms_docks_and_lpns.sql` terpasang & lolos test.
- Backend Go handlers, usecases, repo lolos `go vet`, `go build`, dan unit test.
- Frontend lulus `tsc --noEmit` dan `npm test`.
- Kepatuhan 12 modul (KO-2) terjaga rapi.
- Commit, push branch `main`, CI lulus, dan Zeabur live deployment verified.
