# Payment Gateway API

This document covers the payment gateway integration in the Tayooli ERP
frontend slice. The gates collect customer payments on **Sales Invoices**
(Order-to-Cash) through a **provider abstraction** that currently supports two
gateways: **Pakasir** and **Midtrans**. See
[ADR-008](../adr/008-pakasir-payment-gateway.md) for the architecture
decision, multi-tenant money flow, and compliance notes.

## Overview

Tayooli is multi-tenant. Each tenant may own its own gateway account
(Pakasir project, or Midtrans merchant). The generic flow:

1. UI chooses a gateway (`pakasir` | `midtrans`) and creates a payment for an
   invoice → gets a hosted checkout link (Pakasir payment link, or Midtrans
   Snap redirect URL).
2. The customer pays (QRIS / VA) on the gateway's hosted page.
3. The gateway sends a webhook to its own endpoint → the shared handler
   verifies it (signature/project match + server-side re-check) → marks the
   invoice `Paid`.

**Multi-tenant model:** Pakasir does **not** support per-project settlement —
all projects under one account settle into the account holder's balance and
can only be withdrawn to the account holder's KYC-verified bank (verified from
the official FAQ, 2026-08-07; see
[ADR-008](../adr/008-pakasir-payment-gateway.md)). For production, **each
tenant owns their own gateway account** (own KYC and withdrawal account); the
platform stores only the tenant's credentials. When no per-tenant config
exists, process-level env fallbacks are used — **demo/MVP only**, never for
moving real tenant funds; without any credentials, the system runs in **demo
mode** (simulate endpoint enabled and payment links point at the local demo
checkout page).

## Provider abstraction (`lib/payments/`)

| File | Purpose |
|---|---|
| `lib/payments/types.ts` | `PaymentProvider` interface + shared types |
| `lib/payments/providers/pakasir.ts` | Pakasir provider (wraps `lib/pakasir.ts`) |
| `lib/payments/providers/midtrans.ts` | Midtrans provider (Snap API + SHA512 webhook) |
| `lib/payments/registry.ts` | Provider registry + `resolveProviderConfig` |
| `lib/payments/webhook-handler.ts` | Shared webhook receiver logic |
| `lib/payment-store.ts` | In-memory demo store (tenant configs + transactions) |

Adding another gateway = implement `PaymentProvider` + register it in
`lib/payments/registry.ts`. No route/UI changes required.

---

## POST /api/payments/transactions

Create a payment for an invoice through a selected gateway.

### Request Body

