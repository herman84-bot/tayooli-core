# Task 3 Report: Backend Usecase & REST Handler untuk Shipping Manifests

## Status
DONE

## Commit
`fffd84e2adac20261393f8be9523072dabc2b2d9`
`feat(wms): implement manifest usecase and REST handlers with loading scan and KPI`

## Changed
- `backend/go-core/internal/domain/wms.go`:
  - Embedded `WMSManifestRepository` into composite interface `WMSRepository`.
- `backend/go-core/internal/usecase/wms/manifest_usecase.go`:
  - `CreateShippingManifest`: validates request via `req.Validate()`, validates warehouse write access with `ValidateWarehouseWriteAccess`, calls repository.
  - `GetShippingManifest`: delegates to `u.repo.GetShippingManifestByID`.
  - `ListShippingManifests`: delegates to `u.repo.ListShippingManifests` with tenant, warehouse, status, expedition filters.
  - `ScanDOLoading`: validates non-empty barcode, delegates to `u.repo.ScanDOLoading`.
  - `DispatchShippingManifest`: validates driver signature via `req.Validate()`, calls atomic `u.repo.DispatchShippingManifest`.
  - `GetWMSOutboundKPIs`: delegates to `u.repo.GetWMSOutboundKPIs`.
- `backend/go-core/internal/usecase/wms/manifest_usecase_test.go`:
  - Unit tests verifying validation invariants (empty DOs, auditor write rejection, invalid loading barcode, missing/short SVG signature) and repository delegation.
- `backend/go-core/internal/handler/wms_handler.go`:
  - Added 6 shipping manifest methods to `WMSUsecase` interface.
  - Registered manifest REST endpoints and aliases under `/wms/manifests`, `/wms/kpi`, `/wms/delivery-orders/manifests`, and `/wms/outbound/manifests`.
  - Updated error mapper `handleWMSError` for:
    - `domain.ErrManifestNotFound` -> StatusNotFound (404)
    - `domain.ErrInvalidManifestStatus` -> StatusConflict (409)
    - `domain.ErrManifestSignatureRequired` -> StatusBadRequest (400)
    - `domain.ErrDOMisload` -> StatusUnprocessableEntity (422)
    - `domain.ErrManifestEmpty` -> StatusBadRequest (400)
- `backend/go-core/internal/handler/wms_manifest_handler.go`:
  - `ListShippingManifests`: extracts auth, parses query params (`warehouse_id`, `status`, `expedition_name`), returns JSON data list.
  - `CreateShippingManifest`: parses and limits JSON payload, calls usecase, returns 201 Created.
  - `GetShippingManifest`: extracts UUID param `id`, returns manifest detail.
  - `ScanDOLoading`: extracts manifest ID and barcode, returns updated manifest detail.
  - `DispatchShippingManifest`: extracts manifest ID and SVG signature, returns dispatched manifest.
  - `GetWMSOutboundKPIs`: extracts query params, returns 8 SOP outbound KPIs.
- `backend/go-core/internal/handler/wms_handler_test.go`:
  - Added shipping manifest mock hooks and receivers to `mockWMSUsecase`.
- `backend/go-core/internal/handler/wms_manifest_handler_test.go`:
  - Endpoint HTTP tests with httptest covering all positive and negative failure cases:
    - `GET /api/v1/wms/manifests`
    - `POST /api/v1/wms/manifests` (valid, empty DOs, malformed payload)
    - `GET /api/v1/wms/manifests/{id}` (valid, invalid UUID, not found)
    - `POST /api/v1/wms/manifests/{id}/loading-scan` (success, misload 422)
    - `POST /api/v1/wms/manifests/{id}/dispatch` (success, signature required 400)
    - `GET /api/v1/wms/kpi`
    - `GET /api/v1/wms/delivery-orders/manifests` (alias)

## Verified
- `go vet ./...` in `backend/go-core`: clean (exit code 0).
- `go build ./...` in `backend/go-core`: clean build (exit code 0).
- `go test -count=1 -v ./internal/usecase/wms -run "TestManifest"`: 5/5 subtests PASS.
- `go test -count=1 -v ./internal/handler -run "TestWMSManifest"`: 7/7 subtests PASS.
- `go test -count=1 ./internal/handler/...`: PASS (exit code 0).
- `go test -count=1 ./internal/usecase/wms/...`: PASS (exit code 0).

## Issues
None.

## Next
Task 4: Frontend API Client & State Hooks (`lib/api.ts`, `hooks/useWMSManifests.ts`, `hooks/useWMSLedger.ts`).
