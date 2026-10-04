# ADR-005: PO & GR CRUD API and Codebase Refactor (Phase 5A)

Date: 2026-06-28
Status: Accepted

---

## Context

Phase 4 built the 3-way match engine but provided no HTTP interface to create or
query purchase orders and goods receipts, making end-to-end flow impossible to test
or use. Additionally, the codebase had grown three duplications that needed
addressing before more handlers were added:

1. The `SET LOCAL app.current_tenant_id` transaction preamble was copied verbatim
   into every Postgres repository file.
2. UUID path-parameter parsing and decimal amount validation were duplicated across
   invoice, PO, and GR HTTP handlers with inconsistent error messages.
3. RLS policies on the new `purchase_orders` and `goods_receipts` tables did not
   guard against the `current_setting` function returning an empty string,
   a Postgres-level runtime error that would surface as an opaque 500.

---

## Decisions

### 1. PORepository and GRRepository as separate domain interfaces

Purchase orders and goods receipts are distinct domain objects with independent
lifecycles: a PO can be created and remain open for weeks before any GR is
submitted, and a single PO may have multiple partial GRs over time. Merging them
into a shared repository interface would introduce a coupling that makes neither
mock-testable in isolation. Separate `PORepository` and `GRRepository` interfaces
in `internal/domain/` enforce the boundary so each can be faked independently
in use-case unit tests without instantiating unrelated infrastructure.

### 2. GR usecase verifies PO ownership before insert

A goods receipt is only meaningful in the context of a PO that belongs to the same
tenant. Without an explicit `SELECT tenant_id FROM purchase_orders WHERE id = $1`
check before the GR insert, a caller who knows a foreign tenant's PO UUID could
link their own GR to it, corrupting 3-way match accounting across a tenant boundary.
The ownership check is performed inside the GR use-case (not the repository) so the
business rule is visible at the domain layer, not buried in SQL. Both "PO not found"
and "PO belongs to another tenant" surface identically as 404 to prevent a caller
from inferring another tenant's PO existence.

### 3. setTenantLocally extracted to postgres/shared.go

Every Postgres repository must call `SET LOCAL app.current_tenant_id = $1` at the
start of each transaction to activate RLS. This call existed as a copy in
`invoice_repo.go`, `po_repo.go`, and `gr_repo.go`. Divergence risk is concrete:
a future change to add observability tracing or error wrapping to this call would
have to be made identically in three files, and a missed copy would silently break
RLS on one entity type. Extracting `setTenantLocally(ctx, tx, tenantID)` to
`internal/infra/postgres/shared.go` makes it a single authoritative implementation.

### 4. parseUUIDParam and parseDecimalAmount extracted to handler/helpers.go

UUID path-parameter parsing and decimal amount string validation were implemented
independently in the invoice, PO, and GR handler files. Independent implementations
drift: the invoice handler was returning `"invalid invoice id"` while a draft GR
handler returned `"invalid id"`, producing inconsistent error messages for the same
failure class. A single `parseUUIDParam(r, "id")` and `parseDecimalAmount(s)`
in `internal/handler/helpers.go` guarantees that all handlers emit the same error
strings, which matters because client error-handling code and the OpenAPI spec
both reference these exact strings.

### 5. Trailing-zero normalization in parseDecimalAmount

The initial implementation used `amt.Exponent() < -4` to reject amounts with more
than 4 decimal places. This test operates on the decimal's internal representation,
not its mathematical value: the string `"1.23450000"` is stored with exponent `-8`
even though it is numerically equal to `1.2345`, a value with exactly 4 significant
decimal places. Using `!amt.Equal(amt.Truncate(4))` tests whether truncating to 4
decimal places changes the value — it does not for trailing zeros — so valid inputs
with trailing zeros are accepted while genuinely over-precise values like `"1.23456"`
are still rejected.

### 6. NULLIF guard in RLS policies

`current_setting('app.current_tenant_id', true)` suppresses the Postgres error
that would occur if the variable is unset, but it returns an empty string instead.
The RLS expression `tenant_id = current_setting(...)::uuid` then attempts to cast
an empty string to UUID, which throws a Postgres runtime error rather than
evaluating to false. Wrapping the call in `NULLIF(..., '')` converts the empty
string to NULL, making the RLS predicate `tenant_id = NULL`, which evaluates to
FALSE for every row. This ensures that any connection that reaches a query without
first calling `setTenantLocally` is denied all rows instead of causing a 500.

---

## Consequences

### Positive

- The HTTP surface for the Order-to-Pay workflow is now complete: clients can
  create POs and GRs and retrieve them, enabling end-to-end integration tests.
- `setTenantLocally` is a single implementation; RLS activation behaviour is
  uniform and auditable in one place.
- Handler error messages are consistent across all entity types, matching the
  strings referenced in the OpenAPI spec and client SDKs.
- The NULLIF guard converts a class of silent RLS misconfiguration from a runtime
  500 into a safe, silent deny — fail-closed rather than fail-open.
- `parseDecimalAmount` correctly accepts canonical decimal representations with
  trailing zeros, which are common outputs from accounting software and numeric
  libraries.

### Negative / Trade-offs

- `handler/helpers.go` becomes a shared dependency for all handlers. A breaking
  change to `parseUUIDParam` or `parseDecimalAmount` signature requires updating
  every handler that uses it. This is the standard cost of DRY in a flat package
  structure; the alternative (duplication) had a higher ongoing cost.
- The GR use-case ownership check requires an additional SELECT before the INSERT
  on every GR creation, adding one round-trip to the database. Given that GR
  creation is a low-frequency human-initiated action (not a bulk pipeline step),
  this overhead is acceptable.

---

## References

- `internal/domain/po.go` — PORepository interface
- `internal/domain/gr.go` — GRRepository interface
- `internal/usecase/gr/create.go` — PO ownership verification logic
- `internal/infra/postgres/shared.go` — setTenantLocally helper
- `internal/handler/helpers.go` — parseUUIDParam, parseDecimalAmount
- `migrations/003_purchase_orders.sql` — NULLIF guard in RLS policy
- `migrations/004_goods_receipts.sql` — NULLIF guard in RLS policy
