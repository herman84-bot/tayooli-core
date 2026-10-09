# Task 4 Report: Frontend API Client & State Hooks (`lib/api.ts`, `hooks/useWMSDocksAndLPNs.ts`)

**Sprint:** 5 - Inbound Dock Scheduling & Pallet LPN Containerization  
**Task:** 4 - Frontend API Client & State Hooks  
**Status:** COMPLETED  
**Commit:** `c52ffcb4462345076f72eba0045d5325a1d171dc`  
`feat(wms): add API client methods and TanStack Query hooks for docks and LPNs`  

---

## 1. Summary of Changes

1. **API Client Types & Methods (`lib/api.ts`)**:
   - Added TypeScript domain types matching Go backend models:
     - `DockStatus` (`"AVAILABLE" | "OCCUPIED" | "MAINTENANCE"`)
     - `DockType` (`"INBOUND" | "OUTBOUND" | "CROSS_DOCK"`)
     - `AppointmentStatus` (`"SCHEDULED" | "ARRIVED" | "UNLOADING" | "COMPLETED" | "CANCELLED"`)
     - `LPNStatus` (`"STAGED" | "STORED" | "PICKED" | "SHIPPED" | "DECOMMISSIONED"`)
     - `PalletType` (`"WOODEN" | "PLASTIC" | "METAL" | "CAGE"`)
     - Domain interfaces: `InboundDock`, `CreateDockInput`, `UpdateDockStatusInput`, `DockAppointment`, `CreateAppointmentInput`, `UpdateAppointmentStatusInput`, `AssignDockInput`, `StockLPN`, `StockLPNItem`, `StockLPNDetail`, `CreateLPNInput`, `AddLPNItemInput`, `MoveLPNInput`.
   - Extended `api.wms` object:
     - `docks.list(warehouseId, status)` -> `GET /wms/docks?warehouse_id=...`
     - `docks.create(data)` -> `POST /wms/docks`
     - `docks.updateStatus(id, data)` -> `PATCH /wms/docks/${id}/status`
     - `dockAppointments.list(warehouseId, status)` -> `GET /wms/dock-appointments?warehouse_id=...`
     - `dockAppointments.create(data)` -> `POST /wms/dock-appointments`
     - `dockAppointments.assign(id, dockId)` -> `POST /wms/dock-appointments/${id}/assign`
     - `dockAppointments.updateStatus(id, data)` -> `PATCH /wms/dock-appointments/${id}/status`
     - `lpns.list(warehouseId, status)` -> `GET /wms/lpns?warehouse_id=...`
     - `lpns.create(data)` -> `POST /wms/lpns`
     - `lpns.get(id)` -> `GET /wms/lpns/${id}`
     - `lpns.addItem(id, data)` -> `POST /wms/lpns/${id}/items`
     - `lpns.move(id, data)` -> `POST /wms/lpns/${id}/move`

2. **TanStack Query Hooks (`hooks/useWMSDocksAndLPNs.ts`)**:
   - Created client hook suite with complete query cache invalidation:
     - `useInboundDocks(warehouseId?, status?)`
     - `useCreateDock()` (invalidates `["inbound-docks"]`)
     - `useUpdateDockStatus()` (invalidates `["inbound-docks"]`, `["dock-appointments"]`)
     - `useDockAppointments(warehouseId?, status?)`
     - `useCreateAppointment()` (invalidates `["dock-appointments"]`)
     - `useAssignDock()` (invalidates `["dock-appointments"]`, `["inbound-docks"]`)
     - `useUpdateAppointmentStatus()` (invalidates `["dock-appointments"]`, `["inbound-docks"]`)
     - `useStockLPNs(warehouseId?, status?)`
     - `useStockLPNDetail(id?)` (enabled when `!!id`)
     - `useCreateLPN()` (invalidates `["stock-lpns"]`)
     - `useAddLPNItem()` (invalidates `["stock-lpn", id]`, `["stock-lpns"]`)
     - `useMoveLPN()` (invalidates `["stock-lpn", id]`, `["stock-lpns"]`, `["wms-stock"]`, `["wms-locations"]`, `["wms", "stock"]`, `["wms", "locations"]`)

---

## 2. Verification Evidence

- `npx tsc --noEmit` -> PASS (0 type errors).
- `npm test` -> PASS (35 suites passed, 343 tests passed, 0 failures).

---

## 3. Issues
None.

---

## 4. Next
Task 5: Frontend Dock Board & Antrean Armada (`DockBoardSubView.tsx`).
