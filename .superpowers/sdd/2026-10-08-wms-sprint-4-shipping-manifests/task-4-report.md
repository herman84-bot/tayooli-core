# Task 4 Report: Frontend API Clients & TanStack Query Hooks untuk Shipping Manifests

## Status
DONE

## Commit
`abb6ce3212258fd9d7ed420d82791991fbea60f8`
`feat(wms): add frontend API clients and TanStack Query hooks for manifests`

## Changed
- `lib/api.ts`:
  - Added TypeScript domain interfaces & types:
    - `ShippingManifestStatus`: `"STAGED" | "LOADED" | "DISPATCHED" | "CANCELLED"`
    - `ShippingManifest`: ID, tenant, warehouse, manifest number, expedition, driver details, totals, status, signature, audit timestamps
    - `ShippingManifestItem`: DO ID, DO number, customer name, destination city, package weight/type, loading scan flag & audit
    - `ShippingManifestDetail`: manifest header and list of items
    - `CreateShippingManifestInput`: warehouse ID, expedition, driver info, array of DO IDs, notes
    - `DispatchShippingManifestInput`: driver signature SVG string, notes
    - `WMSOutboundKPISummary`: 8 warehouse KPI metrics (dock-to-stock, receiving accuracy, PO compliance, inbound backlog, order-to-dispatch, picking accuracy, on-time shipment, outbound backlog)
  - Added `manifests` namespace to `api.wms`:
    - `list`: `GET /wms/manifests` with optional `warehouse_id`, `status`, `expedition_name` query params
    - `create`: `POST /wms/manifests` with `CreateShippingManifestInput` payload
    - `get`: `GET /wms/manifests/:id` returning manifest detail
    - `scanLoading`: `POST /wms/manifests/:id/loading-scan` with `{ barcode }`
    - `dispatch`: `POST /wms/manifests/:id/dispatch` with `DispatchShippingManifestInput`
    - `kpis`: `GET /wms/kpi` with optional `warehouse_id` query param
- `hooks/useWMSManifests.ts`:
  - `useShippingManifests(params)`: query key `["shipping-manifests", params]`
  - `useShippingManifestDetail(id)`: query key `["shipping-manifest", id]`, enabled: `!!id`
  - `useCreateShippingManifest()`: mutation invalidating `["shipping-manifests"]` & `["delivery-orders"]`
  - `useScanLoadingDO()`: mutation invalidating `["shipping-manifest", id]` & `["shipping-manifests"]` (with alias `useLoadingScanDO`)
  - `useDispatchShippingManifest()`: mutation invalidating `["shipping-manifests"]`, `["delivery-orders"]`, `["wms-stock"]`, `["wms-outbound-kpi"]`
  - `useWMSOutboundKPI(warehouseId)`: query key `["wms-outbound-kpi", warehouseId]` (with alias `useOutboundKPI`)

## Verified
- `npx tsc --noEmit`: 0 errors (exit code 0).
- `npm test`: 32 test suites passed, 324 tests passed (exit code 0).
