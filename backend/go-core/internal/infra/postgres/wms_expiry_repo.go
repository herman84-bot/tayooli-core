package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/lib/pq"
)

// ListExpiringBatchBalances returns on-hand batch balances in the given
// warehouses whose expiry date is on or before `until` (already expired lots
// included). Only stock physically in the warehouse counts: racks, staging and
// quarantine bins. System locations (vendor, customer, transit, loss, scrap)
// are excluded. Ordered by expiry date (FEFO, OCA wms §1.3).
func (r *WMSRepo) ListExpiringBatchBalances(ctx context.Context, tenantID uuid.UUID, warehouseIDs []uuid.UUID, until time.Time) ([]domain.ExpiringBatchLine, error) {
	if len(warehouseIDs) == 0 {
		return []domain.ExpiringBatchLine{}, nil
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListExpiringBatchBalances: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListExpiringBatchBalances: set tenant: %w", err)
	}

	whIDs := make([]string, len(warehouseIDs))
	for i, id := range warehouseIDs {
		whIDs[i] = id.String()
	}

	rows, err := tx.QueryContext(ctx, `
SELECT b.id, b.batch_number, b.expiry_date, b.status,
       sm.product_id, COALESCE(p.name, ''), COALESCE(p.sku, ''),
       loc.warehouse_id, COALESCE(w.name, ''), loc.id, loc.code, loc.type::text,
       SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty
FROM stock_movements sm
JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
LEFT JOIN warehouses w ON w.id = loc.warehouse_id AND w.tenant_id = loc.tenant_id
LEFT JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
WHERE sm.tenant_id = $1 AND sm.status = 'DONE'
  AND b.expiry_date IS NOT NULL AND b.expiry_date <= $2::date
  AND loc.warehouse_id = ANY($3::uuid[])
  AND loc.type::text NOT IN ('VENDOR', 'CUSTOMER', 'TRANSIT', 'LOSS', 'SCRAP')
GROUP BY b.id, b.batch_number, b.expiry_date, b.status, sm.product_id, p.name, p.sku,
         loc.warehouse_id, w.name, loc.id, loc.code, loc.type
HAVING SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) > 0
ORDER BY b.expiry_date ASC, p.name ASC, loc.code ASC, b.id ASC`,
		tenantID, until.Format("2006-01-02"), pq.Array(whIDs))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListExpiringBatchBalances: query: %w", err)
	}
	defer rows.Close()

	lines := []domain.ExpiringBatchLine{}
	for rows.Next() {
		var l domain.ExpiringBatchLine
		var whID sql.NullString
		var locType string
		if err := rows.Scan(
			&l.BatchID, &l.BatchNumber, &l.ExpiryDate, &l.BatchStatus,
			&l.ProductID, &l.ProductName, &l.ProductSKU,
			&whID, &l.WarehouseName, &l.LocationID, &l.LocationCode, &locType,
			&l.Quantity,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListExpiringBatchBalances: scan: %w", err)
		}
		if p := nullUUIDToPtr(whID); p != nil {
			l.WarehouseID = *p
		}
		l.LocationType = domain.LocationType(locType)
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListExpiringBatchBalances: rows: %w", err)
	}
	return lines, nil
}
