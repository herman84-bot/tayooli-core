# WMS Sprint 4: Shipping Manifests, Loading Scan & Outbound Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Menuntaskan seluruh alur WMS Outbound Tahap 5–6 (konsolidasi multi-DO per ekspedisi, loading scan barcode truk, kanvas tanda tangan sopir digital, pemotongan stok atomik saat serah terima, mode scanner loading truk, dan 8 KPI operasional SOP di dashboard).

**Architecture:** Menerapkan Clean Hexagonal Architecture di backend Go (database/sql + pgx) dengan transaksi atomik pemotongan stok ledger double-entry (`@CUSTOMER`) pada dispatch manifest. Di frontend Next.js 15, modul diintegrasikan ke `/wms/arus-barang?mode=keluar` sebagai tab "Manifest Ekspedisi" untuk mempertahankan batas 12 modul (KO-2).

**Tech Stack:** Go 1.24 (Chi router, pure SQL, PostgreSQL 15 RLS), Next.js 15 (App Router, Tailwind CSS, TanStack Query v5, Zustand, Lucide icons, HTML5 Canvas API).

**Spec:** `docs/superpowers/specs/2026-10-08-wms-sprint-4-shipping-manifests-loading-design.md`

## Global Constraints
- Strict 12 Modul: Dilarang menambah item rute baru di sidebar menu utama; integrasikan ke `/wms/arus-barang?mode=keluar` dan `/wms/scanner`.
- ADR-014 Invariant 1: Setiap mutasi stok keluar wajib membawa `batch_id` dan memotong stok dari lokasi rak internal ke lokasi `@CUSTOMER`.
- Anti-Misload: Loading scan wajib memvalidasi ekspedisi dan nomor DO/AWB; tolak paket beda kurir.
- Atomic Dispatch: Dispatch manifest dan seluruh perubahan status DO ke `SHIPPED` wajib berada dalam 1 transaksi SQL database (`tx.Commit`).

## Review Focus
1. Loading scan memvalidasi koli yang tidak terdaftar di manifest dan mengembalikan pesan error informatif tanpa crash.
2. Dispatch manifest tanpa tanda tangan sopir (string SVG kosong) wajib ditolak dengan HTTP 400.
3. Dispatch manifest berulang (idempotency) tidak boleh mengeksekusi pemotongan stok kedua kali (status transisi guard).
4. DO yang belum lolos loading scan (parsial) diberi konfirmasi khusus atau diblokir sesuai konfigurasi manifest.
5. Pemotongan stok multi-DO dalam 1 manifest me-rollback seluruh transaksi jika satu item mengalami insufficient stock atau kegagalan internal.

---

### Task 1: Database Migration `036_wms_shipping_manifests.sql`

**Files:**
- Create: `backend/go-core/migrations/036_wms_shipping_manifests.sql`
- Modify: `backend/go-core/migrations/migrations.go` (jika ada daftar berkas terdaftar)
- Test: `backend/go-core/internal/infra/postgres/wms_manifest_migration_test.go`

**Interfaces:**
- Consumes: `tenants`, `warehouses`, `delivery_orders`, `users`
- Produces: Table `shipping_manifests`, column `delivery_orders.manifest_id`, `delivery_orders.loading_scanned_at`, `delivery_orders.loading_scanned_by`

- [x] **Step 1: Tulis berkas migrasi SQL `036_wms_shipping_manifests.sql`**
```sql
-- 036_wms_shipping_manifests.sql
-- Sprint 4 Master PRD WMS: Shipping Manifests, Loading Scan, Courier Multi-DO consolidation
-- ADR-014 Invariant 1 & Master PRD §3.2, §4.1

BEGIN;

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

DROP POLICY IF EXISTS shipping_manifests_tenant_isolation ON shipping_manifests;
CREATE POLICY shipping_manifests_tenant_isolation ON shipping_manifests
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_setting', true), '')::uuid OR tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

ALTER TABLE delivery_orders
    ADD COLUMN IF NOT EXISTS manifest_id UUID REFERENCES shipping_manifests(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS loading_scanned_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS loading_scanned_by UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_delivery_orders_manifest ON delivery_orders(tenant_id, manifest_id);

COMMIT;
```

- [x] **Step 2: Jalankan syntax check migrasi melalui go vet/build**
Run: `go build ./...` di `backend/go-core`
Expected: exit 0

