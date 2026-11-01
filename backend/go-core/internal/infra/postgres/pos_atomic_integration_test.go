package postgres_test

// Real PostgreSQL integration test for the atomic POS checkout
// (POSRepo.CreateSale): stock deductions, sales order/invoice and the POS order
// commit together or not at all, concurrent carts do not deadlock and cannot
// oversell. Runs as a non-superuser app role so RLS is enforced. Skipped unless
// TAYOOLI_PG_INTEGRATION_DSN points at a THROWAWAY database.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
	posuc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pos"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/migrations"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const posAtomicAppRole = "tayooli_app_pos_atomic_it"

func TestPOSCheckoutAtomicRealPostgres(t *testing.T) {
	adminDSN := os.Getenv("TAYOOLI_PG_INTEGRATION_DSN")
	if adminDSN == "" {
		t.Skip("set TAYOOLI_PG_INTEGRATION_DSN to run real PostgreSQL integration test")
	}
	ctx := context.Background()
	admin, err := sql.Open("postgres", adminDSN)
	require.NoError(t, err)
	defer admin.Close()
	require.NoError(t, migrations.Run(ctx, admin))

	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP OWNED BY %s`, posAtomicAppRole))
	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP ROLE IF EXISTS %s`, posAtomicAppRole))
	for _, q := range []string{
		fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD 'it'`, posAtomicAppRole),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, posAtomicAppRole),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s`, posAtomicAppRole),
		fmt.Sprintf(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s`, posAtomicAppRole),
	} {
		_, err := admin.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}

	tenantID, userID, warehouseID, rackID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	prodA, prodB, prodC := uuid.New(), uuid.New(), uuid.New()
	short := func(id uuid.UUID) string { return id.String()[:6] }
	for _, s := range []struct {
		q    string
		args []any
	}{
		{`INSERT INTO tenants (id, name) VALUES ($1, 'POS Atomic Tenant')`, []any{tenantID}},
		{`INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES ($1, $2, $3, 'x', 'admin')`, []any{userID, tenantID, "pos-atomic-" + userID.String() + "@example.test"}},
		{`INSERT INTO customers (tenant_id, name) VALUES ($1, 'Pelanggan Umum (POS)')`, []any{tenantID}},
		{`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1, $2, 'Gula 1kg', $3, 15000)`, []any{prodA, tenantID, "GULA-" + short(prodA)}},
		{`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1, $2, 'Teh 25s', $3, 8000)`, []any{prodB, tenantID, "TEH-" + short(prodB)}},
		{`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1, $2, 'Kopi Langka', $3, 50000)`, []any{prodC, tenantID, "KOPI-" + short(prodC)}},
		{`INSERT INTO warehouses (id, tenant_id, code, name) VALUES ($1, $2, $3, 'Gudang POS')`, []any{warehouseID, tenantID, "WH-" + short(warehouseID)}},
		{`INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type) VALUES ($1, $2, $3, 'RAK-POS', 'Rak POS', 'INTERNAL')`, []any{rackID, tenantID, warehouseID}},
		// Legacy inventory rows for A and B (C has none: WMS ledger only).
		{`INSERT INTO inventory (tenant_id, product_id, quantity) VALUES ($1, $2, 100)`, []any{tenantID, prodA}},
		{`INSERT INTO inventory (tenant_id, product_id, quantity) VALUES ($1, $2, 100)`, []any{tenantID, prodB}},
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
	app, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=it dbname=%s sslmode=disable", host, port, posAtomicAppRole, dbname))
	require.NoError(t, err)
	defer app.Close()
	app.SetMaxOpenConns(25)

	wmsRepo := postgres.NewWMSRepo(app)
	posRepo := postgres.NewPOSRepo(app)
	uc := posuc.New(posRepo, postgres.NewProductRepo(app), postgres.NewInventoryRepo(app), wmsRepo,
		postgres.NewCustomerRepo(app), postgres.NewSalesOrderRepo(app), postgres.NewSalesInvoiceRepo(app))
	dec := decimal.NewFromInt

	// WMS stock: A 60, B 60, C 5 in one fresh batch each.
	vendor, err := wmsRepo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeVendor)
	require.NoError(t, err)
	for _, seed := range []struct {
		prod uuid.UUID
		qty  int64
	}{{prodA, 60}, {prodB, 60}, {prodC, 5}} {
		exp := domain.BusinessToday(time.Now()).AddDate(1, 0, 0)
		b, err := wmsRepo.GetOrCreateBatch(ctx, &domain.StockBatch{TenantID: tenantID, ProductID: seed.prod, BatchNumber: "LOT-" + short(seed.prod), ExpiryDate: &exp, Status: domain.StockBatchStatusReleased})
		require.NoError(t, err)
		require.NoError(t, wmsRepo.CreateStockMovement(ctx, &domain.StockMovement{
			ID: uuid.New(), TenantID: tenantID, MovementNumber: "SEED-POS-" + short(seed.prod), ProductID: seed.prod,
			SourceLocationID: vendor.ID, DestLocationID: rackID, Quantity: dec(seed.qty),
			Status: domain.StockMovementStatusDone, ReferenceType: "SEED", ReferenceID: uuid.New(), BatchID: &b.ID,
		}))
	}

	type snapshot struct {
		wmsA, wmsB, wmsC, invA, invB  decimal.Decimal
		orders, items, so, si, posMov int
	}
	// key renders a snapshot by value (decimal internals are not comparable).
	key := func(s snapshot) string {
		return fmt.Sprintf("wmsA=%s wmsB=%s wmsC=%s invA=%s invB=%s orders=%d items=%d so=%d si=%d posMov=%d",
			s.wmsA.String(), s.wmsB.String(), s.wmsC.String(), s.invA.String(), s.invB.String(),
			s.orders, s.items, s.so, s.si, s.posMov)
	}
	snap := func() snapshot {
		t.Helper()
		var s snapshot
		wms := func(p uuid.UUID) decimal.Decimal {
			q, err := wmsRepo.GetStockByLocation(ctx, tenantID, rackID, p)
			require.NoError(t, err)
			return q
		}
		s.wmsA, s.wmsB, s.wmsC = wms(prodA), wms(prodB), wms(prodC)
		require.NoError(t, admin.QueryRowContext(ctx, `SELECT COALESCE(SUM(quantity),0) FROM inventory WHERE tenant_id=$1 AND product_id=$2`, tenantID, prodA).Scan(&s.invA))
		require.NoError(t, admin.QueryRowContext(ctx, `SELECT COALESCE(SUM(quantity),0) FROM inventory WHERE tenant_id=$1 AND product_id=$2`, tenantID, prodB).Scan(&s.invB))
		for _, c := range []struct {
			q   string
			dst *int
		}{
			{`SELECT count(*) FROM pos_orders WHERE tenant_id=$1`, &s.orders},
			{`SELECT count(*) FROM pos_order_items WHERE tenant_id=$1`, &s.items},
			{`SELECT count(*) FROM sales_orders WHERE tenant_id=$1`, &s.so},
			{`SELECT count(*) FROM sales_invoices WHERE tenant_id=$1`, &s.si},
			{`SELECT count(*) FROM stock_movements WHERE tenant_id=$1 AND reference_type='POS_SALE'`, &s.posMov},
		} {
			require.NoError(t, admin.QueryRowContext(ctx, c.q, tenantID).Scan(c.dst))
		}
		return s
	}
	cart := func(lines ...struct {
		p uuid.UUID
		q int64
	}) posuc.CheckoutRequest {
		req := posuc.CheckoutRequest{WarehouseID: &warehouseID}
		for _, l := range lines {
			req.Items = append(req.Items, posuc.CheckoutItemRequest{ProductID: l.p, Quantity: dec(l.q)})
		}
		return req
	}
	type line = struct {
		p uuid.UUID
		q int64
	}

	// 1. Success: everything is written together.
	before := snap()
	res, err := uc.Checkout(ctx, tenantID, userID, cart(line{prodA, 2}, line{prodB, 3}))
	require.NoError(t, err)
	after := snap()
	require.True(t, after.wmsA.Equal(before.wmsA.Sub(dec(2))), "WMS A")
	require.True(t, after.wmsB.Equal(before.wmsB.Sub(dec(3))), "WMS B")
	require.True(t, after.invA.Equal(before.invA.Sub(dec(2))), "legacy A")
	require.True(t, after.invB.Equal(before.invB.Sub(dec(3))), "legacy B")
	require.Equal(t, before.orders+1, after.orders)
	require.Equal(t, before.items+2, after.items)
	require.Equal(t, before.so+1, after.so)
	require.Equal(t, before.si+1, after.si)
	require.Equal(t, before.posMov+2, after.posMov)
	require.NotNil(t, res.SalesOrderID)
	require.NotNil(t, res.SalesInvoiceID)

	// 2. Second line short on WMS stock: the first line must NOT be deducted.
	//    (Old code deducted A's legacy inventory and WMS stock before failing on B.)
	before = snap()
	_, err = uc.Checkout(ctx, tenantID, userID, cart(line{prodA, 5}, line{prodB, 99}))
	require.ErrorIs(t, err, domain.ErrInsufficientStock)
	require.Equal(t, key(before), key(snap()), "refused sale must change nothing")

	// 3. Failure at the very last write (the 2026-11-01 load-test bug): the POS
	//    order insert violates the unique order number. Stock must roll back.
	var existingNumber string
	require.NoError(t, admin.QueryRowContext(ctx, `SELECT order_number FROM pos_orders WHERE tenant_id=$1 LIMIT 1`, tenantID).Scan(&existingNumber))
	cust, err := wmsRepo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
	require.NoError(t, err)
	before = snap()
	orderID := uuid.New()
	err = posRepo.CreateSale(ctx, &domain.POSSale{
		Order: &domain.POSOrder{ID: orderID, TenantID: tenantID, OrderNumber: existingNumber, WarehouseID: &warehouseID,
			TotalAmount: dec(15000), PaymentMethod: "CASH", SaleMode: "DIRECT", Status: "COMPLETED", CashierID: &userID,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		Items:               []domain.POSOrderItem{{ID: uuid.New(), TenantID: tenantID, POSOrderID: orderID, ProductID: prodA, ProductName: "Gula 1kg", SKU: "GULA", Quantity: dec(1), Price: dec(15000), Subtotal: dec(15000), CreatedAt: time.Now().UTC()}},
		InventoryDeductions: []domain.POSInventoryDeduction{{ProductID: prodA, SKU: "GULA", Quantity: dec(1)}},
		StockDeductions: []domain.POSStockDeduction{{WarehouseID: warehouseID, ProductID: prodA, SKU: "GULA", Quantity: dec(1),
			Movement: domain.StockMovement{TenantID: tenantID, MovementNumber: "MV-POS-IT-" + short(orderID), ProductID: prodA, DestLocationID: cust.ID,
				Quantity: dec(1), Status: domain.StockMovementStatusDone, ReferenceType: domain.StockRefPOS, ReferenceID: orderID, ExecutedBy: &userID}}},
	})
	require.Error(t, err, "duplicate order number must fail")
	require.Equal(t, key(before), key(snap()), "failed order insert must roll back stock deductions")

	// 4. 20 concurrent carts with the same two products in opposite order:
	//    no deadlock, every sale recorded once.
	before = snap()
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := cart(line{prodA, 1}, line{prodB, 1})
			if i%2 == 1 {
				c = cart(line{prodB, 1}, line{prodA, 1})
			}
			_, err := uc.Checkout(ctx, tenantID, userID, c)
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err, "concurrent checkout (possible deadlock)")
	}
	after = snap()
	require.True(t, after.wmsA.Equal(before.wmsA.Sub(dec(20))), "WMS A after 20 sales: %s -> %s", before.wmsA, after.wmsA)
	require.True(t, after.wmsB.Equal(before.wmsB.Sub(dec(20))), "WMS B after 20 sales")
	require.True(t, after.invA.Equal(before.invA.Sub(dec(20))), "legacy A after 20 sales")
	require.Equal(t, before.orders+20, after.orders)
	require.Equal(t, before.posMov+40, after.posMov)

	// 5. 12 cashiers race for the last 5 units of C: exactly 5 sales succeed.
	before = snap()
	var okCount, shortCount int
	var mu sync.Mutex
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := uc.Checkout(ctx, tenantID, userID, cart(line{prodC, 1}))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				okCount++
			case errors.Is(err, domain.ErrInsufficientStock):
				shortCount++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	after = snap()
	require.Equal(t, 5, okCount, "exactly the 5 units on hand can be sold")
	require.Equal(t, 7, shortCount)
	require.True(t, after.wmsC.IsZero(), "C stock must be 0, got %s", after.wmsC)
	require.Equal(t, before.orders+5, after.orders, "only successful sales create orders")
	require.Equal(t, before.posMov+5, after.posMov)
}
