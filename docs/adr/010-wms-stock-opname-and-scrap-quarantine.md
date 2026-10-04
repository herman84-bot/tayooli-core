# ADR-010: WMS Physical Inventory Discrepancy Resolution via Double-Entry Movements to @LOSS, Opname Status Lifecycle, and Quarantine Stock Deduction via @SCRAP with Atomic Advisory Locks

- **Status:** Accepted
- **Date:** 2026-09-06
- **Deciders:** Tayooli ERP Architecture Team, WMS Core Team
- **Consulted:** Finance & Accounting Core, Security Engineering, Logistics & Warehouse Operations
- **Informed:** Core Engineering, Product Management

---

## Context

### Background & Operational Problem Statement

In physical supply chain management and multi-warehouse distribution, recorded book inventory (*system inventory*) and shelf inventory (*physical inventory*) inevitably drift apart over time. The primary drivers of inventory variance include:
1. **Shrinkage & Theft**: Pilferage, lost cartons, or undocumented physical removals.
2. **Breakage & In-Facility Damage**: Drops during forklift handling, crushing in high-bay pallet racking, and liquid spills.
3. **Receipt & Dispatch Miscounts**: Receiving 98 units while documenting 100, or dispatching incorrect product variants during peak shipping hours.
4. **Perishability & Batch Expiration**: Food, cosmetics, or chemicals passing their shelf-life dates while in storage.
5. **Misplacement & Ghost Stock**: Goods placed in incorrect bin locations without scanning barcode updates.

In standard relational database architectures, inventory counts are frequently reconciled via **single-entry direct overwrites**:
```sql
-- DANGEROUS / NON-AUDITABLE PATTERN:
UPDATE inventory
SET quantity = :physical_count, updated_at = NOW()
WHERE warehouse_id = :wh_id AND product_id = :prod_id;
```

This legacy approach introduces catastrophic failures in enterprise ERP environments:
- **Destruction of Forensic Audit Trails**: Overwriting balances destroys historical traceability. External auditors, internal loss prevention teams, and tax inspectors cannot determine *when*, *why*, *in which bin*, or *under whose authority* inventory vanished or appeared.
- **Violation of Double-Entry Conservation Principles**: In accounting (PSAK 14 / IAS 2 Inventories), inventory value cannot vanish into or appear from nowhere. Every physical inventory change corresponds to an inventory shrinkage expense (*Beban Selisih Stok / Inventory Shrinkage Loss*) or inventory surplus gain. Overwriting a quantity column breaks double-entry balance conservation.
- **Ghost Stock Pick Allocation**: If damaged or spoiled goods remain in standard picking bins, the order routing system continues allocating them to new Sales Orders and Delivery Orders (*Surat Jalan*), resulting in customer delivery rejections and reverse logistics costs.
- **Concurrency & Negative Inventory Vulnerabilities**: If an operator reports damaged stock while concurrent picker processes are dispatching from the same bin, race conditions cause stock counters to plummet below zero (*negative inventory*).

To address these requirements, Tayooli ERP must support:
1. Periodic and continuous physical cycle counts (**Stock Opname**) with mathematical reconciliation via double-entry movements.
2. An explicit state machine lifecycle ensuring that opname sessions are auditable, verifiable, and permanently immutable upon completion.
3. A formal quarantine deduction system (**Stock Scrap**) that isolates damaged inventory into dedicated virtual or physical quarantine locations using atomic PostgreSQL advisory locks.

---

## Decision

We designed and implemented a production-grade Stock Opname and Scrap Quarantine architecture in Go (`internal/domain/wms.go`, `internal/usecase/wms/wms_usecase.go`, `internal/infra/postgres/wms_repo.go`, `internal/handler/wms_handler.go`) and PostgreSQL migration (`backend/go-core/migrations/022_wms_stock_opname_and_scrap.sql`).

The solution is founded on four architectural decisions:

```
+---------------------------------------------------------------------------------------------------+
|                                  WMS INVENTORY LEDGER SYSTEM                                      |
+---------------------------------------------------------------------------------------------------+
                                                  |
           +--------------------------------------+--------------------------------------+
           |                                                                             |
           v                                                                             v
+-------------------------------+                                         +-------------------------------+
|     STOCK OPNAME CYCLE        |                                         |   DAMAGED GOODS / QUARANTINE  |
+-------------------------------+                                         +-------------------------------+
| 1. Create Opname (DRAFT)      |                                         | 1. Identify Damaged Units     |
| 2. Count Bins (IN_PROGRESS)   |                                         | 2. Acquire Advisory Xact Lock |
| 3. Add Line Items             |                                         | 3. Verify Available Stock     |
|    - System Qty from Ledger   |                                         | 4. Deduct Source Bin          |
|    - Physical Qty from Count  |                                         | 5. Post to @SCRAP / Bay       |
|    - Compute Discrepancy      |                                         | 6. Insert stock_scraps Log    |
| 4. Supervisor Complete/Approve|                                         +-------------------------------+
+-------------------------------+                                                        |
           |                                                                             |
           | Double-Entry Reconciliation                                                | Atomic Transfer
           v                                                                             v
+---------------------------------------------------------------------------------------------------+
|                                     IMMUTABLE STOCK LEDGER                                        |
|                                       (stock_movements)                                           |
+---------------------------------------------------------------------------------------------------+
|  Surplus (Discrepancy > 0):  [@LOSS]           =====================>   [Physical Bin Location]   |
|  Deficit (Discrepancy < 0):  [Physical Bin Location]  ===============>   [@LOSS]                  |
|  Quarantine Scrap:          [Physical Bin Location]  ===============>   [@SCRAP / Quarantine Bay]|
+---------------------------------------------------------------------------------------------------+
```

---

### 1. Physical Discrepancy Resolution via Double-Entry Movements to `@LOSS`

Under the immutable double-entry paradigm established in ADR-009, stock is never mutated in place. All balance adjustments are executed as append-only entries in `stock_movements`.

Every tenant in Tayooli ERP provisions a system-wide virtual location:
- **`@LOSS` (`LocationTypeLoss`)**: Represents the tenant's inventory variance and discrepancy clearing account.

#### Mathematical Formulation & Reconciliation Rules

For each item $i$ recorded during a stock opname session:
1. **System Quantity ($Q_{sys}$)**: Dynamically computed from all posted ledger movements at that location:
   $$Q_{sys} = \sum_{\text{dest}=L} \text{qty} - \sum_{\text{src}=L} \text{qty} \quad (\text{where } \text{status} = \text{'DONE'})$$
2. **Physical Quantity ($Q_{phys}$)**: The actual physical count entered by the warehouse operator ($Q_{phys} \ge 0$).
3. **Discrepancy Quantity ($\Delta$)**:
   $$\Delta = Q_{phys} - Q_{sys}$$

When the stock opname is completed via `POST /api/v1/wms/opnames/{id}/complete`, the reconciliation engine executes the following logic:

#### Case A: Positive Discrepancy / Surplus Recovery ($\Delta > 0$)
- Physical items exceed system records (unrecorded return, misplaced stock found).
- **Movement Direction**: System virtual location `@LOSS` $\to$ Physical Location (`item.location_id`).
- **Movement Quantity**: $\Delta$.
- **Attributes**:
  - `movement_number`: `OPN-SURPLUS-<opname_number>-<index>`
  - `reference_type`: `OPNAME`
  - `reference_id`: `opname.id`
  - `status`: `DONE`
  - `executed_by`: Caller user ID (`approved_by`)
- **Ledger Impact**: The physical bin balance increases by $\Delta$, reconciling it exactly to $Q_{phys}$. The virtual `@LOSS` location reflects an offsetting negative balance (surplus gain).

#### Case B: Negative Discrepancy / Shrinkage Write-Off ($\Delta < 0$)
- Physical items are less than system records (theft, unrecorded damage, shrinkage).
- **Movement Direction**: Physical Location (`item.location_id`) $\to$ System virtual location `@LOSS`.
- **Movement Quantity**: $|\Delta|$.
- **Attributes**:
  - `movement_number`: `OPN-LOSS-<opname_number>-<index>`
  - `reference_type`: `OPNAME`
  - `reference_id`: `opname.id`
  - `status`: `DONE`
  - `executed_by`: Caller user ID (`approved_by`)
- **Ledger Impact**: The physical bin balance decreases by $|\Delta|$, reconciling it exactly to $Q_{phys}$. The virtual `@LOSS` location reflects an offsetting positive balance (accumulated inventory loss).

