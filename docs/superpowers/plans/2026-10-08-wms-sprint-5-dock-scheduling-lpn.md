# WMS Sprint 5: Dock Scheduling, Inbound Queue & Pallet LPN Containerization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Menuntaskan fitur Enterprise WMS tingkat lanjut (Sprint 5): manajemen dermaga bongkar muat (*Dock Scheduling & Inbound Queue*), kontainerisasi palet barang (*License Plate Number - LPN*), pemindahan rak massal berbasis palet (*Forklift LPN Putaway*), cetak label barcode palet LPN, dan integrasi mode scanner PDA.

**Architecture:** Menerapkan Clean Hexagonal Architecture di backend Go (database/sql + pgx) dengan transaksi atomik pemindahan stok rak per LPN (ADR-014 Invariant 1). Di frontend Next.js 15, modul diintegrasikan ke `/wms/arus-barang?mode=masuk` sebagai tab "Antrean Dermaga (Dock)" untuk mempertahankan batas 12 modul (KO-2).

**Tech Stack:** Go 1.24 (Chi router, pure SQL, PostgreSQL 15 RLS), Next.js 15 (App Router, Tailwind CSS, TanStack Query v5, Zustand, Lucide icons, HTML5 Barcode Canvas / SVG print).

**Spec:** `docs/superpowers/specs/2026-10-08-wms-sprint-5-dock-scheduling-lpn-design.md`

## Global Constraints
- Strict 12 Modul: Dilarang menambah item rute baru di sidebar menu utama; integrasikan ke `/wms/arus-barang?mode=masuk` dan `/wms/scanner`.
- ADR-014 Invariant 1: Setiap mutasi stok putaway LPN wajib membawa `batch_id` pada setiap item palet dan memindahkan saldo rak secara atomik.
- Anti-Collision Dock Guard: Satu dock aktif dilarang menerima dua antrean berstatus `UNLOADING` / `OCCUPIED` secara bersamaan.
- Atomic LPN Move: Pemindahan palet LPN (`MoveLPN`) dan seluruh mutasi stok item di dalamnya wajib berada dalam 1 transaksi SQL database (`tx.Commit`).

## Review Focus
1. Validasi tabrakan jadwal dock (*dock collision*) mengembalikan error HTTP 409 jika dock sedang digunakan armada lain.
2. Pemindahan LPN kosong (tanpa item) atau lokasi tujuan tidak valid (bukan internal rack) wajib ditolak.
3. Hak akses gudang (*warehouse permission*) divalidasi pada seluruh endpoint dock, appointment, dan LPN.
4. Payload size limit di semua handler mutasi LPN dan appointment status.
5. Pemindahan palet LPN me-rollback seluruh transaksi jika ada item yang gagal dipindahkan.

---

### Task 1: Database Migration `037_wms_docks_and_lpns.sql`

**Files:**
- Create: `backend/go-core/migrations/037_wms_docks_and_lpns.sql`
- Verify: `backend/go-core/migrations/migrations.go`

**Interfaces:**
- Consumes: `tenants`, `warehouses`, `warehouse_locations`, `products`, `stock_batches`, `users`
- Produces: Tables `inbound_docks`, `dock_appointments`, `stock_lpns`, `stock_lpn_items` with RLS policies and indexes.

- [x] **Step 1: Tulis berkas migrasi SQL `037_wms_docks_and_lpns.sql`**
Skema lengkap:
- `inbound_docks`: `id`, `tenant_id`, `warehouse_id`, `dock_code`, `dock_name`, `dock_type`, `max_tonnage`, `status`, `notes`, timestamps, UNIQUE(tenant_id, warehouse_id, dock_code).
- `dock_appointments`: `id`, `tenant_id`, `warehouse_id`, `dock_id`, `appointment_number`, `vendor_name`, `vehicle_plate`, `driver_name`, `driver_phone`, `po_reference`, `estimated_arrival`, `actual_arrival`, `start_unloading_at`, `completed_at`, `status`, `notes`, `created_by`, timestamps, UNIQUE(tenant_id, appointment_number).
- `stock_lpns`: `id`, `tenant_id`, `warehouse_id`, `lpn_code`, `location_id`, `pallet_type`, `status`, `max_weight_kg`, `total_weight_kg`, `notes`, `created_by`, timestamps, UNIQUE(tenant_id, warehouse_id, lpn_code).
- `stock_lpn_items`: `id`, `tenant_id`, `lpn_id`, `product_id`, `batch_id`, `quantity`, timestamps, UNIQUE(tenant_id, lpn_id, batch_id).
- RLS enabled & forced pada semua 4 tabel dengan policy isolasi tenant `tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid`.

