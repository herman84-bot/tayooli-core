# ADR-014: Enterprise WMS Inbound & Outbound Lifecycle, Batch/FEFO Ledger Lineage, and Staging-to-Bin Physical Traceability

- **Status:** Accepted
- **Date:** 2026-10-07
- **Deciders:** Tayooli ERP Architecture Team, WMS Core Team, Logistics & Warehouse Operations
- **Consulted:** Supply Chain Compliance, Quality Assurance, Frontend Architecture Guild
- **Informed:** Core Engineering, Product Management
- **Supersedes / Extends:** Extends ADR-009 (Multi-Warehouse Ledger) & ADR-012 (Logistics Document Engine)

---

## 1. Context & Operational Background

Under Indonesian logistics, retail distribution, and ISO 9001:2015 quality assurance standards, warehouse operations cannot treat goods movements as instantaneous single-step digital balance updates. Physical stock transitions across discrete operational custody checkpoints:
1. **Physical Pre-Receiving & Unloading Buffer:** Goods entering a distribution facility do not land immediately on storage racks; they sit in a receiving buffer (*Staging Inbound*) while undergoing carton counts, outer-wrapper damage inspection, and quarantine sorting.
2. **Product Lineage (Batch/Lot & Expiry Date):** For food, pharmaceuticals, and fast-moving consumer goods (FMCG), selling or transferring stock without tracking batch numbers and expiration dates violates health regulations and risks substantial financial write-offs. Picking must strictly enforce **FEFO** (*First Expired, First Out*).
3. **Defective Goods Isolation:** Goods damaged during transit cannot be unceremoniously discarded into scrap; they require formal documentation via *Berita Acara Kerusakan (BAK)*, photograph evidence, driver sign-off, and isolation in dedicated `@QUARANTINE` locations.
4. **Outbound 100% Packing Verification:** Dispatches must pass an automated SKU scan validation at the packing bench to achieve zero-defect packing before thermal Airway Bill (AWB) generation.

Historically, `tayooli-core` handled stock receipts and delivery orders as single-step direct mutations (`DRAFT` &rarr; `POSTED` / `SHIPPED`). While sufficient for basic inventory tracking, this does not satisfy enterprise 3PL distribution requirements.

---

## 2. Architectural Decisions & Invariants

### Invariant 1: Append-Only Ledger with Batch Lineage
- The `stock_movements` table remains an immutable, append-only double-entry ledger.
- A foreign key `batch_id UUID REFERENCES stock_batches(id)` is introduced on `stock_movements`.
- **(Owner decision 2026-10-07: traceability is per batch, for ALL products.)** Every ledger row that moves physical stock between locations (receipt, putaway, transfer, pick, dispatch, POS sale, marketplace, opname adjustment, scrap/quarantine) MUST carry `batch_id`. There is no "lot tracking disabled" exemption.
- Products without a supplier batch number receive an auto-generated batch per receipt line (`AUTO-<GR number>-<line>`), so even untracked goods remain traceable to their receipt.
- Existing (pre-migration) stock balances are backfilled into one `LEGACY-<product>` batch per product and location so the NOT NULL rule holds from day one.

### Invariant 1b: Bidirectional Inbound ↔ Outbound Traceability
- Lineage chain: `stock_receipt (GR) → stock_batch → putaway → [transfer] → pick → delivery_order (DO) / POS order → customer`.
- **Forward trace (from a receipt):** the system must answer, for a GR or batch, how much went out, to which customers, through which DOs or POS orders, when, by whom, and how much remains in which rack.
- **Backward trace (from an outbound document):** the system must answer, for a DO line, which batch(es), expiry, GR, supplier/source, and rack it came from.
- Both traces are pure queries over `stock_movements` (`batch_id` + `reference_type/reference_id`). No denormalized copies that could drift.
- A DO line that picks from several batches is split into one line per batch (`delivery_order_items.batch_id NOT NULL`).
- Enforcement: a DB-level `CHECK`/`NOT NULL` on `stock_movements.batch_id` plus Go tests that fail when any movement path writes without a batch.

### Invariant 1c: Single "Barang Masuk & Keluar" Module (UI)
- **(Owner decision 2026-10-07)** Inbound and Outbound live in ONE menu module, **"Barang Masuk & Keluar"**, at route `/wms/arus-barang`, with a `MASUK | KELUAR` toggle at the top.
- `MASUK` shows: receipts → QC → putaway (tabs). `KELUAR` shows: Surat Jalan → picking → packing → manifest/loading (tabs).
- The last selected mode is remembered (`?mode=masuk|keluar` in the URL plus localStorage).
- Legacy routes `/wms/inbound` and `/wms/delivery-orders` redirect to `/wms/arus-barang?mode=masuk` and `?mode=keluar`.
- Merging former modules 3 (Barang Masuk) and 5 (Surat Jalan) reduces the core module count from 13 to **12**.
- Back-dating or destructive editing of existing ledger movements is strictly prohibited.