- [x] **Step 3: Commit migrasi database**
```bash
git add backend/go-core/migrations/036_wms_shipping_manifests.sql
git commit -m "feat(wms): add migration 036 for shipping manifests and loading scan tracking"
```

---

### Task 2: Backend Domain & Repository untuk Shipping Manifests

**Files:**
- Create: `backend/go-core/internal/domain/wms_manifest.go`
- Create: `backend/go-core/internal/infra/postgres/wms_manifest_repo.go`
- Modify: `backend/go-core/internal/domain/wms_outbound.go`
- Modify: `backend/go-core/internal/infra/postgres/wms_repo.go`

**Interfaces:**
- Consumes: `domain.DeliveryOrder`, `domain.StockMovement`
- Produces: `domain.ShippingManifest`, `domain.WMSOutboundRepository` methods (`CreateShippingManifest`, `ListShippingManifests`, `GetShippingManifestByID`, `ScanDOLoading`, `DispatchShippingManifest`, `GetWMSOutboundKPIs`)

- [x] **Step 1: Buat domain model di `internal/domain/wms_manifest.go`**
Definisikan struct `ShippingManifest`, `ShippingManifestItem`, status enum `ShippingManifestStatus`, request/response DTOs, dan error sentinel `ErrManifestNotFound`, `ErrInvalidManifestStatus`, `ErrManifestSignatureRequired`, `ErrDOMisload`.

- [x] **Step 2: Implementasikan repository PostgreSQL di `internal/infra/postgres/wms_manifest_repo.go`**
Implementasikan metode:
- `CreateShippingManifest`: insert header manifest, update `delivery_orders.manifest_id` dan ubah status DO ke `STAGED`.
- `GetShippingManifestByID`: select manifest dan load daftar DO terlampir dengan status loading-nya.
- `ListShippingManifests`: query dengan filter tenant, warehouse, status, expedition.
- `ScanDOLoading`: verifikasi barcode cocok dengan DO di manifest, set `loading_scanned_at` dan `loading_scanned_by`.
- `DispatchShippingManifest`: transaksi atomik multi-DO, potong stok buku besar via `DeductLocationStock`, update status DO ke `SHIPPED`, update status manifest ke `DISPATCHED` beserta signature SVG, catat audit logs.

- [x] **Step 3: Tulis unit test untuk domain & repository wms_manifest**
Run: `go test -v ./internal/domain/...`
Expected: PASS

- [x] **Step 4: Commit domain & repository**
```bash
git add backend/go-core/internal/domain/wms_manifest.go backend/go-core/internal/infra/postgres/wms_manifest_repo.go
git commit -m "feat(wms): implement domain models and postgres repository for shipping manifests"
```

---

### Task 3: Backend Usecase & REST Handler untuk Shipping Manifests

**Files:**
- Create: `backend/go-core/internal/usecase/wms/manifest_usecase.go`
- Create: `backend/go-core/internal/handler/wms_manifest_handler.go`
- Modify: `backend/go-core/internal/handler/wms_handler.go`
- Test: `backend/go-core/internal/handler/wms_manifest_handler_test.go`

**Interfaces:**
- Consumes: `domain.WMSOutboundRepository`, `domain.WMSRepository`
- Produces: HTTP endpoints di `/api/v1/wms/outbound/manifests/*` dan `/api/v1/wms/outbound/kpi`

- [ ] **Step 1: Tulis unit test usecase manifest**
Verifikasi bahwa pembuatan manifest memvalidasi DO status `PACKED`, loading scan menolak DO yang tidak ada di manifest, dan dispatch mewajibkan tanda tangan.

- [ ] **Step 2: Implementasikan `internal/usecase/wms/manifest_usecase.go`**
Logika bisnis:
- `CreateShippingManifest`: validasi warehouse write access, verifikasi seluruh DO milik tenant dan berstatus `PACKED`.
- `ScanDOLoading`: pencocokan nomor DO / resi AWB, update status koli.
- `DispatchShippingManifest`: validasi signature non-kosong, pemotongan stok keluar atomik.
- `GetOutboundKPIs`: menghitung 8 metrik SLA & kepatuhan WMS.

