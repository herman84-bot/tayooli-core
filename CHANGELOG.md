# Changelog

All notable changes to Tayooli ERP are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- Demo auth fallback for the Next.js preview: `app/api/v1/auth/*` route
  handlers (`login` / `me` / `logout`) that proxy to the Go backend when
  `NEXT_PUBLIC_API_URL`/`API_BASE_URL` is configured, otherwise serve a
  small in-memory demo session (seed users admin/accountant/approver@test.com,
  password `password123`) with an HMAC-signed `lumina_auth` cookie. Demo mode
  is disabled in production unless `AUTH_DEMO=true`
  (`lib/auth/demo-auth.ts` + unit tests)
- Multi-gateway payment abstraction (`lib/payments/`): `PaymentProvider`
  interface + registry, supporting **Pakasir** and **Midtrans** behind one
  generic API (`/api/payments/*`), shared webhook handler with per-provider
  authenticity checks (project slug match / SHA512 signature), and a local
  demo checkout page (`/payments/demo/[orderId]`)
- Midtrans integration (Snap API): server-side transaction creation with the
  tenant's Server Key, hosted redirect URL, SHA512 webhook signature
  verification, and server-to-server status re-check
- Generic per-tenant gateway config: `tenant_payment_configs.provider` column
  plus Midtrans credential columns in `schema_pakasir.sql`; admin page moved
  to `/dashboard/payment-gateways` with gateway selector per tenant
- Pakasir payment gateway integration for Sales Invoices (Order-to-Cash):
  hosted checkout payment links (QRIS / Virtual Account), webhook receiver
  with amount + order_id verification and server-to-server re-verification,
  and a demo-only simulate endpoint
- Per-tenant Pakasir provider configuration (Model B multi-tenant money flow):
  `tenant_payment_configs` table with RLS (`schema_pakasir.sql`), per-tenant
  slug/API key resolution with global env fallback, and cross-tenant webhook
  isolation (project mismatch rejected with 403)
- Payment Gateway admin page (`/dashboard/pakasir-config`) to manage tenant
  Pakasir projects and settlement bank metadata
- API documentation for the payment gateway (`docs/api/pakasir.md`) and
  architecture decision record (`docs/adr/008-pakasir-payment-gateway.md`)
- Unit tests for Pakasir helpers, webhook payload parsing, per-tenant config
  store, and payment lifecycle (`__tests__/pakasir.test.ts`)
- Unit tests for the Midtrans provider and multi-gateway registry
  (`__tests__/midtrans.test.ts`)
- AI Serving (Module 2): FastAPI service (`ai-worker/service.py`) implementing
  the `/api/v1/inference/*` contract (ingest + status) with deterministic
  anomaly scoring and GL account suggestion; OCR pipeline extracted to
  `ai-worker/ocr.py` and run as a background thread inside the serving process
- AI inference frontend: `api.inference` client, `useInference` hooks, and a
  "Run AI analysis" action in the invoice detail drawer showing the anomaly
  badge + suggested GL account
- Products & Inventory wiring: `products`/`inventory` tables (migration
  `011_products_inventory.sql`), product CRUD + stock-adjustment endpoints
  registered in the Go router, and a new `/dashboard/products` page
  (DataTable, create drawer with Zod validation, inventory adjust controls)
- Order-to-Cash E2E spec (`frontend/e2e/o2c-cycle.spec.ts`) covering
  Sales Order → Sales Invoice → gateway payment → Paid
- Production readiness: root Next.js standalone `Dockerfile`,
  `docker-compose.yml` (Postgres + Kafka + API + AI worker + frontend), and
  `k8s/` manifests (namespace, configmap, secrets example, deployments for
  finance-service / ai-worker / lumina-api, HPA, ingress); CI builds the
  frontend from the repo root, builds/pushes the ai-worker image, and creates
  the app secret before deploying
- AI Serving API documentation (`docs/api/inference.md`)
- Complete Playwright E2E setup: `@playwright/test` devDependency,
  `playwright.config.ts` with webServer entries (Next.js always; Go API +
  Python AI worker when `E2E_FULL_STACK=1`), and a `test:e2e` script
- E2E test data (migration `012_e2e_test_users.sql`): seeded users
  admin/accountant/approver (password123) + demo vendor, so the suite runs
  against a real backend
- E2E specs realigned to the current app: auth-flow (login redirect, invalid
  credentials, logout), invoice/approval flow (create invoice via
  `/dashboard/invoices/new`), O2P page smoke tests, and the O2C full cycle
- Login redirects to `/dashboard` after successful sign-in
  (`components/auth/LoginForm.tsx`)
- CI e2e job now provisions the full stack (Postgres + Kafka services, Go
  backend and AI worker via webServer) so every spec runs in CI
