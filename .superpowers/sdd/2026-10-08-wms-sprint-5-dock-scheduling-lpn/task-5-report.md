# Task 5 Report: Frontend Dock Board & Antrean Armada (`DockBoardSubView.tsx`)

**Status:** DONE  
**Commit:** `ed219b1`  
**Date:** 2026-10-08  

## Summary of Changes

1. **Created `components/wms/DockBoardSubView.tsx`**:
   - Integrated with TanStack Query hooks from `@/hooks/useWMSDocksAndLPNs` (`useInboundDocks`, `useCreateDock`, `useUpdateDockStatus`, `useDockAppointments`, `useCreateAppointment`, `useAssignDock`, `useUpdateAppointmentStatus`).
   - Four metrics cards:
     - `Total Dermaga`
     - `Dermaga Tersedia` (AVAILABLE)
     - `Sedang Bongkar` (OCCUPIED / Unloading)
     - `Antrean Menunggu` (SCHEDULED / ARRIVED)
   - Visual dock bay grid:
     - Status badges: AVAILABLE (emerald), OCCUPIED (amber), MAINTENANCE (rose).
     - Active docked appointment card summary (driver, plate, vendor, status).
     - Card actions: Toggle maintenance mode, Release dock when unloading completes.
     - "Tambah Dermaga" modal dialog to add dock bay.
   - Inbound queue & appointment management:
     - Filter tabs: `Semua`, `Terjadwal (SCHEDULED)`, `Tiba (ARRIVED)`, `Bongkar (UNLOADING)`, `Selesai (COMPLETED)`.
     - Appointment actions: `Tiba`, `Alokasikan Dock`, `Mulai Bongkar`, `Selesai`.
     - "Jadwalkan Armada Baru" modal dialog for appointment scheduling.
     - Friendly alert handling for `ErrDockOccupied` (409):
       `"Dermaga ini sedang digunakan oleh armada lain! Silakan pilih dermaga lain."`

2. **Updated `app/(app)/wms/arus-barang/page.tsx`**:
   - Extended `MasukTab` type to `"penerimaan" | "putaway" | "trace" | "qc" | "dock"`.
   - Added sub-tab button `Antrean Dermaga (Dock)` in Inbound flow.
   - Rendered `<DockBoardSubView warehouseId={selectedWarehouseId} />` when `masukTab === "dock"`.

3. **Created `__tests__/dock-board.test.tsx`**:
   - 11 unit tests covering:
     - Metrics calculation & rendering.
     - Dock cards rendering with active appointment display.
     - Maintenance mode toggling.
     - Dock release action.
     - Appointments table rendering & filtering.
     - Appointment lifecycle status updates (Tiba, Mulai Bongkar, Selesai).
     - Dock creation modal submission.
     - Appointment creation modal submission.
     - Dock allocation modal submission.
     - Friendly error handling for dock collision (409 / occupied).

## Verification Evidence

- `npx tsc --noEmit`: 0 errors (Pass)
- `npm test -- __tests__/dock-board.test.tsx`: 11 passed, 0 failed (Pass)
- `npm test -- __tests__/arus-barang.test.tsx`: 7 passed, 0 failed (Pass)
