# ADR-011: Omnichannel Marketplace Sales Order Ingestion, Canonical Channel Normalization, Idempotent Batch Processing, and In-Place Frictionless SKU Multiplier Resolution

- **Status:** Accepted
- **Date:** 2026-09-09
- **Deciders:** Tayooli ERP Architecture Team, WMS Core Team, Omnichannel Retail Guild
- **Consulted:** Finance & Accounting Core, Logistics & Fulfillment Operations, Security Engineering
- **Informed:** Core Engineering, Product Management

---

## Context

### Background & Operational Problem Statement

As Tayooli ERP expanded to serve high-volume distribution networks, FMCG distributors, and modern retail merchants across Southeast Asia (particularly Indonesia), an increasing proportion of order fulfillment shifted to omnichannel e-commerce platforms: **Shopee, Tokopedia, TikTok Shop, Lazada, and Blibli**.

Prior to this architecture, enterprise clients faced critical operational bottlenecks when fulfilling e-commerce marketplace orders:

1. **Disparate Channel Formats & Reporting Asymmetry:**
   Each marketplace exports order reports with incompatible tabular schemas, conflicting column headers (e.g. `No. Pesanan` in Shopee vs `Order ID` in Tokopedia vs `Order SN` in TikTok Shop), and regional date/currency formatting (such as Indonesian currency prefixes `Rp`, dot-based thousands separators `1.250.000`, and comma-based decimal markers `50.000,00`). Manual transcription was error-prone, labor-intensive, and introduced multi-hour fulfillment delays.

2. **Channel SKU Impedance Mismatch & Bundling Multipliers:**
   Marketplace seller listings frequently use marketing-driven SKUs (e.g. `ABC-KOPISUSU-240ML` or promotional bundles `KOPISUSU-DUS`) that do not correspond 1-to-1 with internal warehouse master SKUs (`MAS000123`). Furthermore, e-commerce orders frequently sell bundle packages (e.g. 1 Carton/Dus = 24 Pcs). Without an automated conversion multiplier engine, warehouse staff either deducted incorrect unit quantities or manually recalculated inventory on paper.

3. **Absence of Import Idempotency & Duplicate Order Hazards:**
   E-commerce fulfillment managers frequently re-download and re-upload daily or weekly order export spreadsheets. In standard tabular import systems lacking channel-level deduplication, re-uploading an export file risks creating duplicate sales orders, inflating financial revenue figures, and triggering catastrophic double-deductions of warehouse physical stock.

4. **Rigid Failure Modes for Unmapped SKUs:**
   In conventional ERP systems, encountering a single unmapped or unrecognized external SKU causes the entire CSV batch upload to abort with a generic error. In a 500-order daily export, a single newly launched promotional item would paralyze shipping operations for all 499 valid orders.

5. **Security Liabilities in File Ingestion:**
   Unbounded file uploads introduce severe Denial-of-Service (DoS) and memory exhaustion risks. Furthermore, external marketplace order exports contain untrusted buyer input (customer names, delivery notes, shipping addresses) that can harbor **CSV Formula Injection (CWE-1236 / DDE Injection)** payloads (e.g. `=cmd|' /C calc'!A0` or `@SUM(...)`). If exported or opened in spreadsheet software like Microsoft Excel or LibreOffice, these formulas can execute arbitrary commands or exfiltrate sensitive customer data.

To resolve these challenges, Tayooli ERP required an enterprise-grade omnichannel sales order ingestion engine capable of bounded streaming, formula sanitization, canonical channel normalization, idempotent batch tracking, multiplier-based stock deduction, and frictionless in-place unmapped SKU resolution.

---

## Decision

We designed and implemented a production-grade Omnichannel Marketplace Sales Order Ingestion and SKU Mapping architecture in Go (`internal/domain/wms.go`, `internal/usecase/wms/wms_usecase.go`, `internal/infra/postgres/wms_repo.go`, `internal/handler/wms_handler.go`) and PostgreSQL migration (`backend/go-core/migrations/023_wms_marketplace_sales_import.sql`).

