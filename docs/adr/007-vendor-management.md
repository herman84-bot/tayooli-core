# ADR-007: Vendor Management Module

Date: 2026-07-11
Status: Accepted

---

## Context

Tayooli's Order-to-Pay pipeline references vendors extensively: invoices
carry a `vendor_id`, purchase orders are issued to vendors, and goods receipts
track deliveries from vendors. However, there is no first-class vendor master
data module. The `vendor_id` field on invoices and POs is a free-text string
with no referential integrity -- any value can be inserted, and there is no
centralized place to manage vendor contact details, bank accounts, or
performance history.

This creates several problems:

1. **Data quality.** Vendor names are inconsistent across invoices and POs
   ("PT Acme", "Acme Supplies", "ACME-001") because there is no canonical
   source.
2. **Payment risk.** Bank account information is entered ad-hoc on each
   invoice or payment order, increasing the risk of routing payments to the
   wrong account.
3. **No performance tracking.** There is no mechanism to track vendor
   reliability (delivery timeliness, quality) to inform procurement decisions.
4. **No lifecycle management.** Vendors cannot be deactivated when a
   relationship ends. Old vendor references remain dangling.

---

## Decision

### 1. Vendor as a first-class tenant-scoped entity

Vendors are stored in a `vendors` table with `tenant_id` as a mandatory
foreign key, enabling Row-Level Security. Each vendor has a UUID primary key,
a unique `code` (per tenant) for human-readable identification, and optional
bank details for payment routing.

**Rationale:** Vendors are referenced by invoices, POs, and GRs. Making them
a first-class entity with referential integrity (foreign keys from
`invoices.vendor_id`, `purchase_orders.vendor_id`) eliminates data quality
issues and enables joins for reporting.

### 2. Soft-delete instead of hard-delete

The `DELETE /api/v1/vendors/{id}` endpoint sets `is_active = false` rather
than removing the row. This preserves referential integrity for historical
invoices, POs, and goods receipts that reference the vendor.

**Rationale:** Hard-deleting a vendor would break foreign key constraints on
`invoices.vendor_id`, `purchase_orders.vendor_id`, and
`goods_receipts.vendor_id`. Soft-delete avoids cascading deletes or orphaned
references while signaling that the vendor relationship is no longer active.
List endpoints filter on `is_active = true` by default; the detail endpoint
returns soft-deleted vendors so historical records remain viewable.

### 3. Guard against deleting vendors with open transactions

The soft-delete endpoint returns `409 Conflict` when the vendor has open
(non-terminal) purchase orders or pending invoices. This prevents deactivating
a vendor that is mid-transaction.

**Rationale:** Deactivating a vendor with in-flight orders would create
confusion for procurement and finance teams. The guard forces the caller to
resolve or close outstanding transactions first, maintaining data consistency
across the Order-to-Pay pipeline.

### 4. Public rating system (any authenticated user)

Any authenticated tenant user can rate a vendor on a 1-5 scale via
`POST /api/v1/vendors/{id}/rate`. The vendor's running average `rating` and
`rating_count` are updated atomically.

**Rationale:** Vendor performance feedback should be collected from all
stakeholders -- warehouse staff (delivery quality), accountants (invoice
accuracy), procurement (responsiveness) -- not gated behind an admin-only
endpoint. The 1-5 integer scale is simple enough for quick feedback while
providing enough granularity for meaningful ranking.

### 5. RBAC: admin + accountant for writes, admin-only for delete

Create and update operations require `admin` or `accountant` roles. Delete is
restricted to `admin` only. Read and rate endpoints are open to any
authenticated tenant user.

**Rationale:** Vendor creation and updates are accounting-adjacent operations
(accountants manage vendor master data as part of their AP workflow). Delete is
a destructive action (even as soft-delete) that warrants the higher `admin`
gate. Read access is unrestricted because vendor information is needed across
departments (procurement, warehouse, finance). Rating is unrestricted to
encourage broad feedback collection.

### 6. Paginated list with search

`GET /api/v1/vendors` supports `q` (ILIKE search on `name` and `code`),
`page`, and `page_size` query parameters. Response includes `total`, `page`,
and `page_size` for client-side pagination controls.

