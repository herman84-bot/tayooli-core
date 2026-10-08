# Task 3 Report: Backend Usecase & REST Handler untuk Docks & LPNs

**Sprint:** 5 - Inbound Dock Scheduling & Pallet LPN Containerization  
**Task:** 3 - Backend Usecase & REST Handler untuk Docks & LPNs  
**Status:** COMPLETED  
**Standard/Pola Acuan:** OCA/wms dock scheduling, Sentry WMS containerization, ADR-014 Invariant 1 (Double-Entry Ledger per Batch).

---

## 1. Summary of Changes

1. **Usecase Layer (`backend/go-core/internal/usecase/wms/dock_lpn_usecase.go`)**:
   - Implemented 14 methods on `*Usecase`:
     - Docks: `CreateDock`, `GetDock`, `ListDocks`, `UpdateDockStatus`
     - Appointments: `CreateAppointment`, `GetAppointment`, `ListAppointments`, `AssignDockToAppointment`, `UpdateAppointmentStatus`
     - Stock LPNs: `CreateLPN`, `GetLPN`, `ListLPNs`, `AddLPNItem`, `MoveLPN`
   - Strict access control enforcement:
     - Read operations enforce `u.ValidateWarehouseAccess`.
     - Mutation operations enforce `u.ValidateWarehouseWriteAccess` (rejecting `auditor` with `domain.ErrForbidden` and unassigned staff with `domain.ErrUnauthorizedWarehouse`).
   - Every input DTO validated via `req.Validate()`.
   - `MoveLPN` fetches LPN first to determine its `WarehouseID`, performs write access verification, then delegates atomic movement to the repository.

2. **REST Handler & Route Registration (`backend/go-core/internal/handler/`)**:
   - Updated `WMSUsecase` interface with all 14 methods in `wms_handler.go`.
   - Updated `handleWMSError` mapping:
     - `domain.ErrDockNotFound` -> `404 Not Found`
     - `domain.ErrDockOccupied` -> `409 Conflict`
     - `domain.ErrAppointmentNotFound` -> `404 Not Found`
     - `domain.ErrLPNNotFound` -> `404 Not Found`
     - `domain.ErrLPNEmpty` -> `422 Unprocessable Entity`
     - `domain.ErrInvalidLocationType` -> `422 Unprocessable Entity`
   - Registered endpoints under `/wms`:
     - `/docks`, `/docks/{id}`, `/docks/{id}/status`
     - `/dock-appointments`, `/dock-appointments/{id}`, `/dock-appointments/{id}/assign`, `/dock-appointments/{id}/status`
     - `/lpns`, `/lpns/{id}`, `/lpns/{id}/items`, `/lpns/{id}/move`
   - Implemented `backend/go-core/internal/handler/wms_dock_lpn_handler.go` with `http.MaxBytesReader(w, r.Body, 1<<20)`, tenant/user auth guards, and safe UUID parsing.

3. **Unit & HTTP Test Coverage**:
   - `backend/go-core/internal/usecase/wms/dock_lpn_usecase_test.go`:
     - Access control (admin vs warehouse vs auditor).
     - Input validation errors (`domain.ErrInvalidInput`, `domain.ErrInvalidStatus`).
     - Delegation to repo and atomic putaway.
   - `backend/go-core/internal/handler/wms_dock_lpn_handler_test.go`:
     - Route matching, 401 unauthenticated guard, 413 payload limit (> 1MB), 400 bad JSON/UUID, 404, 409, 422, 200/201 success responses.

---

## 2. Verification Evidence

- `go vet ./...` -> Clean, 0 errors.
- `go build ./...` -> Clean, exit 0.
- `go test -v -count=1 ./internal/usecase/wms -run "TestDockLPNUsecase"` -> **PASS** (11/11 subtests pass).
- `go test -v -count=1 ./internal/handler -run "TestDockLPNHandlers"` -> **PASS** (4/4 test groups pass).
- Full regression run: `go test ./internal/usecase/wms/... ./internal/handler/...` -> **ALL PASS**.
