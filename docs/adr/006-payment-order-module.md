# ADR-006: Payment Order Module

Date: 2026-07-11
Status: Accepted

---

## Context

Tayooli's Order-to-Pay pipeline currently ends at invoice approval. Once an
invoice passes OCR processing and 3-way match (or human review), it reaches
`approved` status and the workflow stops. There is no mechanism to:

1. Initiate an actual payment to a vendor from an approved invoice.
2. Track whether a payment has been executed, preventing duplicate payments to
   the same vendor for the same invoice.
3. Enforce an approval gate on payments separate from invoice approval (e.g.
   a CFO must approve the payment before treasury executes it).
4. Emit structured events when money moves, enabling downstream reconciliation,
   accounting ledger updates, and vendor notification workflows.

Without a payment order layer, finance teams manage payments through
spreadsheets or external banking portals with no audit trail in the ERP system.
This creates a gap where an invoice can be marked `approved` but the payment
status is unknown, making month-end close unreliable.

---

## Decision

### 1. Payment Order as a separate domain entity (not a status on Invoice)

A payment order is modeled as an independent entity with its own lifecycle,
linked to an invoice via `invoice_id` foreign key. The invoice retains its
own status (`approved`, `rejected`, etc.) and is not mutated when a payment
order is created, approved, or paid.

**Rationale:** Separation of concerns. An invoice can exist without a payment
order (the user may choose to pay later or via external means). Multiple
payment orders against the same invoice should be prevented, but this is
enforced by a database constraint rather than by coupling the two lifecycles.
This also avoids adding payment-related columns (`payment_method`,
`approved_by`, `paid_at`) to the invoice table, keeping it focused on
document ingestion and matching.

### 2. Three-role RBAC model: treasury, cfo, admin

Payment operations are split across two specialized roles:

- **treasury**: Can create payment orders and mark them as paid. This role
  represents the team that executes payments.
- **cfo**: Can approve and reject payment orders. This role represents the
  authorization gate before money moves.
- **admin**: Full access to all operations (superset of both roles).

The `admin` role retains access to every route for operational override
purposes, consistent with the existing RBAC pattern across all modules.

**Rationale:** Separation of duties. The person who initiates a payment
(treasury) should not be the same person who authorizes it (cfo). This
mirrors real-world finance controls where payment execution and payment
approval are performed by different individuals to prevent fraud.

### 3. One active payment order per invoice

A partial unique index on `(invoice_id)` where `status IN ('draft', 'approved')`
ensures that at most one active payment order can exist for any given invoice.
Once a payment order reaches `paid` or `rejected` status, a new one can be
created against the same invoice.

```sql
CREATE UNIQUE INDEX idx_payment_orders_active_invoice
ON payment_orders (invoice_id)
WHERE status IN ('draft', 'approved');
```

**Rationale:** Double-payment prevention. Without this constraint, two treasury
users could independently create payment orders for the same invoice, leading
to duplicate payments. The partial index approach allows the history of past
payment orders to be retained while preventing concurrent active orders.

### 4. SQL WHERE guard on the pay transition

The `pay` transition uses an atomic UPDATE with a `WHERE status = 'approved'`
predicate. If the row is already `paid` (due to a concurrent request), the
UPDATE matches zero rows and the handler returns `409 Conflict`.

```sql
UPDATE payment_orders
SET status = 'paid', paid_at = NOW(), reference_number = $3, updated_at = NOW()
WHERE id = $1
  AND tenant_id = $2
  AND status = 'approved'
```

**Rationale:** Eliminates time-of-check/time-of-use (TOCTOU) race conditions.
No optimistic locking (version column) is needed because the `WHERE status`
predicate provides equivalent atomicity for single-record transitions.

### 5. Audit logging on every status transition

Each transition writes an entry to the existing `audit_logs` table
(append-only). The write is best-effort: if the audit insert fails, the
payment order transition still succeeds. This matches the existing audit
pattern used by the invoice module.

