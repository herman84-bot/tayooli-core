package postgres_test

// Real-database integration test for Barang Masuk (stock receipts).
//
// Skipped unless TAYOOLI_PG_INTEGRATION_DSN points at a THROWAWAY PostgreSQL 15
// database. It runs every embedded migration (001..latest) exactly the way the
// backend does at startup, then exercises create -> post -> cancel against the
// real schema, RLS and stock_movements ledger. Never point it at production.
//
// The app connects as a non-superuser so FORCE ROW LEVEL SECURITY is actually
// enforced (superusers bypass RLS, which would hide policy bugs).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
	wmsuc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/migrations"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const appRole = "tayooli_app_it"

func TestStockReceiptRealPostgres(t *testing.T) {
	adminDSN := os.Getenv("TAYOOLI_PG_INTEGRATION_DSN")
	if adminDSN == "" {
		t.Skip("set TAYOOLI_PG_INTEGRATION_DSN to a throwaway PostgreSQL 15 database to run")
	}
	ctx := context.Background()

	admin, err := sql.Open("postgres", adminDSN)
	require.NoError(t, err)
	defer admin.Close()
	require.NoError(t, admin.PingContext(ctx))

	// Clean throwaway database schema before running migrations
	_, _ = admin.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO postgres; GRANT ALL ON SCHEMA public TO public;")

	// 1. Migrations exactly as production startup runs them.
	require.NoError(t, migrations.Run(ctx, admin), "migrations.Run must succeed on a fresh database")

	var applied int
	require.NoError(t, admin.QueryRowContext(ctx,
		`SELECT count(*) FROM schema_migrations WHERE filename = '028_wms_stock_receipts.sql'`).Scan(&applied))
	require.Equal(t, 1, applied)
	for _, tbl := range []string{"stock_receipts", "stock_receipt_items"} {
		var forced bool
		require.NoError(t, admin.QueryRowContext(ctx,
			`SELECT relforcerowsecurity FROM pg_class WHERE relname = $1`, tbl).Scan(&forced))
		require.True(t, forced, "%s must have FORCE ROW LEVEL SECURITY", tbl)
	}

	// Re-running must be a no-op (already recorded in schema_migrations).
	require.NoError(t, migrations.Run(ctx, admin))

	// Production upgrade path: a database already at 027 receives 028 on deploy.
	// Roll 028 back to "pending", then let the real runner apply it again.
	for _, q := range []string{
		`DROP TABLE IF EXISTS stock_receipt_items CASCADE`,
		`DROP TABLE IF EXISTS stock_batches CASCADE`,
		`DROP TABLE IF EXISTS stock_receipts CASCADE`,
		// 029 and 033 ALTER the 028 tables; dropping them also drops those columns,
		// so both must be re-applied too or the schema is silently incomplete.
		`DELETE FROM schema_migrations WHERE filename IN ('028_wms_stock_receipts.sql', '029_wms_receipt_source_types.sql', '033_wms_batches_staging_putaway.sql')`,
	} {
		_, err := admin.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}
	require.NoError(t, migrations.Run(ctx, admin), "028 must apply cleanly on a database already at 027")
	var exists bool
	require.NoError(t, admin.QueryRowContext(ctx,
		`SELECT to_regclass('public.stock_receipts') IS NOT NULL AND to_regclass('public.stock_receipt_items') IS NOT NULL`).Scan(&exists))
	require.True(t, exists, "028 tables must exist after upgrade run (not silently skipped)")

	// 2. Non-superuser app role so RLS is enforced.
	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP OWNED BY %s`, appRole))
	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP ROLE IF EXISTS %s`, appRole))
	for _, q := range []string{
		fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD 'it'`, appRole),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, appRole),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s`, appRole),
		fmt.Sprintf(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s`, appRole),
	} {
		_, err := admin.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}

	// Seed tenant, user, product, warehouse, internal location (as admin; seed bypasses RLS).
	tenantID, otherTenant, userID := uuid.New(), uuid.New(), uuid.New()
	productID, warehouseID, binID := uuid.New(), uuid.New(), uuid.New()
	seed := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO tenants (id, name) VALUES ($1, 'IT Tenant'), ($2, 'Other Tenant')`, []any{tenantID, otherTenant}},
		{`INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES ($1, $2, $3, 'x', 'admin')`,
			[]any{userID, tenantID, "it-" + userID.String() + "@example.test"}},
		{`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1, $2, 'Kopi Bubuk 200g', 'KOPI-200', 25000)`,
			[]any{productID, tenantID}},
		{`INSERT INTO warehouses (id, tenant_id, code, name) VALUES ($1, $2, 'GU', 'Gudang Utama')`, []any{warehouseID, tenantID}},
		{`INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type) VALUES ($1, $2, $3, 'A-01', 'Rak A1', 'INTERNAL')`,
			[]any{binID, tenantID, warehouseID}},
	}
	for _, s := range seed {
		_, err := admin.ExecContext(ctx, s.q, s.args...)
		require.NoError(t, err, s.q)
	}

	// 3. App connection as the RLS-bound role.
	var host, port, dbname string
	require.NoError(t, admin.QueryRowContext(ctx,
		`SELECT current_database(), COALESCE(inet_server_port()::text, '5432')`).Scan(&dbname, &port))
	host = os.Getenv("TAYOOLI_PG_INTEGRATION_HOST")
	if host == "" {
		host = "localhost"
	}
	app, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%s user=%s password=it dbname=%s sslmode=disable", host, port, appRole, dbname))
	require.NoError(t, err)
	defer app.Close()
	require.NoError(t, app.PingContext(ctx))

	repo := postgres.NewWMSRepo(app)
	uc := wmsuc.New(repo)

	stockAt := func(loc uuid.UUID) decimal.Decimal {
		q, err := repo.GetStockByLocation(ctx, tenantID, loc, productID)
		require.NoError(t, err)
		return q
	}
	scrapLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
	require.NoError(t, err)

	reason := "Kemasan sobek"
	ref := "SJ-001"
	req := wmsuc.StockReceiptRequest{
		WarehouseID:    warehouseID,
		DestLocationID: binID,
		SupplierName:   "CV Sumber Rejeki",
		SupplierRef:    &ref,
		Items: []wmsuc.StockReceiptItemRequest{{
			ProductID:    productID,
			AcceptedQty:  decimal.NewFromInt(10),
			RejectedQty:  decimal.NewFromInt(2),
			RejectReason: &reason,
		}},
	}

	// 4. Create DRAFT: no stock effect.
	rc, items, err := uc.CreateStockReceipt(ctx, tenantID, userID, "admin", req)
	require.NoError(t, err)
	require.Equal(t, domain.StockReceiptStatusDraft, rc.Status)
	require.Len(t, items, 1)
	require.True(t, stockAt(binID).IsZero(), "draft must not change stock")

	got, gotItems, err := uc.GetStockReceipt(ctx, tenantID, userID, "admin", rc.ID)
	require.NoError(t, err)
	require.Equal(t, "Kopi Bubuk 200g", gotItems[0].ProductName)
	require.Equal(t, "KOPI-200", gotItems[0].ProductSKU)
	require.Equal(t, 1, got.ItemCount)
	require.True(t, got.TotalAcceptedQty.Equal(decimal.NewFromInt(10)))

	// Edit draft (replace items) works.
	req.Items[0].AcceptedQty = decimal.NewFromInt(12)
	_, _, err = uc.UpdateStockReceipt(ctx, tenantID, userID, "admin", rc.ID, req)
	require.NoError(t, err)

	// Unknown product -> validation error (not a 500).
	bad := req
	bad.Items = []wmsuc.StockReceiptItemRequest{{ProductID: uuid.New(), AcceptedQty: decimal.NewFromInt(1)}}
	_, _, err = uc.CreateStockReceipt(ctx, tenantID, userID, "admin", bad)
	var vErr *domain.StockReceiptValidationError
	require.True(t, errors.As(err, &vErr), "unknown product should be a validation error, got %v", err)

	// 5. POST: accepted -> staging, rejected -> @SCRAP, atomically with batch.
	stgLoc, err := repo.GetOrCreateStagingLocation(ctx, tenantID, warehouseID)
	require.NoError(t, err)

	posted, err := uc.PostStockReceipt(ctx, tenantID, userID, "admin", rc.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StockReceiptStatusPosted, posted.Status)
	require.True(t, stockAt(stgLoc.ID).Equal(decimal.NewFromInt(12)), "staging stock = %s", stockAt(stgLoc.ID))
	require.True(t, stockAt(scrapLoc.ID).Equal(decimal.NewFromInt(2)), "scrap stock = %s", stockAt(scrapLoc.ID))

	var movCount int
	require.NoError(t, admin.QueryRowContext(ctx,
		`SELECT count(*) FROM stock_movements WHERE reference_type = 'GOODS_RECEIPT' AND reference_id = $1`, rc.ID).Scan(&movCount))
	require.Equal(t, 2, movCount)

	// Fetch created batch_id for test simulation
	var batchID uuid.UUID
	require.NoError(t, admin.QueryRowContext(ctx,
		`SELECT batch_id FROM stock_receipt_items WHERE receipt_id = $1 LIMIT 1`, rc.ID).Scan(&batchID))
	require.NotEqual(t, uuid.Nil, batchID)

	// Double post and edit-after-post are rejected; no extra ledger rows.
	_, err = uc.PostStockReceipt(ctx, tenantID, userID, "admin", rc.ID)
	require.ErrorIs(t, err, domain.ErrStockReceiptNotDraft)
	_, _, err = uc.UpdateStockReceipt(ctx, tenantID, userID, "admin", rc.ID, req)
	require.ErrorIs(t, err, domain.ErrStockReceiptNotDraft)
	require.NoError(t, admin.QueryRowContext(ctx,
		`SELECT count(*) FROM stock_movements WHERE reference_id = $1`, rc.ID).Scan(&movCount))
	require.Equal(t, 2, movCount)

	// 6. RLS: another tenant cannot see the receipt.
	_, _, err = uc.GetStockReceipt(ctx, otherTenant, userID, "admin", rc.ID)
	require.ErrorIs(t, err, domain.ErrStockReceiptNotFound)

	// 7. Cancel blocked when stock already consumed (simulate a consumption: stgLoc -> @CUSTOMER).
	custLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
	require.NoError(t, err)
	require.NoError(t, repo.CreateStockMovement(ctx, &domain.StockMovement{
		ID: uuid.New(), TenantID: tenantID, MovementNumber: "IT-SALE-1", ProductID: productID,
		SourceLocationID: stgLoc.ID, DestLocationID: custLoc.ID, Quantity: decimal.NewFromInt(5),
		Status: domain.StockMovementStatusDone, ReferenceType: "IT", ReferenceID: uuid.New(),
		BatchID: &batchID,
	}))
	_, err = uc.CancelStockReceipt(ctx, tenantID, userID, "admin", rc.ID, "salah input")
	require.ErrorIs(t, err, domain.ErrStockReceiptStockConsumed)
	require.True(t, stockAt(stgLoc.ID).Equal(decimal.NewFromInt(7)), "failed cancel must not change stock")

	// Return the consumed units, then cancel succeeds and fully reverses.
	require.NoError(t, repo.CreateStockMovement(ctx, &domain.StockMovement{
		ID: uuid.New(), TenantID: tenantID, MovementNumber: "IT-RETURN-1", ProductID: productID,
		SourceLocationID: custLoc.ID, DestLocationID: stgLoc.ID, Quantity: decimal.NewFromInt(5),
		Status: domain.StockMovementStatusDone, ReferenceType: "IT", ReferenceID: uuid.New(),
		BatchID: &batchID,
	}))
	cancelled, err := uc.CancelStockReceipt(ctx, tenantID, userID, "admin", rc.ID, "salah input")
	require.NoError(t, err)
	require.Equal(t, domain.StockReceiptStatusCancelled, cancelled.Status)
	require.True(t, stockAt(stgLoc.ID).IsZero(), "staging after cancel = %s", stockAt(stgLoc.ID))
	require.True(t, stockAt(scrapLoc.ID).IsZero(), "scrap after cancel = %s", stockAt(scrapLoc.ID))

	// Ledger is append-only: 2 post + 2 reversal rows, nothing deleted.
	require.NoError(t, admin.QueryRowContext(ctx,
		`SELECT count(*) FROM stock_movements WHERE reference_id = $1`, rc.ID).Scan(&movCount))
	require.Equal(t, 4, movCount)

	_, err = uc.CancelStockReceipt(ctx, tenantID, userID, "admin", rc.ID, "lagi")
	require.ErrorIs(t, err, domain.ErrStockReceiptAlreadyCancelled)

	// List filter by status.
	st := domain.StockReceiptStatusCancelled
	list, err := uc.ListStockReceipts(ctx, tenantID, userID, "admin", &warehouseID, &st, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
}