- Auto-created approval requests for high-value invoices: creating an invoice
  at/above `ApprovalAmountThreshold` (100,000,000, IDR reference) now
  creates a pending approval request (target_type `invoice`) as a best-effort
  side-effect in the invoice facade, wired in `main.go` via
  `WithApprovalRequester`; the requester is the authenticated user
  (`CreateInvoiceParams.CreatedBy`)
- `approval_requests` table migrated (`013_approval_requests.sql`) with RLS —
  previously it only existed in the CI inline bootstrap SQL
- Approval E2E spec deepened: asserts a pending approval request appears
  automatically for a high-value invoice and that no request is created for a
  low-value invoice
- Dashboard Analytics module (`GET /api/v1/dashboard/summary`) — read-only
  aggregated metrics endpoint available to any authenticated role:
  invoice counts by status, payment totals, active vendors, purchase order and
  goods receipt counts, 6-month invoice trend, and top 5 vendors by amount
- All dashboard queries execute inside a single transaction with dual-layer
  tenant isolation (application WHERE clause + RLS `SET LOCAL`)
- Amount fields serialized as strings (`NUMERIC(20,4)`) to preserve decimal
  precision in JSON
- API documentation for Dashboard Analytics (`docs/api/dashboard.md`)
- Payment Order module (`/api/v1/payment-orders`) with full CRUD and lifecycle:
  create, approve, pay, reject endpoints with role-based access control
  (`admin`, `treasury`, `cfo`)
- One active payment order per invoice enforcement via partial unique index
  (`WHERE status IN ('draft', 'approved')`)
- Double-payment prevention through SQL WHERE guard on the pay transition
- Kafka events for all payment order lifecycle transitions
  (`payment_order.created`, `payment_order.approved`, `payment_order.paid`,
  `payment_order.rejected`)
- Audit logging for every payment order status transition (append-only,
  best-effort)
- Defense-in-depth role enforcement at both router middleware and use-case layers
- API documentation for Payment Orders (`docs/api/payment-orders.md`)
- Architecture Decision Record: payment order module design, RBAC model, and
  double-payment prevention (`docs/adr/006-payment-order-module.md`)
- Vendor Management module (`/api/v1/vendors`) with full CRUD, soft-delete,
  and public rating system (1-5 scale)
- Paginated vendor listing with full-text search on `name` and `code`
  (`?q=search&page=1&page_size=20`)
- Vendor rating aggregation: running average `rating` and `rating_count`
  updated atomically on each rating submission
- Open-transaction guard on vendor delete: prevents soft-deleting vendors with
  open purchase orders or pending invoices (returns `409 Conflict`)
- Immutable vendor code: normalized to uppercase at creation, cannot be
  changed via PUT
- Kafka events for vendor lifecycle (`vendor.created`, `vendor.updated`,
  `vendor.deleted`, `vendor.rated`) keyed by vendor UUID
- RBAC enforcement: create/update restricted to `admin` and `accountant`,
  delete restricted to `admin` only; read and rate open to any authenticated
  user
- API documentation for Vendor Management (`docs/api/vendors.md`)
- Architecture Decision Record: vendor master data design, soft-delete model,
  rating system, and RBAC (`docs/adr/007-vendor-management.md`)
- Go backend Clean/Hexagonal Architecture foundation
  (`cmd/api/`, `internal/domain/`, `internal/usecase/`, `internal/handler/`, `internal/infra/`)
- JWT-based multi-tenant authentication middleware (`internal/middleware/tenant.go`)
  — HS256, mandatory `exp` claim, `tenant_id` UUID injected into request context
- Invoice CRUD endpoints (list, create, approve) served under `/api/v1/invoices`
- Dual-layer tenant isolation: application-level `WHERE tenant_id = $n` clause
  plus PostgreSQL `SET LOCAL app.current_tenant_id` activating RLS policies
- Kafka event publishing (`invoice.created`, `invoice.approved`) keyed by
  invoice UUID for ordered per-entity consumption
- Structured logging with zerolog (JSON in production, console in development)
- PostgreSQL schema v1: `tenants`, `users`, `invoices`, `audit_logs` tables
  with Row-Level Security enabled on all tables (`migrations/001_init_schema.sql`)
- Seed data migration for local development (`migrations/002_seed_data.sql`)
- OpenAPI 3.1 specification for Invoice API (`docs/openapi.yaml`)
- Architecture Decision Record: hexagonal arch, JWT, dual-layer RLS, sqlc,
  Kafka keying (`docs/adr/001-go-backend-hexagonal-jwt-rls.md`)

---

## [0.1.0] — 2026-06-26

### Added

- Initial project scaffold: Docker Compose, Go module, Next.js 14 app shell,
  PostgreSQL 15 with multi-tenant schema, Apache Kafka in KRaft mode
- LuminaFlow ERP core architecture documentation (`ARCHITECTURE.md`)
- AI Agent orchestration rules and CTO pipeline (`CLAUDE.md`)
