# Benchmark Reference: OCA WMS (Odoo Community Association)
## Business Rules, ABC Velocity Putaway & Wave Order Release Patterns

- **Source Repository:** `https://github.com/OCA/wms`
- **License:** AGPL-3.0
- **Purpose:** Primary business logic reference for ISO 9001:2015 warehouse governance, dock management, ABC velocity putaway allocation, and wave picking in `tayooli-core`.

---

## 1. Core Logistics Modules & Domain Rules

### 1.1 Inbound Docking & Gate Management (`shopfloor_reception_dock`)
- **Concept:** Physical vehicle gate verification.
- **Rules:**
  1. No unloading begins before gate check-in and dock assignment.
  2. Dock bays are assigned based on vehicle type (box truck, trailer, van) and status (empty, busy, reserved).
  3. Dock arrival creates an operational window with an SLA timer (target: $\le 15$ minutes check-in to docking).

### 1.2 ABC Velocity Putaway Strategy (`stock_storage_type_putaway_abc`)
- **Concept:** Automatic bin allocation based on SKU velocity mapping (Pareto 80/20 principle).
- **Classification Rules:**
  - **Category A (Fast-Moving):** Top 20% of SKUs generating 80% of pick volume.
    - *Allocation rule:* Ground level racks adjacent to main outbound staging aisles. Lowest travel time.
  - **Category B (Medium-Moving):** Next 30% of SKUs generating 15% of pick volume.
    - *Allocation rule:* Middle levels and secondary aisles.
  - **Category C (Slow-Moving):** Remaining 50% of SKUs generating 5% of pick volume.
    - *Allocation rule:* Top levels or deep warehouse storage zones.
- **Physical Balancing Constraint:**
  - Weight balancing: Heavy goods ($> 25\text{ kg}$) must never be suggested above level 1 regardless of velocity class.

### 1.3 Expiry Date Management & FEFO Rotation
- **Rules:**
  1. Each received lot stores manufacturing date (`mfg_date`) and expiry date (`expiry_date`).
  2. Stock availability query sorts by `expiry_date ASC`.
  3. Goods with remaining shelf life below minimum customer tolerance (e.g., $< 30$ days) are automatically quarantined from sales orders.

### 1.4 Wave Planning & Release Channels (`stock_release_channel`)
- **Concept:** Consolidation of discrete orders into operational waves.
- **Grouping Criteria:**
  - By carrier/courier (e.g., JNE wave, SiCepat wave, Truck delivery wave).
  - By destination route / geographic delivery zone.
  - By shipping cut-off time (e.g., afternoon 16:00 dispatch).
- **Cluster & Batch Picking (`shopfloor_cluster_picking_repack`):**
  - Pickers carry multi-tote trolleys through the warehouse in a single serpentine walk path, picking items for multiple orders simultaneously.
  - Totes are transferred to the packing bench for per-order sorting and repackaging.

---

## 2. Adaptation Blueprint for Tayooli Core

| OCA WMS Concept | Tayooli Core Equivalent | Implementation File |
| :--- | :--- | :--- |
| `reception_dock` | `inbound_docks` & `dock_appointments` | `backend/go-core/migrations/035_...` |
| `putaway_abc` | Putaway Suggestion Engine (`Fast/Medium/Slow`) | `backend/go-core/internal/usecase/wms/putaway_usecase.go` |
| `stock_release_channel` | `pick_waves` grouped by expedition & route | `backend/go-core/internal/usecase/wms/wave_usecase.go` |
| `cluster_picking` | Sequential aisle walking order in Picking List | `app/(app)/wms/outbound/picking` |