**Rationale:** Vendor lists can grow to hundreds or thousands of entries for
larger tenants. Full-text search on name and code covers the two most common
lookup patterns (fuzzy name search and exact code lookup). Pagination prevents
unbounded result sets.

### 7. Immutable vendor code

The `code` field is set at creation time and cannot be changed via the update
endpoint. It is normalized to uppercase and unique per tenant.

**Rationale:** The vendor code is used as a stable identifier in integrations,
reports, and cross-references. Allowing code changes would break downstream
systems that cache or reference the code. If a code correction is needed, the
vendor can be soft-deleted and recreated.

### 8. Kafka events for downstream integration

Four topics are published: `vendor.created`, `vendor.updated`,
`vendor.deleted`, `vendor.rated`. Events are keyed by vendor UUID for ordered
per-entity consumption.

**Rationale:** Downstream systems (search indexing, notification service,
analytics) need to react to vendor lifecycle changes. Kafka provides durable,
replayable event delivery consistent with the existing pattern used by
invoice and payment order modules.

---

## Alternatives Considered

### Vendor code as a mutable field

Allowing code changes via PUT. Rejected because vendor codes are used as
stable identifiers in external integrations and reporting. Mutable codes
would require cascading updates across all referencing tables and downstream
systems.

### Hard-delete with cascade

Removing the vendor row and cascading the delete to all referencing invoices,
POs, and GRs. Rejected because it destroys historical financial records,
violates audit trail requirements, and makes month-end close unreliable.

### Role-gated rating (admin-only or purchaser-only)

Restricting the rating endpoint to specific roles. Rejected because vendor
performance feedback should be multi-stakeholder. Warehouse, accounting, and
procurement all interact with vendors and should all be able to contribute
ratings.

### Vendor groups / categories

Adding a `category` or `group` field to organize vendors (e.g. "raw materials",
"services", "logistics"). Deferred to a future iteration. The current scope
covers master data CRUD and basic performance tracking. Category management
can be layered on without schema changes (via a separate `vendor_categories`
table and join).

---

## Consequences

### Positive

- Vendor master data is now a single source of truth, eliminating naming
  inconsistencies across invoices, POs, and GRs.
- Bank account information is centralized and managed by authorized users,
  reducing payment routing errors.
- The rating system provides a lightweight vendor performance signal that
  informs procurement decisions without requiring a separate vendor scorecard
  system.
- Soft-delete preserves referential integrity and audit trails while allowing
  inactive vendors to be excluded from active workflows.
- The open-transaction guard prevents deactivating vendors with in-flight
  orders, avoiding downstream confusion.
- Kafka events enable downstream consumers (search indexing, notifications,
  analytics) without coupling them to the vendor module.

### Negative / Trade-offs

- The running average rating is vulnerable to gaming (e.g. a single user
  submitting repeated ratings). A future iteration could add a "one rating per
  user per vendor" constraint, but this requires tracking rater identity and
  adds complexity deferred for now.

- Soft-deleted vendors remain in the database indefinitely. A future data
  retention policy or archival job may be needed for tenants with high vendor
  churn to prevent table bloat.

- The `code` immutability constraint means typos in the code field require
  soft-delete and recreation, which is a minor friction for administrators.

- The ILIKE-based search on `name` and `code` does not support fuzzy matching,
  typo tolerance, or relevance ranking. For tenants with very large vendor
  lists, a full-text search index (PostgreSQL `tsvector` or external search
  service) may be needed.

---

## References

- `docs/api/vendors.md` -- full API documentation
- `internal/domain/vendor.go` -- VendorRepository interface
- `internal/usecase/vendor/` -- use-case implementations
- `internal/handler/vendor.go` -- HTTP handlers
- `migrations/` -- vendors table and RLS policies
- ADR-001 (Go Backend Hexagonal Architecture) -- establishes the Clean
  Architecture pattern reused here
- ADR-003 (Invoice Status Lifecycle) -- establishes the soft-delete and
  status guard patterns referenced here
- ADR-006 (Payment Order Module) -- establishes the RBAC and Kafka event
  patterns reused here