The solution is founded on five architectural pillars:

```
+-----------------------------------------------------------------------------------------------------------------------+
|                                    OMNICHANNEL SALES ORDER INGESTION ENGINE                                           |
+-----------------------------------------------------------------------------------------------------------------------+
                                                          |
                 +----------------------------------------+----------------------------------------+
                 |                                                                                 |
                 v                                                                                 v
   [JSON Payload / Inline CSV]                                                        [Multipart Form / text/csv]
                 |                                                                                 |
                 +----------------------------------------+----------------------------------------+
                                                          |
                                                          v
                                  +-----------------------------------------------+
                                  | 5MB BOUNDED STREAM READER & DDE SANITIZER     |
                                  | - http.MaxBytesReader(w, r.Body, 5*1024*1024) |
                                  | - Neutralize '=','+','@' -> prepend "'"       |
                                  | - Clean 'Rp', dot thousands, comma decimals   |
                                  +-----------------------------------------------+
                                                          |
                                                          v
                                  +-----------------------------------------------+
                                  | PILLAR 1: OPENOMS CHANNEL NORMALIZATION       |
                                  | - Dynamic Header Synonym Matching             |
                                  | - Canonical Models: MarketplaceOrder & Items  |
                                  | - Financials: Total, Shipping, Fee, Net       |
                                  +-----------------------------------------------+
                                                          |
                                                          v
                                  +-----------------------------------------------+
                                  | PILLAR 2: ERPNEXT BATCH & IDEMPOTENCY GATE    |
                                  | - Session: marketplace_import_batches         |
                                  | - UNIQUE(tenant_id, channel, external_order)  |
                                  | - Deduplication: Skip existing -> failed_orders|
                                  +-----------------------------------------------+
                                                          |
                 +----------------------------------------+----------------------------------------+
                 |                                                                                 |
     [All SKUs Resolved]                                                             [Has Unmapped SKUs]
                 |                                                                                 |
                 v                                                                                 v
+-----------------------------------------------+                                 +-------------------------------+
| PILLAR 3: MULTIPLIER & STOCK DEDUCTION        |                                 | PILLAR 4: FRICTIONLESS SKU RES|
| - Packaging Multiplier: Qty * Multiplier      |                                 | - Set status = UNMAPPED_SKU   |
|   (e.g., 2 Dus * 24 pcs/Dus = 48 pcs)         |                                 | - Increment unmapped_skus     |
| - Acquire Advisory Lock (pg_advisory_xact_lock|                                 | - Stock deduction deferred    |
| - Double-Entry Move: Bin ====> @CUSTOMER      |                                 +-------------------------------+
| - StockMovement ref: MARKETPLACE              |                                                 |
| - If stock insufficient -> STOCK_INSUFFICIENT |                                                 | 1-Click Link
+-----------------------------------------------+                                                 | via SKU Mapping API
                 |                                                                                v
                 |                                                                +-------------------------------+
                 |                                                                | Reprocess In-Place:           |
                 |                                                                | 1. Update order items         |
                 |                                                                | 2. Re-evaluate order          |
                 |                                                                | 3. Deduct stock -> COMPLETED  |
                 +----------------------------------------+-----------------------+-------------------------------+
                                                          |
                                                          v
                                  +-----------------------------------------------+
                                  | PERSISTENCE & AUDIT TRAIL                     |
                                  | - Batch status: COMPLETED / FAILED            |
                                  | - Postgres RLS tenant isolation               |
                                  +-----------------------------------------------+
```

---

### Pillar 1: OpenOMS Channel Normalization & Canonical Domain Models

Marketplace sales channels produce radically heterogeneous data formats. Following design patterns from **OpenOMS** (Open Order Management System), Tayooli ERP decouples external channel format specifics from internal fulfillment logic by translating all incoming payloads into canonical domain entities.

