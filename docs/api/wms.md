# WMS & Multi-Warehouse Management API

This document details the REST API endpoints for the **Warehouse Management System (WMS) & Multi-Warehouse Module** in Tayooli ERP. The system provides double-entry stock movement tracking, micro-location hierarchy (zones, racks, bins, pallets/LPNs), omnichannel barcode resolution, inter-warehouse transfers with in-transit states, outbound delivery orders (*Surat Jalan*) with race-free inventory deduction, physical inventory cycle counts (*Stock Opname*) with `@LOSS` discrepancy reconciliation, damaged goods quarantine (*Scrap*) with atomic advisory locks, and omnichannel marketplace sales order batch import (Shopee, Tokopedia, TikTok Shop, Lazada, Blibli) with composite idempotency, multiplier resolution, and frictionless in-place SKU mapping.

---

## Table of Contents

1. [Authentication & Multi-Tenancy](#authentication--multi-tenancy)
2. [RBAC & Warehouse Scoping Matrix](#rbac--warehouse-scoping-matrix)
3. [Concurrency Control & Advisory Locks](#concurrency-control--advisory-locks)
4. [Domain Entities & Enums](#domain-entities--enums)
5. [Endpoints Specification](#endpoints-specification)
   - 1. [GET /api/v1/wms/warehouses](#1-get-apiv1wmswarehouses)
   - 2. [POST /api/v1/wms/warehouses](#2-post-apiv1wmswarehouses)
   - 3. [GET /api/v1/wms/warehouses/{id}](#3-get-apiv1wmswarehousesid)
   - 4. [GET /api/v1/wms/locations](#4-get-apiv1wmslocations)
   - 5. [POST /api/v1/wms/locations](#5-post-apiv1wmslocations)
   - 6. [GET /api/v1/wms/barcodes/resolve](#6-get-apiv1wmsbarcodesresolve)
   - 7. [GET /api/v1/wms/transfers](#7-get-apiv1wmstransfers)
   - 8. [POST /api/v1/wms/transfers](#8-post-apiv1wmstransfers)
   - 9. [GET /api/v1/wms/transfers/{id}](#9-get-apiv1wmstransfersid)
   - 10. [POST /api/v1/wms/transfers/{id}/dispatch](#10-post-apiv1wmstransfersiddispatch)
   - 11. [POST /api/v1/wms/transfers/{id}/receive](#11-post-apiv1wmstransfersidreceive)
   - 12. [GET /api/v1/wms/delivery-orders](#12-get-apiv1wmsdelivery-orders)
   - 13. [POST /api/v1/wms/delivery-orders](#13-post-apiv1wmsdelivery-orders)
   - 14. [GET /api/v1/wms/delivery-orders/{id}](#14-get-apiv1wmsdelivery-ordersid)
   - 15. [POST /api/v1/wms/delivery-orders/{id}/dispatch](#15-post-apiv1wmsdelivery-ordersiddispatch)
   - 16. [GET /api/v1/wms/opnames](#16-get-apiv1wmsopnames)
   - 17. [POST /api/v1/wms/opnames](#17-post-apiv1wmsopnames)
   - 18. [GET /api/v1/wms/opnames/{id}](#18-get-apiv1wmsopnamesid)
   - 19. [POST /api/v1/wms/opnames/{id}/items](#19-post-apiv1wmsopnamesiditems)
   - 20. [POST /api/v1/wms/opnames/{id}/complete](#20-post-apiv1wmsopnamesidcomplete)
   - 21. [GET /api/v1/wms/scraps](#21-get-apiv1wmsscraps)
   - 22. [POST /api/v1/wms/scraps](#22-post-apiv1wmsscraps)
   - 23. [POST /api/v1/wms/marketplace/import](#23-post-apiv1wmsmarketplaceimport)
   - 24. [GET /api/v1/wms/marketplace/batches](#24-get-apiv1wmsmarketplacebatches)
   - 25. [GET /api/v1/wms/marketplace/orders](#25-get-apiv1wmsmarketplaceorders)
   - 26. [GET /api/v1/wms/marketplace/orders/{id}](#26-get-apiv1wmsmarketplaceordersid)
   - 27. [POST /api/v1/wms/marketplace/sku-mappings](#27-post-apiv1wmsmarketplacesku-mappings)
   - 28. [GET /api/v1/wms/marketplace/sku-mappings](#28-get-apiv1wmsmarketplacesku-mappings)
6. [Error Responses Catalog](#error-responses-catalog)

---

## Authentication & Multi-Tenancy

All `/api/v1/wms/*` endpoints require an HTTP `Authorization` header containing a valid JSON Web Token (`Bearer <JWT>`):

```http
Authorization: Bearer <jwt_token>
```

The JWT is validated by `TenantMiddleware` (`internal/middleware/tenant.go`), which injects the following claims into the request context:
- `tenant_id` (UUID): Mandates PostgreSQL Row-Level Security (RLS) scoping via `app.current_tenant_id`. Every query is isolated per tenant.
- `user_id` (UUID): Used to check operator warehouse assignments in `user_warehouses` and to audit movement executions.
- `role` (string): Evaluated against role-based and warehouse-based access control policies.

Requests lacking a valid token or missing tenant claims return `401 Unauthorized`.

---

## RBAC & Warehouse Scoping Matrix

In addition to tenant-level security, WMS enforces **granular warehouse-level access control** (`ValidateWarehouseReadAccess` and `ValidateWarehouseWriteAccess` in `internal/usecase/wms/wms_usecase.go`):

### Role Definitions

| Role | Warehouse Scope | Read Permissions | Write Permissions |
|---|---|---|---|
| `admin`, `owner` | Tenant-wide | All warehouses and locations | Full write access to all warehouses, locations, transfers, delivery orders, stock opnames, and scraps |
| `regional_manager` | Regional cluster | All warehouses in the regional of assigned warehouses | Write access to warehouses within their assigned regional cluster |
| `warehouse` (staf gudang) | Assigned warehouses | Only warehouses explicitly assigned in `user_warehouses` | Write access only to assigned warehouses |
| `auditor` | Tenant-wide | All warehouses and locations (read-only) | **Strictly forbidden** (`403 Forbidden`) from all mutations, dispatches, opname counting/completion, and scrap reporting |

### Endpoint Access Matrix

| Endpoint | Method | Allowed Roles | Warehouse Scoping Rule |
|---|---|---|---|
| `/api/v1/wms/warehouses` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Filtered to accessible warehouses |
| `/api/v1/wms/warehouses` | POST | `admin`, `owner` | Tenant-level write |
| `/api/v1/wms/warehouses/{id}` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | User must have read access to target warehouse |
| `/api/v1/wms/locations` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Scoped to accessible warehouses |
| `/api/v1/wms/locations` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Must have write access to `warehouse_id`; system locations require `admin`/`owner` |
| `/api/v1/wms/barcodes/resolve` | GET | Any authenticated role in tenant | Tenant-wide catalog resolution |
| `/api/v1/wms/transfers` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Scoped to transfers involving accessible warehouses |
| `/api/v1/wms/transfers` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to `from_warehouse_id` |
| `/api/v1/wms/transfers/{id}` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Read access to either `from_warehouse_id` or `to_warehouse_id` |
| `/api/v1/wms/transfers/{id}/dispatch` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to `from_warehouse_id` |
| `/api/v1/wms/transfers/{id}/receive` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to `to_warehouse_id` |
| `/api/v1/wms/delivery-orders` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Scoped to accessible warehouses |
| `/api/v1/wms/delivery-orders` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to `warehouse_id` |
| `/api/v1/wms/delivery-orders/{id}` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Read access to DO `warehouse_id` |
| `/api/v1/wms/delivery-orders/{id}/dispatch` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to DO `warehouse_id` |
| `/api/v1/wms/opnames` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Scoped to accessible warehouses (or filtered by `warehouse_id` query param) |
| `/api/v1/wms/opnames` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to request `warehouse_id` |
| `/api/v1/wms/opnames/{id}` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Read access to opname's `warehouse_id` |
| `/api/v1/wms/opnames/{id}/items` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to opname's `warehouse_id`; location must belong to same warehouse |
| `/api/v1/wms/opnames/{id}/complete` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to opname's `warehouse_id` |
| `/api/v1/wms/scraps` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Scoped to accessible warehouses (or filtered by `warehouse_id` query param) |
| `/api/v1/wms/scraps` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to `warehouse_id`; source location must belong to `warehouse_id` |
| `/api/v1/wms/marketplace/import` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Write access to `warehouse_id`. `auditor` strictly forbidden (`403 Forbidden`). |
| `/api/v1/wms/marketplace/batches` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Scoped to accessible warehouses (or filtered by `warehouse_id` query param). |
| `/api/v1/wms/marketplace/orders` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Scoped to accessible warehouses (or filtered by `warehouse_id`, `batch_id`, `status` query params). |
| `/api/v1/wms/marketplace/orders/{id}` | GET | `admin`, `owner`, `regional_manager`, `warehouse`, `auditor` | Read access to order's `warehouse_id`. |
| `/api/v1/wms/marketplace/sku-mappings` | POST | `admin`, `owner`, `regional_manager`, `warehouse` | Tenant-wide SKU mapping write. `auditor` strictly forbidden (`403 Forbidden`). |
| `/api/v1/wms/marketplace/sku-mappings` | GET | Any authenticated role in tenant | Tenant-wide SKU mapping catalog resolution. |

---

## Concurrency Control & Advisory Locks

To guarantee **zero negative inventory** and eliminate overselling during concurrent dispatch requests, the stock deduction engine (`WMSRepo.DeductLocationStock`) acquires a PostgreSQL transaction-scoped advisory lock before querying stock balances:

```sql
SELECT pg_advisory_xact_lock(hashtext($tenant_id::text || $location_id::text || $product_id::text));
```

### Guarantees
1. **Serialization Per SKU-Location**: Multiple concurrent dispatches for the same SKU at the same bin location are serialized deterministically.
2. **Immediate Release**: The lock is bound to the database transaction (`xact`) and is released automatically when the transaction commits or aborts.
3. **Atomic Ledger Verification**: Available balance is dynamically computed from `stock_movements`. If available stock is less than requested quantity, the transaction rolls back and returns `422 Unprocessable Entity` (`domain.ErrInsufficientStock`).

---

## Domain Entities & Enums

### Location Types (`location_type`)
- `INTERNAL`: Physical storage location inside a warehouse (Zone, Rack, Bin, Pallet).
- `VENDOR`: System virtual source for inbound goods receipts.
- `CUSTOMER`: System virtual destination for outbound customer dispatches.
- `TRANSIT`: System virtual holding location for goods moving between warehouses.
- `LOSS`: System virtual location for stock opname/inventory discrepancy adjustments.
- `SCRAP`: System virtual location for damaged, defective, or quarantined inventory.

### Stock Transfer Statuses (`transfer_status`)
- `DRAFT`: Initial draft created by warehouse operator.
- `PENDING_APPROVAL`: Awaiting manager approval (if approval policy enabled).
- `APPROVED`: Approved for dispatch.
- `DISPATCHED` / `IN_TRANSIT`: Stock deducted from source location and placed in `@TRANSIT`.
- `RECEIVED`: Stock received at destination warehouse and moved from `@TRANSIT` to destination location.
- `REJECTED`: Transfer rejected or canceled.

### Delivery Order Statuses (`delivery_order_status`)
- `DRAFT`: Newly generated order (Surat Jalan).
- `CONFIRMED`: Confirmed against Sales Order.
- `PICKED`: Items picked from warehouse racks.
- `PACKED`: Items packed into cartons/pallets.
- `SHIPPED`: Goods dispatched; stock deducted from warehouse bin to `@CUSTOMER`.
- `DELIVERED`: Recipient acknowledged receipt.
- `RETURNED`: Shipment returned by courier or customer.
- `CANCELLED`: Delivery order cancelled.

### Stock Opname Statuses (`stock_opname_status`)
- `DRAFT`: Initial draft count session created. Line items and physical counts can be added or adjusted.
- `IN_PROGRESS`: Physical count actively being conducted across warehouse bins.
- `COMPLETED`: Stock count completed, verified, and approved. Discrepancy movements posted to/from `@LOSS`. Session is permanently immutable.
- `CANCELLED`: Opname session voided or canceled without ledger impact.

### Stock Movement Reference Types (`reference_type`)
- `TRANSFER`: Inter-warehouse stock transfer movement.
- `DELIVERY_ORDER`: Outbound sales delivery order dispatch (*Surat Jalan*).
- `GOODS_RECEIPT`: Inbound vendor purchase receipt.
- `ADJUSTMENT`: Manual stock balance adjustment.
- `OPNAME`: Physical count discrepancy reconciliation movement to/from system virtual location `@LOSS`.
- `SCRAP`: Damaged, defective, or quarantined inventory deduction to `@SCRAP` or dedicated physical quarantine bay.
- `MARKETPLACE`: Omnichannel marketplace sales order stock deduction from warehouse storage to system virtual location `@CUSTOMER`.

### Marketplace Channels (`channel`)
- `SHOPEE`: Shopee marketplace order export / ingestion.
- `TOKOPEDIA`: Tokopedia marketplace order export / ingestion.
- `TIKTOK`: TikTok Shop marketplace order export / ingestion.
- `LAZADA`: Lazada marketplace order export / ingestion.
- `BLIBLI`: Blibli marketplace order export / ingestion.
- `OTHER`: Generic e-commerce CSV or manual multi-channel order ingestion.

### Marketplace Batch Statuses (`status`)
- `PENDING`: Initial import batch created and awaiting background processing.
- `PROCESSING`: Order parsing, SKU resolution, and stock deduction actively in progress.
- `COMPLETED`: Ingestion batch completed; all valid orders processed and stored.
- `FAILED`: Ingestion batch failed completely (zero orders processed, or all duplicate/invalid).

### Marketplace Order Statuses (`status`)
- `PENDING`: Order ingested and validated, awaiting inventory allocation.
- `PROCESSING`: Inventory deduction and order fulfillment currently executing.
- `COMPLETED`: Order successfully mapped, stock deducted to `@CUSTOMER`, fulfillment completed.
- `FAILED`: Order processing encountered an unrecoverable system failure.
- `UNMAPPED_SKU`: Order contains one or more line items with external SKUs not mapped to internal warehouse products. Stock deduction deferred until SKU is mapped.
- `STOCK_INSUFFICIENT`: Order contains valid mapped products, but warehouse stock is lower than required quantity. Order flagged without failing the entire batch.

### SKU Mapping Types (`mapping_type`)
- `MARKETPLACE`: Channel-specific external SKU mapping with packaging multiplier (e.g. Shopee / Tokopedia bundle).
- `CUSTOMER`: B2B customer-specific SKU alias.
- `VENDOR`: Vendor catalog SKU alias for purchase orders and inbound receipts.

---

## Endpoints Specification

### 1. GET /api/v1/wms/warehouses

Lists all warehouses accessible to the authenticated user based on role and warehouse assignment scoping.

- **URL:** `/api/v1/wms/warehouses`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:** None

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "a0000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "regional_id": "c0000000-0000-0000-0000-000000000001",
      "code": "WH-JKT-01",
      "name": "Jakarta Central Hub",
      "address": "Jl. Gatot Subroto No. 12, Jakarta Selatan",
      "is_active": true,
      "created_at": "2026-09-01T08:00:00Z",
      "updated_at": "2026-09-01T08:00:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Warehouses listed successfully (returns empty array `[]` if none).
- `401 Unauthorized`: Missing or invalid JWT token.
- `500 Internal Server Error`: Database query failure.

---

### 2. POST /api/v1/wms/warehouses

Creates a new physical or operational warehouse entity.

- **URL:** `/api/v1/wms/warehouses`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json`
- **RBAC Requirement:** `admin` or `owner` role.
- **Max Body Size:** 1 MB (`413` if exceeded).

#### Request Body
```json
{
  "code": "WH-SBY-01",
  "name": "Surabaya Distribution Center",
  "regional_id": "c0000000-0000-0000-0000-000000000002",
  "address": "Kawasan Industri Rungkut Blok A-5, Surabaya",
  "is_active": true
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `code` | string | **Yes** | Unique identifier for warehouse. Automatically uppercase-normalized. |
| `name` | string | **Yes** | Descriptive warehouse name. |
| `regional_id` | UUID | No | Regional administrative grouping ID. |
| `address` | string | No | Physical street address. |
| `is_active` | boolean | No | Active status flag (defaults to `true`). |

#### Response: `201 Created`
```json
{
  "id": "a0000000-0000-0000-0000-000000000002",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "regional_id": "c0000000-0000-0000-0000-000000000002",
  "code": "WH-SBY-01",
  "name": "Surabaya Distribution Center",
  "address": "Kawasan Industri Rungkut Blok A-5, Surabaya",
  "is_active": true,
  "created_at": "2026-09-06T10:00:00Z",
  "updated_at": "2026-09-06T10:00:00Z"
}
```

#### Status Codes
- `201 Created`: Warehouse created successfully.
- `400 Bad Request`: Missing `code` or `name`, or invalid JSON payload.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: User is not an `admin` or `owner`.
- `413 Request Entity Too Large`: Body size exceeds 1 MB.
- `500 Internal Server Error`: Duplicate code or database error.

---

### 3. GET /api/v1/wms/warehouses/{id}

Retrieves details for a single warehouse by ID.

- **URL:** `/api/v1/wms/warehouses/{id}`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Warehouse unique ID.
- **RBAC Requirement:** User must have read access to target warehouse.

#### Response: `200 OK`
```json
{
  "id": "a0000000-0000-0000-0000-000000000001",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "regional_id": "c0000000-0000-0000-0000-000000000001",
  "code": "WH-JKT-01",
  "name": "Jakarta Central Hub",
  "address": "Jl. Gatot Subroto No. 12, Jakarta Selatan",
  "is_active": true,
  "created_at": "2026-09-01T08:00:00Z",
  "updated_at": "2026-09-01T08:00:00Z"
}
```

#### Status Codes
- `200 OK`: Warehouse retrieved successfully.
- `400 Bad Request`: Invalid UUID syntax.
- `401 Unauthorized`: Missing tenant context.
- `403 Forbidden`: User does not have access to this warehouse (`unauthorized warehouse access`).
- `404 Not Found`: Warehouse ID does not exist in tenant (`warehouse not found`).
- `500 Internal Server Error`: Internal database error.

---

### 4. GET /api/v1/wms/locations

Lists warehouse locations (zones, racks, bins, pallets, or virtual locations).

- **URL:** `/api/v1/wms/locations`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `warehouse_id` (UUID, optional): Filter locations for a specific warehouse. When omitted, returns all locations belonging to warehouses the user is authorized to view (plus virtual system locations).

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "b0000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "warehouse_id": "a0000000-0000-0000-0000-000000000001",
      "parent_id": null,
      "code": "RACK-A01-01",
      "barcode": "LOC-A01-01",
      "name": "Rack A Shelf 1 Bin 1",
      "type": "INTERNAL",
      "is_pallet": false,
      "pallet_number": null,
      "max_capacity": "500.00",
      "created_at": "2026-09-01T08:30:00Z",
      "updated_at": "2026-09-01T08:30:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Locations retrieved successfully.
- `400 Bad Request`: `invalid warehouse_id uuid`.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: User does not have read access to the specified warehouse.
- `500 Internal Server Error`: Internal database error.

---

### 5. POST /api/v1/wms/locations

Creates a location in the warehouse hierarchy (Zone, Rack, Bin, Pallet/LPN, or Virtual Location).

- **URL:** `/api/v1/wms/locations`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json`
- **RBAC Requirement:** Caller must have write access to `warehouse_id`. Creating non-warehouse system locations (`warehouse_id = null`) requires `admin` or `owner`. `auditor` role is rejected.
- **Max Body Size:** 1 MB.

#### Request Body
```json
{
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "parent_id": "b0000000-0000-0000-0000-000000000000",
  "code": "RACK-A01-02",
  "barcode": "LOC-A01-02",
  "name": "Rack A Shelf 1 Bin 2",
  "type": "INTERNAL",
  "is_pallet": false,
  "pallet_number": null,
  "max_capacity": "250.00"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `warehouse_id` | UUID | No | Owning warehouse ID (null for system virtual locations). |
| `parent_id` | UUID | No | Parent location ID (for hierarchy: Zone $\to$ Rack $\to$ Bin). |
| `code` | string | **Yes** | Unique location code within warehouse. |
| `barcode` | string | No | Physical scannable barcode label on the bin. |
| `name` | string | **Yes** | Descriptive name. |
| `type` | string | No | Enum: `INTERNAL`, `VENDOR`, `CUSTOMER`, `TRANSIT`, `LOSS`, `SCRAP`. Defaults to `INTERNAL`. |
| `is_pallet` | boolean | No | Whether this location represents a mobile pallet/LPN. Defaults to `false`. |
| `pallet_number`| string | No | Unique Pallet / LPN number if `is_pallet = true`. |
| `max_capacity` | string (decimal) | No | Maximum storage capacity in units or weight. |

#### Response: `201 Created`
```json
{
  "id": "b0000000-0000-0000-0000-000000000002",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "parent_id": "b0000000-0000-0000-0000-000000000000",
  "code": "RACK-A01-02",
  "barcode": "LOC-A01-02",
  "name": "Rack A Shelf 1 Bin 2",
  "type": "INTERNAL",
  "is_pallet": false,
  "pallet_number": null,
  "max_capacity": "250.00",
  "created_at": "2026-09-06T10:15:00Z",
  "updated_at": "2026-09-06T10:15:00Z"
}
```

#### Status Codes
- `201 Created`: Location created successfully.
- `400 Bad Request`: Missing `code` or `name`, or invalid JSON payload.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: Caller lacks write access to warehouse or is `auditor`.
- `413 Request Entity Too Large`: Body size exceeds 1 MB.
- `500 Internal Server Error`: Duplicate code within warehouse or DB constraint failure.

---

### 6. GET /api/v1/wms/barcodes/resolve

Resolves a scanned code against the omnichannel product resolution engine. Checks:
1. Internal SKU in `products` table (`source: "SKU"`, `multiplier: 1.0`).
2. Physical packaging barcode in `product_barcodes` (`source: "BARCODE"`, with UOM multiplier).
3. Marketplace or customer external SKU in `product_sku_mappings` (`source: "MAPPING"`, with channel multiplier).

- **URL:** `/api/v1/wms/barcodes/resolve`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `code` (string, **required**): Scanned barcode, internal SKU, or marketplace SKU.

#### Response: `200 OK`
```json
{
  "product_id": "33333333-3333-3333-3333-333333333333",
  "sku": "PROD-A",
  "name": "Indomie Goreng Original 85g",
  "barcode": "8998866200227",
  "external_sku": "MKT-TOKOPEDIA-INDOMIE-DUS",
  "multiplier": "40.0000",
  "source": "BARCODE"
}
```

#### Status Codes
- `200 OK`: Code resolved successfully.
- `400 Bad Request`: Missing `code` query parameter or empty string.
- `401 Unauthorized`: Missing or invalid JWT token.
- `404 Not Found`: No product matches the provided barcode or SKU in this tenant (`barcode or external sku not found`).
- `500 Internal Server Error`: Database query failure.

---

### 7. GET /api/v1/wms/transfers

Lists inter-warehouse stock transfers.

- **URL:** `/api/v1/wms/transfers`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `warehouse_id` (UUID, optional): Filter transfers where `from_warehouse_id` or `to_warehouse_id` matches the given warehouse ID.

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "d0000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "transfer_number": "TR-2026-0001",
      "from_warehouse_id": "a0000000-0000-0000-0000-000000000001",
      "to_warehouse_id": "a0000000-0000-0000-0000-000000000002",
      "status": "DRAFT",
      "requested_by": "11111111-1111-1111-1111-111111111111",
      "approved_by": null,
      "vehicle_plate": "B 1234 XYZ",
      "driver_name": "Budi Santoso",
      "dispatched_at": null,
      "received_at": null,
      "notes": "Weekly replenishment",
      "created_at": "2026-09-06T11:00:00Z",
      "updated_at": "2026-09-06T11:00:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Transfers listed successfully.
- `400 Bad Request`: `invalid warehouse_id uuid`.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: User unauthorized to read specified warehouse.
- `500 Internal Server Error`: Database error.

---

### 8. POST /api/v1/wms/transfers

Creates a draft inter-warehouse stock transfer with line items.

- **URL:** `/api/v1/wms/transfers`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json`
- **RBAC Requirement:** Caller must have write access to `from_warehouse_id`. `auditor` role is rejected with `403`.
- **Max Body Size:** 1 MB.

#### Request Body
```json
{
  "from_warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "to_warehouse_id": "a0000000-0000-0000-0000-000000000002",
  "transfer_number": "TR-2026-0002",
  "vehicle_plate": "B 5678 KLM",
  "driver_name": "Agus Salim",
  "notes": "Inter-branch transfer",
  "items": [
    {
      "product_id": "33333333-3333-3333-3333-333333333333",
      "requested_qty": "20.0000",
      "source_location_id": "b0000000-0000-0000-0000-000000000001",
      "dest_location_id": "b0000000-0000-0000-0000-000000000002"
    }
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `from_warehouse_id` | UUID | **Yes** | Originating warehouse ID. |
| `to_warehouse_id` | UUID | **Yes** | Destination warehouse ID. |
| `transfer_number` | string | No | Custom transfer number (auto-generated if empty). |
| `vehicle_plate` | string | No | Vehicle plate number. |
| `driver_name` | string | No | Driver name. |
| `notes` | string | No | Operational transfer notes. |
| `items` | array | **Yes** | Minimum 1 item required. |
| `items[].product_id` | UUID | **Yes** | Product UUID. |
| `items[].requested_qty` | string (decimal) | **Yes** | Transfer quantity (> 0). |
| `items[].source_location_id` | UUID | No | Must belong to `from_warehouse_id` (anti-spoofing). |
| `items[].dest_location_id` | UUID | No | Must belong to `to_warehouse_id` (anti-spoofing). |

#### Response: `201 Created`
```json
{
  "id": "d0000000-0000-0000-0000-000000000002",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "transfer_number": "TR-2026-0002",
  "from_warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "to_warehouse_id": "a0000000-0000-0000-0000-000000000002",
  "status": "DRAFT",
  "requested_by": "11111111-1111-1111-1111-111111111111",
  "approved_by": null,
  "vehicle_plate": "B 5678 KLM",
  "driver_name": "Agus Salim",
  "dispatched_at": null,
  "received_at": null,
  "notes": "Inter-branch transfer",
  "created_at": "2026-09-06T11:10:00Z",
  "updated_at": "2026-09-06T11:10:00Z"
}
```

#### Status Codes
- `201 Created`: Transfer created successfully.
- `400 Bad Request`: Empty items, zero or negative requested quantity, or malformed JSON.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: Caller lacks write access to `from_warehouse_id` or location does not belong to the specified warehouse.
- `404 Not Found`: Target warehouse does not exist.
- `413 Request Entity Too Large`: Body exceeds 1 MB.
- `500 Internal Server Error`: Database error.

---

### 9. GET /api/v1/wms/transfers/{id}

Retrieves transfer details and line items.

- **URL:** `/api/v1/wms/transfers/{id}`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Transfer ID.
- **RBAC Requirement:** Caller must have read access to either `from_warehouse_id` or `to_warehouse_id`.

#### Response: `200 OK`
```json
{
  "transfer": {
    "id": "d0000000-0000-0000-0000-000000000001",
    "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "transfer_number": "TR-2026-0001",
    "from_warehouse_id": "a0000000-0000-0000-0000-000000000001",
    "to_warehouse_id": "a0000000-0000-0000-0000-000000000002",
    "status": "DRAFT",
    "requested_by": "11111111-1111-1111-1111-111111111111",
    "approved_by": null,
    "vehicle_plate": "B 1234 XYZ",
    "driver_name": "Budi Santoso",
    "dispatched_at": null,
    "received_at": null,
    "notes": "Weekly replenishment",
    "created_at": "2026-09-06T11:00:00Z",
    "updated_at": "2026-09-06T11:00:00Z"
  },
  "items": [
    {
      "id": "e0000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "transfer_id": "d0000000-0000-0000-0000-000000000001",
      "product_id": "33333333-3333-3333-3333-333333333333",
      "requested_qty": "10.0000",
      "sent_qty": "0.0000",
      "received_qty": "0.0000",
      "source_location_id": "b0000000-0000-0000-0000-000000000001",
      "dest_location_id": "b0000000-0000-0000-0000-000000000002",
      "created_at": "2026-09-06T11:00:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Transfer and items retrieved.
- `400 Bad Request`: Invalid transfer UUID syntax.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: User has no access to either source or destination warehouse.
- `404 Not Found`: Transfer not found (`stock transfer not found`).
- `500 Internal Server Error`: Database query failure.

---

### 10. POST /api/v1/wms/transfers/{id}/dispatch

Dispatches a transfer. Validates available stock under a PostgreSQL advisory transaction lock (`pg_advisory_xact_lock`), deducts stock from `source_location_id`, records double-entry movement to `@TRANSIT`, and updates transfer status to `IN_TRANSIT`.

- **URL:** `/api/v1/wms/transfers/{id}/dispatch`
- **Method:** `POST`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Transfer ID.
- **RBAC Requirement:** Caller must have write access to `from_warehouse_id`. `auditor` role is rejected.
- **Allowed Pre-Statuses:** `DRAFT` or `APPROVED`. Attempting to dispatch from `PENDING_APPROVAL`, `IN_TRANSIT`, `RECEIVED`, or `REJECTED` returns `400 Bad Request`.

#### Response: `200 OK`
```json
{
  "id": "d0000000-0000-0000-0000-000000000001",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "transfer_number": "TR-2026-0001",
  "from_warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "to_warehouse_id": "a0000000-0000-0000-0000-000000000002",
  "status": "IN_TRANSIT",
  "requested_by": "11111111-1111-1111-1111-111111111111",
  "approved_by": null,
  "vehicle_plate": "B 1234 XYZ",
  "driver_name": "Budi Santoso",
  "dispatched_at": "2026-09-06T11:25:00Z",
  "received_at": null,
  "notes": "Weekly replenishment",
  "created_at": "2026-09-06T11:00:00Z",
  "updated_at": "2026-09-06T11:25:00Z"
}
```

#### Status Codes
- `200 OK`: Transfer dispatched; stock moved to `@TRANSIT`.
- `400 Bad Request`: Invalid transfer status transition (`invalid stock transfer status transition`).
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: Caller lacks write access to source warehouse.
- `404 Not Found`: Transfer or location not found.
- `413 Request Entity Too Large`: Body exceeds 1 MB.
- `422 Unprocessable Entity`: Available stock at source bin is insufficient (`insufficient stock at location`).
- `500 Internal Server Error`: Ledger insertion or status update failure.

---

### 11. POST /api/v1/wms/transfers/{id}/receive

Confirms receipt of in-transit goods at the destination warehouse. Moves stock from system `@TRANSIT` location to destination bin (`dest_location_id`) and updates transfer status to `RECEIVED`.

- **URL:** `/api/v1/wms/transfers/{id}/receive`
- **Method:** `POST`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Transfer ID.
- **RBAC Requirement:** Caller must have write access to `to_warehouse_id`. `auditor` role is rejected.
- **Allowed Pre-Statuses:** `IN_TRANSIT` or `DISPATCHED`.

#### Response: `200 OK`
```json
{
  "id": "d0000000-0000-0000-0000-000000000001",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "transfer_number": "TR-2026-0001",
  "from_warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "to_warehouse_id": "a0000000-0000-0000-0000-000000000002",
  "status": "RECEIVED",
  "requested_by": "11111111-1111-1111-1111-111111111111",
  "approved_by": null,
  "vehicle_plate": "B 1234 XYZ",
  "driver_name": "Budi Santoso",
  "dispatched_at": "2026-09-06T11:25:00Z",
  "received_at": "2026-09-06T13:40:00Z",
  "notes": "Weekly replenishment",
  "created_at": "2026-09-06T11:00:00Z",
  "updated_at": "2026-09-06T13:40:00Z"
}
```

#### Status Codes
- `200 OK`: Transfer received; stock moved into destination warehouse location.
- `400 Bad Request`: Transfer is not currently in transit (`invalid stock transfer status transition`).
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: Caller lacks write access to destination warehouse.
- `404 Not Found`: Transfer or destination location not found.
- `413 Request Entity Too Large`: Body exceeds 1 MB.
- `500 Internal Server Error`: Ledger insertion or status update failure.

---

### 12. GET /api/v1/wms/delivery-orders

Lists outbound Delivery Orders (*Surat Jalan*).

- **URL:** `/api/v1/wms/delivery-orders`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `warehouse_id` (UUID, optional): Filter delivery orders for a specific warehouse.

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "f0000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "sales_order_id": "44444444-4444-4444-4444-444444444444",
      "warehouse_id": "a0000000-0000-0000-0000-000000000001",
      "do_number": "DO-2026-0001",
      "status": "DRAFT",
      "expedition_name": "JNE Trucking",
      "tracking_number": "JNE-TRACK-9912",
      "driver_name": "Agus Salim",
      "vehicle_plate": "B 9876 ABC",
      "recipient_name": "PT Mitra Sentosa Abadi",
      "received_date": null,
      "created_at": "2026-09-06T12:00:00Z",
      "updated_at": "2026-09-06T12:00:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Delivery orders listed successfully.
- `400 Bad Request`: `invalid warehouse_id uuid`.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: User lacks read access to the specified warehouse.
- `500 Internal Server Error`: Database query error.

---

### 13. POST /api/v1/wms/delivery-orders

Creates an outbound Delivery Order (*Surat Jalan*) tied to a Sales Order. Status is strictly initialized to `DRAFT`.

- **URL:** `/api/v1/wms/delivery-orders`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json`
- **RBAC Requirement:** Caller must have write access to `warehouse_id`. `auditor` role is rejected.
- **Max Body Size:** 1 MB.

#### Request Body
```json
{
  "sales_order_id": "44444444-4444-4444-4444-444444444444",
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "do_number": "DO-2026-0001",
  "expedition_name": "JNE Trucking",
  "tracking_number": "JNE-TRACK-9912",
  "driver_name": "Agus Salim",
  "vehicle_plate": "B 9876 ABC",
  "recipient_name": "PT Mitra Sentosa Abadi",
  "items": [
    {
      "product_id": "33333333-3333-3333-3333-333333333333",
      "quantity": "5.0000",
      "location_id": "b0000000-0000-0000-0000-000000000001"
    }
  ]
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `sales_order_id` | UUID | **Yes** | Referencing Sales Order. |
| `warehouse_id` | UUID | **Yes** | Dispatching warehouse ID. |
| `do_number` | string | No | Custom DO number (auto-generated if empty). |
| `expedition_name` | string | No | Logistics partner / courier service. |
| `tracking_number` | string | No | Airway bill / tracking code. |
| `driver_name` | string | No | Courier driver name. |
| `vehicle_plate` | string | No | Vehicle plate number. |
| `recipient_name` | string | No | Receiving party contact or company name. |
| `items` | array | **Yes** | Minimum 1 line item required. |
| `items[].product_id` | UUID | **Yes** | Product UUID. |
| `items[].quantity` | string (decimal) | **Yes** | Quantity to dispatch (> 0). |
| `items[].location_id` | UUID | **Yes** | Picking location. Must belong to `warehouse_id`. |

#### Response: `201 Created`
```json
{
  "id": "f0000000-0000-0000-0000-000000000001",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "sales_order_id": "44444444-4444-4444-4444-444444444444",
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "do_number": "DO-2026-0001",
  "status": "DRAFT",
  "expedition_name": "JNE Trucking",
  "tracking_number": "JNE-TRACK-9912",
  "driver_name": "Agus Salim",
  "vehicle_plate": "B 9876 ABC",
  "recipient_name": "PT Mitra Sentosa Abadi",
  "received_date": null,
  "created_at": "2026-09-06T12:00:00Z",
  "updated_at": "2026-09-06T12:00:00Z"
}
```

#### Status Codes
- `201 Created`: Delivery order draft created.
- `400 Bad Request`: Empty items list, non-positive quantity, or malformed JSON.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: Caller lacks write access or location belongs to another warehouse.
- `413 Request Entity Too Large`: Body exceeds 1 MB.
- `500 Internal Server Error`: Database error.

---

### 14. GET /api/v1/wms/delivery-orders/{id}

Retrieves delivery order header and line items.

- **URL:** `/api/v1/wms/delivery-orders/{id}`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Delivery order ID.
- **RBAC Requirement:** Caller must have read access to the DO's `warehouse_id`.

#### Response: `200 OK`
```json
{
  "delivery_order": {
    "id": "f0000000-0000-0000-0000-000000000001",
    "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "sales_order_id": "44444444-4444-4444-4444-444444444444",
    "warehouse_id": "a0000000-0000-0000-0000-000000000001",
    "do_number": "DO-2026-0001",
    "status": "DRAFT",
    "expedition_name": "JNE Trucking",
    "tracking_number": "JNE-TRACK-9912",
    "driver_name": "Agus Salim",
    "vehicle_plate": "B 9876 ABC",
    "recipient_name": "PT Mitra Sentosa Abadi",
    "received_date": null,
    "created_at": "2026-09-06T12:00:00Z",
    "updated_at": "2026-09-06T12:00:00Z"
  },
  "items": [
    {
      "id": "80000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "delivery_order_id": "f0000000-0000-0000-0000-000000000001",
      "product_id": "33333333-3333-3333-3333-333333333333",
      "quantity": "5.0000",
      "location_id": "b0000000-0000-0000-0000-000000000001",
      "created_at": "2026-09-06T12:00:00Z",
      "product_name": "Kopi Arabika Gayo 250g",
      "product_sku": "KOP-GAYO-250",
      "location_code": "WH1-RACK-A-01"
    }
  ]
}
```

##### Enriched Item Fields (Official Document & Picking Slip Support)
The `items` array is enriched via repository-level SQL `LEFT JOIN`s against `products` and `warehouse_locations` to support zero-latency rendering of official picking slips and *Surat Jalan* print layouts without N+1 client-side query roundtrips:
- `product_name` (string, nullable): Official catalog title of the product (`products.name`).
- `product_sku` (string, nullable): Unique Master SKU code (`products.sku`) used for barcode and line-item cross-checking.
- `location_code` (string, nullable): Physical picking micro-location code (`warehouse_locations.code`, e.g. rack/bin `WH1-RACK-A-01`) indicating exactly where warehouse staff must pull inventory for dispatch.

#### Status Codes
- `200 OK`: Delivery order and line items retrieved.
- `400 Bad Request`: Invalid DO UUID syntax.
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: User not authorized to view this warehouse.
- `404 Not Found`: Delivery order not found (`delivery order not found`).
- `500 Internal Server Error`: Database error.

---

### 15. POST /api/v1/wms/delivery-orders/{id}/dispatch

Dispatches a delivery order. Transitions order status to `SHIPPED` and executes double-entry stock deduction from warehouse picking location to system virtual location `@CUSTOMER` under a PostgreSQL advisory transaction lock.

- **URL:** `/api/v1/wms/delivery-orders/{id}/dispatch`
- **Method:** `POST`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Delivery order ID.
- **RBAC Requirement:** Caller must have write access to DO `warehouse_id`. `auditor` role is rejected.
- **Allowed Pre-Statuses:** `DRAFT`, `CONFIRMED`, `PACKED`. Attempting to dispatch already `SHIPPED`, `DELIVERED`, `CANCELLED`, or `RETURNED` orders returns `400 Bad Request`.

#### Response: `200 OK`
```json
{
  "id": "f0000000-0000-0000-0000-000000000001",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "sales_order_id": "44444444-4444-4444-4444-444444444444",
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "do_number": "DO-2026-0001",
  "status": "SHIPPED",
  "expedition_name": "JNE Trucking",
  "tracking_number": "JNE-TRACK-9912",
  "driver_name": "Agus Salim",
  "vehicle_plate": "B 9876 ABC",
  "recipient_name": "PT Mitra Sentosa Abadi",
  "received_date": null,
  "created_at": "2026-09-06T12:00:00Z",
  "updated_at": "2026-09-06T12:45:00Z"
}
```

#### Status Codes
- `200 OK`: Delivery order dispatched; stock moved to `@CUSTOMER`.
- `400 Bad Request`: Invalid status transition (`invalid status for this operation`).
- `401 Unauthorized`: Missing or invalid JWT token.
- `403 Forbidden`: Caller lacks write access to warehouse or location belongs to another warehouse.
- `404 Not Found`: Delivery order or location not found.
- `413 Request Entity Too Large`: Body exceeds 1 MB.
- `422 Unprocessable Entity`: Available stock at bin is less than order quantity (`insufficient stock at location`).
- `500 Internal Server Error`: Movement ledger insertion or update failure.

---

### 16. GET /api/v1/wms/opnames

Lists stock opname (physical inventory count) sessions accessible to the authenticated user. Results can be filtered to a specific warehouse.

- **URL:** `/api/v1/wms/opnames`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `warehouse_id` (UUID, optional): Filter stock opname sessions by warehouse ID. If provided, caller must have read access to the specified warehouse. If omitted, returns sessions across all warehouses accessible to the caller.
- **RBAC Requirement:**
  - `admin`, `owner`: View opname sessions across all tenant warehouses.
  - `regional_manager`: View opname sessions in warehouses within their assigned regional cluster.
  - `warehouse`: View opname sessions only for warehouses explicitly assigned in `user_warehouses`.
  - `auditor`: Tenant-wide read-only view.

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "11111111-1111-1111-1111-111111111111",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "warehouse_id": "a0000000-0000-0000-0000-000000000001",
      "opname_number": "OPN-2026-0001",
      "status": "DRAFT",
      "conducted_by": "u0000000-0000-0000-0000-000000000001",
      "approved_by": null,
      "notes": "Q3 Routine Cycle Count - Central Hub",
      "created_at": "2026-09-06T08:00:00Z",
      "updated_at": "2026-09-06T08:00:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Stock opnames listed successfully (returns empty array `[]` if none found).
- `400 Bad Request`: Invalid `warehouse_id` UUID syntax (`invalid warehouse_id uuid`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks read access to the requested warehouse (`unauthorized warehouse access`).
- `404 Not Found`: Warehouse specified in `warehouse_id` does not exist or belongs to another tenant (`warehouse not found`).
- `500 Internal Server Error`: Database query failure (`internal server error`).

---

### 17. POST /api/v1/wms/opnames

Creates a new physical stock opname session for a designated warehouse. The session is initialized in `DRAFT` status and assigned to the authenticated user (`conducted_by`).

- **URL:** `/api/v1/wms/opnames`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json`
- **RBAC Requirement:**
  - `admin`, `owner`: May create opname sessions for any warehouse.
  - `regional_manager`: May create opname sessions for warehouses in their assigned regional cluster.
  - `warehouse`: May create opname sessions only for assigned warehouses in `user_warehouses`.
  - `auditor`: Strictly rejected with `403 Forbidden` (`forbidden`).
- **Max Body Size:** 1 MB (`413` if exceeded).

#### Request Body
```json
{
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "opname_number": "OPN-2026-0001",
  "notes": "Q3 Routine Cycle Count - Central Hub"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `warehouse_id` | UUID | **Yes** | Target warehouse ID where physical counting will take place. |
| `opname_number` | string | No | Custom opname identification code. If omitted, automatically generated as `OPN-<timestamp_ms>`. |
| `notes` | string | No | Optional description, scope, or operational notes. |

#### Response: `201 Created`
```json
{
  "id": "11111111-1111-1111-1111-111111111111",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "opname_number": "OPN-2026-0001",
  "status": "DRAFT",
  "conducted_by": "u0000000-0000-0000-0000-000000000001",
  "approved_by": null,
  "notes": "Q3 Routine Cycle Count - Central Hub",
  "created_at": "2026-09-06T08:00:00Z",
  "updated_at": "2026-09-06T08:00:00Z"
}
```

#### Status Codes
- `201 Created`: Stock opname session created successfully with `DRAFT` status.
- `400 Bad Request`: Malformed JSON or missing required `warehouse_id` (`invalid input`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks write access to target warehouse (`unauthorized warehouse access`) or has `auditor` role (`forbidden`).
- `404 Not Found`: Warehouse not found or belongs to another tenant (`warehouse not found`).
- `413 Request Entity Too Large`: Body exceeds 1 MB limit (`request entity too large`).
- `500 Internal Server Error`: Database insertion failure (`internal server error`).

---

### 18. GET /api/v1/wms/opnames/{id}

Retrieves the header details and recorded line items (counted products, locations, physical quantities, system quantities, and discrepancies) of a stock opname session.

- **URL:** `/api/v1/wms/opnames/{id}`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Stock opname ID.
- **RBAC Requirement:**
  - Caller must have read access to the warehouse associated with the opname (`admin`, `owner`, `regional_manager`, `warehouse`, `auditor`).

#### Response: `200 OK`
```json
{
  "opname": {
    "id": "11111111-1111-1111-1111-111111111111",
    "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "warehouse_id": "a0000000-0000-0000-0000-000000000001",
    "opname_number": "OPN-2026-0001",
    "status": "DRAFT",
    "conducted_by": "u0000000-0000-0000-0000-000000000001",
    "approved_by": null,
    "notes": "Q3 Routine Cycle Count - Central Hub",
    "created_at": "2026-09-06T08:00:00Z",
    "updated_at": "2026-09-06T08:00:00Z"
  },
  "items": [
    {
      "id": "22222222-2222-2222-2222-222222222222",
      "opname_id": "11111111-1111-1111-1111-111111111111",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "product_id": "b0000000-0000-0000-0000-000000000001",
      "location_id": "l0000000-0000-0000-0000-000000000001",
      "system_qty": "100.0000",
      "physical_qty": "95.0000",
      "discrepancy_qty": "-5.0000",
      "notes": "5 units damaged/missing",
      "created_at": "2026-09-06T08:30:00Z"
    },
    {
      "id": "22222222-2222-2222-2222-222222222223",
      "opname_id": "11111111-1111-1111-1111-111111111111",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "product_id": "b0000000-0000-0000-0000-000000000002",
      "location_id": "l0000000-0000-0000-0000-000000000002",
      "system_qty": "50.0000",
      "physical_qty": "52.0000",
      "discrepancy_qty": "2.0000",
      "notes": "2 units surplus found unbagged",
      "created_at": "2026-09-06T08:35:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Stock opname session and associated items retrieved successfully.
- `400 Bad Request`: Invalid opname UUID format (`invalid opname id uuid`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks read access to the opname's warehouse (`unauthorized warehouse access`).
- `404 Not Found`: Opname not found or belongs to another tenant (`stock opname not found`).
- `500 Internal Server Error`: Database query failure (`internal server error`).

---

### 19. POST /api/v1/wms/opnames/{id}/items

Records a counted product and physical quantity against a warehouse location in a stock opname session. The system dynamically reads the current theoretical stock (`system_qty`) from the immutable ledger `stock_movements` and computes `discrepancy_qty = physical_qty - system_qty`.

- **URL:** `/api/v1/wms/opnames/{id}/items`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json`
- **Path Parameters:**
  - `id` (UUID, required): Stock opname ID.
- **RBAC Requirement:**
  - Write access to opname's `warehouse_id`. `auditor` role is strictly rejected (`403 Forbidden`).
- **Allowed Pre-Statuses:** `DRAFT`, `IN_PROGRESS`. Adding items to an opname with status `COMPLETED` or `CANCELLED` returns `400 Bad Request` (`invalid stock opname status transition`).
- **Max Body Size:** 1 MB (`413` if exceeded).

#### Request Body
```json
{
  "product_id": "b0000000-0000-0000-0000-000000000001",
  "location_id": "l0000000-0000-0000-0000-000000000001",
  "physical_qty": "95.0000",
  "notes": "5 units damaged/missing"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `product_id` | UUID | **Yes** | Master product ID being counted. |
| `location_id` | UUID | **Yes** | Storage bin or location where items were physically verified. Must belong to the opname's warehouse. |
| `physical_qty` | string / number | **Yes** | Actual physical count. Must be $\ge 0$. |
| `notes` | string | No | Optional observation remarks or physical conditions noted. |

#### Response: `201 Created`
```json
{
  "id": "22222222-2222-2222-2222-222222222222",
  "opname_id": "11111111-1111-1111-1111-111111111111",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "product_id": "b0000000-0000-0000-0000-000000000001",
  "location_id": "l0000000-0000-0000-0000-000000000001",
  "system_qty": "100.0000",
  "physical_qty": "95.0000",
  "discrepancy_qty": "-5.0000",
  "notes": "5 units damaged/missing",
  "created_at": "2026-09-06T08:30:00Z"
}
```

#### Status Codes
- `201 Created`: Physical count item recorded with computed `system_qty` and `discrepancy_qty`.
- `400 Bad Request`: Malformed JSON, missing fields, negative physical quantity (`invalid input`), or opname already sealed (`invalid stock opname status transition`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks write access to warehouse (`unauthorized warehouse access`), `auditor` role (`forbidden`), or location belongs to another warehouse (`unauthorized warehouse access`).
- `404 Not Found`: Opname or location not found (`stock opname not found`, `location not found`).
- `413 Request Entity Too Large`: Body exceeds 1 MB limit (`request entity too large`).
- `500 Internal Server Error`: Database query or insert failure (`internal server error`).

---

### 20. POST /api/v1/wms/opnames/{id}/complete

Finalizes and reconciles a stock opname session. Posts double-entry stock ledger movements to or from system virtual location `@LOSS` for every item with a non-zero discrepancy, stamps `approved_by` with the caller ID, and sets opname status to `COMPLETED`.

- **URL:** `/api/v1/wms/opnames/{id}/complete`
- **Method:** `POST`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Stock opname ID.
- **RBAC Requirement:** Caller must have write access to opname's `warehouse_id`. `auditor` role is rejected (`403 Forbidden`).
- **Allowed Pre-Statuses:** `DRAFT`, `IN_PROGRESS`. Attempting to complete an already `COMPLETED` or `CANCELLED` opname returns `400 Bad Request` (`invalid stock opname status transition`).
- **Max Body Size:** 1 MB (`413` if exceeded).

#### Reconciliation Mechanics:
1. **Surplus Recovery (`discrepancy_qty > 0`)**: Physical items exceed system count. Generates double-entry movement from `@LOSS` to `item.location_id` with `quantity = discrepancy_qty`.
2. **Deficit Write-Off (`discrepancy_qty < 0`)**: Physical items are less than system count. Generates double-entry movement from `item.location_id` to `@LOSS` with `quantity = abs(discrepancy_qty)`.
3. **Zero Discrepancy (`discrepancy_qty == 0`)**: No movement is posted, preserving ledger cleanliness.
4. **All Movements**: Stamped with `status = 'DONE'`, `reference_type = 'OPNAME'`, `reference_id = opname.id`, `executed_by = caller_id`.

#### Response: `200 OK`
```json
{
  "id": "11111111-1111-1111-1111-111111111111",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "opname_number": "OPN-2026-0001",
  "status": "COMPLETED",
  "conducted_by": "u0000000-0000-0000-0000-000000000001",
  "approved_by": "u0000000-0000-0000-0000-000000000002",
  "notes": "Q3 Routine Cycle Count - Central Hub",
  "created_at": "2026-09-06T08:00:00Z",
  "updated_at": "2026-09-06T09:00:00Z"
}
```

#### Status Codes
- `200 OK`: Stock opname finalized; discrepancy movements posted to/from `@LOSS`; ledger reconciled.
- `400 Bad Request`: Invalid opname UUID or opname already finalized (`invalid stock opname status transition`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: Caller lacks write access to warehouse (`unauthorized warehouse access`) or has `auditor` role (`forbidden`).
- `404 Not Found`: Opname not found or belongs to another tenant (`stock opname not found`).
- `413 Request Entity Too Large`: Body exceeds 1 MB limit (`request entity too large`).
- `500 Internal Server Error`: Ledger transaction or opname status update failure (`internal server error`).

---

### 21. GET /api/v1/wms/scraps

Lists recorded damaged goods and quarantine scrap records. Results can be filtered to a specific warehouse.

- **URL:** `/api/v1/wms/scraps`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `warehouse_id` (UUID, optional): Filter scrap entries by warehouse ID. If provided, caller must have read access to the specified warehouse. If omitted, returns scraps across all warehouses accessible to the caller.
- **RBAC Requirement:**
  - `admin`, `owner`: View scrap records across all tenant warehouses.
  - `regional_manager`: View scrap records for warehouses in their assigned regional cluster.
  - `warehouse`: View scrap records only for assigned warehouses in `user_warehouses`.
  - `auditor`: Tenant-wide read-only view.

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "33333333-3333-3333-3333-333333333333",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "scrap_number": "SCRAP-2026-0001",
      "warehouse_id": "a0000000-0000-0000-0000-000000000001",
      "product_id": "b0000000-0000-0000-0000-000000000001",
      "source_location_id": "l0000000-0000-0000-0000-000000000001",
      "scrap_location_id": "s0000000-0000-0000-0000-000000000001",
      "quantity": "5.0000",
      "reason": "Water damage from roof leakage during monsoon",
      "reported_by": "u0000000-0000-0000-0000-000000000001",
      "created_at": "2026-09-06T11:00:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Scrap records listed successfully (returns empty array `[]` if none found).
- `400 Bad Request`: Invalid `warehouse_id` UUID syntax (`invalid warehouse_id uuid`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks read access to the requested warehouse (`unauthorized warehouse access`).
- `404 Not Found`: Warehouse specified in `warehouse_id` does not exist or belongs to another tenant (`warehouse not found`).
- `500 Internal Server Error`: Database query failure (`internal server error`).

---

### 22. POST /api/v1/wms/scraps

Quarantines and deducts damaged, broken, expired, or defective inventory from a warehouse storage bin. Executes an atomic double-entry deduction under a PostgreSQL transaction advisory lock to move stock to `@SCRAP` (or a dedicated physical scrap bay), and writes an audit log in `stock_scraps`.

- **URL:** `/api/v1/wms/scraps`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json`
- **RBAC Requirement:**
  - Write access to `warehouse_id`. `auditor` role is strictly rejected (`403 Forbidden`).
- **Max Body Size:** 1 MB (`413` if exceeded).
- **Concurrency & Advisory Lock:**
  - Acquires `pg_advisory_xact_lock(hashtext(tenant_id || source_location_id || product_id))`.
  - Re-evaluates available stock balance at `source_location_id` within the locked transaction.
  - If available stock $< \text{quantity}$, rolls back and returns `422 Unprocessable Entity` (`insufficient stock at location`).

#### Request Body
```json
{
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "product_id": "b0000000-0000-0000-0000-000000000001",
  "source_location_id": "l0000000-0000-0000-0000-000000000001",
  "scrap_location_id": "s0000000-0000-0000-0000-000000000001",
  "quantity": "5.0000",
  "reason": "Water damage from roof leakage during monsoon",
  "scrap_number": "SCRAP-2026-0001"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `warehouse_id` | UUID | **Yes** | Warehouse where damaged goods were identified. |
| `product_id` | UUID | **Yes** | Master product ID being quarantined. |
| `source_location_id` | UUID | **Yes** | Warehouse internal storage location where damaged items are currently stored. Must belong to `warehouse_id`. |
| `scrap_location_id` | UUID | No | Target quarantine location. If omitted or null, automatically defaults to tenant system virtual location `@SCRAP`. If a physical internal location is specified, it must belong to `warehouse_id`. |
| `quantity` | string / number | **Yes** | Positive quantity to deduct and quarantine ($> 0$). |
| `reason` | string | **Yes** | Non-empty explanation for damage/quarantine (e.g. broken during forklift handling, expired batch). |
| `scrap_number` | string | No | Custom scrap reference identifier. Defaults to `SCRAP-<timestamp_ms>` if omitted. |

#### Response: `201 Created`
```json
{
  "id": "33333333-3333-3333-3333-333333333333",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "scrap_number": "SCRAP-2026-0001",
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "product_id": "b0000000-0000-0000-0000-000000000001",
  "source_location_id": "l0000000-0000-0000-0000-000000000001",
  "scrap_location_id": "s0000000-0000-0000-0000-000000000001",
  "quantity": "5.0000",
  "reason": "Water damage from roof leakage during monsoon",
  "reported_by": "u0000000-0000-0000-0000-000000000001",
  "created_at": "2026-09-06T11:00:00Z"
}
```

#### Status Codes
- `201 Created`: Damaged inventory atomically deducted and transferred to quarantine; scrap record created.
- `400 Bad Request`: Malformed JSON, missing required fields, empty reason, zero/negative quantity, identical source and destination locations (`invalid input`), or invalid UUID syntax.
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks write access to warehouse (`unauthorized warehouse access`), `auditor` role (`forbidden`), or source/scrap location belongs to another warehouse (`unauthorized warehouse access`).
- `404 Not Found`: Warehouse, product, source location, or custom scrap location not found (`warehouse not found`, `location not found`).
- `413 Request Entity Too Large`: Body exceeds 1 MB limit (`request entity too large`).
- `422 Unprocessable Entity`: Available stock at source bin is strictly less than requested scrap quantity (`insufficient stock at location`).
- `500 Internal Server Error`: Movement ledger transaction or scrap audit insert failure (`internal server error`).

---

### 23. POST /api/v1/wms/marketplace/import

Imports sales orders from marketplace channels (Shopee, Tokopedia, TikTok Shop, Lazada, Blibli, or generic formats) in batch. Automatically parses line items, resolves external SKUs against internal warehouse products with multiplier conversion, deducts warehouse inventory to system virtual location `@CUSTOMER` via double-entry `StockMovement`, and logs an import session.

- **URL:** `/api/v1/wms/marketplace/import`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json` **OR** `multipart/form-data` **OR** `text/csv`
- **RBAC Requirement:**
  - Write access to `warehouse_id`. Allowed roles: `admin`, `owner`, `regional_manager`, `warehouse`.
  - `auditor` role is strictly rejected (`403 Forbidden`).
- **Payload Limit:**
  - Bounded to **5 MB** (`http.MaxBytesReader`). Requests exceeding 5 MB return `413 Request Entity Too Large`.
- **Security & Data Sanitization:**
  - **Formula Injection Mitigation (CWE-1236):** Any string value beginning with `=`, `+`, or `@` is prefixed with `'` to neutralize spreadsheet execution.
  - **Locale-Aware Currency Parsing:** Automatically strips `Rp`, `rp`, `IDR`, and handles dot-thousands/comma-decimals or international number representations.
- **Idempotency Guarantees:**
  - Scoped by composite unique key `(tenant_id, channel, external_order_id)`.
  - Re-uploading an identical CSV safely skips duplicate orders without double-counting revenue, order lines, or deducting duplicate inventory. Duplicate orders increment `failed_orders` on the batch session.
- **Order Lifecycle & Stock Allocation:**
  - **Fully Mapped with Stock:** Transitions order to `COMPLETED` and immediately creates a `StockMovement` deducting $Q \times \text{Multiplier}$ from the warehouse primary location to `@CUSTOMER` (`reference_type = 'MARKETPLACE'`).
  - **Unmapped SKU:** If any line item cannot be resolved, order transitions to `UNMAPPED_SKU` and increments `unmapped_skus` on the batch. Inventory deduction is safely deferred until the SKU is mapped via `POST /api/v1/wms/marketplace/sku-mappings`.
  - **Insufficient Stock:** If warehouse available inventory is lower than required quantity, order transitions to `STOCK_INSUFFICIENT` without failing the rest of the batch.

#### Supported CSV Format Headers (Case-Insensitive)

The parser automatically detects header synonyms, trims UTF-8 BOM markers (`\ufeff`), and accepts underscores or spaces:

| Field | Supported Header Synonyms |
|---|---|
| **External Order ID** | `nomor pesanan`, `no. pesanan`, `no pesanan`, `order id`, `order sn`, `order_id`, `external order id`, `external_order_id` |
| **External SKU** | `nomor referensi sku`, `no. referensi sku`, `sku induk`, `sku`, `seller sku`, `item sku`, `product sku`, `external sku`, `external_sku` |
| **Item Name** | `nama produk`, `nama barang`, `product name`, `item name`, `product` |
| **Quantity** | `jumlah`, `quantity`, `qty`, `jumlah produk` |
| **Unit Price** | `harga awal`, `harga satuan`, `unit price`, `deal price`, `harga`, `price`, `unit_price` |
| **Subtotal** | `total harga produk`, `subtotal`, `total price`, `jumlah harga` |
| **Total Amount** | `total pembayaran`, `total amount`, `grand total`, `total pesanan`, `total` |
| **Shipping Fee** | `ongkos kirim dibayar pembeli`, `ongkir`, `shipping fee`, `biaya pengiriman`, `shipping_fee` |
| **Marketplace Fee** | `biaya layanan`, `biaya transaksi`, `marketplace fee`, `service fee`, `marketplace_fee` |
| **Customer Name** | `nama pembeli`, `customer name`, `username (pembeli)`, `nama penerima`, `customer` |
| **Customer Phone** | `nomor telepon pembeli`, `no. telepon`, `no telepon`, `phone number`, `phone`, `telepon` |
| **Shipping Address** | `alamat pengiriman`, `shipping address`, `alamat penerima`, `alamat`, `address` |
| **Courier** | `opsi pengiriman`, `kurir`, `shipping option`, `jasa kirim`, `courier` |
| **Tracking Number** | `no. resi`, `no resi`, `tracking number`, `nomor pelacakan`, `tracking`, `airway bill` |

---

#### Ingestion Mode 1: JSON Payload (`Content-Type: application/json`)

##### Request Body (Structured JSON)
```json
{
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "channel": "SHOPEE",
  "file_name": "shopee_orders_20260909.json",
  "orders": [
    {
      "external_order_id": "240909SHP001",
      "order_date": "2026-09-09T10:15:30Z",
      "customer_name": "Budi Santoso",
      "customer_phone": "081234567890",
      "shipping_address": "Jl. Sudirman No. 45, Jakarta Pusat",
      "courier": "J&T Express",
      "tracking_number": "JT123456789ID",
      "total_amount": "50000.0000",
      "shipping_fee": "10000.0000",
      "marketplace_fee": "2500.0000",
      "net_amount": "57500.0000",
      "items": [
        {
          "external_sku": "KOPISUSU-BOTOL-250ML",
          "item_name": "Kopi Susu Gula Aren 250ml",
          "quantity": "2.0000",
          "unit_price": "25000.0000",
          "subtotal": "50000.0000"
        }
      ]
    }
  ]
}
```

##### Request Body (Inline CSV Data String)
```json
{
  "warehouse_id": "a0000000-0000-0000-0000-000000000001",
  "channel": "TOKOPEDIA",
  "file_name": "tokopedia_export.csv",
  "csv_data": "Nomor Pesanan,Nomor Referensi SKU,Nama Produk,Jumlah,Harga Satuan,Total Pembayaran\nTKP-2026-001,KOPISUSU-DUS,Kopi Susu Dus (24 pcs),2,120000,240000\n"
}
```

---

#### Ingestion Mode 2: Multipart Form (`Content-Type: multipart/form-data`)

- `warehouse_id` (string/UUID, form-field): Destination warehouse ID.
- `channel` (string, form-field): `SHOPEE`, `TOKOPEDIA`, `TIKTOK`, `LAZADA`, `BLIBLI`, `OTHER`.
- `file` (binary, file-field): CSV / TSV file (max 5 MB).

---

#### Ingestion Mode 3: Raw CSV Stream (`Content-Type: text/csv`)

- **Query Parameters:**
  - `warehouse_id` (UUID, required): `?warehouse_id=a0000000-0000-0000-0000-000000000001`
  - `channel` (string, required): `&channel=SHOPEE`
  - `file_name` (string, optional): `&file_name=shopee_batch_01.csv` (defaults to `import.csv`)
- **Body:** Raw CSV text bytes directly in the HTTP request payload.

---

#### Response: `201 Created`
```json
{
  "data": {
    "batch": {
      "id": "e0000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "batch_number": "BATCH-MKT-SHOPEE-1725876930000000",
      "channel": "SHOPEE",
      "warehouse_id": "a0000000-0000-0000-0000-000000000001",
      "file_name": "shopee_orders_20260909.json",
      "total_orders": 1,
      "processed_orders": 1,
      "failed_orders": 0,
      "unmapped_skus": 0,
      "status": "COMPLETED",
      "uploaded_by": "u0000000-0000-0000-0000-000000000001",
      "created_at": "2026-09-09T10:15:30Z"
    },
    "orders": [
      {
        "id": "f0000000-0000-0000-0000-000000000001",
        "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        "batch_id": "e0000000-0000-0000-0000-000000000001",
        "warehouse_id": "a0000000-0000-0000-0000-000000000001",
        "channel": "SHOPEE",
        "external_order_id": "240909SHP001",
        "order_date": "2026-09-09T10:15:30Z",
        "customer_name": "Budi Santoso",
        "customer_phone": "081234567890",
        "shipping_address": "Jl. Sudirman No. 45, Jakarta Pusat",
        "courier": "J&T Express",
        "tracking_number": "JT123456789ID",
        "total_amount": "50000.0000",
        "shipping_fee": "10000.0000",
        "marketplace_fee": "2500.0000",
        "net_amount": "57500.0000",
        "status": "COMPLETED",
        "sales_order_id": null,
        "created_at": "2026-09-09T10:15:30Z",
        "items": [
          {
            "id": "11111111-1111-1111-1111-111111111111",
            "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
            "order_id": "f0000000-0000-0000-0000-000000000001",
            "external_sku": "KOPISUSU-BOTOL-250ML",
            "product_id": "b0000000-0000-0000-0000-000000000001",
            "item_name": "Kopi Susu Gula Aren 250ml",
            "quantity": "2.0000",
            "unit_price": "25000.0000",
            "subtotal": "50000.0000",
            "is_mapped": true
          }
        ]
      }
    ]
  }
}
```

#### Status Codes
- `201 Created`: Batch imported and processed successfully.
- `400 Bad Request`: Missing `warehouse_id`, malformed JSON, invalid multipart form, unparseable CSV data, or item quantity $\le 0$ (`invalid input`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks write access to warehouse (`unauthorized warehouse access`) or `auditor` role (`forbidden`).
- `404 Not Found`: Warehouse not found (`warehouse not found`).
- `409 Conflict`: Direct duplicate marketplace order submitted (`duplicate marketplace order`).
- `413 Request Entity Too Large`: File or request body exceeds 5 MB limit (`request entity too large`).
- `500 Internal Server Error`: Database insertion error or transaction failure (`internal server error`).

---

### 24. GET /api/v1/wms/marketplace/batches

Lists historical marketplace import batches and ingestion statistics for accessible warehouses.

- **URL:** `/api/v1/wms/marketplace/batches`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `warehouse_id` (UUID, optional): Filter import sessions by warehouse.
- **RBAC Requirement:**
  - Scoped to accessible warehouses based on role. `auditor` is allowed read access.

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "e0000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "batch_number": "BATCH-MKT-SHOPEE-1725876930000000",
      "channel": "SHOPEE",
      "warehouse_id": "a0000000-0000-0000-0000-000000000001",
      "file_name": "shopee_orders_20260909.json",
      "total_orders": 150,
      "processed_orders": 148,
      "failed_orders": 2,
      "unmapped_skus": 3,
      "status": "COMPLETED",
      "uploaded_by": "u0000000-0000-0000-0000-000000000001",
      "created_at": "2026-09-09T10:15:30Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: Batches retrieved successfully (empty array `[]` if none).
- `400 Bad Request`: Invalid `warehouse_id` UUID parameter.
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks read access to specified warehouse (`unauthorized warehouse access`).
- `500 Internal Server Error`: Database query failure (`internal server error`).

---

### 25. GET /api/v1/wms/marketplace/orders

Lists normalized canonical marketplace orders with filtering by warehouse, batch session, and fulfillment status.

- **URL:** `/api/v1/wms/marketplace/orders`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `warehouse_id` (UUID, optional): Filter by warehouse.
  - `batch_id` (UUID, optional): Filter by import batch ID.
  - `status` (string, optional): Filter by order status (`PENDING`, `PROCESSING`, `COMPLETED`, `FAILED`, `UNMAPPED_SKU`, `STOCK_INSUFFICIENT`).
- **RBAC Requirement:**
  - Scoped to accessible warehouses based on role. `auditor` is allowed read access.

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "f0000000-0000-0000-0000-000000000001",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "batch_id": "e0000000-0000-0000-0000-000000000001",
      "warehouse_id": "a0000000-0000-0000-0000-000000000001",
      "channel": "TOKOPEDIA",
      "external_order_id": "TKP-2026-001",
      "order_date": "2026-09-09T11:00:00Z",
      "customer_name": "Siti Nurhaliza",
      "customer_phone": "081987654321",
      "shipping_address": "Jl. Melati No. 12, Surabaya",
      "courier": "SiCepat REG",
      "tracking_number": "004012345678",
      "total_amount": "240000.0000",
      "shipping_fee": "15000.0000",
      "marketplace_fee": "6000.0000",
      "net_amount": "249000.0000",
      "status": "COMPLETED",
      "sales_order_id": null,
      "created_at": "2026-09-09T11:00:00Z",
      "items": [
        {
          "id": "22222222-2222-2222-2222-222222222222",
          "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
          "order_id": "f0000000-0000-0000-0000-000000000001",
          "external_sku": "KOPISUSU-DUS",
          "product_id": "b0000000-0000-0000-0000-000000000001",
          "item_name": "Kopi Susu Dus (24 pcs)",
          "quantity": "2.0000",
          "unit_price": "120000.0000",
          "subtotal": "240000.0000",
          "is_mapped": true
        }
      ]
    }
  ]
}
```

#### Status Codes
- `200 OK`: Orders retrieved successfully (empty array `[]` if none match).
- `400 Bad Request`: Invalid UUID syntax in query parameters.
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks read access to target warehouse (`unauthorized warehouse access`).
- `500 Internal Server Error`: Database query failure (`internal server error`).

---

### 26. GET /api/v1/wms/marketplace/orders/{id}

Retrieves single marketplace order details, including all line item mappings and fulfillment status.

- **URL:** `/api/v1/wms/marketplace/orders/{id}`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Path Parameters:**
  - `id` (UUID, required): Marketplace order unique identifier.
- **RBAC Requirement:**
  - User must have read access to the order's `warehouse_id`.

#### Response: `200 OK`
```json
{
  "data": {
    "id": "f0000000-0000-0000-0000-000000000001",
    "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "batch_id": "e0000000-0000-0000-0000-000000000001",
    "warehouse_id": "a0000000-0000-0000-0000-000000000001",
    "channel": "TIKTOK",
    "external_order_id": "TT-ORDER-888",
    "order_date": "2026-09-09T12:00:00Z",
    "customer_name": "Rian Pratama",
    "customer_phone": "081311223344",
    "shipping_address": "Jl. Asia Afrika No. 8, Bandung",
    "courier": "J&T Cargo",
    "tracking_number": "JTC987654321",
    "total_amount": "75000.0000",
    "shipping_fee": "0.0000",
    "marketplace_fee": "3750.0000",
    "net_amount": "71250.0000",
    "status": "UNMAPPED_SKU",
    "sales_order_id": null,
    "created_at": "2026-09-09T12:00:00Z",
    "items": [
      {
        "id": "33333333-3333-3333-3333-333333333333",
        "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
        "order_id": "f0000000-0000-0000-0000-000000000001",
        "external_sku": "NEW-TIKTOK-VIRAL-SKU",
        "product_id": null,
        "item_name": "TikTok Viral Item 250ml",
        "quantity": "2.0000",
        "unit_price": "37500.0000",
        "subtotal": "75000.0000",
        "is_mapped": false
      }
    ]
  }
}
```

#### Status Codes
- `200 OK`: Marketplace order retrieved successfully.
- `400 Bad Request`: Invalid UUID syntax in path parameter (`invalid order id uuid`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: User lacks read access to order's warehouse (`unauthorized warehouse access`).
- `404 Not Found`: Order not found or belongs to another tenant (`marketplace order not found`).
- `500 Internal Server Error`: Database query failure (`internal server error`).

---

### 27. POST /api/v1/wms/marketplace/sku-mappings

Creates or updates a channel SKU mapping linking an external marketplace SKU to an internal warehouse product. Implements **frictionless in-place resolution**: when a mapping is saved, the engine automatically backfills previously unmapped order items in `marketplace_order_items`, discovers affected `UNMAPPED_SKU` orders, and completes stock deductions to `@CUSTOMER` if all order lines are resolved.

- **URL:** `/api/v1/wms/marketplace/sku-mappings`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <jwt_token>`
  - `Content-Type: application/json`
- **RBAC Requirement:**
  - Allowed roles: `admin`, `owner`, `regional_manager`, `warehouse`.
  - `auditor` role is strictly rejected (`403 Forbidden`).
- **Payload Limit:**
  - Bounded to **5 MB** (`http.MaxBytesReader`). Requests exceeding 5 MB return `413 Request Entity Too Large`.

#### Request Body
```json
{
  "product_id": "b0000000-0000-0000-0000-000000000001",
  "mapping_type": "MARKETPLACE",
  "channel_name": "TOKOPEDIA",
  "external_sku": "KOPISUSU-DUS",
  "external_name": "Kopi Susu Dus (24 pcs)",
  "multiplier": "24.0000"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `product_id` | UUID | **Yes** | Internal master product ID in `products` table. |
| `mapping_type` | string | No | Mapping type: `MARKETPLACE` (default), `CUSTOMER`, or `VENDOR`. |
| `channel_name` | string | **Yes** | Marketplace channel name (`SHOPEE`, `TOKOPEDIA`, `TIKTOK`, `LAZADA`, `BLIBLI`, `OTHER`). |
| `external_sku` | string | **Yes** | Non-empty seller SKU as listed on the marketplace channel. |
| `external_name` | string | No | Optional human-readable name/title on marketplace. |
| `multiplier` | string / number | No | Quantity conversion multiplier ($> 0$). Defaults to `1.0000` if omitted or $\le 0$. E.g., `24` indicates 1 bundle/carton contains 24 internal pieces. |

#### Automatic In-Place Reprocessing Flow
1. **Upsert Mapping:** Stores record in `product_sku_mappings` with unique constraint `(tenant_id, channel_name, external_sku)`.
2. **Backfill Items:** Executes `UPDATE marketplace_order_items SET product_id = $1, is_mapped = TRUE WHERE external_sku = $2 AND channel = $3`.
3. **Reprocess Pending Orders:** Queries all orders in `UNMAPPED_SKU` status containing `external_sku`.
4. **Trigger Stock Deduction:** If all line items in an order are now mapped, executes `DeductLocationStock` for each item ($Q \times \text{Multiplier}$) to virtual `@CUSTOMER` and updates order status to `COMPLETED`. If stock is insufficient, updates status to `STOCK_INSUFFICIENT`.

#### Response: `201 Created`
```json
{
  "data": {
    "id": "44444444-4444-4444-4444-444444444444",
    "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "product_id": "b0000000-0000-0000-0000-000000000001",
    "mapping_type": "MARKETPLACE",
    "channel_name": "TOKOPEDIA",
    "external_sku": "KOPISUSU-DUS",
    "external_name": "Kopi Susu Dus (24 pcs)",
    "multiplier": "24.0000",
    "created_at": "2026-09-09T13:00:00Z",
    "updated_at": "2026-09-09T13:00:00Z"
  }
}
```

#### Status Codes
- `201 Created`: SKU mapping created/updated, order items linked, and pending unmapped orders reprocessed.
- `400 Bad Request`: Missing `product_id`, `channel_name`, or `external_sku`; malformed JSON; invalid UUID syntax (`invalid input`).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `403 Forbidden`: `auditor` role or unauthorized role attempting to create mappings (`forbidden`).
- `404 Not Found`: Internal product not found in tenant catalog (`product not found`).
- `413 Request Entity Too Large`: Request body exceeds 5 MB (`request entity too large`).
- `500 Internal Server Error`: Database upsert or inventory movement error (`internal server error`).

---

### 28. GET /api/v1/wms/marketplace/sku-mappings

Lists external channel SKU mappings configured for the authenticated tenant.

- **URL:** `/api/v1/wms/marketplace/sku-mappings`
- **Method:** `GET`
- **Headers:** `Authorization: Bearer <jwt_token>`
- **Query Parameters:**
  - `channel_name` (string, optional): Filter mappings by marketplace channel (`SHOPEE`, `TOKOPEDIA`, `TIKTOK`, `LAZADA`, `BLIBLI`, `OTHER`). Accepts `channel` as an alias.
- **RBAC Requirement:**
  - Any authenticated role in the tenant (`admin`, `owner`, `regional_manager`, `warehouse`, `auditor`).

#### Response: `200 OK`
```json
{
  "data": [
    {
      "id": "44444444-4444-4444-4444-444444444444",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "product_id": "b0000000-0000-0000-0000-000000000001",
      "mapping_type": "MARKETPLACE",
      "channel_name": "TOKOPEDIA",
      "external_sku": "KOPISUSU-DUS",
      "external_name": "Kopi Susu Dus (24 pcs)",
      "multiplier": "24.0000",
      "created_at": "2026-09-09T13:00:00Z",
      "updated_at": "2026-09-09T13:00:00Z"
    }
  ]
}
```

#### Status Codes
- `200 OK`: SKU mappings retrieved successfully (empty array `[]` if none).
- `401 Unauthorized`: Missing or invalid JWT token (`missing tenant context`).
- `500 Internal Server Error`: Database query failure (`internal server error`).

---

## Error Responses Catalog

Tayooli ERP WMS uses standard HTTP status envelopes formatted as:

```json
{
  "error": {
    "code": "<Status Text>",
    "message": "<Human-readable error description>",
    "details": null
  }
}
```

### Complete Status Codes Reference

| HTTP Code | Error Code Text | Common Trigger Scenarios |
|---|---|---|
| **`200 OK`** | *(Success)* | Read operations, status queries, transfer/DO dispatches, stock opname completion (`POST /api/v1/wms/opnames/{id}/complete`), marketplace batch/order/mapping retrieval. |
| **`201 Created`** | *(Success)* | New warehouse, location, transfer, delivery order, stock opname session, opname count line item, quarantine scrap entry, marketplace order batch import, or channel SKU mapping created. |
| **`400 Bad Request`** | `Bad Request` | Malformed JSON; missing required fields (`code`, `name`, `items`, `warehouse_id`, `product_id`, `reason`, `external_sku`, `channel_name`); zero or negative quantities; identical source and scrap quarantine locations; invalid status transition (e.g., dispatching `PENDING_APPROVAL` transfer, `CANCELLED` delivery order, or adding items / completing an already `COMPLETED`/`CANCELLED` stock opname: `invalid stock opname status transition`); invalid UUID format in path or query parameters. |
| **`401 Unauthorized`** | `Unauthorized` | Missing `Authorization: Bearer` header; expired or tampered token; token missing tenant context (`missing tenant context`). |
| **`403 Forbidden`** | `Forbidden` | User lacks permission for warehouse (`unauthorized warehouse access`); `auditor` role attempting any write or state-changing action (`forbidden`); location spoofing across warehouses. |
| **`404 Not Found`** | `Not Found` | Entity does not exist or belongs to another tenant: `warehouse not found`, `location not found`, `barcode or external sku not found`, `stock transfer not found`, `delivery order not found`, `stock opname not found`, `stock scrap not found`, `marketplace import batch not found`, `marketplace order not found`, `sku mapping not found`. |
| **`409 Conflict`** | `Conflict` | Duplicate entity detected: `duplicate marketplace order` (external order ID already imported for this channel and tenant). |
| **`413 Request Entity Too Large`** | `Request Entity Too Large` | HTTP request payload exceeds configured size boundary: 1 MB for standard WMS requests, 5 MB for marketplace CSV/JSON imports and SKU mappings (`request entity too large`). |
| **`422 Unprocessable Entity`** | `Unprocessable Entity` | Business logic violation: available stock at bin is strictly less than requested dispatch or scrap quantity (`insufficient stock at location`). |
| **`500 Internal Server Error`** | `Internal Server Error` | Database connection lost, lock acquisition failure, or unexpected backend error (`internal server error`). |
