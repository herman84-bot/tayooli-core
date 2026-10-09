# Task 2 Report: Backend Domain & Repository untuk Shipping Manifests

## Status
DONE

## Commit
`13e4f6851111ca17404285d29caeb07a622d8aaf`
`feat(wms): implement domain models and postgres repository for shipping manifests`

## Changed
- `backend/go-core/internal/domain/wms_manifest.go`:
  - Domain status enum `ShippingManifestStatus` (`STAGED`, `LOADED`, `DISPATCHED`, `CANCELLED`).
  - Added constant `DeliveryOrderStatusStaged` for delivery order staging transition.
  - Domain model `ShippingManifest` with warehouse, courier, driver info, package aggregates, SVG signature, actor timestamps.
  - Domain line item `ShippingManifestItem` with DO number, customer, city, weight, scan flags and timestamps.
  - Detail struct `ShippingManifestDetail` and request DTOs (`CreateShippingManifestRequest`, `LoadingScanRequest`, `DispatchShippingManifestRequest`).
  - Outbound operational metric summary `WMSOutboundKPISummary` covering 8 core SOP KPIs.
  - Domain sentinel errors: `ErrManifestNotFound`, `ErrInvalidManifestStatus`, `ErrManifestSignatureRequired`, `ErrDOMisload`, `ErrManifestEmpty`.
  - Interface `WMSManifestRepository` specifying 6 operations.
- `backend/go-core/internal/domain/wms_manifest_test.go`:
  - Unit tests validating `CreateShippingManifestRequest` and `DispatchShippingManifestRequest` invariants.
- `backend/go-core/internal/infra/postgres/wms_manifest_repo.go`:
  - Compile-time interface assertion: `var _ domain.WMSManifestRepository = (*WMSRepo)(nil)`.
  - `CreateShippingManifest`: validates warehouse and DO status (`PACKED` / `STAGED`), atomic `MAN-YYYYMMDD-XXXX` sequence generation, sets `manifest_id` and status `STAGED` on DOs, inserts manifest, logs audit trail.
  - `GetShippingManifestByID`: selects manifest header with warehouse and actor names, and attaches linked delivery order items.
  - `ListShippingManifests`: dynamic multi-filter query (tenant, warehouse, status, expedition) with RLS isolation.
  - `ScanDOLoading`: matches barcode against DO number or tracking number, updates loading timestamp/actor, promotes manifest to `LOADED` when all DOs scanned.
  - `DispatchShippingManifest`: row-lock `FOR UPDATE`, signature validation, double-entry stock deduction to `@CUSTOMER` location for all DO line items with batch preservation, updates DOs to `SHIPPED` and manifest to `DISPATCHED`, atomic single transaction commit with audit trails.
  - `GetWMSOutboundKPIs`: calculates 8 metrics (`DockToStockAvgMinutes`, `ReceivingAccuracyPct`, `POCompliancePct`, `BacklogInboundCount`, `OrderToDispatchAvgHours`, `PickingAccuracyPct`, `OnTimeShipmentPct`, `BacklogOutboundCount`) with safe division and rounding.
- `backend/go-core/internal/infra/postgres/wms_manifest_repo_test.go`:
  - Unit tests verifying interface adherence and fast-path parameter validation.

## Verified
- `go vet ./...` in `backend/go-core`: clean (exit code 0).
- `go build ./...` in `backend/go-core`: clean build (exit code 0).
- `go test -c -o test_domain.exe ./internal/domain/...`: compiled cleanly with 0 errors.
- `go test -c -o test_postgres.exe ./internal/infra/postgres/...`: compiled cleanly with 0 errors.

## Issues
None.

## Next
Task 3: Backend Usecase & REST Handler untuk Shipping Manifests (`manifest_usecase.go`, `wms_manifest_handler.go`, unit & handler tests).