#### 1. Canonical Domain Entities

Three core entities manage the lifecycle:

1. **`MarketplaceImportBatch` (`marketplace_import_batches`)**:
   Tracks an ingestion session, execution metrics, and error rates:
   - `id` (UUID), `tenant_id` (UUID), `batch_number` (string)
   - `channel` (enum: `SHOPEE`, `TOKOPEDIA`, `TIKTOK`, `LAZADA`, `BLIBLI`, `OTHER`)
   - `warehouse_id` (UUID): Target warehouse responsible for fulfilling the batch
   - `file_name` (string): Original source filename
   - `total_orders`, `processed_orders`, `failed_orders`, `unmapped_skus` (integers)
   - `status` (`PENDING`, `PROCESSING`, `COMPLETED`, `FAILED`)
   - `uploaded_by` (UUID), `created_at` (TIMESTAMPTZ)

2. **`MarketplaceOrder` (`marketplace_orders`)**:
   Represents a normalized sales order across any marketplace:
   - `id` (UUID), `tenant_id` (UUID), `batch_id` (UUID, nullable)
   - `warehouse_id` (UUID), `channel` (MarketplaceChannel)
   - `external_order_id` (string): Unique identifier assigned by the marketplace
   - `order_date` (TIMESTAMPTZ)
   - `customer_name`, `customer_phone`, `shipping_address`, `courier`, `tracking_number`
   - `total_amount`, `shipping_fee`, `marketplace_fee`, `net_amount`
   - `status` (`PENDING`, `PROCESSING`, `COMPLETED`, `FAILED`, `UNMAPPED_SKU`, `STOCK_INSUFFICIENT`)
   - `sales_order_id` (UUID, nullable)

3. **`MarketplaceOrderItem` (`marketplace_order_items`)**:
   Represents a single line item within an imported order:
   - `id` (UUID), `tenant_id` (UUID), `order_id` (UUID)
   - `external_sku` (string): Seller SKU string as presented on the channel
   - `product_id` (UUID, nullable): Internal product reference (populated when mapped)
   - `item_name` (string), `quantity` (NUMERIC), `unit_price` (NUMERIC), `subtotal` (NUMERIC)
   - `is_mapped` (boolean): Flag indicating whether the external SKU is linked to a master product

#### 2. Multi-Channel Ingestion Adapters

The endpoint `POST /api/v1/wms/marketplace/import` provides a unified entry point accepting three content representations:
- **`application/json`**: High-performance REST payload for programmatic integrations, carrying pre-structured orders or an inline CSV string (`csv_data`).
- **`multipart/form-data`**: Optimized for drag-and-drop web file uploaders, accepting direct binary CSV uploads along with `warehouse_id` and `channel`.
- **`text/csv`**: Direct streaming upload with query parameters (`?warehouse_id=...&channel=...`).

#### 3. Canonical Financial Normalization

Marketplace fee accounting varies across channels. The normalization engine enforces standard financial balancing:
$$\text{Net Amount} = \text{Total Amount} + \text{Shipping Fee} - \text{Marketplace Fee}$$
If the source report omits line subtotals or gross totals, the engine derives:
$$\text{Subtotal}_i = \text{Quantity}_i \times \text{Unit Price}_i, \quad \text{Total Amount} = \sum_{i} \text{Subtotal}_i$$

---

### Pillar 2: ERPNext Batch Import Session & Idempotency via Composite Unique Key

Inspired by **ERPNext's Data Import Session Architecture**, every import operation is tracked as a first-class batch entity (`marketplace_import_batches`) that guarantees **strict idempotency**.

#### 1. Composite Unique Constraint

At the database schema layer, duplicate order ingestion is prevented via a composite unique constraint:

```sql
ALTER TABLE marketplace_orders 
ADD CONSTRAINT unique_marketplace_orders_tenant_channel_ext_order 
UNIQUE (tenant_id, channel, external_order_id);
```

#### 2. Two-Tier Deduplication Gate

When a batch is uploaded:
1. **Pre-Insert Existence Check:** The application queries `GetMarketplaceOrderByExternalID(ctx, tenantID, channel, externalOrderID)`. If the order already exists for that channel in that tenant:
   - The duplicate order is safely skipped.
   - The batch counter `failed_orders` is incremented.
   - No duplicate order items or stock movements are created.
2. **Database Constraint Fallback:** In high-concurrency environments where duplicate orders are submitted simultaneously in parallel threads, PostgreSQL's `unique_marketplace_orders_tenant_channel_ext_order` constraint catches race conditions, triggering `domain.ErrDuplicateMarketplaceOrder`.

#### 3. Batch Lifecycle & Idempotency Guarantee

- If an entire spreadsheet is re-uploaded (e.g. 100 orders that were already processed):
  - `total_orders = 100`, `processed_orders = 0`, `failed_orders = 100`.
  - Batch status settles as `FAILED`.
  - Warehouse stock balances and historical records remain completely untouched.
- If a delta spreadsheet is uploaded (e.g. 90 already processed, 10 new orders):
  - The 90 duplicates are ignored; the 10 new orders are ingested and fulfilled.
  - `processed_orders = 10`, `failed_orders = 90`, batch status settles as `COMPLETED`.

---

### Pillar 3: Multiplier Resolution & Double-Entry Stock Movement to `@CUSTOMER`

In omnichannel retail, products are regularly sold in wholesale packaging or promotional bundles (e.g. 1 Dus / Carton = 24 cans, 1 Pack = 6 units).

#### 1. Packaging Multiplier Mathematical Model

The mapping between external channel SKUs and internal warehouse master products is governed by `product_sku_mappings`:

```sql
CREATE TABLE product_sku_mappings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id   UUID NOT NULL REFERENCES products(id, tenant_id) ON DELETE RESTRICT,
    mapping_type sku_mapping_type NOT NULL DEFAULT 'MARKETPLACE',
    channel_name VARCHAR(100) NOT NULL,
    external_sku VARCHAR(150) NOT NULL,
    multiplier   NUMERIC(20, 4) NOT NULL DEFAULT 1.0000 CHECK (multiplier > 0),
    ...
    UNIQUE (tenant_id, channel_name, external_sku)
);
```

When an order item with external quantity $Q_{\text{ext}}$ is processed, the physical quantity $Q_{\text{phys}}$ deducted from the warehouse is computed as:
$$Q_{\text{phys}} = Q_{\text{ext}} \times M_{\text{sku}}$$

*Example:* An order of 2 Dus of `KOPISUSU-DUS` with multiplier $M=24$ results in:
$$Q_{\text{phys}} = 2 \times 24 = 48 \text{ Pcs}$$
deducted from internal master product `MAS000123`.

#### 2. Double-Entry Stock Movement to `@CUSTOMER`

In accordance with ADR-009, inventory balances in Tayooli ERP are never mutated via destructive `UPDATE inventory SET qty = qty - x`. Instead, every deduction posts an immutable double-entry record to `stock_movements`:
- **`source_location_id`:** Primary physical storage location inside the selected warehouse (or `@DEFAULT`).
- **`dest_location_id`:** System virtual location `@CUSTOMER` (`LocationTypeCustomer`).
- **`reference_type`:** `MARKETPLACE`.
- **`reference_id`:** `order.ID`.
- **`quantity`:** $Q_{\text{phys}}$ ($48.0000$).
- **`status`:** `DONE`.

#### 3. Advisory Locking & Zero Negative Inventory