- [x] **Step 2: Uji sintaks & kompilasi backend Go**
Run: `go build ./...` di `backend/go-core`
Expected: exit 0

- [x] **Step 3: Commit migrasi database**
```bash
git add backend/go-core/migrations/037_wms_docks_and_lpns.sql
git commit -m "feat(wms): add migration 037 for inbound docks, appointments, and pallet LPNs"
```

---

### Task 2: Backend Domain & Repository untuk Docks & LPNs

**Files:**
- Create: `backend/go-core/internal/domain/wms_dock_lpn.go`
- Create: `backend/go-core/internal/infra/postgres/wms_dock_lpn_repo.go`
- Modify: `backend/go-core/internal/domain/wms.go`
- Test: `backend/go-core/internal/domain/wms_dock_lpn_test.go`
- Test: `backend/go-core/internal/infra/postgres/wms_dock_lpn_repo_test.go`

**Interfaces:**
- Consumes: `domain.WarehouseLocation`, `domain.StockMovement`
- Produces: `domain.InboundDock`, `domain.DockAppointment`, `domain.StockLPN`, `domain.StockLPNItem`, `domain.StockLPNDetail`, `domain.WMSDockLPNRepository` methods

- [x] **Step 1: Definisikan domain models di `internal/domain/wms_dock_lpn.go`**
- Enums: `DockStatus`, `AppointmentStatus`, `LPNStatus`, `PalletType`.
- Structs: `InboundDock`, `DockAppointment`, `StockLPN`, `StockLPNItem`, `StockLPNDetail`.
- Request DTOs: `CreateDockRequest`, `UpdateDockStatusRequest`, `CreateAppointmentRequest`, `AssignDockRequest`, `UpdateAppointmentStatusRequest`, `CreateLPNRequest`, `AddLPNItemRequest`, `MoveLPNRequest`.
- Sentinel errors: `ErrDockNotFound`, `ErrDockOccupied`, `ErrAppointmentNotFound`, `ErrLPNNotFound`, `ErrLPNEmpty`, `ErrInvalidLocationType`.
- Interface `WMSDockLPNRepository`.

- [x] **Step 2: Implementasikan repository PostgreSQL di `internal/infra/postgres/wms_dock_lpn_repo.go`**
Metode pada `*WMSRepo`:
- `CreateDock`, `ListDocks`, `UpdateDockStatus`
- `CreateAppointment`, `ListAppointments`, `AssignDockToAppointment` (dengan dock collision guard), `UpdateAppointmentStatus`
- `CreateLPN`, `GetLPNByID`, `ListLPNs`, `AddLPNItem`
- `MoveLPN`: Transaksi atomik mengunci LPN, memindahkan setiap item batch ke lokasi tujuan rak (`DeductLocationStock` / mutasi rak internal), memperbarui `stock_lpns.location_id` dan status `STORED`, serta mencatat audit trail.

- [x] **Step 3: Tulis unit test untuk domain & repository docks/lpns**
Run: `go test -v ./internal/domain -run "TestDockLPN"`
Expected: PASS

- [x] **Step 4: Commit domain & repository**
```bash
git add backend/go-core/internal/domain/wms_dock_lpn.go backend/go-core/internal/infra/postgres/wms_dock_lpn_repo.go backend/go-core/internal/domain/wms.go
git commit -m "feat(wms): implement domain models and postgres repository for docks and pallet LPNs"
```

---

### Task 3: Backend Usecase & REST Handler untuk Docks & LPNs

**Files:**
- Create: `backend/go-core/internal/usecase/wms/dock_lpn_usecase.go`
- Create: `backend/go-core/internal/handler/wms_dock_lpn_handler.go`
- Modify: `backend/go-core/internal/handler/wms_handler.go`
- Test: `backend/go-core/internal/usecase/wms/dock_lpn_usecase_test.go`
- Test: `backend/go-core/internal/handler/wms_dock_lpn_handler_test.go`

**Interfaces:**
- Consumes: `domain.WMSDockLPNRepository`, `domain.WMSRepository`
- Produces: REST Endpoints `/api/v1/wms/docks/*`, `/api/v1/wms/dock-appointments/*`, `/api/v1/wms/lpns/*`