#### Case C: Zero Discrepancy ($\Delta = 0$)
- Physical count perfectly matches system quantity.
- **Action**: No ledger movement is generated, preventing synthetic database bloat while certifying count accuracy.

---

### 2. Stock Opname Status Lifecycle & Immutability

To guarantee data integrity and support segregation of duties, stock opname follows a deterministic state machine:

```
       +---------------------------------------------+
       |                                             |
       v                                             |
   [ DRAFT ] -----------------> [ IN_PROGRESS ]      |
       |                              |              |
       |                              |              |
       +--------------+---------------+              |
                      |                              |
                      v (POST /{id}/complete)        v (Void / Cancel)
               [ COMPLETED ]                   [ CANCELLED ]
             (SEALED / IMMUTABLE)             (NO MOVEMENTS)
```

#### Status Enum (`stock_opname_status`)
1. **`DRAFT`**: Initial session initialized by warehouse staff via `POST /api/v1/wms/opnames`. Header contains `warehouse_id`, auto-generated or custom `opname_number`, and `conducted_by` set to the initiating operator.
2. **`IN_PROGRESS`**: Physical verification underway. Warehouse operators scan bins and products, submitting line items via `POST /api/v1/wms/opnames/{id}/items`.
3. **`COMPLETED`**: Finalized and reconciled by warehouse supervisor or manager via `POST /api/v1/wms/opnames/{id}/complete`.
   - Discrepancy movements are generated to/from `@LOSS`.
   - `approved_by` is stamped with the authenticated user ID.
   - `updated_at` is updated to transaction completion timestamp.
4. **`CANCELLED`**: Opname session aborted without posting adjustments.

#### Immutability & Concurrency Guardrails
- **Post-Completion Lockout**: Once an opname reaches `COMPLETED` or `CANCELLED`, all modification attempts (`POST /api/v1/wms/opnames/{id}/items` and re-invoking `POST /api/v1/wms/opnames/{id}/complete`) are rejected immediately with `domain.ErrInvalidOpnameStatus` (HTTP 400).
- **Physical Count Validation**: `physical_qty` must be non-negative ($\ge 0$). Submissions with negative quantities return `domain.ErrInvalidInput` (HTTP 400).
- **Warehouse Boundary Enclosure**: Every location submitted in `AddOpnameItem` is verified against the opname's `warehouse_id`. Attempting to record items against locations belonging to a different warehouse is rejected with `domain.ErrUnauthorizedWarehouse` (HTTP 403), preventing cross-warehouse data corruption.

---

### 3. Quarantine Stock Deduction via `@SCRAP` with Atomic Advisory Locks

When goods are physically damaged, spoiled, or expired, warehouse personnel must immediately isolate them to prevent pickers from allocating them to active orders.

We established the **Stock Scrap** quarantine pipeline (`POST /api/v1/wms/scraps`):

#### Quarantine Destinations
1. **Default Tenant Virtual Scrap Location (`@SCRAP`)**:
   - System virtual location of type `LocationTypeScrap`.
   - Used for zero-configuration, company-wide damaged goods write-offs.
2. **Dedicated Physical Quarantine Bay / Scrap Cage**:
   - Operators can specify an explicit `scrap_location_id`.
   - If specified, the engine validates that the target location exists, is marked as `SCRAP` or is an internal location belonging strictly to the same warehouse (`targetLoc.WarehouseID == req.WarehouseID`).
   - Prevents shipping damaged goods across warehouse boundaries without a formal transfer.
   - **Self-Transfer Prohibition**: Enforces `source_location_id <> scrap_location_id` at both application layer (`domain.ErrInvalidInput`) and database schema constraint (`chk_scrap_distinct_locations`).

#### Atomic Advisory Locking Protocol (`WMSRepo.DeductLocationStock`)

To eliminate race conditions where multiple operators or concurrent dispatch jobs attempt to scrap or ship the same inventory simultaneously, the deduction executes inside an isolated transaction using a 64-bit PostgreSQL transaction advisory lock:

```sql
SELECT pg_advisory_xact_lock(hashtext($tenant_id::text || $source_location_id::text || $product_id::text));
```