### Invariant 2: Two-Step Inbound Movement (Staging Gate)
- Receiving goods from external suppliers (`VENDOR`) or production lines (`PRODUCTION`) must first book movements into a warehouse location of type `STAGING_INBOUND`.
- Stock in `STAGING_INBOUND` is accounted for in financial valuation but is marked as **Unallocated / In-Transit** (not available for sales picking).
- Transition to **Available for Sale** occurs exclusively when a `putaway_task` moves stock from `STAGING_INBOUND` to a definitive rack location (`location_type = 'INTERNAL'`).

### Invariant 3: Formal Quarantine & Discrepancy Governance
- Rejected or damaged items during Inbound QC must move to a location of type `QUARANTINE`.
- Creation of a quarantine ledger entry requires an associated `qc_inspections` record and a formal *Berita Acara Kerusakan (BAK)* document with driver signature.
- Direct movement from incoming shipments to `@SCRAP` is disallowed without an intervening QC inspection.

### Invariant 4: FEFO Picking & Packing Station Verification
- Outbound sales orders and delivery orders automatically reserve inventory lines ordered by `expiry_date ASC, created_at ASC` (FEFO).
- Packing benches operate with an interactive barcode validation terminal: a Delivery Order cannot transition to `PACKED` until 100% of line items are physically scanned and verified.
- Discrepancies generate a *Shortage Ticket* and initiate an automated re-pick from backup storage.

### Invariant 5: Dual Print Engine (Laser A4 + Direct Thermal 100x150mm)
- Official legal documents (Surat Jalan 3-ply, Manifest Angkutan, and Berita Acara Kerusakan) use client-side browser print rendering on standard A4 paper with isolated `@media print` stylesheets.
- Logistics labels (Pallet LPN and Carrier AWB) use high-contrast monochrome layouts calibrated for standard $100\text{mm} \times 150\text{mm}$ (4" x 6") direct thermal label printers.

---

## 3. Database Schema Blueprint

```
                      +-------------------+
                      |   stock_batches   |
                      |-------------------|
                      | id (PK)           |
                      | product_id (FK)   |
                      | batch_number      |
                      | expiry_date       |
                      +---------+---------+
                                | 1
                                |
                                | N
+-------------------+ 1       N |       1 +---------------------+
| stock_receipts    +-----------+---------+ stock_movements     |
|-------------------|                     |---------------------|
| id (PK)           |                     | id (PK)             |
| status            |                     | source_location_id  |
| ...               |                     | dest_location_id    |
+---------+---------+                     | batch_id (FK)       |
          | 1                             | quantity            |
          |                               +---------------------+
          | N                                        ▲
+---------+-----------+                              │
| putaway_tasks       +------------------------------+
|---------------------|  (Generates movements from
| staging_location_id |   STAGING to INTERNAL Bin)
| confirmed_bin_id    |
| batch_id (FK)       |
+---------------------+
```

---

## 4. Anti-Hallucination & AI Continuation Protocol

To preserve system integrity across multiple development sessions and AI agents:
1. **Never bypass Staging or Batch constraints:** An agent must not write code that directly dumps incoming items into arbitrary internal bins without respecting the two-step staging workflow.
2. **Never delete or recreate core ledger tables:** Modifications must take the form of incremental PostgreSQL migrations (`033_...`, `034_...`, `035_...`).
3. **Mandatory Test Verification:** Every pull request or milestone commit must include automated unit tests in Go (`go test ./...`) and frontend component checks (`npm run test`) before deployment.

---

## 5. Consequences

### Positive
- Full compliance with client SOP documents (`ALUR PROSES INBOUND.pdf` & `ALUR PROSES OUTBOUND.pdf`) and ISO 9001:2015.
- Accurate lot-level expiry tracking, eliminating accidental shipments of expired products.
- Seamless audit trail for carrier handovers, damaged goods claims, and warehouse throughput KPIs.

### Negative / Trade-Offs
- Increased user interactions: operators must perform two scan steps (Staging & Putaway) instead of a single form confirmation.
- Additional database tables and migration overhead.
- Requires physical handheld barcode terminals (PDA) and thermal label printers on the warehouse floor for maximum efficiency.
