# ADR-008: Pakasir Payment Gateway (Per-Tenant Provider Config)

Date: 2026-08-07
Status: Accepted

---

## Context

Tayooli is a multi-tenant ERP. Tenants invoice their customers (Sales
Invoices / Order-to-Cash) and need a way to collect payments online. We chose
**Pakasir** (pakasir.com, PT Geksa) as the payment gateway — an Indonesian
payment-link platform supporting QRIS and Virtual Account channels (BRI, BNI,
Permata, CIMB, Maybank, BNC, etc.) with webhook notifications and IDR
settlement.

The central architecture question was **who holds the payment credentials and
where the money settles**, because this is a multi-tenant SaaS product:

1. **Model A — one gateway account owned by the platform.** All tenant
   payments settle into the platform's bank account first, then the platform
   distributes funds to tenants.
2. **Model B — per-tenant gateway accounts/projects.** Each tenant has its own
   Pakasir project (slug + api_key) and money settles directly to the tenant's
   bank account. The platform never holds tenant funds.

Pakasir's official documentation confirms that **one Pakasir account can host
multiple projects** ("dengan satu akun Pakasir, Anda dapat mengintegrasikan
untuk banyak website/aplikasi"), each project having its own **slug + API key**
and its own **sandbox mode** and **webhook URL**.

**Settlement model — verified from the official website & FAQ (checked
2026-08-07; docs updated 21 Jul 2026, FAQ updated 16 Jul 2026):**

- Pakasir is **not itself a payment gateway**. It builds payment links on top
  of a BI-licensed payment gateway: "Dana disimpan dan diproses oleh Payment
  Gateway yang kami gunakan" — funds are stored by that gateway and appear as
  a **single Pakasir account balance**.
- All transactions (across **all** projects under the account) move into that
  balance on **H+1 at 12:00 WIB** ("saldo tertunda" becomes the main balance).
- Withdrawals go to **15 national banks / 4 eWallets**, and the receiving
  account **must match the account holder's KYC identity (KTP)** — KYC is
  mandatory and per-account. No per-project settlement bank is documented.

**Conclusion:** Pakasir does **not** support per-project settlement accounts.
A "per-tenant project under the platform's account" still settles into the
account holder's balance and can only be withdrawn to the account holder's
KYC-verified bank account — i.e. it is **Model A in substance** and would make
the platform a fund holder. The only way money settles directly to a tenant's
own bank account is for **the tenant to own their own Pakasir account**
(own KYC, own withdrawal account); the platform then only stores their
slug + api_key.

---

## Decision

### 1. Gateway: Pakasir (accepted, supersedes generic gateway abstraction)

Pakasir was selected over Stripe because the product targets Indonesian
businesses: IDR currency, QRIS, local Virtual Accounts, and a simple
payment-link integration without lengthy merchant onboarding. Stripe has
limited local payment method coverage in Indonesia.

### 2. Per-tenant provider configuration as the target model (Model B)

Production target: **one Pakasir project per tenant**, stored in the
`tenant_payment_configs` table (see `schema_pakasir.sql`), keyed by
`tenant_id`. Every transaction records its owning `tenant_id`, and the payment
link is built from the **tenant's own slug**. Webhooks are isolated per tenant:

- The webhook payload's `project` field **must** match the tenant's registered
  slug; a mismatch is rejected with `403`.
- When the tenant has an API key, the webhook is re-verified server-to-server
  via the Pakasir Transaction Detail API before marking the invoice paid
  (official best practice — never trust the webhook body alone).

**Rationale:** Model B keeps the platform a pure software provider. It never
holds or distributes tenant funds, avoiding the regulatory burden of a payment
service provider (PJP) license in Indonesia, simplifying reconciliation (each
tenant's ledger matches their own bank statement), and isolating fraud/KYC risk
per tenant.

**Clarification (verified 2026-08-07):** "one Pakasir project per tenant"
must be read as **one Pakasir *account* per tenant**. Projects under the
platform's single account all settle to the account holder's bank, so they do
not satisfy Model B. In production each tenant registers their own Pakasir
account (their own KYC + withdrawal account) and the platform stores only the
tenant's project slug + api_key in `tenant_payment_configs`.

### 3. Phased rollout, with a single global fallback for demo/dev

A process-level fallback config (`PAKASIR_PROJECT_SLUG` / `PAKASIR_API_KEY`)
is used **only** when no per-tenant config exists. This keeps the demo and
single-tenant deployments working while the multi-tenant configuration
matures. The resolution order is always:

```
per-tenant config (tenant_payment_configs / in-memory mirror)
  → process-level env fallback
  → demo mode (no key, simulate endpoint enabled)
```

### 4. Demo/persistence strategy for the sandbox

The sandbox cannot run the Go backend + PostgreSQL + Kafka, so the frontend
vertical slice persists payment state in an **in-memory store**
(`lib/payment-store.ts`, globalThis singleton). It mirrors
`tenant_payment_configs` and the transaction table and is deliberately small
and isolated so it can be replaced by real Postgres/Go repositories without
touching route handlers or UI.

### 5. Security invariants

- `pakasir_api_key` must be **encrypted at rest** in production (pgcrypto/KMS)
  and is never returned by API responses (only `hasApiKey: boolean`).
- Webhook handlers verify amount + order_id (demo) and re-verify via the
  Transaction Detail API (live) before any state change.
- Cross-tenant webhook attempts are rejected (`403 project mismatch`).
- The simulate endpoint is disabled automatically once the owning tenant (or
  the global fallback) has a real API key.

---

## Consequences

**Positive**

- Money never touches the platform → no PJP licensing burden, cleaner
  accounting, per-tenant KYC/fraud isolation.
- The codebase already supports both single-key (demo) and per-tenant (live)
  modes; migration is configuration-only.
- Webhook `project` matching gives a hard tenant-isolation boundary at the
  payment layer.

**Negative / Open questions**

- **Resolved (verified from official website/FAQ, 2026-08-07):** Pakasir has
  **no per-project settlement**. All projects under one account pool into the
  account holder's balance, and withdrawals are KYC-locked to the account
  holder's bank. Therefore "per-tenant project under the platform account" is
  **not** Model B — it would make the platform a fund holder. Production model:
  **each tenant owns their own Pakasir account** (own KYC, own withdrawal
  account); the platform stores slug + api_key per tenant only. The demo
  fallback (one global key, platform-held) is explicitly **demo/MVP-only** and
  must not be used to move real tenant funds.
- In-memory demo state is lost on server restart / redeploy; production needs
  Postgres persistence and the Go backend wiring.
- Gateway fee handling (per-tenant `gateway_fee_percent`) is captured in the
  schema for reporting but not yet reconciled against settlement statements.

---

## Related

- `schema_pakasir.sql` — `tenant_payment_configs` table + RLS
- `docs/api/pakasir.md` — payment gateway API reference
- `lib/pakasir.ts`, `lib/payment-store.ts`, `lib/payments/*` — helpers +
  provider abstraction + in-memory mirror
- ADR-006 (Payment Order Module) — the O2P domain this gateway complements