#### Execution Steps:
1. Open database transaction (`BEGIN`).
2. Set PostgreSQL RLS context: `SELECT set_config('app.current_tenant_id', $1, true)`.
3. Acquire transaction-scoped advisory lock on `(tenant_id, source_location_id, product_id)`. Any concurrent dispatch or scrap on the exact same SKU-bin blocks until this transaction completes.
4. Dynamically compute current available stock from `stock_movements` for the locked location.
5. **Ledger Validation**: If `current_stock < requested_quantity`, abort immediately, roll back transaction, and return `domain.ErrInsufficientStock` (HTTP 422).
6. Insert double-entry movement into `stock_movements`:
   - `source_location_id`: Source storage bin.
   - `dest_location_id`: Target quarantine location (`@SCRAP` or physical quarantine bay).
   - `quantity`: Requested scrap quantity.
   - `reference_type`: `SCRAP`.
   - `reference_id`: `stock_scraps.id`.
   - `status`: `DONE`.
7. Insert audit record into `stock_scraps` table capturing `warehouse_id`, `product_id`, `quantity`, mandatory `reason`, and `reported_by`.
8. Commit transaction (`COMMIT`), automatically releasing the advisory lock.

This protocol guarantees **zero negative inventory** and ensures damaged items are immediately deducted from pickable stock.

---

### 4. Database Schema & Row-Level Security (RLS)

All tables are implemented in PostgreSQL migration `022_wms_stock_opname_and_scrap.sql`:

```sql
-- 1. stock_opnames: Session headers
CREATE TABLE IF NOT EXISTS stock_opnames (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    warehouse_id   UUID NOT NULL,
    opname_number  VARCHAR(100) NOT NULL,
    status         VARCHAR(50) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED')),
    conducted_by   UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    notes          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (tenant_id, opname_number),
    UNIQUE (id, tenant_id)
);

-- 2. stock_opname_items: Line items & count comparisons
CREATE TABLE IF NOT EXISTS stock_opname_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opname_id       UUID NOT NULL,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id      UUID NOT NULL,
    location_id     UUID NOT NULL,
    system_qty      NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (system_qty >= 0),
    physical_qty    NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (physical_qty >= 0),
    discrepancy_qty NUMERIC(20, 4) NOT NULL DEFAULT 0,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (opname_id, tenant_id) REFERENCES stock_opnames(id, tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    UNIQUE (id, tenant_id)
);

-- 3. stock_scraps: Quarantine audit ledger
CREATE TABLE IF NOT EXISTS stock_scraps (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    scrap_number       VARCHAR(100) NOT NULL,
    warehouse_id       UUID NOT NULL,
    product_id         UUID NOT NULL,
    source_location_id UUID NOT NULL,
    scrap_location_id  UUID NOT NULL,
    quantity           NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    reason             TEXT NOT NULL,
    reported_by        UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (product_id, tenant_id) REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (source_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (scrap_location_id, tenant_id) REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    CONSTRAINT chk_scrap_distinct_locations CHECK (source_location_id <> scrap_location_id),
    UNIQUE (tenant_id, scrap_number),
    UNIQUE (id, tenant_id)
);
```

#### Row-Level Security (RLS) Enforcement
Each table enables and forces PostgreSQL RLS:
```sql
ALTER TABLE stock_opnames ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_opnames FORCE ROW LEVEL SECURITY;
CREATE POLICY stock_opnames_tenant_isolation_select ON stock_opnames
    FOR SELECT USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);
-- Repeated for INSERT, UPDATE, DELETE across all three tables
```
This guarantees complete tenant isolation even if application code contains logic defects.

---

### 5. Granular Role-Based Access Control (RBAC)

Access control operates across tenant, regional, and warehouse assignment dimensions:

| Role | Warehouse Read Permissions | Warehouse Write Permissions | Opname / Scrap Authorization |
|---|---|---|---|
| `admin`, `owner` | Tenant-wide across all warehouses | Tenant-wide across all warehouses | Full authority to initiate, record counts, approve/complete opnames, and execute scraps. |
| `regional_manager` | All warehouses within assigned regional cluster | All warehouses within assigned regional cluster | Can manage and complete opnames and scraps for facilities in their region. |
| `warehouse` (staf gudang) | Only warehouses assigned in `user_warehouses` | Only warehouses assigned in `user_warehouses` | Can conduct opnames and report scraps strictly within assigned facilities. |
| `auditor` | Tenant-wide read-only | **Strictly forbidden** (`403 Forbidden`) | Allowed to inspect opname discrepancies and scrap history; barred from creating, editing, or completing sessions. |

