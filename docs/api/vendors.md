# Vendor Management API

This document covers all Vendor Management endpoints. Vendors are the master
data records that represent suppliers, service providers, and other external
parties transacted with. They are referenced by invoices, purchase orders, and
goods receipts throughout the Order-to-Pay pipeline.

## Table of Contents

1. [Authentication](#authentication)
2. [RBAC Role Matrix](#rbac-role-matrix)
3. [Vendor Object](#vendor-object)
4. [GET /api/v1/vendors](#get-apiv1vendors)
5. [GET /api/v1/vendors/{id}](#get-apiv1vendorsid)
6. [POST /api/v1/vendors](#post-apiv1vendors)
7. [PUT /api/v1/vendors/{id}](#put-apiv1vendorsid)
8. [DELETE /api/v1/vendors/{id}](#delete-apiv1vendorsid)
9. [POST /api/v1/vendors/{id}/rate](#post-apiv1vendorsidrate)
10. [Kafka Events](#kafka-events)

---

## Authentication

All `/api/v1/vendors` endpoints require a `Bearer` JWT in the `Authorization`
header. `TenantMiddleware` validates the token (HS256, expiry enforced) and
extracts two claims that drive every downstream operation:

| Claim | Type | Purpose |
|---|---|---|
| `tenant_id` | UUID string | Scopes every DB query to a single tenant |
| `role` | string | Checked by `RequireRole` on gated routes |

`tenant_id` is **never** accepted from the request body or query string.

**Common auth errors**

| Status | Body | When |
|---|---|---|
| `401` | `{"error": "missing authorization header"}` | No `Authorization` header or wrong prefix |
| `401` | `{"error": "invalid token"}` | Token expired, signature invalid, or non-HS256 algorithm |
| `401` | `{"error": "invalid token claims"}` | Claims cannot be parsed |
| `401` | `{"error": "missing tenant_id in token"}` | `tenant_id` claim absent |

---

## RBAC Role Matrix

All `GET` (read) routes are open to every authenticated user within a tenant.
Write routes are gated by `RequireRole` middleware. The `admin` role has access
to every write route.

| Route | Method | Required Roles | Purpose |
|---|---|---|---|
| `/api/v1/vendors` | GET | any | List vendors (paginated, searchable) |
| `/api/v1/vendors/{id}` | GET | any | Get vendor detail |
| `/api/v1/vendors` | POST | `admin`, `accountant` | Create vendor |
| `/api/v1/vendors/{id}` | PUT | `admin`, `accountant` | Update vendor |
| `/api/v1/vendors/{id}` | DELETE | `admin` | Soft-delete vendor |
| `/api/v1/vendors/{id}/rate` | POST | any | Rate vendor (1-5) |

A request with a role not listed for a write route receives `403 Forbidden`
with body `{"error": "forbidden"}`.

---

## Vendor Object

All endpoints that return a vendor return this shape.

```json
{
  "id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "PT Acme Supplies Indonesia",
  "code": "VENDOR-ACME-001",
  "email": "ap@acme-supplies.co.id",
  "phone": "+62-21-555-0123",
  "address": "Jl. Sudirman No. 123, Jakarta Selatan 12190",
  "tax_id": "01.234.567.8-012.000",
  "bank_account": "1234567890",
  "bank_name": "Bank Central Asia",
  "rating": "4.25",
  "rating_count": 8,
  "is_active": true,
  "created_at": "2026-06-15T08:00:00Z",
  "updated_at": "2026-07-10T14:30:00Z"
}
```

**Field notes**

- `rating` is always a decimal string (e.g. `"4.25"`), not a JSON number. It is
  the running average of all ratings submitted via the rate endpoint. Range is
  `[1.00, 5.00]`.
- `rating_count` is the total number of ratings submitted. When `rating_count`
  is `0`, `rating` is `"0.00"`.
- `is_active` is `false` for soft-deleted vendors. Soft-deleted vendors are
  excluded from list results by default.
- `bank_account` and `bank_name` are **omitted** from the response when their
  value is `NULL` in the database.
- `tax_id` is **omitted** from the response when its value is `NULL` in the
  database.
- `tenant_id` is always the tenant from the JWT -- never a user-supplied value.
- `code` is unique per tenant and case-insensitive. It is normalized to
  uppercase on creation.

---

## GET /api/v1/vendors

List all active vendors for the current tenant. Supports pagination and
full-text search on name and code.

**Auth:** Bearer JWT -- any authenticated tenant user (no role restriction)

### Query parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `q` | string | *(none)* | Full-text search on `name` and `code` (case-insensitive, ILIKE) |
| `page` | integer | `1` | Page number (1-based) |
| `page_size` | integer | `20` | Items per page (max `100`) |

### Example

```
GET /api/v1/vendors?q=acme&page=1&page_size=10
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoic3RhZmYiLCJleHAiOjE3ODI3MjAwMDB9.sig
```

### Responses

**200 OK** -- Paginated vendor list.

```json
{
  "data": [
    {
      "id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
      "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
      "name": "PT Acme Supplies Indonesia",
      "code": "VENDOR-ACME-001",
      "email": "ap@acme-supplies.co.id",
      "phone": "+62-21-555-0123",
      "address": "Jl. Sudirman No. 123, Jakarta Selatan 12190",
      "rating": "4.25",
      "rating_count": 8,
      "is_active": true,
      "created_at": "2026-06-15T08:00:00Z",
      "updated_at": "2026-07-10T14:30:00Z"
    }
  ],
  "page": 1,
  "page_size": 10,
  "total": 1
}
```

When no vendors match, `data` is `[]` and `total` is `0`.

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid page"}` | `page` is not a positive integer |
| `400` | `{"error": "invalid page_size"}` | `page_size` is not between 1 and 100 |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## GET /api/v1/vendors/{id}

Get a single vendor by ID. The vendor must belong to the caller's tenant.
Includes soft-deleted vendors (so that referenced vendors remain viewable in
historical records like invoices).

**Auth:** Bearer JWT -- any authenticated tenant user (no role restriction)

### Path parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | The vendor ID |

### Example

```
GET /api/v1/vendors/d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoic3RhZmYiLCJleHAiOjE3ODI3MjAwMDB9.sig
```

### Responses

**200 OK** -- Single vendor object.

```json
{
  "id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "PT Acme Supplies Indonesia",
  "code": "VENDOR-ACME-001",
  "email": "ap@acme-supplies.co.id",
  "phone": "+62-21-555-0123",
  "address": "Jl. Sudirman No. 123, Jakarta Selatan 12190",
  "tax_id": "01.234.567.8-012.000",
  "bank_account": "1234567890",
  "bank_name": "Bank Central Asia",
  "rating": "4.25",
  "rating_count": 8,
  "is_active": true,
  "created_at": "2026-06-15T08:00:00Z",
  "updated_at": "2026-07-10T14:30:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid vendor id"}` | `id` path segment is not a valid UUID |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `404` | `{"error": "vendor not found"}` | No vendor with this ID exists in the caller's tenant |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## POST /api/v1/vendors

Create a new vendor. The record is inserted with `is_active: true` and a
running `rating` of `"0.00"`. On success, a `vendor.created` event is
published to Kafka.

**Auth:** Bearer JWT with `role` claim `admin` or `accountant`

### Request body

`Content-Type: application/json` -- max 64 KiB.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Vendor legal or trading name (max 255 chars) |
| `code` | string | yes | Unique vendor code, e.g. `"VENDOR-ACME-001"`. Normalized to uppercase. |
| `email` | string | yes | Primary contact email (valid email format) |
| `phone` | string | no | Phone number with country code |
| `address` | string | no | Full mailing address (max 1000 chars) |
| `tax_id` | string | no | Tax identification number (NPWP or equivalent) |
| `bank_account` | string | no | Bank account number for payments |
| `bank_name` | string | no | Bank name for payments |

`tenant_id` is derived from the JWT; do not include it in the body.

### Example

```
POST /api/v1/vendors
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoiYWNjb3VudGFudCIsImV4cCI6MTc4MjcyMDAwMH0.sig
Content-Type: application/json

{
  "name": "PT Acme Supplies Indonesia",
  "code": "VENDOR-ACME-001",
  "email": "ap@acme-supplies.co.id",
  "phone": "+62-21-555-0123",
  "address": "Jl. Sudirman No. 123, Jakarta Selatan 12190",
  "tax_id": "01.234.567.8-012.000",
  "bank_account": "1234567890",
  "bank_name": "Bank Central Asia"
}
```

### Responses

**201 Created** -- Vendor created. Body is the vendor object with `is_active`
set to `true`, `rating` to `"0.00"`, and `rating_count` to `0`.

```json
{
  "id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "PT Acme Supplies Indonesia",
  "code": "VENDOR-ACME-001",
  "email": "ap@acme-supplies.co.id",
  "phone": "+62-21-555-0123",
  "address": "Jl. Sudirman No. 123, Jakarta Selatan 12190",
  "tax_id": "01.234.567.8-012.000",
  "bank_account": "1234567890",
  "bank_name": "Bank Central Asia",
  "rating": "0.00",
  "rating_count": 0,
  "is_active": true,
  "created_at": "2026-07-11T10:00:00Z",
  "updated_at": "2026-07-11T10:00:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid request body"}` | Malformed JSON or body unreadable |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `403` | `{"error": "forbidden"}` | JWT `role` is not `admin` or `accountant` |
| `409` | `{"error": "vendor code already exists"}` | A vendor with the same `code` already exists in this tenant |
| `413` | `{"error": "request body too large"}` | Body exceeds 64 KiB |
| `422` | `{"error": "invalid vendor data"}` | Business rule violation (e.g. blank name, blank code, invalid email) |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## PUT /api/v1/vendors/{id}

Update an existing vendor. Only provided fields are updated; omitted fields
retain their current values. The `code` field cannot be changed after creation.

On success, a `vendor.updated` event is published to Kafka.

**Auth:** Bearer JWT with `role` claim `admin` or `accountant`

### Path parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | The vendor ID |

### Request body

`Content-Type: application/json` -- max 64 KiB.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | no | Vendor legal or trading name (max 255 chars) |
| `email` | string | no | Primary contact email (valid email format) |
| `phone` | string | no | Phone number with country code |
| `address` | string | no | Full mailing address (max 1000 chars) |
| `tax_id` | string | no | Tax identification number (NPWP or equivalent) |
| `bank_account` | string | no | Bank account number for payments |
| `bank_name` | string | no | Bank name for payments |

`code` and `tenant_id` are immutable and must not be included in the request
body. Including `code` is silently ignored.

### Example

```
PUT /api/v1/vendors/d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoiYWNjb3VudGFudCIsImV4cCI6MTc4MjcyMDAwMH0.sig
Content-Type: application/json

{
  "name": "PT Acme Supplies Indonesia (Updated)",
  "bank_account": "9876543210",
  "bank_name": "Bank Mandiri"
}
```

### Responses

**200 OK** -- Vendor updated. Body is the full vendor object after the update.

```json
{
  "id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "PT Acme Supplies Indonesia (Updated)",
  "code": "VENDOR-ACME-001",
  "email": "ap@acme-supplies.co.id",
  "phone": "+62-21-555-0123",
  "address": "Jl. Sudirman No. 123, Jakarta Selatan 12190",
  "tax_id": "01.234.567.8-012.000",
  "bank_account": "9876543210",
  "bank_name": "Bank Mandiri",
  "rating": "4.25",
  "rating_count": 8,
  "is_active": true,
  "created_at": "2026-06-15T08:00:00Z",
  "updated_at": "2026-07-11T11:00:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid request body"}` | Malformed JSON or body unreadable |
| `400` | `{"error": "invalid vendor id"}` | `id` path segment is not a valid UUID |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `403` | `{"error": "forbidden"}` | JWT `role` is not `admin` or `accountant` |
| `404` | `{"error": "vendor not found"}` | No vendor with this ID exists in the caller's tenant |
| `413` | `{"error": "request body too large"}` | Body exceeds 64 KiB |
| `422` | `{"error": "invalid vendor data"}` | Business rule violation (e.g. blank name, invalid email) |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## DELETE /api/v1/vendors/{id}

Soft-delete a vendor. Sets `is_active` to `false` rather than removing the
row. Soft-deleted vendors are excluded from list results but remain accessible
by ID (so that historical invoices and POs can still resolve the vendor
reference).

A vendor cannot be soft-deleted if it has open (non-terminal) purchase orders
or pending invoices linked to it. The request returns `409 Conflict` in that
case.

On success, a `vendor.deleted` event is published to Kafka.

**Auth:** Bearer JWT with `role` claim `admin` only

### Path parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | The vendor ID |

### Request body

None.

### Example

```
DELETE /api/v1/vendors/d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoiYWRtaW4iLCJleHAiOjE3ODI3MjAwMDB9.sig
```

### Responses

**200 OK** -- Vendor soft-deleted. Body confirms the updated state.

```json
{
  "id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "is_active": false,
  "updated_at": "2026-07-11T12:00:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid vendor id"}` | `id` path segment is not a valid UUID |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `403` | `{"error": "forbidden"}` | JWT `role` is not `admin` |
| `404` | `{"error": "vendor not found"}` | No vendor with this ID exists in the caller's tenant |
| `409` | `{"error": "vendor has open purchase orders or pending invoices"}` | Vendor has linked open POs or pending invoices |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## POST /api/v1/vendors/{id}/rate

Submit a rating for a vendor. Any authenticated tenant user can rate. The
rating is a whole number from 1 to 5 inclusive. The vendor's running average
`rating` and `rating_count` are updated atomically.

On success, a `vendor.rated` event is published to Kafka.

**Auth:** Bearer JWT -- any authenticated tenant user (no role restriction)

### Path parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | The vendor ID |

### Request body

`Content-Type: application/json` -- max 1 KiB.

| Field | Type | Required | Description |
|---|---|---|---|
| `score` | integer | yes | Rating score from 1 to 5 inclusive |

### Example

```
POST /api/v1/vendors/d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef/rate
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoic3RhZmYiLCJleHAiOjE3ODI3MjAwMDB9.sig
Content-Type: application/json

{
  "score": 4
}
```

### Responses

**200 OK** -- Rating recorded. Returns the updated rating summary.

```json
{
  "id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "rating": "4.30",
  "rating_count": 10
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid request body"}` | Malformed JSON or body unreadable |
| `400` | `{"error": "score must be between 1 and 5"}` | `score` is outside the 1-5 range or not an integer |
| `400` | `{"error": "invalid vendor id"}` | `id` path segment is not a valid UUID |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `404` | `{"error": "vendor not found"}` | No vendor with this ID exists in the caller's tenant |
| `409` | `{"error": "vendor is not active"}` | Vendor has been soft-deleted |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## Kafka Events

Event publication is best-effort and non-fatal: if the Kafka broker is
unavailable, the database write still succeeds and the error is logged. The
database is the source of truth for all vendor states.

All events are keyed by vendor UUID for ordered per-entity consumption.

### Topic: `vendor.created`

Published by: `POST /api/v1/vendors`

```json
{
  "vendor_id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "PT Acme Supplies Indonesia",
  "code": "VENDOR-ACME-001",
  "email": "ap@acme-supplies.co.id"
}
```

### Topic: `vendor.updated`

Published by: `PUT /api/v1/vendors/{id}`

```json
{
  "vendor_id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "name": "PT Acme Supplies Indonesia (Updated)",
  "code": "VENDOR-ACME-001"
}
```

### Topic: `vendor.deleted`

Published by: `DELETE /api/v1/vendors/{id}`

```json
{
  "vendor_id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001"
}
```

### Topic: `vendor.rated`

Published by: `POST /api/v1/vendors/{id}/rate`

```json
{
  "vendor_id": "d4e5f6a7-8b9c-0d1e-2f3a-456789abcdef",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "score": 4,
  "new_rating": "4.30",
  "rating_count": 10
}
```