- [ ] **Step 3: Implementasikan REST handler di `internal/handler/wms_manifest_handler.go` & daftarkan rute di `wms_handler.go`**
Daftarkan rute:
- `GET /delivery-orders/manifests` & `POST /delivery-orders/manifests`
- `GET /delivery-orders/manifests/{id}`
- `POST /delivery-orders/manifests/{id}/loading-scan`
- `POST /delivery-orders/manifests/{id}/dispatch`
- `GET /kpi`

- [ ] **Step 4: Jalankan test handler & usecase**
Run: `go test -v ./internal/usecase/wms/... ./internal/handler/...`
Expected: PASS

- [ ] **Step 5: Commit backend usecase & handlers**
```bash
git add backend/go-core/internal/usecase/wms/manifest_usecase.go backend/go-core/internal/handler/wms_manifest_handler.go backend/go-core/internal/handler/wms_manifest_handler_test.go backend/go-core/internal/handler/wms_handler.go
git commit -m "feat(wms): implement manifest usecase and REST handlers with loading scan and KPI"
```

---

### Task 4: Frontend API Client & State Hooks

**Files:**
- Modify: `lib/api.ts`
- Create: `hooks/useWMSManifests.ts`
- Modify: `hooks/useWMSLedger.ts`

**Interfaces:**
- Consumes: Backend endpoints `/api/v1/wms/outbound/manifests`
- Produces: TypeScript types `ShippingManifest`, `ShippingManifestDetail`, TanStack query hooks `useShippingManifests`, `useCreateShippingManifest`, `useDispatchShippingManifest`, `useLoadingScanDO`, `useOutboundKPI`

- [ ] **Step 1: Tambahkan types dan client calls di `lib/api.ts`**
Tambahkan `ShippingManifest`, `CreateShippingManifestRequest`, `DispatchShippingManifestRequest`, dan `api.wms.manifests.*`.

- [ ] **Step 2: Buat hook `hooks/useWMSManifests.ts`**
TanStack Query hooks dengan invalidasi query `["shipping-manifests"]`, `["delivery-orders"]`, `["wms-stock"]`.

- [ ] **Step 3: Jalankan typecheck frontend**
Run: `npx tsc --noEmit`
Expected: 0 error

- [ ] **Step 4: Commit frontend API client & hooks**
```bash
git add lib/api.ts hooks/useWMSManifests.ts
git commit -m "feat(wms): add frontend API clients and TanStack Query hooks for manifests"
```

---

### Task 5: Frontend Manifest UI (`ShippingManifestsPanel.tsx` & Kanvas TTD)

**Files:**
- Create: `components/wms/ShippingManifestsPanel.tsx`
- Create: `components/wms/SignatureCanvas.tsx`
- Create: `components/wms/PrintShippingManifest.tsx`
- Modify: `components/wms/DeliveryOrdersPanel.tsx`

**Interfaces:**
- Consumes: `useShippingManifests`, `useCreateShippingManifest`, `useDispatchShippingManifest`
- Produces: Komponen Tab "Manifest Ekspedisi" terpasang di panel arus keluar

- [ ] **Step 1: Buat komponen `SignatureCanvas.tsx`**
Kanvas tanda tangan HTML5 berbasis mouse/touch dengan tombol Clear, Undo, dan ekspor data URL / SVG string.

- [ ] **Step 2: Buat komponen cetak A4 `PrintShippingManifest.tsx`**
Layout resmi lembar manifest serah terima ekspedisi: data ekspedisi, armada, tabel daftar Surat Jalan, total koli/berat, kotak tanda tangan ganda (Gudang & Kurir).

- [ ] **Step 3: Buat `ShippingManifestsPanel.tsx` & hubungkan ke `DeliveryOrdersPanel.tsx`**
- Tabel daftar manifest dengan filter ekspedisi dan status.
- Modal Buat Manifest dengan seleksi multi-DO yang sudah dikemas (`PACKED`).
- Drawer Detail Manifest: checklist koli ter-scan, progress pemuatan, tombol Buka Kanvas TTD untuk Dispatch.

- [ ] **Step 4: Tulis Jest test untuk SignatureCanvas & ShippingManifestsPanel**
Test render, interaksi tab, dan ekspor signature.

- [ ] **Step 5: Jalankan pengujian Jest & typecheck**
Run: `npx tsc --noEmit` && `npm test -- __tests__/wms-outbound-components.test.tsx`
Expected: PASS

