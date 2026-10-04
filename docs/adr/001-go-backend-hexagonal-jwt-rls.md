# ADR-001: Go Backend — Hexagonal Architecture, JWT Multi-tenancy, Dual-Layer RLS, sqlc, and Kafka Invoice Events

Date: 2026-06-26
Status: Accepted

---

## Context

Tayooli ERP is a multi-tenant B2B finance platform.  The Go backend must:

1. Serve multiple tenant organisations from a single deployment without any
   data leakage between them.
2. Scale to high invoice throughput while maintaining low p99 latency.
3. Stay maintainable as the domain grows (new document types, workflow states,
   AI pipeline integrations).
4. Integrate with an async event bus (Apache Kafka) so downstream services
   (Python AI worker, notification service, audit pipeline) can react to
   invoice lifecycle changes.

Several architectural decisions were made during the foundation phase.  This
ADR captures the rationale for each, so future engineers understand the
constraints and trade-offs that led to the current design.

---

## Decision 1 — Hexagonal / Clean Architecture for the Go backend

### What was decided

The codebase is organised into four layers with strict import direction
(outer layers may import inner; inner layers must not import outer):

```
cmd/api/        — entrypoint; wires dependencies and starts HTTP server
internal/
  domain/       — pure Go structs, interfaces, sentinel errors (no imports)
  usecase/      — business logic, orchestrates domain interfaces
  handler/      — HTTP adapter; decodes requests, calls usecases, encodes responses
  infra/        — concrete implementations (postgres, kafka)
```

The `domain` package exposes interfaces (`InvoiceRepository`, `EventPublisher`)
that usecases depend on.  The `infra` package contains the concrete
implementations.  Wiring happens in `cmd/api/main.go`.

### Why

- **Testability**: usecases and handlers can be unit-tested with mock
  implementations without a real database or Kafka broker.
- **Replaceability**: swapping the Postgres driver or the Kafka client requires
  changes only inside `infra/`, not in business logic.
- **Boundary enforcement**: the domain package has zero external dependencies,
  making it trivially auditable.

### Alternatives considered

- **Flat package layout** — rejected; collapses layers and makes dependency
  direction unenforceable as the codebase grows.
- **MVC** — rejected; the "controller" anti-pattern tends to push business
  logic into HTTP handlers, making it hard to test without spinning up an
  HTTP server.

---

## Decision 2 — JWT-based multi-tenancy (not session cookies, not API keys)

### What was decided

Every request to `/api/v1/*` must include an `Authorization: Bearer <JWT>`
header.  The JWT is:

- Signed with HS256 using the `JWT_SECRET` environment variable.
- Required to carry an `exp` claim (the parser uses `WithExpirationRequired()`).
- Required to carry a `tenant_id` claim (UUID string).

The `TenantMiddleware` (`internal/middleware/tenant.go`) validates the token
and places the parsed `tenant_id` into the request context under a typed key.
All downstream code retrieves the tenant ID exclusively from the context via
`middleware.GetTenantID(ctx)`.

### Why

- **Stateless**: no server-side session store needed; horizontal scaling is
  trivial.
- **Self-contained tenant claim**: the tenant boundary is established at the
  network edge (the middleware), not scattered across handler or usecase code.
- **Signing algorithm pinned**: `WithValidMethods([]string{"HS256"})` prevents
  the `alg:none` attack and algorithm-confusion attacks (e.g. RS256/HS256 swap).
- **Expiry enforced**: `WithExpirationRequired()` ensures tokens without `exp`
  are rejected — protects against indefinite access if a token is leaked.

### Alternatives considered

- **Session cookies** — rejected; requires a session store (Redis), adds
  infrastructure complexity, and complicates mobile/API client integrations.
- **API keys** — rejected; long-lived keys are harder to rotate and lack
  built-in expiry semantics; per-tenant key management adds operational burden.
- **RS256 JWT** — deferred; RS256 is preferable when a separate Identity
  Provider issues tokens.  At current scale, a shared HS256 secret is simpler
  and sufficient.  Migration to RS256 (or JWK endpoint verification) is
  straightforward because the middleware is the single validation point.

---

## Decision 3 — Dual-layer tenant isolation: app WHERE clause + PostgreSQL RLS SET LOCAL

### What was decided

Every database operation runs inside an explicit transaction that does two
things before any query:

1. **Application layer**: every SQL query carries a `WHERE tenant_id = $n`
   clause (parameterized).
2. **Database layer**: `SET LOCAL app.current_tenant_id = '<uuid>'` is called
   at the start of each transaction, activating per-table RLS policies:

```sql
CREATE POLICY tenant_isolation ON invoices
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
```

`SET LOCAL` (not `SET SESSION`) scopes the variable to the transaction.  When
the transaction commits or rolls back the variable is cleared automatically,
preventing cross-tenant leakage when connection pool connections are reused.

### Why

- **Defence in depth**: a bug in application-layer filtering (wrong variable,
  missing clause) cannot leak data to another tenant because the database RLS
  policy acts as a second independent check.
- **SET LOCAL over SET SESSION**: `SET SESSION` persists for the lifetime of
  the connection.  With a connection pool, a connection returned to the pool
  after serving tenant A could be reused for tenant B while still carrying
  tenant A's session variable — a critical data-leak vector.