To prevent race conditions during concurrent dispatches or parallel batch uploads, the stock deduction engine executes `DeductLocationStock`:
```sql
SELECT pg_advisory_xact_lock(hashtext($tenant_id::text || $location_id::text || $product_id::text));
```
If available stock at the location is less than $Q_{\text{phys}}$:
- The transaction does not abort the entire batch.
- The order status is transitioned to `STOCK_INSUFFICIENT`.
- Warehouse stock remains non-negative.
- The batch session marks the order as processed, allowing warehouse supervisors to replenish stock and fulfill the backorder later.

---

### Pillar 4: In-Place Frictionless Unmapped SKU Resolution

Traditional ERP systems suffer from catastrophic batch rejection: if an export contains unknown SKUs, the entire batch fails. Inspired by **Dolibarr's Product Aliasing Pattern**, Tayooli ERP adopts a **zero-rejection, frictionless deferred resolution** workflow:

```
+-----------------------------------------------------------------------------------------------+
|                       IN-PLACE UNMAPPED SKU RESOLUTION WORKFLOW                               |
+-----------------------------------------------------------------------------------------------+
                                                |
                                                v
               [Batch Upload: Order contains unknown SKU "VIRAL-BOBA"]
                                                |
                                                v
                +---------------------------------------------------------------+
                | Ingest Order:                                                 |
                | - item.product_id = NULL, item.is_mapped = FALSE              |
                | - order.status = 'UNMAPPED_SKU'                               |
                | - batch.unmapped_skus++                                       |
                | - Stock deduction deferred (No inventory deducted)            |
                +---------------------------------------------------------------+
                                                |
                                                | Operator clicks "Map SKU" in UI
                                                v
                +---------------------------------------------------------------+
                | POST /api/v1/wms/marketplace/sku-mappings                     |
                | Payload: {                                                    |
                |   "channel_name": "TIKTOK",                                   |
                |   "external_sku": "VIRAL-BOBA",                               |
                |   "product_id": "b0000000-...",                               |
                |   "multiplier": 1                                             |
                | }                                                             |
                +---------------------------------------------------------------+
                                                |
                                                v
                +---------------------------------------------------------------+
                | AUTOMATIC ATOMIC REPROCESSING:                                |
                | 1. Upsert product_sku_mappings                                |
                | 2. In-place backfill marketplace_order_items:                  |
                |    UPDATE marketplace_order_items                             |
                |    SET product_id = $prod, is_mapped = TRUE                   |
                |    WHERE channel = 'TIKTOK' AND external_sku = 'VIRAL-BOBA'   |
                | 3. Query all pending orders with status 'UNMAPPED_SKU'        |
                | 4. If all items in order are now mapped:                      |
                |    - Deduct stock: Bin ====> @CUSTOMER                        |
                |    - If stock sufficient -> UPDATE order status = 'COMPLETED' |
                |    - If stock insufficient -> status = 'STOCK_INSUFFICIENT'   |
                +---------------------------------------------------------------+
                                                |
                                                v
                [Order Completed Without Re-Uploading File or Editing CSV!]
```

#### Operational Advantages:
1. **Zero Downtime for Fulfillment:** 99% of valid orders in a batch are fulfilled and dispatched immediately.
2. **Zero Re-upload Friction:** Operators do not need to download the CSV, edit columns in Excel, and re-upload. Mapping the SKU once resolves all historical and pending orders automatically.
3. **Continuous Catalog Evolution:** The ERP learns channel aliases progressively as new products launch.

---

### Pillar 5: Security Hardening & Data Sanitization

#### 1. 5MB Bounded Streaming & Memory Protection
To prevent heap exhaustion and denial-of-service (DoS) attacks via multi-gigabyte file uploads:
- Handlers wrap the HTTP request body with Go's `http.MaxBytesReader`:
  ```go
  r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024) // Strictly 5MB
  ```
- If a payload exceeds 5MB, the stream immediately terminates with HTTP `413 Request Entity Too Large` (`"request entity too large"`).

