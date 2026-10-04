# Payment Orders API

This document covers the Payment Order endpoints that finalize the Order-to-Pay
workflow. A payment order is created against an approved invoice and progresses
through an approval and payment lifecycle.

## Table of Contents

1. [Authentication](#authentication)
2. [RBAC Role Matrix](#rbac-role-matrix)
3. [Payment Order Object](#payment-order-object)
4. [Status Lifecycle](#status-lifecycle)
5. [GET /api/v1/payment-orders](#get-apiv1payment-orders)
6. [GET /api/v1/payment-orders/{id}](#get-apiv1payment-ordersid)
7. [POST /api/v1/payment-orders](#post-apiv1payment-orders)
8. [POST /api/v1/payment-orders/{id}/approve](#post-apiv1payment-ordersidapprove)
9. [POST /api/v1/payment-orders/{id}/pay](#post-apiv1payment-ordersidpay)
10. [POST /api/v1/payment-orders/{id}/reject](#post-apiv1payment-ordersidreject)
11. [Error Responses](#error-responses)
12. [Kafka Events](#kafka-events)
13. [Architecture Notes](#architecture-notes)

---

## Authentication

All `/api/v1/payment-orders` endpoints require a `Bearer` JWT in the
`Authorization` header. `TenantMiddleware` validates the token (HS256, expiry
enforced) and extracts two claims that drive every downstream operation:

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
| `/api/v1/payment-orders` | GET | any authenticated | List payment orders |
| `/api/v1/payment-orders/{id}` | GET | any authenticated | Get single payment order |
| `/api/v1/payment-orders` | POST | `admin`, `treasury` | Create payment order |
| `/api/v1/payment-orders/{id}/approve` | POST | `admin`, `cfo` | Approve payment order |
| `/api/v1/payment-orders/{id}/pay` | POST | `admin`, `treasury` | Mark as paid |
| `/api/v1/payment-orders/{id}/reject` | POST | `admin`, `cfo`, `treasury` | Reject payment order |

**Role definitions (payment-order specific)**

| Role | Description |
|---|---|
| `admin` | Full access to every write route across all modules |
| `treasury` | Can create payment orders and mark them as paid |
| `cfo` | Can approve and reject payment orders |

A request with a role not listed for a write route receives `403 Forbidden`
with body `{"error": "forbidden"}`.

---

## Payment Order Object

All endpoints that return a payment order use this shape.

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "invoice_id": "b5c6d7e8-f9a0-1234-bcde-f01234567891",
  "amount": "15000.00",
  "currency": "IDR",
  "payment_method": "bank_transfer",
  "reference_number": "PAY-2026-0001",
  "status": "draft",
  "notes": "Urgent vendor payment",
  "created_by": "user-uuid-001",
  "approved_by": null,
  "paid_at": null,
  "created_at": "2026-07-11T10:00:00Z",
  "updated_at": "2026-07-11T10:00:00Z"
}
```

**Field descriptions**

| Field | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Unique payment order identifier |
| `tenant_id` | UUID | no | Owning tenant (from JWT, never from request body) |
| `invoice_id` | UUID | no | The approved invoice this payment order covers |
| `amount` | string (decimal) | no | Payment amount, max 4 decimal places |
| `currency` | string | no | ISO 4217 currency code (e.g. `IDR`, `USD`) |
| `payment_method` | string | no | `bank_transfer`, `virtual_account`, `check`, `cash` |
| `reference_number` | string | yes | External reference or tracking number |
| `status` | string | no | Current status (see [Status Lifecycle](#status-lifecycle)) |
| `notes` | string | yes | Free-text notes |
| `created_by` | UUID | no | User who created the payment order |
| `approved_by` | UUID | yes | User who approved (null until approved) |
| `paid_at` | datetime | yes | Timestamp when marked as paid (null until paid) |
| `created_at` | datetime | no | Creation timestamp (ISO 8601) |
| `updated_at` | datetime | no | Last update timestamp (ISO 8601) |

---

## Status Lifecycle

```
draft  ---[approve]--->  approved  ---[pay]--->  paid
   |
   +------[reject]--->  rejected
```

| Transition | From | To | Trigger |
|---|---|---|---|
| approve | `draft` | `approved` | `POST /{id}/approve` by `admin` or `cfo` |
| pay | `approved` | `paid` | `POST /{id}/pay` by `admin` or `treasury` |
| reject | `draft` | `rejected` | `POST /{id}/reject` by `admin`, `cfo`, or `treasury` |

Invalid transitions (e.g. approving an already-approved order) return `409 Conflict`.

---

## GET /api/v1/payment-orders

List payment orders for the authenticated tenant with cursor-based pagination.

**Auth:** any authenticated user
**Role:** any

### Query Parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | `1` | Page number (1-based) |
| `per_page` | int | `20` | Results per page (max 100) |
| `status` | string | - | Filter by status (`draft`, `approved`, `paid`, `rejected`) |

### Response `200 OK`

```json
{
  "data": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "tenant_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "invoice_id": "b5c6d7e8-f9a0-1234-bcde-f01234567891",
      "amount": "15000.00",
      "currency": "IDR",
      "payment_method": "bank_transfer",
      "reference_number": "PAY-2026-0001",
      "status": "draft",
      "notes": "Urgent vendor payment",
      "created_by": "user-uuid-001",
      "approved_by": null,
      "paid_at": null,
      "created_at": "2026-07-11T10:00:00Z",
      "updated_at": "2026-07-11T10:00:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "per_page": 20
}
```

---

## GET /api/v1/payment-orders/{id}

Retrieve a single payment order by ID.

**Auth:** any authenticated user
**Role:** any

### Path Parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Payment order ID |

### Response `200 OK`

Returns the [Payment Order Object](#payment-order-object).

### Error Responses

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid payment order id"}` | Malformed UUID |
| `404` | `{"error": "payment order not found"}` | ID does not exist or belongs to another tenant |

---

## POST /api/v1/payment-orders

Create a new payment order against an approved invoice.

**Auth:** JWT required
**Role:** `admin`, `treasury`

### Request Body

```json
{
  "invoice_id": "b5c6d7e8-f9a0-1234-bcde-f01234567891",
  "amount": "15000.00",
  "currency": "IDR",
  "payment_method": "bank_transfer",
  "reference_number": "PAY-2026-0001",
  "notes": "Urgent vendor payment"
}
```

| Field | Type | Required | Constraints |
|---|---|---|---|
| `invoice_id` | UUID | yes | Must reference an existing approved invoice belonging to the same tenant |
| `amount` | string (decimal) | yes | Positive value, max 4 decimal places |
| `currency` | string | yes | ISO 4217 currency code |
| `payment_method` | string | yes | One of: `bank_transfer`, `virtual_account`, `check`, `cash` |
| `reference_number` | string | no | External reference or tracking number |
| `notes` | string | no | Free-text notes |

### Response `201 Created`

Returns the created [Payment Order Object](#payment-order-object) with
`status: "draft"`.

### Error Responses

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid request body"}` | Malformed JSON or missing required fields |
| `400` | `{"error": "invalid amount"}` | Amount is zero, negative, or exceeds 4 decimal places |
| `403` | `{"error": "forbidden"}` | User role is not `admin` or `treasury` |
| `404` | `{"error": "invoice not found"}` | `invoice_id` does not exist or belongs to another tenant |
| `409` | `{"error": "active payment order exists"}` | A `draft` or `approved` payment order already exists for this invoice |

---

## POST /api/v1/payment-orders/{id}/approve

Approve a draft payment order. Transitions status from `draft` to `approved`.

**Auth:** JWT required
**Role:** `admin`, `cfo`

### Path Parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Payment order ID |

### Request Body

Empty body `{}`. No fields accepted.

### Response `200 OK`

Returns the updated [Payment Order Object](#payment-order-object) with
`status: "approved"` and `approved_by` populated.

### Error Responses

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid payment order id"}` | Malformed UUID |
| `403` | `{"error": "forbidden"}` | User role is not `admin` or `cfo` |
| `404` | `{"error": "payment order not found"}` | ID does not exist or belongs to another tenant |
| `409` | `{"error": "payment order is not in draft status"}` | Order is already `approved`, `paid`, or `rejected` |

---

## POST /api/v1/payment-orders/{id}/pay

Mark an approved payment order as paid. Transitions status from `approved`
to `paid`.

**Auth:** JWT required
**Role:** `admin`, `treasury`

### Path Parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Payment order ID |

### Request Body

```json
{
  "reference_number": "BANK-TXN-98765"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `reference_number` | string | no | Bank transaction reference or external payment confirmation ID |

### Response `200 OK`

Returns the updated [Payment Order Object](#payment-order-object) with
`status: "paid"`, `paid_at` populated, and `reference_number` updated if
provided.

### Error Responses

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid payment order id"}` | Malformed UUID |
| `403` | `{"error": "forbidden"}` | User role is not `admin` or `treasury` |
| `404` | `{"error": "payment order not found"}` | ID does not exist or belongs to another tenant |
| `409` | `{"error": "payment order is not in approved status"}` | Order is not in `approved` status |

---

## POST /api/v1/payment-orders/{id}/reject

Reject a draft payment order. Transitions status from `draft` to `rejected`.

**Auth:** JWT required
**Role:** `admin`, `cfo`, `treasury`

### Path Parameters

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Payment order ID |

### Request Body

```json
{
  "notes": "Duplicate invoice - already paid via check"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `notes` | string | no | Reason for rejection |

### Response `200 OK`

Returns the updated [Payment Order Object](#payment-order-object) with
`status: "rejected"` and `notes` updated if provided.

### Error Responses

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid payment order id"}` | Malformed UUID |
| `403` | `{"error": "forbidden"}` | User role is not `admin`, `cfo`, or `treasury` |
| `404` | `{"error": "payment order not found"}` | ID does not exist or belongs to another tenant |
| `409` | `{"error": "payment order is not in draft status"}` | Order is not in `draft` status |

---

## Error Responses

All endpoints share a consistent error shape:

```json
{
  "error": "human-readable error message"
}
```

### Summary of error codes

| Status | Meaning | Common causes |
|---|---|---|
| `400` | Bad Request | Malformed JSON, missing required fields, invalid UUID, invalid amount |
| `401` | Unauthorized | Missing or invalid JWT |
| `403` | Forbidden | Valid JWT but insufficient role |
| `404` | Not Found | Resource does not exist or belongs to another tenant (cross-tenant IDs are indistinguishable from missing) |
| `409` | Conflict | Invalid status transition, or a `draft`/`approved` payment order already exists for the target invoice |

---

## Kafka Events

Every status transition publishes an event to the corresponding Kafka topic.
Events are keyed by the payment order UUID for ordered per-entity consumption.

| Topic | Published on | Payload highlights |
|---|---|---|
| `payment_order.created` | `POST /api/v1/payment-orders` | Full payment order object, `invoice_id`, `amount` |
| `payment_order.approved` | `POST /api/v1/payment-orders/{id}/approve` | `id`, `approved_by`, `approved_at` |
| `payment_order.paid` | `POST /api/v1/payment-orders/{id}/pay` | `id`, `paid_at`, `reference_number` |
| `payment_order.rejected` | `POST /api/v1/payment-orders/{id}/reject` | `id`, `notes` |

---

## Architecture Notes

**Double-payment prevention.** A partial unique index on `(invoice_id)` where
`status IN ('draft', 'approved')` ensures only one active payment order can
exist per invoice. A SQL `WHERE` guard on the pay transition additionally
checks that the invoice has not already been marked paid, preventing race
conditions between concurrent pay requests.

**Defense-in-depth role enforcement.** Role checks are applied at two layers:
the `RequireRole` middleware at the router level, and a second role validation
inside the use-case layer. If the middleware is bypassed (e.g. a new route is
added without the middleware), the use-case still rejects unauthorized callers.

**Audit logging.** Every status transition writes an entry to `audit_logs`
(append-only). The audit write is best-effort: if it fails, the payment order
transition still succeeds. The audit entry captures the entity type, action,
and a hash chain for tamper detection.

**Tenant isolation.** All queries include `WHERE tenant_id = $n` at the
application level. PostgreSQL RLS policies provide a second enforcement layer
via `SET LOCAL app.current_tenant_id`. Cross-tenant UUID lookups return `404`
(not `403`) to prevent information leakage.
