# ADR-009: WMS Multi-Warehouse Architecture, Double-Entry Virtual Locations, and Immutable Stock Ledger

Date: 2026-09-06  
Status: Accepted  

---

## Context

### Background & System Evolution

Tayooli ERP originated as a Cloud B2B Accounts Payable (AP) and financial matching platform. Its early architecture centered on invoice lifecycle management: vendor invoice OCR ingestion (via Tesseract/Kafka), automated 3-way matching (Purchase Order vs. Goods Receipt vs. Invoice), approval workflows, and general ledger journal postings.

In that initial design, inventory tracking was minimal:
1. **Unstructured Inventory Storage**: Inventory was represented by a single flat table (`inventory`) with a free-text `warehouse_location` column.
2. **Absence of Physical Ledger Deductions**: Confirming a Goods Receipt or issuing a Sales Order did not record microscopic physical movements or deduct inventory from physical bins.
3. **No Multi-Warehouse Scoping**: Authorization was scoped strictly at the tenant level. Any user with warehouse privileges could view and alter inventory records across all company facilities, creating severe data isolation and physical security liabilities for multi-branch organizations.
4. **No In-Transit Tracking**: Inter-warehouse transfers were either recorded as instantaneous jumps or managed externally on paper, leading to lost inventory and unaccounted discrepancies during transportation.
5. **No Omnichannel Barcode Resolution**: Products were keyed only to a single internal SKU string. Warehouse workers could not scan standard EAN-13/Code 128 retail barcodes, packaging multi-packs (e.g., box of 12), or channel-specific external SKUs from marketplace integrations (Shopee, Tokopedia, TikTok Shop).

As Tayooli ERP expanded to serve distribution networks, retail chains, and manufacturing hubs across Indonesia, this financial invoice-matching paradigm became insufficient. The platform required a high-throughput, race-free, and forensically auditable Warehouse Management System (WMS).

---

## Decision

We designed and implemented a production-grade WMS module in Go (`internal/domain/wms.go`, `internal/usecase/wms/wms_usecase.go`, `internal/infra/postgres/wms_repo.go`, `internal/handler/wms_handler.go`) and PostgreSQL migration (`021_wms_multi_warehouse_locations.sql`) founded on three architectural pillars:

### 1. Odoo-Inspired Double-Entry Virtual Locations

Stock is never created from vacuum or destroyed into nothingness. Every physical or operational movement is modeled as a double-entry debit/credit transfer between a `source_location_id` and a `dest_location_id`.

In addition to physical internal locations, every tenant automatically provisions system virtual locations:
- `@VENDOR` (`LocationTypeVendor`): Source location for inbound vendor receipts.
- `@CUSTOMER` (`LocationTypeCustomer`): Destination location for outbound sales orders and Delivery Orders (*Surat Jalan*).
- `@TRANSIT` (`LocationTypeTransit`): Floating holding location for goods moving between warehouses.
- `@LOSS` (`LocationTypeLoss`): Offset location for stock opname and inventory count write-offs.
- `@SCRAP` (`LocationTypeScrap`): Quarantine location for damaged, expired, or defective goods.

### 2. ERPNext-Inspired Immutable Stock Ledger Entry (SLE)

Inventory balances are not maintained as directly mutated mutable counters prone to drifting. Instead, all movements are recorded in an append-only ledger table: `stock_movements`.

```sql
CREATE TABLE stock_movements (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    movement_number    VARCHAR(100) NOT NULL,
    product_id         UUID NOT NULL REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    source_location_id UUID NOT NULL REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    dest_location_id   UUID NOT NULL REFERENCES warehouse_locations(id, tenant_id) ON DELETE RESTRICT,
    quantity           NUMERIC(20, 4) NOT NULL CHECK (quantity > 0),
    unit_cost          NUMERIC(20, 4) NOT NULL DEFAULT 0 CHECK (unit_cost >= 0),
    status             stock_movement_status NOT NULL DEFAULT 'DONE',
    reference_type     VARCHAR(50) NOT NULL,
    reference_id       UUID NOT NULL,
    executed_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, movement_number),
    UNIQUE (id, tenant_id)
);
```

- **Append-Only Immutability**: Rows in `stock_movements` are strictly append-only. No `UPDATE` or `DELETE` operations are permitted on posted movements.
- **Deterministic Balance Calculation**: The real-time quantity of any SKU at any location is computed deterministically from posted movements:
  ```sql
  SELECT COALESCE(
      SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE -quantity END),
      0
  )
  FROM stock_movements
  WHERE tenant_id = $1
    AND product_id = $3
    AND (source_location_id = $2 OR dest_location_id = $2)
    AND status = 'DONE';
  ```

### 3. PostgreSQL Transaction Advisory Locks for Race-Free Stock Deduction

High-volume order dispatches create race conditions when multiple operators attempt to deduct stock from the same bin simultaneously. Standard optimistic concurrency control (`version` counters) suffers from high abort rates, while blanket table or row locks on master tables cause deadlocks.

We implemented transaction-level PostgreSQL advisory locks scoped strictly to the `(tenant_id, location_id, product_id)` tuple:

```sql
SELECT pg_advisory_xact_lock(hashtext($1::text || $2::text || $3::text));
```

#### Atomic Execution Protocol (`WMSRepo.DeductLocationStock`)
1. Open transaction (`BEGIN`).
2. Set tenant RLS session context (`setTenantLocally`).
3. Acquire transaction-scoped advisory lock: `pg_advisory_xact_lock(...)`.
4. Calculate current available stock at `source_location_id` from `stock_movements`.
5. If `current_stock < requested_quantity`, abort immediately and return `domain.ErrInsufficientStock` (HTTP 422).
6. Insert immutable movement row into `stock_movements`.
7. Commit transaction (`COMMIT`). The lock is automatically released by PostgreSQL upon commit or rollback.