#### 2. CSV Formula Injection Sanitization (CWE-1236 / DDE Injection)
CSV files exported from e-commerce platforms contain arbitrary, untrusted buyer input (e.g. buyer names or delivery notes such as `=cmd|' /C calc'!A0` or `@SUM(...)`). When warehouse supervisors open exported fulfillment records in Microsoft Excel or Google Sheets, spreadsheet engines treat leading formula characters as active formulas.

The ingestion parser sanitizes all incoming string cells before persistence:
```go
sanitizeText := func(val string) string {
    val = strings.TrimSpace(val)
    if len(val) > 0 && (val[0] == '=' || val[0] == '+' || val[0] == '@') {
        return "'" + val
    }
    return val
}
```
Prepending a single quotation mark `'` forces spreadsheet applications to treat the cell strictly as literal text, neutralizing formula execution and data exfiltration vectors.

#### 3. Dual-Layer Multi-Tenant Row-Level Security (RLS)
All marketplace tables enforce PostgreSQL RLS:
```sql
ALTER TABLE marketplace_import_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE marketplace_import_batches FORCE ROW LEVEL SECURITY;

ALTER TABLE marketplace_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE marketplace_orders FORCE ROW LEVEL SECURITY;

ALTER TABLE marketplace_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE marketplace_order_items FORCE ROW LEVEL SECURITY;
```
Every query is scoped both by explicit application parameterization (`WHERE tenant_id = $1`) and PostgreSQL session variables (`current_setting('app.current_tenant_id')`), preventing cross-tenant leakage.

#### 4. Role-Based Access Control (RBAC)
- **`admin`, `owner`**: Unrestricted write and configuration across all warehouses and channels.
- **`regional_manager`**: Write and import access across assigned regional warehouse clusters.
- **`warehouse`**: Write and import access strictly within assigned warehouses (`user_warehouses`).
- **`auditor`**: Read-only inspection of batches, orders, and mappings. Strictly rejected (`403 Forbidden`) from initiating imports or creating SKU mappings.

---

## Alternatives Considered

### 1. Direct Sales Order / Delivery Order Creation vs. Canonical Marketplace Ingestion Staging
- **Alternative:** Directly inserting marketplace orders into existing core tables `sales_orders` and `delivery_orders` without an intermediate staging layer.
- **Rejected Because:**
  - Marketplace orders contain channel-specific metadata (external order IDs, commission fees, platform tracking codes) that would pollute core B2B sales order schemas.
  - An import batch frequently contains invalid or unmapped items. Creating draft sales orders for hundreds of invalid rows clutters the general sales ledger and complicates accounting audits.
  - OpenOMS staging (`marketplace_orders`) provides an isolated sandbox where orders can be validated, held for unmapped SKUs, and reviewed before final journal posting.

### 2. Immediate Batch Abort on Unmapped SKU vs. In-Place Deferred Resolution
- **Alternative:** Rejecting the entire file if any external SKU is not recognized in `products`.
- **Rejected Because:**
  - Marketplace businesses frequently release promotional variants. In peak shopping festivals (11.11, 12.12), a single unmapped SKU would block shipping for hundreds of urgent orders.
  - Deferred in-place resolution allows valid orders to ship immediately while queuing unmapped orders for one-click mapping.

### 3. Destructive Inventory Counter Decrement vs. Double-Entry `@CUSTOMER` Ledger
- **Alternative:** Executing `UPDATE inventory SET stock = stock - qty WHERE product_id = ...`.
- **Rejected Because:**
  - Violates the foundational double-entry principle established in ADR-009.
  - Destroys forensic auditability (inability to track which marketplace order deducted which batch at what timestamp).
  - Susceptible to concurrent race conditions and negative inventory drift.

### 4. Unbounded In-Memory File Buffering vs. 5MB Bounded MaxBytesReader
- **Alternative:** Using `io.ReadAll(r.Body)` without size constraints.
- **Rejected Because:**
  - Vulnerable to heap exhaustion attacks where malicious or malformed multi-gigabyte files crash the Go API service.
  - 5MB accommodates over 50,000 order lines while strictly capping heap utilization.

