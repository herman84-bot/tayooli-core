# SPEC-WMS-SPRINT-4: KONSOLIDASI STAGING, MANIFEST EKSPEDISI & LOADING SCAN TRUK

- **Dokumen ID:** `SPEC-WMS-2026-10-08-SPRINT-4-MANIFESTS`
- **Status:** DRAFT PROPOSED FOR REVIEW
- **Sprint:** Sprint 4 (WMS Outbound Tahap 5–6)
- **Dasar Arsitektur:** Master PRD §3.2, §4.1, §4.3, §4.4, §5 & ADR-014 Invariant 1 (Double-Entry Atomic Stock Ledger)
- **Target Repository:** `tayooli-core`

---

## 1. TUJUAN & RUANG LINGKUP

Mengimplementasikan alur fisik penyerahan barang keluar (Outbound Tahap 5 & 6) agar tidak ada barang keluar gudang tanpa validasi muat armada (*loading scan*) dan bukti serah terima resmi (*shipping manifest* ber-tanda tangan kurir).

### 1.1 Invariant Kunci (Non-Negotiable)
1. **Double-Entry Atomic Ledger (ADR-014):** Pemotongan stok gudang ke `@CUSTOMER` terjadi atomik saat manifest diubah menjadi `DISPATCHED`. Seluruh DO di dalam manifest berpindah status ke `SHIPPED` dalam 1 transaksi database (`tx.Commit`).
2. **Anti-Misload Guard:** Barcode scanner pada loading scan memvalidasi nomor DO/AWB terhadap ekspedisi dan nomor manifest. Paket kurir lain otomatis ditolak.
3. **Legal Dispatch Artifact:** Manifest wajib merekam nama sopir, plat nomor armada, kontak, dan kanvas tanda tangan digital sopir (SVG).
4. **12 Modul Kepatuhan (KO-2):** UI terintegrasi ke `/wms/arus-barang?mode=keluar` sebagai tab "Manifest Ekspedisi" (tidak menambah item menu sidebar baru).

---

## 2. SKEMA DATABASE & MIGRASI (`036_wms_shipping_manifests.sql`)

### 2.1 Tabel `shipping_manifests`
```sql
CREATE TABLE IF NOT EXISTS shipping_manifests (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id         UUID NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
    manifest_number      VARCHAR(100) NOT NULL,
    expedition_name      VARCHAR(100) NOT NULL,
    driver_name          VARCHAR(100) NOT NULL,
    vehicle_plate        VARCHAR(50) NOT NULL,
    driver_phone         VARCHAR(50) NULL,
    total_packages       INTEGER NOT NULL DEFAULT 0 CHECK (total_packages >= 0),
    total_weight_kg      NUMERIC(10, 3) NOT NULL DEFAULT 0 CHECK (total_weight_kg >= 0),
    status               VARCHAR(50) NOT NULL DEFAULT 'STAGED' 
                         CHECK (status IN ('STAGED', 'LOADED', 'DISPATCHED', 'CANCELLED')),
    driver_signature_svg TEXT NULL,
    notes                TEXT NULL,
    created_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    dispatched_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    dispatched_at        TIMESTAMPTZ NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, manifest_number)
);

CREATE INDEX IF NOT EXISTS idx_shipping_manifests_tenant_wh 
    ON shipping_manifests(tenant_id, warehouse_id);
CREATE INDEX IF NOT EXISTS idx_shipping_manifests_status 
    ON shipping_manifests(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_shipping_manifests_expedition 
    ON shipping_manifests(tenant_id, expedition_name);

ALTER TABLE shipping_manifests ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipping_manifests FORCE ROW LEVEL SECURITY;
CREATE POLICY shipping_manifests_tenant_isolation ON shipping_manifests
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
```

### 2.2 Relasi pada `delivery_orders`
```sql
ALTER TABLE delivery_orders
    ADD COLUMN IF NOT EXISTS manifest_id UUID REFERENCES shipping_manifests(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS loading_scanned_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS loading_scanned_by UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_orders_manifest ON delivery_orders(tenant_id, manifest_id);
```

---

## 3. ARSITEKTUR BACKEND GO (REST API & USECASE)

### 3.1 Domain Model (`internal/domain/wms_manifest.go`)
- `type ShippingManifestStatus string` (`STAGED`, `LOADED`, `DISPATCHED`, `CANCELLED`)
- `type ShippingManifest struct`: metadata manifest, list item DO ringkas, timestamp, actor.
- `type CreateManifestRequest struct`: `WarehouseID`, `ExpeditionName`, `DriverName`, `VehiclePlate`, `DriverPhone`, `DeliveryOrderIDs []uuid.UUID`, `Notes`.
- `type LoadingScanRequest struct`: `Barcode string` (nomor DO atau nomor resi AWB).
- `type DispatchManifestRequest struct`: `DriverSignatureSVG string`, `Notes *string`.
- `type WMSOutboundKPISummary struct`: 8 KPI SOP (Dock-to-Stock, Receiving Accuracy, PO Compliance, Backlog Inbound, Order-to-Dispatch, Picking Accuracy, On-Time Shipment, Backlog Outbound).

