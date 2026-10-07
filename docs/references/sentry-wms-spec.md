# Benchmark Reference: Sentry WMS (hightower-systems/sentry-wms)
## Technical Architecture, Schema Blueprint & Endpoint Patterns

- **Source Repository:** `https://github.com/hightower-systems/sentry-wms`
- **License:** Apache 2.0
- **Purpose:** Primary engineering reference for physical warehouse execution, barcode scanning workflows, and REST API contracts in `tayooli-core`.

---

## 1. Core Architectural Pillars

### 1.1 Bin Type Separation (Staging vs Pickable)
Sentry WMS strictly controls picking eligibility through bin types:
- **`Staging`:** Non-pickable bin. Used for Inbound Dock receiving and QC holding. Inventory lands here immediately upon physical arrival. Inventory in `Staging` is financially accounted for, but the order allocation engine cannot allocate items from it.
- **`PickableStaging`:** Pickable buffer. Cross-docking staging area where pickers can pull fresh inventory before formal putaway.
- **`Pickable`:** Standard warehouse shelf bins, pallet racks, and bulk storage. Valid target for putaway and picking.

*Adaptation in Tayooli Core:*  
Mapped to `warehouse_locations.type` with enum values `'STAGING_INBOUND'`, `'STAGING_OUTBOUND'`, `'QUARANTINE'`, and `'INTERNAL'` (pickable).

### 1.2 Two-Step Inbound (Receiving &rarr; Putaway)
Inbound is split into two atomic operations:
1. **Receive:** PO barcode scan &rarr; Item count validation &rarr; Stock books into `Staging` bin.
2. **Putaway:** Operator takes items from `Staging` &rarr; System suggests optimal storage bin &rarr; Operator scans destination bin barcode &rarr; Stock transfers to `Pickable` bin.

### 1.3 Split Packing vs Shipping Stations
Sentry WMS enforces a strict physical separation between packing benches and shipping docks:
1. **Packing Station (`/api/packing`):**
   - Operator scans order barcode.
   - Operator scans each physical product barcode (100% SKU match validation).
   - Prevents mispicks from being sealed in boxes.
   - Outputs: Sealed package & Packing Slip.
2. **Shipping Desk (`/api/shipping`):**
   - Operator enters carrier, tracking number, and weight.
   - Outputs: Carrier Airway Bill (AWB) thermal label.

---

## 2. API Endpoint Contracts (Adopted by Tayooli Core)

### Receiving Surface
```text
GET  /api/receiving/po/<barcode>     # Scan PO -> Expected line items
POST /api/receiving/receive          # Book items into staging bin
POST /api/receiving/cancel           # Rollback receipt & restore PO line
```

### Putaway Surface
```text
GET  /api/putaway/pending/<wh_id>    # List items in staging awaiting putaway
GET  /api/putaway/suggest/<item_id>  # Algorithmic bin recommendation
POST /api/putaway/confirm            # Confirm transfer from staging to rack bin
```

### Picking & Wave Surface
```text
POST /api/picking/wave-create        # Group orders into wave batch
POST /api/picking/create-batch       # Create pick tasks in walk-path order
GET  /api/picking/batch/<id>/next    # Get next pick task (aisle, rack, SKU, qty)
POST /api/picking/confirm            # Confirm pick with barcode validation
POST /api/picking/short              # Report short/missing stock during pick
POST /api/picking/complete-batch     # Mark wave batch complete
```

### Packing Surface (100% Barcode Verification)
```text
GET  /api/packing/order/<barcode>    # Retrieve order items and expected weights
POST /api/packing/verify             # Scan single SKU barcode -> Validate match
POST /api/packing/complete           # Seal order, lock package, generate slip
```

### Shipping Surface
```text
POST /api/shipping/fulfill           # Record carrier, tracking number, mark fulfilled
```

---

## 3. Database Schema Blueprint & Patterns

### 3.1 Concurrency & Inventory Row Locking
When mutating inventory balances (receiving, picking, adjusting), Sentry WMS uses explicit row-level locks:
```sql
-- Atomic lock during allocation/picking:
SELECT on_hand, allocated 
FROM inventory 
WHERE item_id = $1 AND bin_id = $2 
FOR UPDATE;
```
*Rule in Tayooli Core:* All ledger entries in `stock_movements` must execute inside a PostgreSQL transaction with tenant context set and row-level locks on affected stock balances.

### 3.2 Audit Log Hash Chain
Sentry WMS protects warehouse transactions with an append-only audit trail and SHA-256 tamper verification:
- Each audit log row hashes its payload combined with the previous row's hash.
- Retroactive modifications break the chain.