This protocol guarantees zero negative inventory without holding long-lived application mutexes or causing database-wide lock contention.

### 4. Physical Location Hierarchy & Scoped Multi-Tenant RBAC

Physical infrastructure is modeled hierarchically:
`Regional` $\to$ `Warehouse` $\to$ `WarehouseLocation` (`Zone` $\to$ `Rack` $\to$ `Bin` $\to$ `Pallet/LPN`).

Access control enforces role and warehouse boundaries:
- **`admin` / `owner`**: Full cross-warehouse read and write access across the entire tenant.
- **`regional_manager`**: Read and write access restricted to warehouses within their assigned regional cluster.
- **`warehouse` (staf gudang)**: Scoped strictly to warehouses explicitly assigned to them in `user_warehouses`. Cross-warehouse access and cross-warehouse location spoofing are strictly rejected with `domain.ErrUnauthorizedWarehouse` (HTTP 403).
- **`auditor`**: Tenant-wide read-only access. Any attempt to invoke write or state-changing endpoints (create warehouse, create location, dispatch transfer, receive transfer, dispatch DO) is rejected with `domain.ErrForbidden` (HTTP 403).

### 5. Omnichannel Barcode & SKU Resolution

Warehouse workers scan heterogeneous physical barcodes. The resolution engine (`ResolveBarcode`) resolves any input string through a 3-tier waterfall:
1. **Internal Product SKU** (`products.sku`): Direct 1:1 match (`source: "SKU"`, `multiplier: 1.0`).
2. **Physical Barcode** (`product_barcodes`): EAN-13, EAN-8, Code 128, or QR codes with packaging UOM multipliers (e.g., Box of 24, Carton of 144) (`source: "BARCODE"`).
3. **Omnichannel SKU Mapping** (`product_sku_mappings`): External marketplace SKUs (Shopee, Tokopedia, TikTok Shop, Lazada) or customer-specific SKU aliases (`source: "MAPPING"`).

---

## Alternatives Considered

### Alternative 1: Mutable Balance Row with Optimistic Locking
- **Approach**: Maintain a `current_stock` column in `products` or `inventory` and update with `UPDATE inventory SET stock = stock - X WHERE id = Y AND stock >= X`.
- **Why Rejected**: While simple, it destroys the audit trail. In an ERP, accountants and auditors must know *why*, *when*, *by whom*, and *via what document* every single unit moved. Furthermore, under concurrent dispatch spikes, optimistic retries cause cascading database failures.

### Alternative 2: Kafka Event-Sourced Balance Projections
- **Approach**: Publish stock movement events to Kafka and have an asynchronous consumer update read-optimized projection tables.
- **Why Rejected**: Eventual consistency is hazardous for warehouse dispatches. If an e-commerce flash sale occurs, asynchronous lag of even 200ms allows double-allocation and physical over-selling (negative inventory on the shelf). Warehouse deductions require ACID serializability at the moment of dispatch.

### Alternative 3: Tenant-Only Authorization Scoping
- **Approach**: Allow any warehouse operator within a tenant to select any warehouse dropdown.
- **Why Rejected**: Large enterprises operating warehouses in multiple provinces (e.g., Jakarta, Surabaya, Medan) cannot allow operators in one warehouse to dispatch or receive items intended for another branch. Physical security mandates strict per-warehouse authorization scoping.

---

## Consequences

### Positive Consequences

1. **Strict Multi-Tenancy & Zero Data Leakage**: Enforced at the database layer via PostgreSQL Row-Level Security (`ALTER TABLE ... FORCE ROW LEVEL SECURITY`) and at the application layer via Go JWT middleware context.
2. **Guaranteed Zero Negative Inventory**: Race conditions are structurally prevented by transaction-level advisory locks and atomic ledger balance checks prior to movement insertion.
3. **Forensic Auditability**: Every item movement records `movement_number`, `source_location_id`, `dest_location_id`, `quantity`, `reference_type` (`TRANSFER`, `DELIVERY_ORDER`, `GOODS_RECEIPT`), `reference_id`, and `executed_by`.
4. **Resilient Physical Operations**: Multi-warehouse transfer tracking provides explicit `IN_TRANSIT` status, preventing stock from vanishing during transit.
5. **Seamless Omnichannel Scanning**: Barcode resolution dynamically handles multi-pack conversions and marketplace channel aliases.

### Operational Trade-offs & Mitigations

1. **Advisory Lock Hash Collisions**:
   - *Risk*: `pg_advisory_xact_lock` accepts a 64-bit integer derived from `hashtext($1 || $2 || $3)`. Hash collisions could theoretically serialize independent stock deductions.
   - *Mitigation*: The collision space in 64-bit integer hashing is negligible for operational warehouse concurrency. The lock duration is sub-millisecond, releasing immediately on `COMMIT`.
2. **Ledger Table Growth**:
   - *Risk*: High-velocity warehouses generate millions of rows in `stock_movements`.
   - *Mitigation*: Comprehensive compound indexes (`idx_stock_movements_tenant_src`, `idx_stock_movements_tenant_dst`, `idx_stock_movements_tenant_prod`, `idx_stock_movements_tenant_created`) ensure sub-5ms balance queries. Future partitioning by `tenant_id` and `created_at` can be executed transparently.