- [x] **Step 1: Implementasikan usecase di `internal/usecase/wms/dock_lpn_usecase.go`**
Logika bisnis:
- Validasi warehouse access & write access untuk seluruh operasi dermaga dan LPN.
- Validasi anti-collision sebelum mengalokasikan dock ke appointment.
- Validasi lokasi rak tujuan pada pemindahan LPN (`MoveLPN`).

- [x] **Step 2: Implementasikan REST handler di `internal/handler/wms_dock_lpn_handler.go` & rute di `wms_handler.go`**
Handler lengkap dengan `http.MaxBytesReader`, ekstraksi tenant/user context, error mapper HTTP:
- `ErrDockOccupied` -> 409
- `ErrDockNotFound`, `ErrAppointmentNotFound`, `ErrLPNNotFound` -> 404
- `ErrLPNEmpty`, `ErrInvalidLocationType` -> 422 / 400

- [x] **Step 3: Tulis pengujian usecase & HTTP handler**
Run: `go test -v ./internal/usecase/wms -run "TestDockLPNUsecase"`
Run: `go test -v ./internal/handler -run "TestDockLPNHandlers"`
Expected: PASS

- [x] **Step 4: Commit usecase & handlers**
```bash
git add backend/go-core/internal/usecase/wms/dock_lpn_usecase.go backend/go-core/internal/handler/wms_dock_lpn_handler.go backend/go-core/internal/handler/wms_handler.go
git commit -m "feat(wms): implement usecases and REST handlers for dock scheduling and LPN management"
```

---

### Task 4: Frontend API Client & State Hooks

**Files:**
- Modify: `lib/api.ts`
- Create: `hooks/useWMSDocksAndLPNs.ts`

**Interfaces:**
- Consumes: Backend endpoints `/api/v1/wms/docks`, `/api/v1/wms/dock-appointments`, `/api/v1/wms/lpns`
- Produces: Types `InboundDock`, `DockAppointment`, `StockLPN`, `StockLPNDetail`, hooks `useInboundDocks`, `useDockAppointments`, `useStockLPNs`, mutations `useCreateDock`, `useAssignDock`, `useMoveLPN`, dll.

- [ ] **Step 1: Tambahkan types dan client methods di `lib/api.ts`**
Tambahkan types dock/appointment/LPN dan objek `api.wms.docks`, `api.wms.dockAppointments`, `api.wms.lpns`.

- [ ] **Step 2: Buat TanStack Query hooks di `hooks/useWMSDocksAndLPNs.ts`**
Hooks lengkap dengan query key invalidation untuk `["inbound-docks"]`, `["dock-appointments"]`, `["stock-lpns"]`, `["wms-stock"]`.

- [ ] **Step 3: Uji typecheck TypeScript**
Run: `npx tsc --noEmit`
Expected: 0 error

- [ ] **Step 4: Commit API client & hooks**
```bash
git add lib/api.ts hooks/useWMSDocksAndLPNs.ts
git commit -m "feat(wms): add API client methods and TanStack Query hooks for docks and LPNs"
```

---

### Task 5: Frontend Dock Board & Antrean Armada (`DockBoardSubView.tsx`)

**Files:**
- Create: `components/wms/DockBoardSubView.tsx`
- Modify: `app/(app)/wms/arus-barang/page.tsx`
- Test: `__tests__/dock-board.test.tsx`

**Interfaces:**
- Consumes: `useInboundDocks`, `useDockAppointments`, `useAssignDock`
- Produces: Komponen Papan Dermaga (Dock Board) terpasang di panel arus barang masuk

- [ ] **Step 1: Buat `components/wms/DockBoardSubView.tsx`**
- Matriks visual dermaga (Dock 1, Dock 2, dll) dengan badge status (AVAILABLE / UNLOADING / OCCUPIED).
- Kolom daftar antrean armada (*Inbound Queue*) dengan rincian PO/Surat Jalan, nomor kendaraan, sopir, ETA.
- Tombol aksi alokasi dermaga & update status (Tiba, Mulai Bongkar, Selesai).
- Modal "Jadwalkan Armada Baru".

- [ ] **Step 2: Integrasikan ke `app/(app)/wms/arus-barang/page.tsx`**
Tambahkan tab ke-3 di mode MASUK: `[Daftar Penerimaan (GRN) | Inspeksi QC & Karantina | Antrean Dermaga (Dock)]` tanpa menambah rute sidebar baru (KO-2).

- [ ] **Step 3: Jalankan Jest tests & typecheck**
Run: `npx tsc --noEmit` && `npm test -- __tests__/dock-board.test.tsx`
Expected: PASS

