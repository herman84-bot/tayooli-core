package postgres_test

// Real PostgreSQL integration test for batch expiry control (OCA wms §1.3):
// expired batches are blocked from outbound (delivery order dispatch, POS
// warehouse deduction) but may still be scrapped, and the expiry report lists
// them per batch and location. Skipped unless TAYOOLI_PG_INTEGRATION_DSN points
// at a THROWAWAY database.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
	wmsuc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/migrations"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const expiryAppRole = "tayooli_app_expiry_it"

func TestExpiryControlRealPostgres(t *testing.T) {
	adminDSN := os.Getenv("TAYOOLI_PG_INTEGRATION_DSN")
	if adminDSN == "" {
		t.Skip("set TAYOOLI_PG_INTEGRATION_DSN to run real PostgreSQL integration test")
	}
	ctx := context.Background()
	admin, err := sql.Open("postgres", adminDSN)
	require.NoError(t, err)
	defer admin.Close()
	require.NoError(t, migrations.Run(ctx, admin))

	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP OWNED BY %s`, expiryAppRole))
	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP ROLE IF EXISTS %s`, expiryAppRole))
	for _, q := range []string{
		fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD 'it'`, expiryAppRole),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, expiryAppRole),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s`, expiryAppRole),
		fmt.Sprintf(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s`, expiryAppRole),
	} {
		_, err := admin.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}

	tenantID, otherTenantID, userID := uuid.New(), uuid.New(), uuid.New()
	productID, warehouseID, rackLocID := uuid.New(), uuid.New(), uuid.New()
	sku := "SUSU-" + productID.String()[:6]
	for _, s := range []struct {
		q    string
		args []any
	}{
		{`INSERT INTO tenants (id, name) VALUES ($1, 'Expiry Tenant')`, []any{tenantID}},
		{`INSERT INTO tenants (id, name) VALUES ($1, 'Other Tenant')`, []any{otherTenantID}},
		{`INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES ($1, $2, $3, 'x', 'admin')`, []any{userID, tenantID, "expiry-" + userID.String() + "@example.test"}},
		{`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1, $2, 'Susu UHT 1L', $3, 20000)`, []any{productID, tenantID, sku}},
		{`INSERT INTO warehouses (id, tenant_id, code, name) VALUES ($1, $2, $3, 'Gudang Expiry')`, []any{warehouseID, tenantID, "WH-" + warehouseID.String()[:4]}},
		{`INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type) VALUES ($1, $2, $3, 'RACK-EXP', 'Rak Expiry', 'INTERNAL')`, []any{rackLocID, tenantID, warehouseID}},
	} {
		_, err := admin.ExecContext(ctx, s.q, s.args...)
		require.NoError(t, err, s.q)
	}

	var dbname, port string
	require.NoError(t, admin.QueryRowContext(ctx, `SELECT current_database(), COALESCE(inet_server_port()::text, '5432')`).Scan(&dbname, &port))
	host := os.Getenv("TAYOOLI_PG_INTEGRATION_HOST")
	if host == "" {
		host = "localhost"
	}
	app, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=it dbname=%s sslmode=disable", host, port, expiryAppRole, dbname))
	require.NoError(t, err)
	defer app.Close()

	repo := postgres.NewWMSRepo(app)
	uc := wmsuc.New(repo)
	dec := decimal.NewFromInt
	recipient, driver, plate, expedition := "Toko Sehat", "Pak Budi", "B 1234 XYZ", "Armada Sendiri"

	// Batch SOON expires in 3 days, FRESH in 120 days (calendar dates, WIB).
	today := domain.BusinessToday(time.Now())
	soonExp := today.AddDate(0, 0, 3)
	freshExp := today.AddDate(0, 0, 120)
	soon := &domain.StockBatch{ID: uuid.New(), TenantID: tenantID, ProductID: productID, BatchNumber: "LOT-SOON", ExpiryDate: &soonExp, Status: domain.StockBatchStatusReleased}
	fresh := &domain.StockBatch{ID: uuid.New(), TenantID: tenantID, ProductID: productID, BatchNumber: "LOT-FRESH", ExpiryDate: &freshExp, Status: domain.StockBatchStatusReleased}
	soon, err = repo.GetOrCreateBatch(ctx, soon)
	require.NoError(t, err)
	fresh, err = repo.GetOrCreateBatch(ctx, fresh)
	require.NoError(t, err)

	vendor, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeVendor)
	require.NoError(t, err)
	for i, seed := range []struct {
		batch *domain.StockBatch
		qty   int64
	}{{soon, 10}, {fresh, 5}} {
		require.NoError(t, repo.CreateStockMovement(ctx, &domain.StockMovement{
			ID: uuid.New(), TenantID: tenantID, MovementNumber: fmt.Sprintf("SEED-EXP-%d", i), ProductID: productID,
			SourceLocationID: vendor.ID, DestLocationID: rackLocID, Quantity: dec(seed.qty),
			Status: domain.StockMovementStatusDone, ReferenceType: "SEED", ReferenceID: uuid.New(), BatchID: &seed.batch.ID,
		}))
	}

	batchQty := func(batchID uuid.UUID) decimal.Decimal {
		t.Helper()
		bals, err := repo.ListBatchBalances(ctx, tenantID, domain.BatchBalanceFilter{LocationIDs: []uuid.UUID{rackLocID}})
		require.NoError(t, err)
		for _, b := range bals {
			if b.BatchID == batchID {
				return b.Quantity
			}
		}
		return decimal.Zero
	}

	// 1. While LOT-SOON is still valid, FEFO allocates the delivery order to it and it gets packed.
	do, err := uc.CreateDeliveryOrder(ctx, tenantID, userID, "admin", wmsuc.CreateDeliveryOrderRequest{
		WarehouseID: warehouseID, RecipientName: &recipient, DriverName: &driver, VehiclePlate: &plate, ExpeditionName: &expedition, DONumber: "DO-EXP-001",
		Items: []wmsuc.CreateDeliveryOrderItemRequest{{ProductID: productID, Quantity: dec(4)}},
	})
	require.NoError(t, err)
	_, items, err := uc.GetDeliveryOrder(ctx, tenantID, userID, "admin", do.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, soon.ID, *items[0].BatchID, "FEFO picks the earliest-expiring valid batch")
	for i := 0; i < 4; i++ {
		_, err := uc.ScanPackStationItem(ctx, tenantID, userID, "admin", do.ID, domain.PackScanRequest{Barcode: sku, Quantity: dec(1)})
		require.NoError(t, err)
	}
	_, err = uc.CompletePackStation(ctx, tenantID, userID, "admin", do.ID, domain.PackCompleteRequest{})
	require.NoError(t, err)

	// 2. Ten days later LOT-SOON has expired. Pin the expiry clock forward.
	later := time.Now().Add(10 * 24 * time.Hour)
	prevClock := domain.ExpiryClock
	domain.ExpiryClock = func() time.Time { return later }
	t.Cleanup(func() { domain.ExpiryClock = prevClock })

	// 2a. Dispatching the packed DO (explicit expired batch) is blocked; stock untouched.
	_, err = uc.DispatchDeliveryOrder(ctx, tenantID, userID, "admin", do.ID)
	require.ErrorIs(t, err, domain.ErrExpiredStockBlocked)
	require.True(t, batchQty(soon.ID).Equal(dec(10)), "expired stock must not leave on dispatch")

	// 2b. A new DO is allocated to the fresh batch only.
	do2, err := uc.CreateDeliveryOrder(ctx, tenantID, userID, "admin", wmsuc.CreateDeliveryOrderRequest{
		WarehouseID: warehouseID, RecipientName: &recipient, DONumber: "DO-EXP-002",
		Items: []wmsuc.CreateDeliveryOrderItemRequest{{ProductID: productID, Quantity: dec(2)}},
	})
	require.NoError(t, err)
	_, items2, err := uc.GetDeliveryOrder(ctx, tenantID, userID, "admin", do2.ID)
	require.NoError(t, err)
	require.Len(t, items2, 1)
	require.Equal(t, fresh.ID, *items2[0].BatchID, "expired batch must be skipped by FEFO")

	// 2c. A DO that only expired stock could cover is rejected with the expiry error.
	_, err = uc.CreateDeliveryOrder(ctx, tenantID, userID, "admin", wmsuc.CreateDeliveryOrderRequest{
		WarehouseID: warehouseID, RecipientName: &recipient, DONumber: "DO-EXP-003",
		Items: []wmsuc.CreateDeliveryOrderItemRequest{{ProductID: productID, Quantity: dec(9)}},
	})
	require.ErrorIs(t, err, domain.ErrExpiredStockBlocked)

	// 2d. POS warehouse deduction takes fresh stock and refuses expired stock.
	posMov := func(qty int64) *domain.StockMovement {
		cust, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
		require.NoError(t, err)
		return &domain.StockMovement{
			ID: uuid.New(), TenantID: tenantID, MovementNumber: "MV-POS-" + uuid.NewString()[:8], ProductID: productID,
			DestLocationID: cust.ID, Quantity: dec(qty), Status: domain.StockMovementStatusDone,
			ReferenceType: domain.StockRefPOS, ReferenceID: uuid.New(), ExecutedBy: &userID,
		}
	}
	require.NoError(t, repo.DeductWarehouseStock(ctx, tenantID, warehouseID, productID, dec(1), posMov(1)))
	require.True(t, batchQty(fresh.ID).Equal(dec(4)))
	require.True(t, batchQty(soon.ID).Equal(dec(10)))
	err = repo.DeductWarehouseStock(ctx, tenantID, warehouseID, productID, dec(6), posMov(6))
	require.ErrorIs(t, err, domain.ErrExpiredStockBlocked)

	// 3. The expiry report shows LOT-SOON as EXPIRED (10 on hand) and omits LOT-FRESH (outside 30 days).
	rep, err := uc.GetExpiryReport(ctx, tenantID, userID, "admin", &warehouseID, 30)
	require.NoError(t, err)
	require.Len(t, rep.Lines, 1)
	line := rep.Lines[0]
	require.Equal(t, "LOT-SOON", line.BatchNumber)
	require.Equal(t, domain.ExpirySeverityExpired, line.Severity)
	require.Equal(t, -7, line.DaysLeft)
	require.Equal(t, "RACK-EXP", line.LocationCode)
	require.Equal(t, "Gudang Expiry", line.WarehouseName)
	require.Equal(t, "Susu UHT 1L", line.ProductName)
	require.True(t, line.Quantity.Equal(dec(10)))
	require.Equal(t, 1, rep.Summary.ExpiredBatches)

	// A wider window also lists LOT-FRESH as WARNING.
	rep, err = uc.GetExpiryReport(ctx, tenantID, userID, "admin", &warehouseID, 365)
	require.NoError(t, err)
	require.Len(t, rep.Lines, 2)
	require.Equal(t, domain.ExpirySeverityWarning, rep.Lines[1].Severity)

	// RLS: another tenant cannot see these batches.
	lines, err := repo.ListExpiringBatchBalances(ctx, otherTenantID, []uuid.UUID{warehouseID}, today.AddDate(1, 0, 0))
	require.NoError(t, err)
	require.Empty(t, lines)

	// 4. Expired stock can still be scrapped (leaves the rack through SCRAP).
	scrapLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
	require.NoError(t, err)
	require.NoError(t, repo.DeductLocationStock(ctx, tenantID, rackLocID, productID, dec(10), &domain.StockMovement{
		ID: uuid.New(), TenantID: tenantID, MovementNumber: "SCRAP-EXP-1", ProductID: productID,
		SourceLocationID: rackLocID, DestLocationID: scrapLoc.ID, Quantity: dec(10),
		Status: domain.StockMovementStatusDone, ReferenceType: domain.StockRefScrap, ReferenceID: uuid.New(), ExecutedBy: &userID,
	}))
	require.True(t, batchQty(soon.ID).IsZero(), "scrap drains the expired lot first")
	require.True(t, batchQty(fresh.ID).Equal(dec(4)))

	rep, err = uc.GetExpiryReport(ctx, tenantID, userID, "admin", &warehouseID, 30)
	require.NoError(t, err)
	require.Empty(t, rep.Lines, "scrapped stock no longer appears in the report")
}
