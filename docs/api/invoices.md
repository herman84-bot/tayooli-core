# Order-to-Pay API

This document covers all Order-to-Pay endpoints: invoices, purchase orders, and
goods receipts. It defines the full invoice status lifecycle, the RBAC role
matrix for write routes, and every request/response shape.

## Table of Contents

1. [Authentication](#authentication)
2. [RBAC Role Matrix](#rbac-role-matrix)
3. [Invoice Object](#invoice-object)
4. [Invoice Status Lifecycle](#status-lifecycle)
5. [POST /api/v1/invoices](#post-apiv1invoices)
6. [POST /api/v1/invoices/{id}/approve](#post-apiv1invoicesidapprove)
7. [POST /api/v1/invoices/{id}/reject](#post-apiv1invoicesidreject)
8. [Kafka Events](#kafka-events)
9. [Purchase Orders API](#purchase-orders-api)
10. [Goods Receipts API](#goods-receipts-api)

---

## Authentication

All `/api/v1` endpoints require a `Bearer` JWT in the `Authorization` header.
`TenantMiddleware` validates the token (HS256, expiry enforced) and extracts
two claims that drive every downstream operation:

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
| `/api/v1/invoices` | POST | `admin`, `accountant` | Create invoice |
| `/api/v1/invoices/{id}/approve` | POST | `admin`, `approver` | Approve invoice |
| `/api/v1/invoices/{id}/reject` | POST | `admin`, `approver` | Reject invoice |
| `/api/v1/purchase-orders` | POST | `admin`, `purchaser` | Create purchase order |
| `/api/v1/goods-receipts` | POST | `admin`, `warehouse` | Create goods receipt |

**Role definitions**

| Role | Description |
|---|---|
| `admin` | Full access to every write route across all modules |
| `accountant` | Can create invoices |
| `approver` | Can approve and reject invoices |
| `purchaser` | Can create purchase orders |
| `warehouse` | Can create goods receipts |

A request with a role not listed for a write route receives `403 Forbidden`
with body `{"error": "forbidden"}`.

---

## Invoice Object

All endpoints that return an invoice return this shape.

```json
{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "vendor_id": "VENDOR-ACME-001",
  "invoice_number": "INV-2026-0042",
  "amount": "84500.0000",
  "currency": "IDR",
  "status": "pending_review",
  "ai_confidence_score": 0.91,
  "po_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "match_result": "amount_mismatch",
  "due_date": "2026-07-15T00:00:00Z",
  "created_at": "2026-06-28T10:30:00Z",
  "updated_at": "2026-06-28T11:05:00Z"
}
```

**Field notes**

- `amount` is always a decimal string (e.g. `"84500.0000"`), not a JSON number.
  Parse it with a decimal library. Casting to `float64` risks truncation on large
  values because the backing column is `NUMERIC(20,4)`.
- `po_id`, `match_result`, and `due_date` are **omitted** from the response when
  their value is `NULL` in the database.
- `ai_confidence_score` is `0` until the OCR pipeline sets it. Range is `[0.0, 1.0]`.
- `tenant_id` is always the tenant from the JWT -- never a user-supplied value.

### Status enum

| Value | Description |
|---|---|
| `pending` | Created, awaiting OCR pipeline |
| `ai_processed` | OCR succeeded; 3-way match engine has run or is running |
| `ai_failed` | OCR pipeline error; no confidence score or extracted text available |
| `pending_review` | 3-way match produced a mismatch; manual review required |
| `approved` | Invoice approved (auto or manual). Terminal state. |
| `rejected` | Invoice rejected by a human approver. Terminal state. |

---

## Status Lifecycle

```
                       Kafka: invoice.created
                              |
                     [Python OCR Worker]
                         |         |
                   ai_processed  ai_failed --------------------------------+
                         |                                                  |
               [Go 3-way Match Engine]                                     |
                   |           |                                           |
               approved   pending_review                                   |
            (auto-approve)     |                                           |
                               |                                           |
                  ------------+--------------------------------------------+
                  Manual approve (RequireRole: admin | approver)
                  Allowed source states: ai_processed, pending_review, ai_failed
                               |
                           approved  (terminal)

                  Manual reject (RequireRole: admin | approver)
                  Allowed source states: pending, pending_review
                               |
                           rejected  (terminal)
```

**Transition table**

| From | To | Trigger | Role required |
|---|---|---|---|
| `pending` | `ai_processed` | Kafka OCR result (status=ai_processed) | System |
| `pending` | `ai_failed` | Kafka OCR result (status=ai_failed) | System |
| `ai_processed` | `approved` | 3-way match passed | System |
| `ai_processed` | `pending_review` | 3-way match mismatch | System |
| `ai_processed` | `approved` | Manual approve | admin, approver |
| `pending_review` | `approved` | Manual approve | admin, approver |
| `ai_failed` | `approved` | Manual approve | admin, approver |
| `pending` | `rejected` | Manual reject | admin, approver |
| `pending_review` | `rejected` | Manual reject | admin, approver |

Note: `approved` and `rejected` are terminal. No further transitions are
permitted by any party.

---

## POST /api/v1/invoices

Create a new invoice. The record is inserted with `status: pending` and an
`invoice.created` event is published to Kafka, which triggers the Python OCR
pipeline asynchronously.

**Auth:** Bearer JWT with `role` claim `admin` or `accountant`

### Request body

`Content-Type: application/json` -- max 1 MB.

| Field | Type | Required | Description |
|---|---|---|---|
| `vendor_id` | string | yes | Identifies the vendor; must match your vendor master data |
| `invoice_number` | string | yes | Vendor-assigned invoice number, e.g. `"INV-2026-0042"` |
| `amount` | string | yes | Decimal string, e.g. `"84500.0000"`. Do not send as a JSON number. |
| `currency` | string | yes | ISO 4217 code, e.g. `"IDR"` or `"USD"` |
| `due_date` | string (RFC3339) | no | Payment due date, e.g. `"2026-07-15T00:00:00Z"` |

`tenant_id` is derived from the JWT; do not include it in the body.

### Example

```
POST /api/v1/invoices
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoiYWNjb3VudGFudCIsImV4cCI6MTc4MjcyMDAwMH0.sig
Content-Type: application/json

{
  "vendor_id": "VENDOR-ACME-001",
  "invoice_number": "INV-2026-0042",
  "amount": "84500.0000",
  "currency": "IDR",
  "due_date": "2026-07-15T00:00:00Z"
}
```

### Responses

**201 Created** -- Invoice created. Body is the invoice object with `"status": "pending"`.

```json
{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "vendor_id": "VENDOR-ACME-001",
  "invoice_number": "INV-2026-0042",
  "amount": "84500.0000",
  "currency": "IDR",
  "status": "pending",
  "ai_confidence_score": 0,
  "due_date": "2026-07-15T00:00:00Z",
  "created_at": "2026-06-28T10:30:00Z",
  "updated_at": "2026-06-28T10:30:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid request body"}` | Malformed JSON or body unreadable |
| `400` | `{"error": "amount must be a valid positive decimal string (e.g. \"1234.56\")"}` | `amount` is empty, non-numeric, zero, or negative |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `403` | `{"error": "forbidden"}` | JWT `role` is not `admin` or `accountant` |
| `422` | `{"error": "invalid invoice data"}` | Business rule violation (e.g. blank vendor) |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## POST /api/v1/invoices/{id}/approve

Manually approve an invoice. Allowed only when the invoice status is
`ai_processed`, `pending_review`, or `ai_failed`. On success, an
`invoice.approved` Kafka event is published.

Approving an `ai_failed` invoice bypasses document verification because no OCR
data is available. Callers with the `approver` or `admin` role accept that risk
explicitly by calling this endpoint.

**Auth:** Bearer JWT with `role` claim `admin` or `approver`

### Path parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | The invoice ID |

### Request body

None.

### Example

```
POST /api/v1/invoices/3fa85f64-5717-4562-b3fc-2c963f66afa6/approve
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoiYXBwcm92ZXIiLCJleHAiOjE3ODI3MjAwMDB9.sig
```

### Responses

**200 OK** -- Invoice approved. Body is the invoice object with `"status": "approved"`.

```json
{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "vendor_id": "VENDOR-ACME-001",
  "invoice_number": "INV-2026-0042",
  "amount": "84500.0000",
  "currency": "IDR",
  "status": "approved",
  "ai_confidence_score": 0.91,
  "po_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "match_result": "amount_mismatch",
  "due_date": "2026-07-15T00:00:00Z",
  "created_at": "2026-06-28T10:30:00Z",
  "updated_at": "2026-06-28T11:05:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid invoice id"}` | `id` path segment is not a valid UUID |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `403` | `{"error": "forbidden"}` | JWT `role` is not `admin` or `approver` |
| `404` | `{"error": "invoice not found"}` | No invoice with this ID exists in the caller's tenant |
| `409` | `{"error": "invoice is not in pending state"}` | Status is `pending`, `approved`, or `rejected` |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## POST /api/v1/invoices/{id}/reject

Manually reject an invoice. Allowed only when the invoice status is `pending`
or `pending_review`. On success, an `invoice.rejected` Kafka event is published.

`ai_processed` and `ai_failed` invoices cannot be rejected via this endpoint
because they have already left the initial review gate; use the approve endpoint
or contact support for corrections.

**Auth:** Bearer JWT with `role` claim `admin` or `approver`

### Path parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | The invoice ID |

### Request body

None.

### Example

```
POST /api/v1/invoices/3fa85f64-5717-4562-b3fc-2c963f66afa6/reject
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoiYWRtaW4iLCJleHAiOjE3ODI3MjAwMDB9.sig
```

### Responses

**200 OK** -- Invoice rejected. Body is the invoice object with `"status": "rejected"`.

```json
{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "vendor_id": "VENDOR-ACME-001",
  "invoice_number": "INV-2026-0042",
  "amount": "84500.0000",
  "currency": "IDR",
  "status": "rejected",
  "ai_confidence_score": 0,
  "due_date": "2026-07-15T00:00:00Z",
  "created_at": "2026-06-28T10:30:00Z",
  "updated_at": "2026-06-28T11:10:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid invoice id"}` | `id` path segment is not a valid UUID |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `403` | `{"error": "forbidden"}` | JWT `role` is not `admin` or `approver` |
| `404` | `{"error": "invoice not found"}` | No invoice with this ID exists in the caller's tenant |
| `409` | `{"error": "invoice cannot be rejected in its current state"}` | Status is `ai_processed`, `ai_failed`, `approved`, or `rejected` |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## Kafka Events

Event publication is best-effort and non-fatal: if the Kafka broker is
unavailable, the database write still succeeds and the error is logged. The
database is the source of truth for all invoice states.

All `amount` values in Kafka payloads are decimal strings. Python consumers must
parse them with `decimal.Decimal`, never cast to `float`.

### Topic: `invoice.created`

Published by: `POST /api/v1/invoices`
Consumed by: Python OCR worker

```json
{
  "invoice_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "amount": "84500.0000",
  "status": "pending",
  "raw_document_bytes": "<base64-encoded document, optional>",
  "mime_type": "application/pdf"
}
```

`raw_document_bytes` and `mime_type` are omitted when no document is attached.

### Topic: `invoice.approved`

Published by: `POST /api/v1/invoices/{id}/approve` and the 3-way match engine
(auto-approve path).

```json
{
  "invoice_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "amount": "84500.0000",
  "match_result": "amount_mismatch"
}
```

### Topic: `invoice.rejected`

Published by: `POST /api/v1/invoices/{id}/reject`

```json
{
  "invoice_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "amount": "84500.0000"
}
```

---

## Purchase Orders API

Purchase orders track procurement requests from vendors. A PO feeds into the
3-way match engine: the invoice amount and quantity are compared against the PO
and its linked goods receipts.

### PO Object

All endpoints that return a purchase order return this shape.

```json
{
  "id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "vendor_id": "VENDOR-ACME-001",
  "po_number": "PO-2026-0187",
  "amount": "125000.0000",
  "qty": 50,
  "currency": "IDR",
  "status": "open",
  "created_at": "2026-07-01T09:15:00Z",
  "updated_at": "2026-07-01T09:15:00Z"
}
```

**Field notes**

- `amount` is always a decimal string (e.g. `"125000.0000"`), not a JSON number.
  Parse it with a decimal library. Casting to `float64` risks truncation on large
  values because the backing column is `NUMERIC(20,4)`.
- `tenant_id` is always the tenant from the JWT -- never a user-supplied value.
- `qty` is the total ordered quantity (integer, positive).

### PO Status enum

| Value | Description |
|---|---|
| `open` | PO created, awaiting goods receipt |
| `partially_received` | Some goods received, not the full quantity |
| `received` | Full quantity received via goods receipts |
| `closed` | PO fully invoiced and matched; terminal state |
| `cancelled` | PO cancelled before completion; terminal state |

---

### POST /api/v1/purchase-orders

Create a new purchase order. The record is inserted with `status: open`.

**Auth:** Bearer JWT with `role` claim `admin` or `purchaser`

#### Request body

`Content-Type: application/json` -- max 64 KiB.

| Field | Type | Required | Description |
|---|---|---|---|
| `vendor_id` | string | yes | Identifies the vendor; must match your vendor master data |
| `po_number` | string | yes | Unique PO number, e.g. `"PO-2026-0187"` |
| `amount` | string | yes | Decimal string, e.g. `"125000.0000"`. Do not send as a JSON number. |
| `qty` | integer | yes | Ordered quantity. Must be a positive integer. |
| `currency` | string | yes | ISO 4217 code, e.g. `"IDR"` or `"USD"` |

`tenant_id` is derived from the JWT; do not include it in the body.

#### Example

```
POST /api/v1/purchase-orders
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoicHVyY2hhc2VyIiwiZXhwIjoxNzgyNzIwMDAwfQ.sig
Content-Type: application/json

{
  "vendor_id": "VENDOR-ACME-001",
  "po_number": "PO-2026-0187",
  "amount": "125000.0000",
  "qty": 50,
  "currency": "IDR"
}
```

#### Responses

**201 Created** -- Purchase order created. Body is the PO object with `"status": "open"`.

```json
{
  "id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "vendor_id": "VENDOR-ACME-001",
  "po_number": "PO-2026-0187",
  "amount": "125000.0000",
  "qty": 50,
  "currency": "IDR",
  "status": "open",
  "created_at": "2026-07-01T09:15:00Z",
  "updated_at": "2026-07-01T09:15:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid request body"}` | Malformed JSON or body unreadable |
| `400` | `{"error": "amount must be a valid positive decimal string (e.g. \"1234.56\")"}` | `amount` is empty, non-numeric, zero, or negative |
| `400` | `{"error": "amount exceeds maximum allowed value"}` | `amount` exceeds `9999999999999999.9999` |
| `400` | `{"error": "amount may not have more than 4 decimal places"}` | `amount` has more than 4 decimal places |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `403` | `{"error": "forbidden"}` | JWT `role` is not `admin` or `purchaser` |
| `413` | `{"error": "request body too large"}` | Body exceeds 64 KiB |
| `422` | `{"error": "invalid purchase order data"}` | Business rule violation (e.g. blank vendor_id, blank po_number) |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

### GET /api/v1/purchase-orders

List all purchase orders for the current tenant.

**Auth:** Bearer JWT -- any authenticated tenant user (no role restriction)

#### Example

```
GET /api/v1/purchase-orders
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoic3RhZmYiLCJleHAiOjE3ODI3MjAwMDB9.sig
```

#### Responses

**200 OK** -- Array of PO objects. Returns `[]` when no purchase orders exist.

```json
[
  {
    "id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
    "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
    "vendor_id": "VENDOR-ACME-001",
    "po_number": "PO-2026-0187",
    "amount": "125000.0000",
    "qty": 50,
    "currency": "IDR",
    "status": "open",
    "created_at": "2026-07-01T09:15:00Z",
    "updated_at": "2026-07-01T09:15:00Z"
  }
]
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

### GET /api/v1/purchase-orders/{id}

Get a single purchase order by ID. The PO must belong to the caller's tenant.

**Auth:** Bearer JWT -- any authenticated tenant user (no role restriction)

#### Path parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | The purchase order ID |

#### Example

```
GET /api/v1/purchase-orders/b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoic3RhZmYiLCJleHAiOjE3ODI3MjAwMDB9.sig
```

#### Responses

**200 OK** -- Single PO object.

```json
{
  "id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "vendor_id": "VENDOR-ACME-001",
  "po_number": "PO-2026-0187",
  "amount": "125000.0000",
  "qty": 50,
  "currency": "IDR",
  "status": "open",
  "created_at": "2026-07-01T09:15:00Z",
  "updated_at": "2026-07-01T09:15:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid purchase order id"}` | `id` path segment is not a valid UUID |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `404` | `{"error": "purchase order not found"}` | No PO with this ID exists in the caller's tenant |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

## Goods Receipts API

Goods receipts record physical delivery of goods against a purchase order. A GR
feeds into the 3-way match engine alongside the PO and invoice to verify that
quantities and amounts align.

### GR Object

All endpoints that return a goods receipt return this shape.

```json
{
  "id": "c5d8e9f0-1a2b-3c4d-5e6f-7890abcdef12",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "po_id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
  "vendor_id": "VENDOR-ACME-001",
  "received_qty": 48,
  "received_amount": "120000.0000",
  "currency": "IDR",
  "status": "pending",
  "received_at": "2026-07-05T14:30:00Z",
  "created_at": "2026-07-05T14:30:00Z"
}
```

**Field notes**

- `received_amount` is always a decimal string (e.g. `"120000.0000"`), not a
  JSON number. Parse it with a decimal library. Casting to `float64` risks
  truncation on large values because the backing column is `NUMERIC(20,4)`.
- `po_id` references the purchase order this receipt is linked to. The PO must
  exist in the same tenant; otherwise creation fails with `404`.
- `tenant_id` is always the tenant from the JWT -- never a user-supplied value.
- `received_qty` is the actual quantity received (integer, positive).

### GR Status enum

| Value | Description |
|---|---|
| `pending` | GR created, awaiting acceptance or rejection |
| `accepted` | GR accepted; used in 3-way match |
| `rejected` | GR rejected (damaged goods, wrong items, etc.) |

---

### POST /api/v1/goods-receipts

Create a new goods receipt linked to an existing purchase order. The record is
inserted with `status: pending`.

The `po_id` must reference a purchase order that belongs to the same tenant. If
no matching PO is found, the request returns `404`.

**Auth:** Bearer JWT with `role` claim `admin` or `warehouse`

#### Request body

`Content-Type: application/json` -- max 64 KiB.

| Field | Type | Required | Description |
|---|---|---|---|
| `po_id` | string (UUID) | yes | The purchase order this receipt is linked to |
| `vendor_id` | string | yes | Vendor identifier; must match the PO vendor |
| `received_qty` | integer | yes | Actual quantity received. Must be a positive integer. |
| `received_amount` | string | yes | Decimal string, e.g. `"120000.0000"`. Do not send as a JSON number. |
| `currency` | string | yes | ISO 4217 code, e.g. `"IDR"` or `"USD"` |

`tenant_id` is derived from the JWT; do not include it in the body.

#### Example

```
POST /api/v1/goods-receipts
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoid2FyZWhvdXNlIiwiZXhwIjoxNzgyNzIwMDAwfQ.sig
Content-Type: application/json

{
  "po_id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
  "vendor_id": "VENDOR-ACME-001",
  "received_qty": 48,
  "received_amount": "120000.0000",
  "currency": "IDR"
}
```

#### Responses

**201 Created** -- Goods receipt created. Body is the GR object with `"status": "pending"`.

```json
{
  "id": "c5d8e9f0-1a2b-3c4d-5e6f-7890abcdef12",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "po_id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
  "vendor_id": "VENDOR-ACME-001",
  "received_qty": 48,
  "received_amount": "120000.0000",
  "currency": "IDR",
  "status": "pending",
  "received_at": "2026-07-05T14:30:00Z",
  "created_at": "2026-07-05T14:30:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid request body"}` | Malformed JSON or body unreadable |
| `400` | `{"error": "invalid po_id"}` | `po_id` is not a valid UUID |
| `400` | `{"error": "amount must be a valid positive decimal string (e.g. \"1234.56\")"}` | `received_amount` is empty, non-numeric, zero, or negative |
| `400` | `{"error": "amount exceeds maximum allowed value"}` | `received_amount` exceeds `9999999999999999.9999` |
| `400` | `{"error": "amount may not have more than 4 decimal places"}` | `received_amount` has more than 4 decimal places |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `403` | `{"error": "forbidden"}` | JWT `role` is not `admin` or `warehouse` |
| `404` | `{"error": "purchase order not found"}` | No PO with the given `po_id` exists in the caller's tenant |
| `413` | `{"error": "request body too large"}` | Body exceeds 64 KiB |
| `422` | `{"error": "invalid goods receipt data"}` | Business rule violation (e.g. blank vendor_id) |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

### GET /api/v1/goods-receipts

List all goods receipts for the current tenant.

**Auth:** Bearer JWT -- any authenticated tenant user (no role restriction)

#### Example

```
GET /api/v1/goods-receipts
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoic3RhZmYiLCJleHAiOjE3ODI3MjAwMDB9.sig
```

#### Responses

**200 OK** -- Array of GR objects. Returns `[]` when no goods receipts exist.

```json
[
  {
    "id": "c5d8e9f0-1a2b-3c4d-5e6f-7890abcdef12",
    "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
    "po_id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
    "vendor_id": "VENDOR-ACME-001",
    "received_qty": 48,
    "received_amount": "120000.0000",
    "currency": "IDR",
    "status": "pending",
    "received_at": "2026-07-05T14:30:00Z",
    "created_at": "2026-07-05T14:30:00Z"
  }
]
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |

---

### GET /api/v1/goods-receipts/{id}

Get a single goods receipt by ID. The GR must belong to the caller's tenant.

**Auth:** Bearer JWT -- any authenticated tenant user (no role restriction)

#### Path parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | The goods receipt ID |

#### Example

```
GET /api/v1/goods-receipts/c5d8e9f0-1a2b-3c4d-5e6f-7890abcdef12
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZW5hbnRfaWQiOiJhMWIyYzNkNC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJyb2xlIjoic3RhZmYiLCJleHAiOjE3ODI3MjAwMDB9.sig
```

#### Responses

**200 OK** -- Single GR object.

```json
{
  "id": "c5d8e9f0-1a2b-3c4d-5e6f-7890abcdef12",
  "tenant_id": "a1b2c3d4-0000-0000-0000-000000000001",
  "po_id": "b4e7f2a1-9c3d-4e8f-a5b6-1234567890ab",
  "vendor_id": "VENDOR-ACME-001",
  "received_qty": 48,
  "received_amount": "120000.0000",
  "currency": "IDR",
  "status": "pending",
  "received_at": "2026-07-05T14:30:00Z",
  "created_at": "2026-07-05T14:30:00Z"
}
```

**Error responses**

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid goods receipt id"}` | `id` path segment is not a valid UUID |
| `401` | *(see Auth section)* | Invalid or missing JWT |
| `404` | `{"error": "goods receipt not found"}` | No GR with this ID exists in the caller's tenant |
| `500` | `{"error": "internal server error"}` | Unexpected server-side error |
