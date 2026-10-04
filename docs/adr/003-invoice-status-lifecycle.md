# ADR-003: Invoice Status Lifecycle and RBAC Guards

## Status

Accepted

## Context

Tayooli's invoice processing pipeline is asynchronous and multi-stage:

1. A user creates an invoice (`POST /api/v1/invoices`). Status becomes `pending`.
2. A Kafka event triggers the Python OCR worker, which updates the invoice to
   `ai_processed` (success) or `ai_failed` (pipeline error).
3. If OCR succeeded, the Go 3-way match engine compares the invoice against a
   Purchase Order and Goods Receipt. A clean match auto-approves the invoice;
   any discrepancy moves it to `pending_review`.
4. At `pending_review` or `ai_failed`, a human approver must intervene.

Before this sprint, two gaps existed:

**Gap 1 — Status bypass.** The approve endpoint accepted any invoice regardless
of status. A caller with database access or a valid JWT could approve a
freshly-created invoice (`status: pending`), skipping OCR and 3-way match
entirely. The same issue existed for reject: a caller could reject an
`approved` invoice, undoing a completed payment cycle.

**Gap 2 — No RBAC on state-changing operations.** Any authenticated user —
regardless of role — could call the approve and reject endpoints. The `role`
claim was extracted from the JWT by `TenantMiddleware` but never enforced.

Both gaps were enforced only at the application-trust level; nothing in the
database or middleware prevented misuse.

Alternatives considered:

- **Separate "override" endpoint for ai_failed** — rejected because it adds a
  second code path with no material security benefit; the RBAC guard on the
  shared approve endpoint provides the same protection with less surface area.
- **Optimistic locking (version column)** — deferred; concurrent approval is
  currently low-frequency. The atomic `WHERE status IN (...)` clause in the
  UPDATE statement provides equivalent safety for single-record transitions.
- **Role stored in the database, not the JWT** — deferred; requires a DB round-trip
  on every request. JWT-embedded roles are acceptable given the current team
  size and token TTL, with the understanding that role revocation requires
  either token blacklisting or short expiry.

## Decision

Apply two status-level guards, enforced atomically at the PostgreSQL layer
via `WHERE status IN (...)` predicates in the UPDATE queries, and add
`RequireRole("admin", "approver")` middleware to the approve and reject routes.

**Approve transition** (`POST /api/v1/invoices/{id}/approve`):

```sql
UPDATE invoices
SET status = 'approved', updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2
  AND status IN ('ai_processed', 'pending_review', 'ai_failed')
```

Allowed source states: `ai_processed`, `pending_review`, `ai_failed`.
Rationale: `ai_processed` and `pending_review` have passed OCR; `ai_failed` is
an explicit human-fallback path for when OCR cannot run.

**Reject transition** (`POST /api/v1/invoices/{id}/reject`):

```sql
UPDATE invoices
SET status = 'rejected', updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2
  AND status IN ('pending', 'pending_review')
```

Allowed source states: `pending`, `pending_review`.
Rationale: an invoice awaiting OCR (`pending`) or flagged for review
(`pending_review`) are the only states where rejection is a valid action.
`ai_processed` invoices have passed OCR and should proceed through the
approval flow rather than be silently rejected. `ai_failed` invoices require
an admin decision (approve with manual review) rather than rejection, because
the vendor document was not verifiable — blanket rejection without review is
the wrong default.

**RBAC guard** — both routes are wrapped in a `chi.Group` with
`RequireRole("admin", "approver")` applied. `RequireRole` reads the `role`
claim that `TenantMiddleware` already placed in the request context. If the
role is absent or does not match, the middleware responds `403 {"error":
"forbidden"}` before the handler is reached.

Route registration (from `cmd/api/main.go`):

```go
r.Group(func(r chi.Router) {
    r.Use(tenantMiddleware.RequireRole("admin", "approver"))
    r.Post("/invoices/{id}/approve", invoiceHandler.ApproveInvoice)
    r.Post("/invoices/{id}/reject", invoiceHandler.RejectInvoice)
})
```

## Consequences

**Positive**

- The OCR + 3-way match pipeline cannot be bypassed on the happy path.
  An invoice must reach `ai_processed` or `pending_review` before a human
  can approve it.
- Status transitions are atomic. The `WHERE status IN (...)` predicate in the
  UPDATE means there is no time-of-check / time-of-use (TOCTOU) window; a
  concurrent request cannot race to a second approval.
- When the UPDATE matches zero rows, the handler distinguishes `404` (invoice
  does not exist in this tenant) from `409` (invoice exists but is in the
  wrong state) by running a secondary existence check in the same transaction.
  Callers receive an actionable error without leaking cross-tenant information.
- `RequireRole` is stateless: it reads from the request context populated by
  `TenantMiddleware` with no additional database call. Latency impact is
  negligible.
- Tenant isolation is enforced at two independent layers: the application-level
  `WHERE tenant_id = $2` in every query, and PostgreSQL Row-Level Security
  (RLS) activated via `SET LOCAL app.current_tenant_id` inside each
  transaction.

**Negative / Risks**

- Approving an `ai_failed` invoice is allowed by design. At the time of
  approval, no OCR confidence score or extracted text exists. The approver is
  relying solely on out-of-band verification (e.g. a physical document check).
  There is no "override reason" field captured in the current schema; the
  audit trail for `ai_failed` approvals contains only the status change
  timestamp and the approver's `tenant_id`, not their identity or justification.
  This is a noted gap to address in a future sprint (add `approved_by` and
  `override_note` columns).

- The `role` claim is embedded in the JWT. If a user's role is downgraded
  (e.g. `approver` → `staff`), their existing token retains the old role until
  it expires. Mitigation options (token blacklist, short TTL, refresh-on-role-change)
  are deferred to a future security sprint.

- The reject endpoint does not accept `ai_processed` or `ai_failed` as source
  states. A vendor invoice that passed OCR but should be declined still requires
  the approver to use the approve path — there is currently no "reject after
  OCR" transition. This was a deliberate scope constraint; a future ADR may
  add this transition if the business process requires it.
