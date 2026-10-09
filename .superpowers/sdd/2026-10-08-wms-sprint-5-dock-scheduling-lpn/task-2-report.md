# Task 2 Report: Backend Domain & Repository untuk Docks & LPNs

## Status
DONE

## Commit
`4b82f7fa40003ed75f174d37957c81585d477bc5`
`feat(wms): implement domain models and postgres repository for docks and pallet LPNs`

## Changed
- `backend/go-core/internal/domain/wms_dock_lpn.go`:
  - Enums: `DockStatus` (`AVAILABLE`, `OCCUPIED`, `MAINTENANCE`), `DockType` (`INBOUND`, `OUTBOUND`, `CROSS_DOCK`), `AppointmentStatus` (`SCHEDULED`, `ARRIVED`, `UNLOADING`, `COMPLETED`, `CANCELLED`), `LPNStatus` (`STAGED`, `STORED`, `PICKED`, `SHIPPED`, `DECOMMISSIONED`), `PalletType` (`WOODEN`, `PLASTIC`, `METAL`, `CAGE`).
  - Constant: `StockRefLPNPutaway = "PUTAWAY-LPN"`.
  - Domain models: `InboundDock`, `DockAppointment`, `StockLPN`, `StockLPNItem`, `StockLPNDetail`.
  - DTOs with `.Validate()`: `CreateDockRequest`, `UpdateDockStatusRequest`, `CreateAppointmentRequest`, `AssignDockRequest`, `UpdateAppointmentStatusRequest`, `CreateLPNRequest`, `AddLPNItemRequest`, `MoveLPNRequest`.
  - Sentinel errors: `ErrDockNotFound`, `ErrDockOccupied`, `ErrAppointmentNotFound`, `ErrLPNNotFound`, `ErrLPNEmpty`, `ErrInvalidLocationType`.
  - Interface: `WMSDockLPNRepository` with 14 persistence operations.
- `backend/go-core/internal/domain/wms.go`:
  - Embedded `WMSDockLPNRepository` into `WMSRepository`.
- `backend/go-core/internal/infra/postgres/wms_dock_lpn_repo.go`:
  - Compile-time check: `var _ domain.WMSDockLPNRepository = (*WMSRepo)(nil)`.
  - Implemented all 14 repository methods:
    1. `CreateDock`: validates request, checks warehouse, auto-generates dock code if blank (`DOCK-XX`), inserts row.
    2. `GetDockByID`: selects dock with joined warehouse name.
    3. `ListDocks`: selects docks filtered by warehouse and optional status.
    4. `UpdateDockStatus`: updates dock operational status and notes.
    5. `CreateAppointment`: validates ETA, checks warehouse and dock, generates `APP-YYYYMMDD-XXXX` sequence, inserts appointment.
    6. `GetAppointmentByID`: selects appointment with joined dock, warehouse, and creator names.
    7. `ListAppointments`: selects appointments for warehouse filtered by optional status.
    8. `AssignDockToAppointment`: checks dock exists, enforces anti-collision guard (no overlapping `ARRIVED`/`UNLOADING` appointment on same dock), assigns dock and updates dock status to `OCCUPIED`.
    9. `UpdateAppointmentStatus`: tracks timestamps (`actual_arrival`, `start_unloading_at`, `completed_at`), guards against overlapping unloading, frees up dock to `AVAILABLE` on completion or cancellation.
    10. `CreateLPN`: validates warehouse and location, generates `LPN-YYYYMMDD-XXXX` sequence, inserts pallet container.
    11. `GetLPNByID`: loads pallet container header and all `stock_lpn_items` with product/batch details.
    12. `ListLPNs`: selects LPNs for warehouse with optional location and status filters.
    13. `AddLPNItem`: validates positive quantity, validates batch belongs to product, upserts into `stock_lpn_items`, recalculates `total_weight_kg`.
    14. `MoveLPN`: atomic single-transaction forklift putaway (`FOR UPDATE` lock, validates target location is `INTERNAL`/`RACK`, checks non-empty items, creates `StockMovement` per item with `batch_id` preserved (ADR-014 Invariant 1) and movement type `PUTAWAY-LPN`, checks balance, updates LPN `location_id` and status to `STORED`, logs audit trail, commits atomically).
- `backend/go-core/internal/domain/wms_dock_lpn_test.go`:
  - Table-driven unit tests for all 8 DTO `.Validate()` methods covering 33 test cases.
- `backend/go-core/internal/infra/postgres/wms_dock_lpn_repo_test.go`:
  - Interface implementation check and fast-path parameter validation tests for `WMSRepo` methods.
- `backend/go-core/internal/usecase/wms/wms_mock_dock_lpn_test.go`:
  - Mock stub methods for `WMSDockLPNRepository` on `mockWMSRepo`.

## Verified
- `backend/go-core`: `go vet ./...` completed clean (exit code 0).
- `backend/go-core`: `go build ./...` built clean (exit code 0).
- `backend/go-core/internal/domain`: `go test -v ./internal/domain -run "Test.*Dock|Test.*Appointment|Test.*LPN"` passed all 33 tests (PASS).
- `backend/go-core/internal/infra/postgres`: test compilation via `go test -c -o test_postgres.exe ./internal/infra/postgres` succeeded with 0 errors.
- `backend/go-core/internal/usecase/wms`: test compilation via `go test -c -o test_usecase.exe ./internal/usecase/wms` succeeded with 0 errors.

## Issues
None.

## Next
Task 3: Backend Usecase & REST Handlers untuk Docks & LPNs (`dock_lpn_usecase.go`, `wms_dock_lpn_handler.go`, rute di `wms_handler.go`).