**Rationale:** Payment audit trails are a regulatory requirement in many
jurisdictions. Best-effort is chosen over transactional (audit in the same
transaction) because audit failures should not block payment execution, and
the audit system can be replayed from Kafka events if needed.

### 6. Kafka events for downstream integration

Four topics are published: `payment_order.created`, `payment_order.approved`,
`payment_order.paid`, `payment_order.rejected`. Events are keyed by payment
order UUID for ordered per-entity consumption.

**Rationale:** Downstream systems (accounting ledger, vendor notification
service, reconciliation engine) need to react to payment lifecycle changes.
Kafka provides durable, replayable event delivery that decouples the payment
module from these consumers.

---

## Alternatives Considered

### Payment as a status on Invoice

Adding `payment_status`, `payment_method`, `paid_at`, and `approved_by` columns
to the `invoices` table. Rejected because it conflates two distinct lifecycles
(document matching vs. money movement) into one entity, violates single-
responsibility, and makes it impossible to track partial payments or multiple
payment attempts per invoice in the future.

### Optimistic locking (version column)

Adding a `version` integer column to `payment_orders` and using `WHERE version = $N` for concurrency control. Deferred because the `WHERE status = '...'`
predicate already prevents the only meaningful race condition (double-pay). A
version column adds complexity without material benefit given the current
transactional volume.

### Role enforcement only at the use-case layer

Removing `RequireRole` middleware and relying solely on the use-case layer for
RBAC checks. Rejected because defense-in-depth is a stated project principle.
The middleware provides a fast-fail at the HTTP layer (no DB call needed), and
the use-case check provides a safety net if a new route is added without the
middleware.

### Single "payment" role

Using one `payment` role for all payment operations instead of splitting into
`treasury` and `cfo`. Rejected because it violates the principle of least
privilege and does not support separation of duties. A single role would allow
the same user to both authorize and execute payments.

---

## Consequences

### Positive

- The Order-to-Pay workflow is now end-to-end: invoice ingestion, OCR, 3-way
  match, approval, payment order creation, payment approval, and payment
  execution are all tracked in the ERP system.
- Double-payment is structurally prevented at the database level (partial
  unique index + SQL WHERE guard), not just at the application level.
- Separation of duties between `treasury` (execute) and `cfo` (authorize)
  provides a fraud-prevention control that mirrors real-world finance
  operations.
- Kafka events enable downstream consumers (accounting, reconciliation,
  vendor notifications) without coupling them to the payment module.
- Every status transition is audit-logged, providing a tamper-evident trail
  for regulatory compliance.

### Negative / Trade-offs

- The partial unique index on `invoice_id` means that once a payment order is
  `paid` or `rejected`, a new one can be created against the same invoice. This
  is intentional for the `rejected` case (retry after rejection) and for the
  `paid` case (rare re-payment scenarios), but it means the application must
  handle the edge case of creating a second payment order for an already-paid
  invoice. The SQL WHERE guard on the pay transition provides the final safety
  net.

- Best-effort audit logging means that in the rare event of an audit write
  failure, the payment order transition succeeds without an audit entry. The
  Kafka event for the same transition provides a secondary audit trail, but
  consumers must be aware that the `audit_logs` table may have gaps.

- The `cfo` role is specific to payment approval. If future modules require
  different approval roles (e.g. `hr_approver` for expense claims), the role
  set will grow. This is acceptable given the current scope but may warrant a
  more flexible permission system in the future.

---

## References

- `docs/api/payment-orders.md` -- full API documentation
- `internal/domain/payment_order.go` -- PaymentOrderRepository interface
- `internal/usecase/paymentorder/` -- use-case implementations
- `internal/handler/payment_order.go` -- HTTP handlers
- `migrations/` -- payment_orders table and RLS policies
- ADR-003 (Invoice Status Lifecycle) -- establishes the status guard pattern
  reused here
- ADR-005 (PO & GR CRUD) -- establishes the handler helper pattern reused here
