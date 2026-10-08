package postgres_test

// Real PG15 integration test for Sprint 3 Outbound: FEFO Allocation,
// Delivery Orders with Customer & Actors, Picking Tasks, and Pack Station 100% Scan (TEST-03).
// Skipped unless TAYOOLI_PG_INTEGRATION_DSN points at a THROWAWAY database.

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

const outboundAppRole = "tayooli_app_outbound_it"

func TestOutboundRealPostgres(t *testing.T) {
	adminDSN := os.Getenv("TAYOOLI_PG_INTEGRATION_DSN")
	if adminDSN == "" {
		t.Skip("set TAYOOLI_PG_INTEGRATION_DSN to run real PostgreSQL integration test")
	}
	ctx := context.Background()
	admin, err := sql.Open("postgres", adminDSN)
	require.NoError(t, err)
	defer admin.Close()
	_, _ = admin.ExecContext(ctx, `DELETE FROM schema_migrations WHERE filename = '035_wms_outbound_waves_and_manifests.sql'`)
	require.NoError(t, migrations.Run(ctx, admin))

	// Verify table RLS on picking_tasks
	for _, tbl := range []string{"picking_tasks", "picking_task_items"} {
		var forced bool
		require.NoError(t, admin.QueryRowContext(ctx, `SELECT relforcerowsecurity FROM pg_class WHERE relname = $1`, tbl).Scan(&forced))
		require.True(t, forced, "%s must FORCE RLS", tbl)
	}

	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP OWNED BY %s`, outboundAppRole))
	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP ROLE IF EXISTS %s`, outboundAppRole))
	for _, q := range []string{
		fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD 'it'`, outboundAppRole),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, outboundAppRole),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s`, outboundAppRole),
		fmt.Sprintf(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s`, outboundAppRole),
	} {
		_, err := admin.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}

	tenantID, userID, customerID := uuid.New(), uuid.New(), uuid.New()
	productID, warehouseID, rackLocID := uuid.New(), uuid.New(), uuid.New()

	for _, s := range []struct {
		q    string
		args []any
	}{
		{`INSERT INTO tenants (id, name) VALUES ($1, 'Outbound Tenant')`, []any{tenantID}},
		{`INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES ($1, $2, $3, 'x', 'admin')`, []any{userID, tenantID, "outbound-" + userID.String() + "@example.test"}},
		{`INSERT INTO customers (id, tenant_id, name) VALUES ($1, $2, 'Toko Berkah')`, []any{customerID, tenantID}},
		{`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1, $2, 'Beras Ramos 5kg', $3, 75000)`, []any{productID, tenantID, "BERAS-" + productID.String()[:6]}},
		{`INSERT INTO warehouses (id, tenant_id, code, name) VALUES ($1, $2, $3, 'Gudang Distribusi')`, []any{warehouseID, tenantID, "WH-" + warehouseID.String()[:4]}},
		{`INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type) VALUES ($1, $2, $3, 'RACK-01', 'Rak A-01', 'INTERNAL')`, []any{rackLocID, tenantID, warehouseID}},
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
	app, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=it dbname=%s sslmode=disable", host, port, outboundAppRole, dbname))
	require.NoError(t, err)
	defer app.Close()

	repo := postgres.NewWMSRepo(app)
	uc := wmsuc.New(repo)
	dec := decimal.NewFromInt

	// Create 2 batches: batch 1 expires in 5 days, batch 2 expires in 60 days
	now := time.Now().UTC()
	exp1 := now.Add(5 * 24 * time.Hour)
	exp2 := now.Add(60 * 24 * time.Hour)

	b1 := &domain.StockBatch{ID: uuid.New(), TenantID: tenantID, ProductID: productID, BatchNumber: "LOT-EXP-EARLY", ExpiryDate: &exp1, Status: domain.StockBatchStatusReleased}
	b2 := &domain.StockBatch{ID: uuid.New(), TenantID: tenantID, ProductID: productID, BatchNumber: "LOT-EXP-LATE", ExpiryDate: &exp2, Status: domain.StockBatchStatusReleased}
	_, err = repo.GetOrCreateBatch(ctx, b1)
	require.NoError(t, err)
	_, err = repo.GetOrCreateBatch(ctx, b2)
	require.NoError(t, err)

	// Seed stock: 10 units in b1, 20 units in b2 at rackLocID
	suppLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeVendor)
	require.NoError(t, err)
	require.NoError(t, repo.CreateStockMovement(ctx, &domain.StockMovement{
		ID: uuid.New(), TenantID: tenantID, MovementNumber: "SEED-1", ProductID: productID,
		SourceLocationID: suppLoc.ID, DestLocationID: rackLocID, Quantity: dec(10),
		Status: domain.StockMovementStatusDone, ReferenceType: "SEED", ReferenceID: uuid.New(), BatchID: &b1.ID,
	}))
	require.NoError(t, repo.CreateStockMovement(ctx, &domain.StockMovement{
		ID: uuid.New(), TenantID: tenantID, MovementNumber: "SEED-2", ProductID: productID,
		SourceLocationID: suppLoc.ID, DestLocationID: rackLocID, Quantity: dec(20),
		Status: domain.StockMovementStatusDone, ReferenceType: "SEED", ReferenceID: uuid.New(), BatchID: &b2.ID,
	}))

	// 1. Create DO with auto-FEFO allocation: request 12 units
	// Should take 10 from b1 (earliest) and 2 from b2 (later)
	doReq := wmsuc.CreateDeliveryOrderRequest{
		CustomerID:  &customerID,
		WarehouseID: warehouseID,
		DONumber:    "DO-IT-001",
		Items: []wmsuc.CreateDeliveryOrderItemRequest{
			{
				ProductID: productID,
				Quantity:  dec(12),
			},
		},
	}
	do, err := uc.CreateDeliveryOrder(ctx, tenantID, userID, "admin", doReq)
	require.NoError(t, err)
	require.NotNil(t, do)
	require.Equal(t, domain.DeliveryOrderStatusDraft, do.Status)
	require.Equal(t, &customerID, do.CustomerID)

	// Verify items were split into FEFO batches
	fetchedDO, items, err := uc.GetDeliveryOrder(ctx, tenantID, userID, "admin", do.ID)
	require.NoError(t, err)
	require.Equal(t, "Toko Berkah", *fetchedDO.CustomerName)
	require.Len(t, items, 2)
	require.Equal(t, b1.ID, *items[0].BatchID)
	require.True(t, items[0].Quantity.Equal(dec(10)))
	require.Equal(t, b2.ID, *items[1].BatchID)
	require.True(t, items[1].Quantity.Equal(dec(2)))

	// 2. Picking Task auto-generated & sorted by shelf order
	pickTask, err := uc.GetPickingTask(ctx, tenantID, userID, "admin", do.ID)
	require.NoError(t, err)
	require.NotNil(t, pickTask)
	require.Len(t, pickTask.Items, 2)
	require.Equal(t, 1, pickTask.Items[0].ShelfOrder)

	// 3. Pack Station Scan 100% verification
	sku := items[0].ProductSKU
	require.NotNil(t, sku)

	// Try completing before scanning -> fails ErrPackStationIncomplete
	_, err = uc.CompletePackStation(ctx, tenantID, userID, "admin", do.ID, domain.PackCompleteRequest{})
	require.ErrorIs(t, err, domain.ErrPackStationIncomplete)

	// Scan items: 10 units for item 1, 2 units for item 2
	for i := 0; i < 10; i++ {
		scanRes, err := uc.ScanPackStationItem(ctx, tenantID, userID, "admin", do.ID, domain.PackScanRequest{
			Barcode:  *sku,
			Quantity: dec(1),
		})
		require.NoError(t, err)
		if i == 9 {
			require.True(t, scanRes.ItemCompleted)
		}
	}
	for i := 0; i < 2; i++ {
		scanRes, err := uc.ScanPackStationItem(ctx, tenantID, userID, "admin", do.ID, domain.PackScanRequest{
			Barcode:  *sku,
			Quantity: dec(1),
		})
		require.NoError(t, err)
		if i == 1 {
			require.True(t, scanRes.OrderCompleted)
		}
	}

	// 4. Complete packing with package dimensions & weight
	weight := decimal.NewFromFloat(15.5)
	packedDO, err := uc.CompletePackStation(ctx, tenantID, userID, "admin", do.ID, domain.PackCompleteRequest{
		PackageWeightKg: &weight,
	})
	require.NoError(t, err)
	require.Equal(t, domain.DeliveryOrderStatusPacked, packedDO.Status)
	require.Equal(t, &userID, packedDO.PackedBy)

	// 5. Dispatch Delivery Order: deducts stock atomically with batch IDs preserved
	shippedDO, err := uc.DispatchDeliveryOrder(ctx, tenantID, userID, "admin", do.ID)
	require.NoError(t, err)
	require.Equal(t, domain.DeliveryOrderStatusShipped, shippedDO.Status)

	// Verify remaining stock at rack
	remStock, err := repo.GetStockByLocation(ctx, tenantID, rackLocID, productID)
	require.NoError(t, err)
	require.True(t, remStock.Equal(dec(18)), "Initial was 30, dispatched 12, remaining must be 18")
}

func TestPickWaveGroupingRealPostgres(t *testing.T) {
	adminDSN := os.Getenv("TAYOOLI_PG_INTEGRATION_DSN")
	if adminDSN == "" {
		t.Skip("set TAYOOLI_PG_INTEGRATION_DSN to run real PostgreSQL integration test")
	}
	ctx := context.Background()
	admin, err := sql.Open("postgres", adminDSN)
	require.NoError(t, err)
	defer admin.Close()

	repo := postgres.NewWMSRepo(admin)
	uc := wmsuc.New(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	whID := uuid.New()

	_, err = admin.ExecContext(ctx, `
		INSERT INTO tenants (id, name, created_at, updated_at) VALUES ($1, 'Tenant Wave', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`, tenantID)
	require.NoError(t, err)

	userEmail := fmt.Sprintf("wave-%s@tayooli.test", userID.String()[:8])
	_, err = admin.ExecContext(ctx, `
		INSERT INTO users (id, tenant_id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, 'x', 'admin', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`, userID, tenantID, userEmail)
	require.NoError(t, err)

	_, err = admin.ExecContext(ctx, `
		INSERT INTO warehouses (id, tenant_id, code, name, created_at, updated_at)
		VALUES ($1, $2, 'WH-WAVE', 'Gudang Wave Release', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`, whID, tenantID)
	require.NoError(t, err)

	p1 := uuid.New()
	p2 := uuid.New()
	_, err = admin.ExecContext(ctx, `
		INSERT INTO products (id, tenant_id, sku, name, price, created_at, updated_at)
		VALUES ($1, $3, 'SKU-WAVE-1', 'Wave Product 1', 10000, NOW(), NOW()),
		       ($2, $3, 'SKU-WAVE-2', 'Wave Product 2', 20000, NOW(), NOW())`, p1, p2, tenantID)
	require.NoError(t, err)

	rackLocID := uuid.New()
	_, err = admin.ExecContext(ctx, `
		INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type, created_at, updated_at)
		VALUES ($1, $2, $3, 'BIN-WV-01', 'Bin Wave 1', 'INTERNAL', NOW(), NOW())`,
		rackLocID, tenantID, whID)
	require.NoError(t, err)

	expJNE := "JNE"
	expSiCepat := "SICEPAT"
	routeJkt := "JABODETABEK"
	otMarketplace := "MARKETPLACE"
	otDirect := "DIRECT_DO"

	do1, err := uc.CreateDeliveryOrder(ctx, tenantID, userID, "admin", wmsuc.CreateDeliveryOrderRequest{
		WarehouseID:    whID,
		OrderType:      &otMarketplace,
		ExpeditionName: &expJNE,
		DONumber:       "DO-WAVE-01",
		Items: []wmsuc.CreateDeliveryOrderItemRequest{
			{ProductID: p1, Quantity: decimal.NewFromInt(2), LocationID: rackLocID},
		},
	})
	require.NoError(t, err)

	do2, err := uc.CreateDeliveryOrder(ctx, tenantID, userID, "admin", wmsuc.CreateDeliveryOrderRequest{
		WarehouseID:    whID,
		OrderType:      &otMarketplace,
		ExpeditionName: &expJNE,
		DONumber:       "DO-WAVE-02",
		Items: []wmsuc.CreateDeliveryOrderItemRequest{
			{ProductID: p2, Quantity: decimal.NewFromInt(3), LocationID: rackLocID},
		},
	})
	require.NoError(t, err)

	do3, err := uc.CreateDeliveryOrder(ctx, tenantID, userID, "admin", wmsuc.CreateDeliveryOrderRequest{
		WarehouseID:    whID,
		OrderType:      &otDirect,
		ExpeditionName: &expSiCepat,
		DONumber:       "DO-WAVE-03",
		Items: []wmsuc.CreateDeliveryOrderItemRequest{
			{ProductID: p1, Quantity: decimal.NewFromInt(1), LocationID: rackLocID},
		},
	})
	require.NoError(t, err)

	// 1. Generate Wave grouped by MARKETPLACE and JNE (PDF-05)
	wave, err := uc.CreatePickWave(ctx, tenantID, userID, "admin", domain.CreatePickWaveRequest{
		WarehouseID:    whID,
		OrderType:      "MARKETPLACE",
		ExpeditionName: &expJNE,
		RouteZone:      &routeJkt,
	})
	require.NoError(t, err)
	require.Equal(t, domain.PickWaveStatusOpen, wave.Status)
	require.Equal(t, 2, wave.TotalOrders, "Should group DO 1 and DO 2, excluding DO 3")
	require.Equal(t, "MARKETPLACE", wave.OrderType)
	require.Equal(t, &expJNE, wave.ExpeditionName)

	// 2. Fetch Wave Detail
	detail, err := uc.GetPickWaveByID(ctx, tenantID, userID, "admin", wave.ID)
	require.NoError(t, err)
	require.Len(t, detail.PickingTasks, 2)
	assertDOIDs := []uuid.UUID{detail.PickingTasks[0].DeliveryOrder.ID, detail.PickingTasks[1].DeliveryOrder.ID}
	require.Contains(t, assertDOIDs, do1.ID)
	require.Contains(t, assertDOIDs, do2.ID)
	require.NotContains(t, assertDOIDs, do3.ID)

	// 3. Release Wave
	releasedWave, err := uc.ReleasePickWave(ctx, tenantID, userID, "admin", wave.ID, &userID)
	require.NoError(t, err)
	require.Equal(t, domain.PickWaveStatusReleased, releasedWave.Status)
	require.Equal(t, &userID, releasedWave.PickerID)

	// Verify linked picking tasks are now IN_PROGRESS
	task1, err := uc.GetPickingTask(ctx, tenantID, userID, "admin", do1.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PickingTaskStatusInProgress, task1.Task.Status)

	// 4. Multi-tenancy isolation: Tenant B cannot list or access wave of Tenant A
	tenantB := uuid.New()
	_, err = admin.ExecContext(ctx, `
		INSERT INTO tenants (id, name, created_at, updated_at) VALUES ($1, 'Tenant B Wave', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`, tenantB)
	require.NoError(t, err)

	wavesB, err := uc.ListPickWaves(ctx, tenantB, userID, "admin", nil, nil, nil, nil)
	require.NoError(t, err)
	require.Empty(t, wavesB)

	_, err = uc.GetPickWaveByID(ctx, tenantB, userID, "admin", wave.ID)
	require.ErrorIs(t, err, domain.ErrPickWaveNotFound)
}