```json
{
  "tenantId": "tenant-acme",
  "provider": "midtrans",
  "invoiceId": "INV-2001",
  "amount": 125000,
  "method": "qris",
  "redirect": "https://app.example.com/invoices"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `tenantId` | string | no | Defaults to `demo-tenant-1` |
| `provider` | string | no | `pakasir` (default) or `midtrans` |
| `invoiceId` | string | yes | ERP invoice id |
| `amount` | number | yes | Positive number (converted to integer) |
| `method` | string | no | `qris` (default) or `va` |
| `redirect` | string | no | Custom return URL after payment (Midtrans callbacks) |

### Response `201 Created`

```json
{
  "tenantId": "tenant-acme",
  "provider": "midtrans",
  "invoiceId": "INV-2001",
  "orderId": "INV-2001-K3QZ",
  "amount": 125000,
  "method": "qris",
  "paymentLink": "https://app.sandbox.midtrans.com/snap/v4/redirection/...",
  "status": "pending",
  "demo": true
}
```

In demo mode (no credentials) `paymentLink` points at the local demo checkout
page: `/payments/demo/{orderId}`.

### Errors

| Status | Body | When |
|---|---|---|
| `400` | `{"error": "invalid json body"}` | Malformed JSON |
| `400` | `{"error": "invoiceId is required"}` | Missing invoice id |
| `400` | `{"error": "amount must be a positive number"}` | Missing/invalid amount |
| `400` | `{"error": "method must be 'qris' or 'va'"}` | Unsupported method |
| `400` | `{"error": "provider must be 'pakasir' or 'midtrans'"}` | Unsupported provider |
| `502` | `{"error": "Gagal membuat pembayaran: ..."}` | Gateway API error |

---

## GET /api/payments/transactions?invoiceId=...

Check the current payment status for an invoice.

### Query Parameters

| Parameter | Type | Required | Notes |
|---|---|---|---|
| `invoiceId` | string | yes | Invoice id to check |

### Response `200 OK`

```json
{
  "tenantId": "tenant-acme",
  "provider": "midtrans",
  "invoiceId": "INV-2001",
  "orderId": "INV-2001-K3QZ",
  "amount": 125000,
  "method": "qris",
  "paymentLink": "...",
  "status": "completed",
  "paymentMethod": "qris",
  "completedAt": "2026-08-07T10:00:00Z",
  "verified": true,
  "demo": false
}
```

When the tenant has credentials, the server re-verifies the transaction
server-to-server with the gateway before returning.

### Errors

| Status | Body | When |
|---|---|---|
| `404` | `{"status": "not_found"}` | No transaction for this invoice |

---

## POST /api/webhooks/pakasir · POST /api/webhooks/midtrans

Webhook receivers (one per gateway, both delegating to the shared handler in
`lib/payments/webhook-handler.ts`). Configure the URL in each gateway's
dashboard (Pakasir project Webhook URL / Midtrans Payment Notification URL).

### Verification flow (shared)

1. `order_id` maps to a stored transaction, which carries its owning tenant
   and provider.
2. The webhook's provider must match the transaction's provider.
3. **Authenticity** — Pakasir: the `project` field must match the tenant's
   slug (`403` otherwise). Midtrans: the `signature_key` must match
   `SHA512(order_id + status_code + gross_amount + server_key)`.
4. With credentials: re-verify via the gateway's transaction status API;
   unverifiable → `502`.
5. Demo mode: the amount must match the created transaction.
6. If paid → invoice marked `Paid` → `200 {"ok": true}`.

### Response

| Status | Body | When |
|---|---|---|
| `200` | `{"ok": true}` | Payment processed (or safely ignored) |
| `400` | `{"error": "malformed webhook payload"}` | Invalid body |
| `400` | `{"error": "amount mismatch"}` | Demo-mode amount mismatch |
| `403` | `{"error": "...verification failed"}` | Signature / project mismatch |
| `403` | `{"error": "webhook provider mismatch: ..."}` | Wrong gateway endpoint |
| `502` | `{"error": "could not verify transaction with ..."}` | Live verification failed |

---

## POST /api/payments/simulate

**Demo-only.** Simulates the completed-payment webhook so the full flow can be
exercised without live merchant credentials. Disabled (`403`) when the owning
tenant has real credentials configured.

### Request Body

```json
{ "orderId": "INV-2001-K3QZ", "paymentMethod": "qris" }
```

### Response `200 OK`

```json
{
  "ok": true,
  "orderId": "INV-2001-K3QZ",
  "invoiceId": "INV-2001",
  "tenantId": "tenant-acme",
  "provider": "midtrans",
  "status": "completed"
}
```

---

## GET /api/payments/configs · POST /api/payments/configs

Manage per-tenant gateway configs (demo mirror of the `tenant_payment_configs`
table — in production this is a system-role-only endpoint writing to Postgres
with credentials encrypted at rest).

### GET

Lists tenant configs. Credentials are **never** returned — only
`hasCredentials: boolean` (plus provider-specific booleans).

### POST — upsert a tenant config

```json
{
  "tenantId": "tenant-acme",
  "provider": "midtrans",
  "serverKey": "Midtrans-Server-Key",
  "clientKey": "Midtrans-Client-Key",
  "isProduction": false,
  "settlementBankName": "BCA",
  "settlementBankAccount": "1234567890",
  "settlementHolderName": "PT Acme Indonesia",
  "gatewayFeePercent": 0.7
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `tenantId` | string | yes | Tenant identifier |
| `provider` | string | no | `pakasir` (default) or `midtrans` |
| `slug` | string | no | Pakasir project slug |
| `apiKey` | string | no | Pakasir project API key (empty = demo) |
| `serverKey` | string | no | Midtrans server key (empty = demo) |
| `clientKey` | string | no | Midtrans client key (snap.js) |
| `isProduction` | boolean | no | Midtrans production vs sandbox |
| `settlementBankName` | string | no | Display/reporting only |
| `settlementBankAccount` | string | no | Display/reporting only |
| `settlementHolderName` | string | no | Display/reporting only |
| `gatewayFeePercent` | number | no | Fee for reporting (default 0.7) |
| `isActive` | boolean | no | Disable payments (default true) |

---

## Environment variables

| Variable | Purpose |
|---|---|
| `PAKASIR_PROJECT_SLUG` / `PAKASIR_API_KEY` | Global fallback Pakasir config |
| `PAKASIR_BASE_URL` / `PAKASIR_API_BASE_URL` | Optional Pakasir URL overrides |
| `MIDTRANS_SERVER_KEY` / `MIDTRANS_CLIENT_KEY` | Global fallback Midtrans config |
| `MIDTRANS_IS_PRODUCTION` | `true` = production, else sandbox |
| `MIDTRANS_BASE_URL` / `MIDTRANS_API_BASE_URL` | Optional Midtrans URL overrides |

Prefer per-tenant config (admin page / `POST /api/payments/configs`) over the
global fallback for production multi-tenant deployments.

---

## Persistence note

The current vertical slice persists payment state in an in-memory store
(`lib/payment-store.ts`). State is lost on server restart or redeploy. For
production, replace the store with Postgres tables (mirrored by
`schema_pakasir.sql` and the Go backend repositories) so payments survive
restarts and are tenant-isolated via RLS.