---

## Consequences

### Positive Consequences
- **Omnichannel Unification:** Seamless fulfillment support for Shopee, Tokopedia, TikTok Shop, Lazada, Blibli, and generic CSV formats.
- **Flawless Idempotency:** Elimination of duplicate orders, revenue double-counting, and phantom stock deductions via composite unique keys `(tenant_id, channel, external_order_id)`.
- **Packaging Accuracy:** Bundling multiplier support ($2 \times 24 = 48\text{ pcs}$) guarantees exact warehouse inventory synchronization.
- **High Operator Velocity:** Frictionless 1-click in-place unmapped SKU resolution eliminates file re-uploads and manual spreadsheet manipulation.
- **Hardened Security:** Built-in 5MB bounded streaming and CSV formula injection sanitization protect backend memory and downstream spreadsheet consumers.
- **Audit Compliance:** Every marketplace fulfillment action generates an immutable `StockMovement` pointing to virtual location `@CUSTOMER`.

### Negative Consequences / Trade-offs
- **Staging Table Storage Overhead:** Retaining `marketplace_orders` and `marketplace_order_items` introduces additional PostgreSQL table storage. *Mitigation:* B-tree indexes on `(tenant_id, created_at)` enable efficient partition archiving for historical batches older than 12 months.
- **Eventual Fulfillment for Unmapped Orders:** Orders in `UNMAPPED_SKU` status remain unfulfilled until an operator maps the SKU. *Mitigation:* Real-time dashboard indicators and batch status counters alert operators immediately after batch ingestion.

---

## Verification & Test Coverage Matrix

The implementation is verified by comprehensive unit and integration test suites:

| Test Case | Target Component | Verifies Architectural Requirement | Status |
|---|---|---|---|
| `AC-1: Idempotent import ignores duplicate orders` | `internal/usecase/wms/wms_test.go` | Composite unique key `(tenant_id, channel, external_order_id)` skips duplicates and prevents double stock deduction | **PASS** |
| `AC-2: Multiplier resolution (2 Dus x 24 = 48 Pcs)` | `internal/usecase/wms/wms_test.go` | Packaging multiplier correctly converts external unit to internal physical deduction ($100 - 48 = 52$) | **PASS** |
| `AC-3: Unmapped SKU handling and in-place resolution` | `internal/usecase/wms/wms_test.go` | Orders with unknown SKUs flag `UNMAPPED_SKU`, defer stock, and reprocess atomically upon mapping creation | **PASS** |
| `AC-4: Stock deduction to @CUSTOMER` | `internal/usecase/wms/wms_test.go` | Double-entry `StockMovement` created with `DestLocation = @CUSTOMER`, `ref = MARKETPLACE`, status `DONE` | **PASS** |
| `Guardrail 3: Insufficient stock flags STOCK_INSUFFICIENT` | `internal/usecase/wms/wms_test.go` | Insufficient stock flags order without failing batch; prevents negative inventory | **PASS** |
| `Security: Auditor cannot import orders or resolve mappings` | `internal/usecase/wms/wms_test.go` | Role `auditor` strictly rejected with `403 Forbidden` for all mutations | **PASS** |
| `Security: 5MB payload limit on import` | `internal/handler/wms_handler_test.go` | Payloads $> 5\text{MB}$ rejected with HTTP `413 Request Entity Too Large` | **PASS** |
| `Security: Formula injection sanitization` | `internal/handler/wms_handler.go` | Text cells starting with `=`, `+`, `@` prepended with `'` | **PASS** |
| `Multi-Tenancy: Cross-tenant isolation` | `internal/usecase/wms/wms_test.go` | Tenant B cannot access, view, or affect Tenant A's batches, orders, or mappings | **PASS** |
