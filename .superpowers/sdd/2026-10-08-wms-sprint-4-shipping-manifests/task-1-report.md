# Task 1 Report: Database Migration `036_wms_shipping_manifests.sql`

## Status
DONE

## Commit
`a97ad50c94e50c3b728162c307d95a9ae1e7337b`
`feat(wms): add migration 036 for shipping manifests and loading scan tracking`

## Changed
- `backend/go-core/migrations/036_wms_shipping_manifests.sql`:
  - Created table `shipping_manifests` with RLS tenant isolation, statuses (`STAGED`, `LOADED`, `DISPATCHED`, `CANCELLED`), SVG signature, driver info, package & weight aggregates.
  - Added indexes: `idx_shipping_manifests_tenant_wh`, `idx_shipping_manifests_status`, `idx_shipping_manifests_expedition`.
  - Added columns to `delivery_orders`: `manifest_id` (FK to `shipping_manifests`), `loading_scanned_at`, `loading_scanned_by` (FK to `users`).
  - Added index: `idx_delivery_orders_manifest`.
  - Embedded DOWN migration block.

## Verified
- `backend/go-core/migrations/migrations.go`: uses `//go:embed *.sql` and automatic directory traversal, no hardcoded file list needed.
- `go vet ./...` in `backend/go-core`: clean (exit code 0, no warnings).
- `go build ./...` in `backend/go-core`: compiled cleanly (exit code 0).

## Issues
None.

## Next
Task 2: Domain models, repository interfaces, and PostgreSQL implementation for shipping manifests and loading scans.
