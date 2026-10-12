package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

// legacyStockUnits is the pre-refactor OR-join query, kept verbatim as the
// reference the new UNION ALL query must match.
const legacyStockUnits = `
SELECT COALESCE(SUM(sm.qty_total), 0) FROM (
  SELECT sm.product_id,
         SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty_total
  FROM stock_movements sm
  JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
  WHERE sm.tenant_id = $1 AND sm.status = 'DONE' AND loc.type NOT IN ('QUARANTINE', 'STAGING_INBOUND', 'STAGING_OUTBOUND')
  GROUP BY sm.product_id, loc.id
  HAVING SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) > 0
) sm`

const legacyStockValue = `
SELECT COALESCE(SUM(sm.qty_val), 0) FROM (
  SELECT SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) * COALESCE(p.price, 0) AS qty_val
  FROM stock_movements sm
  JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
  JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
  WHERE sm.tenant_id = $1 AND sm.status = 'DONE' AND loc.type NOT IN ('QUARANTINE', 'STAGING_INBOUND', 'STAGING_OUTBOUND')
  GROUP BY sm.product_id, p.price, loc.id
  HAVING SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) > 0
) sm`

func TestDashboardStockQuery_MatchesLegacyORJoin(t *testing.T) {
	db := freshApprovalTestDB(t)
	defer db.Close()
	ctx := context.Background()

	tenantID, whID := uuid.New(), uuid.New()
	p1, p2 := uuid.New(), uuid.New()
	vendor, staging, binA, binB, scrap := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck // everything rolls back: no cleanup needed
	_, err = tx.ExecContext(ctx, `SET LOCAL app.current_tenant_id = '`+tenantID.String()+`'`)
	require.NoError(t, err)

	exec := func(q string, args ...any) {
		t.Helper()
		_, err := tx.ExecContext(ctx, q, args...)
		require.NoError(t, err, q)
	}
	exec(`INSERT INTO tenants (id, name) VALUES ($1, 'Dash Tenant')`, tenantID)
	exec(`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1,$3,'P1','DQ-1',1500),($2,$3,'P2','DQ-2',0)`, p1, p2, tenantID)
	exec(`INSERT INTO warehouses (id, tenant_id, code, name) VALUES ($1,$2,'DQ','Dash WH')`, whID, tenantID)
	for _, l := range []struct {
		id         uuid.UUID
		code, kind string
	}{{vendor, "V", "VENDOR"}, {staging, "S", "STAGING_INBOUND"}, {binA, "A", "INTERNAL"}, {binB, "B", "INTERNAL"}, {scrap, "X", "SCRAP"}} {
		exec(`INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type) VALUES ($1,$2,$3,$4,$4,$5)`,
			l.id, tenantID, whID, l.code, l.kind)
	}
	// ADR-014 Invariant 1: every movement carries a batch.
	batch := map[uuid.UUID]uuid.UUID{p1: uuid.New(), p2: uuid.New()}
	for prod, b := range batch {
		exec(`INSERT INTO stock_batches (id, tenant_id, product_id, batch_number) VALUES ($1,$2,$3,'DQ-B')`, b, tenantID, prod)
	}
	n := 0
	move := func(prod, src, dst uuid.UUID, qty string, status string) {
		n++
		exec(`INSERT INTO stock_movements (tenant_id, movement_number, product_id, source_location_id, dest_location_id, quantity, status, reference_type, reference_id, batch_id)
		      VALUES ($1,$2,$3,$4,$5,$6,$7,'TEST',$8,$9)`,
			tenantID, fmt.Sprintf("DQ-%d", n), prod, src, dst, qty, status, uuid.New(), batch[prod])
	}
	move(p1, vendor, staging, "100", "DONE")
	move(p1, staging, binA, "100", "DONE")
	move(p1, binA, binB, "30.5", "DONE")
	move(p1, binA, scrap, "5", "DONE")
	move(p1, binB, binB, "7", "DONE") // self-move edge case
	move(p2, vendor, binA, "12", "DONE")
	move(p2, binA, vendor, "20", "DONE") // drives bin A negative for p2 -> excluded by HAVING
	move(p1, vendor, binA, "999", "CANCELLED")

	var legacyUnits, legacyValue decimal.Decimal
	require.NoError(t, tx.QueryRowContext(ctx, legacyStockUnits, tenantID).Scan(&legacyUnits))
	require.NoError(t, tx.QueryRowContext(ctx, legacyStockValue, tenantID).Scan(&legacyValue))

	var skus, wh, locs, today int64
	var units, value decimal.Decimal
	require.NoError(t, tx.QueryRowContext(ctx, postgres.DashboardWMSCountsSQL, tenantID).
		Scan(&skus, &units, &wh, &locs, &today, &value))

	require.True(t, legacyUnits.Equal(units), "units legacy=%s new=%s", legacyUnits, units)
	require.True(t, legacyValue.Equal(value), "value legacy=%s new=%s", legacyValue, value)
	require.False(t, units.IsZero(), "fixture must produce stock")
}