- [ ] **Step 4: Commit Dock Board**
```bash
git add components/wms/DockBoardSubView.tsx app/(app)/wms/arus-barang/page.tsx __tests__/dock-board.test.tsx
git commit -m "feat(wms): add dock board queue view and schedule management to inbound flow"
```

---

### Task 6: Frontend Pallet LPN & Cetak Stiker Barcode (`PrintLPNLabel.tsx` & LPN Modal)

**Files:**
- Create: `components/wms/PrintLPNLabel.tsx`
- Create: `components/wms/LPNManagementModal.tsx`
- Modify: `components/wms/ReceiptDetailDrawer.tsx` (atau panel penerimaan barang terkait)
- Test: `__tests__/lpn-print-label.test.tsx`

**Interfaces:**
- Consumes: `StockLPNDetail`, `useStockLPNs`
- Produces: Cetak label barcode palet ukuran A6 / 100x150 mm termal dan modal alokasi LPN saat barang masuk.

- [ ] **Step 1: Buat `PrintLPNLabel.tsx`**
Format stiker palet termal 100x150 mm: Barcode LPN besar, kode LPN, warehouse name, daftar batch item dalam palet, total berat kg, tanggal cetak.

- [ ] **Step 2: Buat `LPNManagementModal.tsx`**
Modal untuk membuat palet LPN baru dan menautkan item koli penerimaan barang masuk ke dalam LPN.

- [ ] **Step 3: Uji Jest & typecheck**
Run: `npx tsc --noEmit` && `npm test -- __tests__/lpn-print-label.test.tsx`
Expected: PASS

- [ ] **Step 4: Commit LPN & Print Label**
```bash
git add components/wms/PrintLPNLabel.tsx components/wms/LPNManagementModal.tsx __tests__/lpn-print-label.test.tsx
git commit -m "feat(wms): implement pallet LPN containerization modal and thermal label print"
```

---

### Task 7: Frontend PDA Scanner Mode Pallet Putaway (`/wms/scanner`)

**Files:**
- Modify: `app/(app)/wms/scanner/page.tsx`
- Test: `__tests__/scanner-lpn.test.tsx`

**Interfaces:**
- Consumes: `useStockLPNs`, `useMoveLPN`
- Produces: Mode scanner `PALLET_LPN` untuk forklift / operator MHE

- [ ] **Step 1: Tambahkan mode `PALLET_LPN` di scanner**
- Mode 2 langkah: Scan Barcode Palet LPN -> Tampilkan detail isi koli & rekomendasi rak -> Scan Barcode Rak Tujuan.
- Panggil mutation `moveLPN` untuk memindahkan seluruh isi palet secara atomik.
- Berikan audio feedback sukses / gagal misplacement.

- [ ] **Step 2: Uji Jest & typecheck**
Run: `npx tsc --noEmit` && `npm test -- __tests__/scanner-lpn.test.tsx`
Expected: PASS

- [ ] **Step 3: Commit Scanner LPN Putaway**
```bash
git add app/(app)/wms/scanner/page.tsx __tests__/scanner-lpn.test.tsx
git commit -m "feat(wms): add pallet LPN forklift putaway scan mode to barcode scanner"
```

---

### Task 8: Verifikasi Menyeluruh, Audit AI Debt & Deploy Produksi Zeabur

**Files:**
- All touched files in Sprint 5
- `docs/specs/2026-10-07-enterprise-wms-inbound-outbound-master-prd-plan.md`

- [ ] **Step 1: Jalankan verifikasi otomatis penuh lokal**
- `go vet ./...` && `go build ./...`
- `go test ./internal/...`
- `npx tsc --noEmit`
- `npm test`

- [ ] **Step 2: Jalankan audit `ai-debt-detector`**
Pemeriksaan menyeluruh terhadap penanganan kegagalan rollback, unclosed transactions, edge cases null pointers, dan error mapping.

- [ ] **Step 3: Perbarui living execution checklist di PRD**
Tandai seluruh item Sprint 5 `[x]` di `docs/specs/2026-10-07-enterprise-wms-inbound-outbound-master-prd-plan.md`.

- [ ] **Step 4: Push ke `main` & Pantau CI GitHub Actions**
`git push origin main`

- [ ] **Step 5: Verifikasi Live Produksi Zeabur**
Cek `https://tayooli-backend.zeabur.app/health` dan `https://tayooli.my.id`.
Report bukti status real-time.
