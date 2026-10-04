# Dashboard Analytics API

This document covers the Dashboard Summary endpoint, which returns aggregated
metrics across all modules (invoices, payments, vendors, purchase orders, and
goods receipts) for the authenticated tenant.

## Table of Contents

1. [Authentication](#authentication)
2. [GET /api/v1/dashboard/summary](#get-apiv1dashboardsummary)
3. [Response Object](#response-object)
4. [Field Details](#field-details)
5. [Examples](#examples)
6. [Errors](#errors)

---

## Authentication

All `/api/v1` endpoints require a `Bearer` JWT in the `Authorization` header.
`TenantMiddleware` validates the token (HS256, expiry enforced) and extracts
claims that drive every downstream operation. See
[invoices.md](invoices.md#authentication) for the full authentication reference.

---

## GET /api/v1/dashboard/summary

Returns aggregated dashboard metrics for the current tenant. All data is
scoped to the requesting tenant via dual-layer isolation (application-level
`WHERE tenant_id` plus PostgreSQL Row-Level Security).

**Access:** Any authenticated user within the tenant (no role restriction).

### Response

```
HTTP/1.1 200 OK
Content-Type: application/json
```

```json
{
  "invoices": {
    "total": 142,
    "pending": 18,
    "approved": 105,
    "rejected": 7,
    "pending_review": 12,
    "total_amount": "482150.0000",
    "approved_amount": "391200.0000"
  },
  "payments": {
    "total": 98,
    "paid": 85,
    "paid_amount": "320500.0000",
    "pending_amount": "70700.0000"
  },
  "vendors": {
    "active": 24
  },
  "purchase_orders": {
    "total": 67
  },
  "goods_receipts": {
    "total": 53
  },
  "monthly_trend": [
    { "month": "2026-02", "invoice_count": 15, "total_amount": "42300.0000" },
    { "month": "2026-03", "invoice_count": 22, "total_amount": "61800.0000" },
    { "month": "2026-04", "invoice_count": 19, "total_amount": "55100.0000" },
    { "month": "2026-05", "invoice_count": 28, "total_amount": "87200.0000" },
    { "month": "2026-06", "invoice_count": 31, "total_amount": "92400.0000" },
    { "month": "2026-07", "invoice_count": 27, "total_amount": "79350.0000" }
  ],
  "top_vendors": [
    { "vendor_id": "a1b2c3d4-...", "vendor_name": "PT Maju Bersama", "invoice_count": 18, "total_amount": "124500.0000" },
    { "vendor_id": "e5f6g7h8-...", "vendor_name": "CV Sumber Rejeki", "invoice_count": 12, "total_amount": "87300.0000" }
  ]
}
```

---

## Response Object

| Field | Type | Description |
|---|---|---|
| `invoices` | object | Aggregated invoice metrics (see below) |
| `payments` | object | Aggregated payment order metrics (see below) |
| `vendors` | object | Vendor count metrics |
| `purchase_orders` | object | Purchase order count |
| `goods_receipts` | object | Goods receipt count |
| `monthly_trend` | array | Invoice counts and amounts per month (last 6 months, ascending) |
| `top_vendors` | array | Top 5 vendors by total invoice amount (descending) |

### `invoices` object

| Field | Type | Description |
|---|---|---|
| `total` | integer | Total invoice count for the tenant |
| `pending` | integer | Invoices with status `pending` |
| `approved` | integer | Invoices with status `approved` |
| `rejected` | integer | Invoices with status `rejected` |
| `pending_review` | integer | Invoices with status `pending_review` (AI confidence below threshold) |
| `total_amount` | string | Sum of all invoice amounts (`NUMERIC(20,4)`, serialized as string) |
| `approved_amount` | string | Sum of amounts for approved invoices only |

### `payments` object

| Field | Type | Description |
|---|---|---|
| `total` | integer | Total payment order count |
| `paid` | integer | Payment orders with status `paid` |
| `paid_amount` | string | Sum of paid payment order amounts |
| `pending_amount` | string | Sum of payment order amounts with status `approved` (awaiting payment) |

### `vendors` object

| Field | Type | Description |
|---|---|---|
| `active` | integer | Count of vendors with status `active` |

### `purchase_orders` object

| Field | Type | Description |
|---|---|---|
| `total` | integer | Total purchase order count |

### `goods_receipts` object

| Field | Type | Description |
|---|---|---|
| `total` | integer | Total goods receipt count |

### `monthly_trend[]` object

| Field | Type | Description |
|---|---|---|
| `month` | string | Month in `YYYY-MM` format |
| `invoice_count` | integer | Number of invoices created in that month |
| `total_amount` | string | Sum of invoice amounts for that month |

The array covers the last 6 calendar months (including the current month).
Months with no invoices are omitted. Results are ordered chronologically
(ascending).

### `top_vendors[]` object

| Field | Type | Description |
|---|---|---|
| `vendor_id` | string (UUID) | Vendor identifier |
| `vendor_name` | string | Vendor display name (falls back to `vendor_id` if the vendor record is missing) |
| `invoice_count` | integer | Number of invoices for this vendor |
| `total_amount` | string | Sum of invoice amounts for this vendor |

Limited to the top 5 vendors ranked by `total_amount` descending.

---

## Field Details

### Amount precision

All monetary amounts are stored as `NUMERIC(20,4)` in PostgreSQL and serialized
as **strings** in JSON to avoid floating-point precision loss. Clients should
parse these values with a decimal-aware library (e.g. `Decimal.js`,
`shopspring/decimal` in Go, `decimal.Decimal` in Python).

### Empty arrays

`monthly_trend` and `top_vendors` always return a JSON array -- never `null`.
When no data exists the response contains `[]`.

### Tenant isolation

Every aggregation query runs inside a single database transaction with
`SET LOCAL app.current_tenant_id` applied before any read. This activates
PostgreSQL Row-Level Security policies, ensuring complete data isolation
between tenants.

---

## Examples

### cURL

```bash
curl -s \
  -H "Authorization: Bearer <jwt>" \
  https://api.tayooli.io/api/v1/dashboard/summary | jq
```

### JavaScript (fetch)

```typescript
const res = await fetch("/api/v1/dashboard/summary", {
  headers: { Authorization: `Bearer ${token}` },
});
const data = await res.json();
console.log(data.invoices.approved); // 105
console.log(data.monthly_trend[0].month); // "2026-02"
```

---

## Errors

| Status | Body | When |
|---|---|---|
| `401` | `{"error": "unauthorized"}` | Missing or invalid JWT |
| `500` | `{"error": "internal server error"}` | Unexpected database or server error |

There are no `403` errors for this endpoint -- all authenticated roles have
access.