---

## Alternatives Considered

### Alternative 1: Single-Entry Direct Quantity Override
- **Approach**: Update the existing inventory counter directly (`UPDATE inventory SET stock = physical_qty`).
- **Why Rejected**:
  - Destroys forensic traceability. Accounting and compliance auditors cannot determine what happened to missing stock.
  - Breaks the double-entry invariant where every debit has an equal credit.
  - Fails international financial reporting standards (IFRS / PSAK 14).

### Alternative 2: Discarding Damaged Goods Directly via Loss Write-Off (Skipping Quarantine)
- **Approach**: Route scrap immediately to `@LOSS` instead of maintaining an explicit `@SCRAP` quarantine concept.
- **Why Rejected**:
  - Discrepancy from shrinkage (unknown loss) is fundamentally distinct from documented physical damage or hazardous waste disposal.
  - Physical scrap often requires salvage inspection, return-to-vendor (RMA) credit claims, or destruction certificates under environmental regulations. Quarantining items in `@SCRAP` or physical quarantine bays keeps them visible for operational disposition before final accounting write-off.

### Alternative 3: Optimistic Concurrency Control (OCC) with Retry Loops on Scrap Deductions
- **Approach**: Use a version column or conditional update `WHERE stock >= qty` and retry on conflict.
- **Why Rejected**:
  - High contention occurs when high-volume picking coincides with batch quarantine sweeps (e.g. discovering a compromised pallet of 500 items).
  - OCC retry storms saturate database CPU and connection pools. PostgreSQL advisory locks provide sub-millisecond serialization per SKU-bin with deterministic lock release on transaction commit.

---

## Consequences

### Positive Consequences
1. **Mathematical Ledger Integrity**: The double-entry conservation law is preserved across all warehouse operations. The sum of all location deltas across the tenant equals zero.
2. **Forensic Audit Readiness**: Every unit adjusted via Stock Opname or quarantined via Scrap records exact operator IDs, document reference numbers, timestamps, and reason descriptions.
3. **Zero Negative Inventory**: Race conditions during scrap quarantine deductions are eliminated by PostgreSQL transaction advisory locks.
4. **Segregation of Duties**: Separation between count recording (`conducted_by`) and supervisor approval (`approved_by`) prevents rogue inventory inflation or concealment of theft.
5. **Operational Safety**: Damaged inventory is immediately subtracted from pickable stock, eliminating outbound fulfillment errors.

### Operational Trade-offs & Mitigations
1. **Ledger Table Growth**:
   - *Trade-off*: Large enterprise cycle counts across thousands of SKUs generate numerous rows in `stock_movements`.
   - *Mitigation*: Zero-discrepancy items ($\Delta = 0$) do not generate movement records. Compound indexes on `(tenant_id, dest_location_id, product_id, status)` ensure sub-5ms balance queries.
2. **Advisory Lock Collision Space**:
   - *Trade-off*: `hashtext(...)` maps string identifiers to a 64-bit signed integer for `pg_advisory_xact_lock`.
   - *Mitigation*: The collision probability on 64-bit integers is negligible ($< 10^{-9}$ for millions of combinations). Furthermore, transaction duration is sub-millisecond, meaning even a hash collision causes only a transient wait without deadlocks.

---

## References & Code Map

- **Migration**: `backend/go-core/migrations/022_wms_stock_opname_and_scrap.sql`
- **Domain Definitions**: `backend/go-core/internal/domain/wms.go`
- **Usecase Logic**: `backend/go-core/internal/usecase/wms/wms_usecase.go`
- **PostgreSQL Repository**: `backend/go-core/internal/infra/postgres/wms_repo.go`
- **HTTP Handlers**: `backend/go-core/internal/handler/wms_handler.go`
- **API Specification**: `docs/api/wms.md`
- **Preceding Architecture Decision**: `docs/adr/009-wms-multi-warehouse-and-immutable-stock-ledger.md`
