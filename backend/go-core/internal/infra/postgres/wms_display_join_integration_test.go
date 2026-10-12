package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

// TestWMSListJoinsProductAndReporterNames guards the read-only display fields
// that let the Scrap and Opname pages label rows without loading the full
// product catalog (capped at 1000 rows): product name/SKU and reporter name,
// plus the "all warehouses" (NULL) vs. single-warehouse scrap filter.
func TestWMSListJoinsProductAndReporterNames(t *testing.T) {
	db := freshApprovalTestDB(t)
	defer db.Close()
	ctx := context.Background()

	tenant := uuid.New()
	userID, productID := uuid.New(), uuid.New()
	whA, whB := uuid.New(), uuid.New()
	binA, scrapA, binB, scrapB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	opnameID := uuid.New()

	exec := func(q string, args ...any) {
		t.Helper()
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback() //nolint:errcheck
		_, err = tx.ExecContext(ctx, `SET LOCAL app.current_tenant_id = '`+tenant.String()+`'`)
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, q, args...)
		require.NoError(t, err, q)
		require.NoError(t, tx.Commit())
	}

	exec(`INSERT INTO tenants (id, name) VALUES ($1, 'Join T')`, tenant)
	t.Cleanup(func() {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return
		}
		_, _ = tx.ExecContext(ctx, `SET LOCAL app.current_tenant_id = '`+tenant.String()+`'`)
		for _, q := range []string{
			`DELETE FROM stock_scraps WHERE tenant_id = $1`,
			`DELETE FROM stock_opname_items WHERE tenant_id = $1`,
			`DELETE FROM stock_opnames WHERE tenant_id = $1`,
			`DELETE FROM warehouse_locations WHERE tenant_id = $1`,
			`DELETE FROM warehouses WHERE tenant_id = $1`,
			`DELETE FROM products WHERE tenant_id = $1`,
			`DELETE FROM users WHERE tenant_id = $1`,
			`DELETE FROM tenants WHERE id = $1`,
		} {
			_, _ = tx.ExecContext(ctx, q, tenant)
		}
		_ = tx.Commit()
	})

	exec(`INSERT INTO users (id, tenant_id, email, password_hash, role, full_name) VALUES ($1, $2, $3, 'x', 'warehouse', 'Siti Gudang')`,
		userID, tenant, "join-"+userID.String()+"@example.test")
	exec(`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1, $2, 'Beras 5kg', $3, 1000)`,
		productID, tenant, "BRS-"+productID.String()[:8])
	for _, wh := range []struct{ id, bin, scrap uuid.UUID }{{whA, binA, scrapA}, {whB, binB, scrapB}} {
		exec(`INSERT INTO warehouses (id, tenant_id, code, name) VALUES ($1, $2, $3, 'Gudang')`, wh.id, tenant, "W-"+wh.id.String()[:6])
		exec(`INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type) VALUES ($1, $2, $3, 'R-01', 'Rak', 'INTERNAL')`, wh.bin, tenant, wh.id)
		exec(`INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type) VALUES ($1, $2, $3, 'SCRAP', 'Scrap', 'SCRAP')`, wh.scrap, tenant, wh.id)
		exec(`INSERT INTO stock_scraps (tenant_id, scrap_number, warehouse_id, product_id, source_location_id, scrap_location_id, quantity, reason, reported_by)
		      VALUES ($1, $2, $3, $4, $5, $6, 1, 'Kemasan bocor parah', $7)`,
			tenant, "SCR-"+wh.id.String()[:8], wh.id, productID, wh.bin, wh.scrap, userID)
	}
	exec(`INSERT INTO stock_opnames (id, tenant_id, warehouse_id, opname_number, status, conducted_by) VALUES ($1, $2, $3, 'OPN-1', 'IN_PROGRESS', $4)`,
		opnameID, tenant, whA, userID)
	exec(`INSERT INTO stock_opname_items (opname_id, tenant_id, product_id, location_id, system_qty, physical_qty, discrepancy_qty) VALUES ($1, $2, $3, $4, 5, 4, -1)`,
		opnameID, tenant, productID, binA)

	repo := postgres.NewWMSRepo(db)

	all, err := repo.ListStockScraps(ctx, tenant, nil)
	require.NoError(t, err)
	require.Len(t, all, 2, "nil warehouse filter must return every warehouse")
	for _, s := range all {
		require.Equal(t, "Beras 5kg", s.ProductName)
		require.Equal(t, "BRS-"+productID.String()[:8], s.ProductSKU)
		require.Equal(t, "Siti Gudang", s.ReportedByName)
	}

	onlyA, err := repo.ListStockScraps(ctx, tenant, &whA)
	require.NoError(t, err)
	require.Len(t, onlyA, 1)
	require.Equal(t, whA, onlyA[0].WarehouseID)

	items, err := repo.ListStockOpnameItems(ctx, tenant, opnameID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Beras 5kg", items[0].ProductName)
	require.Equal(t, "BRS-"+productID.String()[:8], items[0].ProductSKU)
}
