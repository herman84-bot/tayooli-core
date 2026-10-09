# Task 1 Report: Database Migration `037_wms_docks_and_lpns.sql`

## Status
DONE

## Commit
`17b7befa8b712fd2237b5e9050bd1a148c96eb7f`
`feat(wms): add migration 037 for inbound docks, appointments, and pallet LPNs`

## Changed
- `backend/go-core/migrations/037_wms_docks_and_lpns.sql`:
  - Created table `inbound_docks` (`id`, `tenant_id`, `warehouse_id`, `dock_code`, `dock_name`, `dock_type`, `max_tonnage`, `status`, `notes`, timestamps, `UNIQUE(tenant_id, warehouse_id, dock_code)`).
  - Created table `dock_appointments` (`id`, `tenant_id`, `warehouse_id`, `dock_id`, `appointment_number`, `vendor_name`, `vehicle_plate`, `driver_name`, `driver_phone`, `po_reference`, `estimated_arrival`, `actual_arrival`, `start_unloading_at`, `completed_at`, `status`, `notes`, `created_by`, timestamps, `UNIQUE(tenant_id, appointment_number)`).
  - Created table `stock_lpns` (`id`, `tenant_id`, `warehouse_id`, `lpn_code`, `location_id`, `pallet_type`, `status`, `max_weight_kg`, `total_weight_kg`, `notes`, `created_by`, timestamps, `UNIQUE(tenant_id, warehouse_id, lpn_code)`).
  - Created table `stock_lpn_items` (`id`, `tenant_id`, `lpn_id`, `product_id`, `batch_id`, `quantity`, timestamps, `UNIQUE(tenant_id, lpn_id, batch_id)`).
  - Enabled and forced RLS tenant isolation policy on all 4 tables with `tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid`.
  - Added indexes: `idx_inbound_docks_wh`, `idx_inbound_docks_status`, `idx_dock_app_wh_status`, `idx_dock_app_eta`, `idx_stock_lpns_wh_loc`, `idx_stock_lpns_status`, `idx_stock_lpn_items_lpn`.
  - Added DOWN migration block.

## Verified
- `backend/go-core/migrations/migrations.go`: embeds `*.sql` cleanly and automatically discovers `037_wms_docks_and_lpns.sql`.
- `go vet ./...` in `backend/go-core`: clean (exit code 0).
- `go build ./...` in `backend/go-core`: compiled cleanly (exit code 0).

## Issues
None.

## Next
Task 2: Backend Domain Models & PostgreSQL Repository for Docks & LPNs (`wms_dock_lpn.go`, `wms_dock_lpn_repo.go`).