- **No superuser bypass**: RLS policies apply to all roles except `SUPERUSER`.
  The application role must not be granted `SUPERUSER`, ensuring the policies
  cannot be accidentally bypassed.

### Alternatives considered

- **Separate schema per tenant** — rejected; operational complexity (DDL
  migrations must run per-tenant schema) and poor fit for high tenant counts.
- **Separate database per tenant** — rejected; unacceptable infrastructure
  overhead for a SaaS product with many small tenants.
- **Application-layer WHERE only (no RLS)** — rejected; single point of
  failure; a missing clause in one query exposes all tenants' data.

---

## Decision 4 — sqlc for type-safe database queries (not GORM or sqlx)

### What was decided

Raw SQL queries are written in `internal/infra/postgres/` using
`database/sql` and hand-written parameterized queries.  `sqlc` (`sqlc.yaml`)
is configured to generate type-safe Go code from `.sql` query files in
`internal/infra/postgres/queries/`.

### Why

- **No magic**: queries are plain SQL, visible in version control, and
  reviewable without knowing an ORM's DSL.
- **Type safety**: `sqlc` generates strongly-typed scan functions, eliminating
  a class of runtime errors from manual `Scan()` calls.
- **No N+1 risk**: ORM lazy-loading patterns cannot occur; every query is
  explicit.
- **Performance predictability**: the executed SQL is exactly what the engineer
  wrote; no hidden JOIN expansion or SELECT *.

### Alternatives considered

- **GORM** — explicitly excluded by the project tech stack rules; introduces
  ORM magic and has historically caused subtle bugs with soft deletes and
  association loading.
- **sqlx** — considered; `sqlx` adds struct scanning helpers but does not
  generate code.  `sqlc` provides stronger compile-time guarantees.
- **pgx directly** — valid alternative; deferred in favour of `database/sql`
  for broader driver compatibility and simpler mock testing.

---

## Decision 5 — Kafka invoice events with key = invoice UUID

### What was decided

After a successful database write, the `CreateUsecase` and `ApproveUsecase`
publish a domain event to Kafka:

| Usecase  | Topic              | Key          | Payload         |
|----------|--------------------|--------------|-----------------|
| Create   | `invoice.created`  | invoice UUID | full Invoice JSON |
| Approve  | `invoice.approved` | invoice UUID | full Invoice JSON |

The Kafka message key is always the **invoice UUID**.

Publishing is **best-effort**: a Kafka failure does not roll back the database
transaction (`_ = publisher.Publish(...)`).

### Why

- **Partition ordering by entity**: using the invoice UUID as the key ensures
  all events for the same invoice are routed to the same Kafka partition.
  Consumers processing events for a single invoice always see them in order.
- **Load distribution**: different invoice UUIDs hash to different partitions,
  distributing load across the broker cluster.
- **Best-effort publishing rationale**: at-least-once delivery with idempotent
  consumers (checking event type + invoice ID) is the preferred pattern.
  Making DB writes fail because Kafka is temporarily unavailable would degrade
  core invoicing availability for an infra dependency — unacceptable.
  An outbox pattern (DB-native event persistence + transactional outbox
  processor) is the recommended upgrade path for strict at-least-once
  guarantees.

### Alternatives considered

- **Random key / no key**: events for the same invoice could arrive
  out-of-order to consumers — rejected.
- **tenant_id as key**: routes all events for a busy tenant to one partition,
  creating a hot-partition bottleneck — rejected.
- **Synchronous Kafka with DB rollback on failure**: couples DB durability to
  Kafka availability — rejected as availability risk.
- **Transactional outbox**: correct for strict at-least-once; deferred to a
  future iteration because it requires an additional background processor
  and schema changes.

---

## Consequences

### Positive

- Clean separation of concerns makes each layer independently testable.
- Dual-layer tenant isolation provides defence in depth with minimal runtime
  overhead (one `SET LOCAL` per transaction).
- JWT statelessness simplifies horizontal scaling.
- Pinned HS256 algorithm and mandatory `exp` claim close common JWT
  attack vectors.
- Kafka keying by invoice UUID enables ordered, parallelised downstream
  processing.

### Negative / Trade-offs

- Hexagonal wiring adds boilerplate in `main.go`; grows linearly with feature
  count.
- `SET LOCAL` requires every DB operation to open an explicit transaction even
  for read-only queries — slight overhead, mitigated by connection pooling.
- Best-effort Kafka publishing means events can be lost if the broker is down
  during a DB write; an outbox pattern is needed for strict guarantees.
- HS256 requires the JWT secret to be shared with every service that needs to
  verify tokens; RS256 / JWKS is the correct long-term path when a dedicated
  Identity Provider is introduced.

---

## References

- `internal/middleware/tenant.go` — JWT validation and tenant context injection
- `internal/infra/postgres/invoice_repo.go` — dual-layer isolation implementation
- `migrations/001_init_schema.sql` — RLS policy definitions
- `internal/usecase/invoice/create.go` — Kafka publishing (best-effort)
- `internal/usecase/invoice/approve.go` — Kafka publishing (best-effort)
- `sqlc.yaml` — sqlc configuration