- [ ] **Step 6: Commit komponen UI manifest**
```bash
git add components/wms/ShippingManifestsPanel.tsx components/wms/SignatureCanvas.tsx components/wms/PrintShippingManifest.tsx components/wms/DeliveryOrdersPanel.tsx
git commit -m "feat(wms): add shipping manifests management panel, signature canvas, and print view"
```

---

### Task 6: Frontend Scanner Mode Loading Truk (`app/(app)/wms/scanner/page.tsx`)

**Files:**
- Modify: `app/(app)/wms/scanner/page.tsx`
- Test: `__tests__/scanner-loading.test.tsx`

**Interfaces:**
- Consumes: `useShippingManifests`, `useLoadingScanDO`
- Produces: Tab `LOADING_TRUCK` di halaman barcode scanner WMS

- [ ] **Step 1: Tambahkan mode `LOADING_TRUCK` di scanner**
- Dropdown pemilih manifest aktif (status `STAGED` / `LOADED`).
- Pemindaian barcode kamera & PDA hardware event:
  - Validasi scan via API `loading-scan`.
  - Notifikasi suara sukses / gagal misload.
  - Tampilan counter jumlah koli yang sudah dimuat ke bak truk.

- [ ] **Step 2: Jalankan test & typecheck**
Run: `npx tsc --noEmit` && `npm test`
Expected: PASS

- [ ] **Step 3: Commit scanner mode loading**
```bash
git add app/(app)/wms/scanner/page.tsx
git commit -m "feat(wms): add truck loading scan mode to warehouse barcode scanner"
```

---

### Task 7: Dashboard 8 SOP KPIs (`app/(app)/dashboard/page.tsx`)

**Files:**
- Modify: `app/(app)/dashboard/page.tsx`
- Modify: `backend/go-core/internal/handler/dashboard_handler.go`
- Modify: `backend/go-core/internal/usecase/dashboard/dashboard_usecase.go`

**Interfaces:**
- Consumes: Timestamp log audit penerimaan, putaway, picking, packing, dispatch
- Produces: Kartu metrik 8 KPI SOP WMS

- [ ] **Step 1: Hitung 8 KPI di backend dashboard usecase**
Metrik: Dock-to-Stock, Receiving Accuracy, PO Compliance, Backlog Inbound; Order-to-Dispatch, Picking Accuracy, On-Time Shipment, Backlog Outbound.

- [ ] **Step 2: Tampilkan widget KPI di UI Dashboard**
Card ringkas yang estetik dan informatif tanpa mengorbankan performa load.

- [ ] **Step 3: Jalankan verifikasi backend & frontend**
Run: `go test ./internal/usecase/dashboard/...` && `npm test`
Expected: PASS

- [ ] **Step 4: Commit KPI dashboard**
```bash
git add app/(app)/dashboard/page.tsx backend/go-core/internal/handler/dashboard_handler.go backend/go-core/internal/usecase/dashboard/dashboard_usecase.go
git commit -m "feat(dashboard): integrate 8 enterprise WMS SOP KPIs into dashboard overview"
```

---

### Task 8: Verifikasi Komprehensif, Audit AI Debt & Deploy Produksi Zeabur

**Files:**
- All touched files in Sprint 4

- [ ] **Step 1: Jalankan seluruh verifikasi otomatis lokal**
Run:
- `go vet ./...` && `go build ./...`
- `go test ./internal/...`
- `npx tsc --noEmit`
- `npm test`

- [ ] **Step 2: Jalankan audit `ai-debt-detector`**
Pemeriksaan menyeluruh terhadap penanganan failure mode, rollback error, edge cases null pointer, dan dependensi.

- [ ] **Step 3: Perbarui living checklist di `docs/specs/2026-10-07-enterprise-wms-inbound-outbound-master-prd-plan.md`**
Tandai seluruh item Sprint 4 menjadi `[x]` beserta nomor commit.

- [ ] **Step 4: Push ke `main` & Pantau CI GitHub Actions**
Run: `git push origin main`
Verifikasi job `Go Tests & Vet`, `Frontend Typecheck & Tests`, dan `Deploy to Zeabur` sukses (status `success`).

- [ ] **Step 5: Verifikasi Live Produksi Zeabur**
Cek `https://tayooli-backend.zeabur.app/health` dan `https://tayooli.my.id`.
Report bukti status real-time.