### 3.2 Endpoints REST API
| Method | Endpoint | Fungsi |
|---|---|---|
| `GET` | `/api/v1/wms/outbound/manifests` | Daftar manifest dengan filter gudang, status, ekspedisi |
| `POST` | `/api/v1/wms/outbound/manifests` | Buat manifest baru menggabungkan DO berstatus `PACKED` / `STAGED` |
| `GET` | `/api/v1/wms/outbound/manifests/{id}` | Detail manifest beserta status loading per DO |
| `POST` | `/api/v1/wms/outbound/manifests/{id}/loading-scan` | Pemindaian barcode paket saat muat bak truk |
| `POST` | `/api/v1/wms/outbound/manifests/{id}/dispatch` | Finalisasi serah terima ekspedisi, simpan TTD, eksekusi pemotongan stok keluar |
| `GET` | `/api/v1/wms/outbound/kpi` | Mengambil data 8 KPI SOP operasional WMS |

### 3.3 Logika Atomik Dispatch Manifest
1. Kunci manifest `FOR UPDATE`.
2. Validasi status harus `LOADED` (seluruh paket di dalamnya sudah lolos scan loading) atau minimal 1 paket jika parsial diizinkan.
3. Untuk setiap `delivery_order` di manifest:
   - Verifikasi status DO `PACKED` / `STAGED`.
   - Potong stok rak internal & buat mutasi stok `DO-SHIP-...` ke `@CUSTOMER`.
   - Update status DO menjadi `SHIPPED` dengan `dispatched_by` dan `dispatched_at`.
   - Catat jejak audit `delivery_order` dan `shipping_manifest`.
4. Update manifest menjadi `DISPATCHED` dengan `driver_signature_svg`.
5. `tx.Commit()`.

---

## 4. ANTARMUKA PENGGUNA (FRONTEND NEXT.JS 15)

### 4.1 Subview Manifest di `/wms/arus-barang?mode=keluar`
- Tambah tab ke-4 di panel Delivery Orders: `Daftar Surat Jalan` | `Gelombang Picking` | `Stasiun Packing` | `Manifest Ekspedisi`.
- **Tombol "Buat Manifest":**
  - Modal form input ekspedisi (JNE, J&T, SiCepat, Truk Internal), sopir, plat nomor.
  - Multi-select DO yang berstatus `PACKED` dengan filter ekspedisi.
- **Detail Manifest & Monitoring Loading:**
  - Progress bar pemuatan (contoh: 8/10 paket termuat).
  - Indikator hijau untuk paket yang sudah discan.
- **Modal Penyelesaian & Tanda Tangan Sopir:**
  - Komponen kanvas HTML5 responsif untuk coretan tanda tangan sopir.
  - Tombol simpan dan terbitkan serah terima.
- **Format Cetak Dokumen A4 (`PrintShippingManifest.tsx`):**
  - KOP dokumen resmi Surat Manifest Ekspedisi.
  - Tabel rincian DO (No. DO, Customer, Kota Tujuan, Berat kg, Koli, Resi AWB).
  - Kotak tanda tangan petugas gudang dan sopir ekspedisi.

### 4.2 PDA Scanner Mode Loading Truk (`/wms/scanner`)
- Tambah tab mode ke-4: `LOADING TRUK`.
- Memilih nomor manifest aktif & ekspedisi.
- Scan kamera / hardware PDA:
  - Bunyi beeper sukses saat nomor DO/AWB terverifikasi di manifest.
  - Bunyi buzzer peringatan jika nomor DO salah ekspedisi / tidak terdaftar.

### 4.3 Widget 8 KPI WMS di Dashboard
- Menampilkan metrik SOP pada kartu ringkas:
  - Inbound: Dock-to-Stock (jam/menit), Akurasi Terima (%), Kepatuhan PO (%), Backlog Inbound.
  - Outbound: Order-to-Dispatch (jam), Akurasi Pick (%), Tepat Waktu (%), Backlog Belum Kirim.

---

## 5. RENCANA PENGUJIAN & VERIFIKASI (QA POLICY COMPLIANCE)

1. **Unit & Integration Test Go:**
   - Test pembuatan manifest multi-DO dengan tenant RLS isolation.
   - Test loading scan valid vs invalid barcode DO.
   - Test atomic dispatch: memastikan stok terpotong, DO menjadi `SHIPPED`, dan rollback jika satu baris gagal.
2. **Jest Frontend:**
   - Test render daftar manifest dan modal pembuatan.
   - Test kanvas tanda tangan (ekspor SVG string).
   - Test komponen cetak `PrintShippingManifest`.
3. **Adversarial QA:**
   - Manifest kosong (0 DO).
   - Scan barcode DO yang sudah di-dispatch atau milik gudang lain.
   - Dispatch tanpa tanda tangan sopir.
   - Dispatch ganda (idempotency check).

---

## 6. DEFINITION OF DONE SPRINT 4
- Migrasi database `036_wms_shipping_manifests.sql` terpasang & lolos test.
- Backend Go handlers, usecases, repo lolos `go vet` dan unit test.
- Frontend lulus `tsc --noEmit` dan `npm test`.
- Audit utang kode via `ai-debt-detector` bersih.
- Commit, push branch `main`, CI lulus, dan Zeabur live deployment verified.
