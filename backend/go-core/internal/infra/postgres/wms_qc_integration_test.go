package postgres_test

// Real PG15 integration test for Sprint 2 QC Inbound / Karantina / BAK.
// Skipped unless TAYOOLI_PG_INTEGRATION_DSN points at a THROWAWAY database.
// Runs as a non-superuser so FORCE ROW LEVEL SECURITY is enforced.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
	wmsuc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/migrations"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const qcAppRole = "tayooli_app_qc_it"

func TestQCQuarantineRealPostgres(t *testing.T) {
	adminDSN := os.Getenv("TAYOOLI_PG_INTEGRATION_DSN")
	if adminDSN == "" {
		t.Skip("set TAYOOLI_PG_INTEGRATION_DSN to a throwaway PostgreSQL 15 database to run")
	}
	ctx := context.Background()
	admin, err := sql.Open("postgres", adminDSN)
	require.NoError(t, err)
	defer admin.Close()
	require.NoError(t, migrations.Run(ctx, admin))

	for _, tbl := range []string{"qc_inspections", "qc_inspection_items"} {
		var forced bool
		require.NoError(t, admin.QueryRowContext(ctx, `SELECT relforcerowsecurity FROM pg_class WHERE relname = $1`, tbl).Scan(&forced))
		require.True(t, forced, "%s must FORCE RLS", tbl)
	}

	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP OWNED BY %s`, qcAppRole))
	_, _ = admin.ExecContext(ctx, fmt.Sprintf(`DROP ROLE IF EXISTS %s`, qcAppRole))
	for _, q := range []string{
		fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD 'it'`, qcAppRole),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, qcAppRole),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s`, qcAppRole),
		fmt.Sprintf(`GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s`, qcAppRole),
	} {
		_, err := admin.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}

	tenantID, otherTenant, userID := uuid.New(), uuid.New(), uuid.New()
	productID, warehouseID, binID := uuid.New(), uuid.New(), uuid.New()
	for _, s := range []struct {
		q    string
		args []any
	}{
		{`INSERT INTO tenants (id, name) VALUES ($1, 'QC Tenant'), ($2, 'QC Other')`, []any{tenantID, otherTenant}},
		{`INSERT INTO users (id, tenant_id, email, password_hash, role) VALUES ($1, $2, $3, 'x', 'admin')`, []any{userID, tenantID, "qc-" + userID.String() + "@example.test"}},
		{`INSERT INTO products (id, tenant_id, name, sku, price) VALUES ($1, $2, 'Teh Celup', $3, 10000)`, []any{productID, tenantID, "TEH-" + productID.String()[:8]}},
		{`INSERT INTO warehouses (id, tenant_id, code, name) VALUES ($1, $2, $3, 'Gudang QC')`, []any{warehouseID, tenantID, "QC-" + warehouseID.String()[:6]}},
		{`INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, code, name, type) VALUES ($1, $2, $3, 'R-01', 'Rak 1', 'INTERNAL')`, []any{binID, tenantID, warehouseID}},
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
	app, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=it dbname=%s sslmode=disable", host, port, qcAppRole, dbname))
	require.NoError(t, err)
	defer app.Close()

	repo := postgres.NewWMSRepo(app)
	uc := wmsuc.New(repo)
	dec := decimal.NewFromInt
	newPosted := func(qty int64) (uuid.UUID, uuid.UUID) {
		rc, _, err := uc.CreateStockReceipt(ctx, tenantID, userID, "admin", wmsuc.StockReceiptRequest{
			WarehouseID: warehouseID, DestLocationID: binID, SupplierName: "PT Teh",
			Items: []wmsuc.StockReceiptItemRequest{{ProductID: productID, AcceptedQty: dec(qty)}},
		})
		require.NoError(t, err)
		_, err = uc.PostStockReceipt(ctx, tenantID, userID, "admin", rc.ID)
		require.NoError(t, err)
		var batchID uuid.UUID
		require.NoError(t, admin.QueryRowContext(ctx, `SELECT batch_id FROM stock_receipt_items WHERE receipt_id = $1`, rc.ID).Scan(&batchID))
		return rc.ID, batchID
	}
	stg, err := repo.GetOrCreateStagingLocation(ctx, tenantID, warehouseID)
	require.NoError(t, err)
	qrn, err := repo.GetOrCreateQuarantineLocation(ctx, tenantID, warehouseID)
	require.NoError(t, err)
	require.Equal(t, domain.LocationTypeQuarantine, qrn.Type)
	again, err := repo.GetOrCreateQuarantineLocation(ctx, tenantID, warehouseID)
	require.NoError(t, err)
	require.Equal(t, qrn.ID, again.ID, "quarantine bin must be idempotent per warehouse")
	stockAt := func(loc uuid.UUID) decimal.Decimal {
		q, err := repo.GetStockByLocation(ctx, tenantID, loc, productID)
		require.NoError(t, err)
		return q
	}

	// 1. Damaged goods: staged 10, counted 9 (shortage 1), 3 damaged -> 3 to QRN, BAK numbered.
	rcID, batchID := newPosted(10)
	driver, why := "Pak Joko", "Dus basah"
	in := domain.QCInspectionInput{
		InspectionMode: domain.QCModeFull, GrossCartons: 2, DriverName: &driver, DriverSigned: true,
		Items: []domain.QCLineInput{{BatchID: batchID, CheckedQty: dec(9), DamagedQty: dec(3), DamageReason: &why}},
	}

	// Concurrent double-submit: exactly one wins, ledger moves once.
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = uc.SubmitQCInspection(ctx, tenantID, userID, "admin", rcID, in)
		}(i)
	}
	wg.Wait()
	okCount := 0
	for _, e := range errs {
		if e == nil {
			okCount++
		} else {
			require.ErrorIs(t, e, domain.ErrQCAlreadyInspected)
		}
	}
	require.Equal(t, 1, okCount)
	require.True(t, stockAt(stg.ID).Equal(dec(7)), "staging = %s", stockAt(stg.ID))
	require.True(t, stockAt(qrn.ID).Equal(dec(3)), "quarantine = %s", stockAt(qrn.ID))

	det, err := uc.GetReceiptQC(ctx, tenantID, userID, "admin", rcID)
	require.NoError(t, err)
	require.NotNil(t, det)
	require.Equal(t, domain.QCStatusQuarantined, det.Inspection.Status)
	require.NotNil(t, det.Inspection.BAKNumber)
	require.Contains(t, *det.Inspection.BAKNumber, "BAK-")
	require.Equal(t, userID, det.Inspection.InspectorID)
	require.True(t, det.Inspection.ShortageQty.Equal(dec(1)))
	require.Len(t, det.Items, 1)
	require.Equal(t, "Teh Celup", det.Items[0].ProductName)

	var auditCount int
	require.NoError(t, admin.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs WHERE entity_type = 'qc_inspection' AND entity_id = $1`, det.Inspection.ID).Scan(&auditCount))
	require.Equal(t, 1, auditCount)

	// 2. Quarantine is not sellable/pickable: warehouse-level deduction ignores QRN,
	//    and a direct location deduction from QRN is refused.
	_, err = repo.GetStockByLocation(ctx, tenantID, qrn.ID, productID)
	require.NoError(t, err)
	err = repo.DeductWarehouseStock(ctx, tenantID, warehouseID, productID, dec(1), &domain.StockMovement{
		ID: uuid.New(), TenantID: tenantID, MovementNumber: "IT-POS-" + uuid.NewString()[:8], ProductID: productID,
		Quantity: dec(1), Status: domain.StockMovementStatusDone, ReferenceType: domain.StockRefPOS, ReferenceID: uuid.New(),
	})
	require.ErrorIs(t, err, domain.ErrInsufficientStock, "only staging+quarantine stock exists; neither is sellable")
	custLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
	require.NoError(t, err)
	err = repo.DeductLocationStock(ctx, tenantID, qrn.ID, productID, dec(1), &domain.StockMovement{
		ID: uuid.New(), TenantID: tenantID, MovementNumber: "IT-PICK-" + uuid.NewString()[:8], ProductID: productID,
		DestLocationID: custLoc.ID, Quantity: dec(1), Status: domain.StockMovementStatusDone,
		ReferenceType: domain.StockRefDeliveryOrder, ReferenceID: uuid.New(), BatchID: &batchID,
	})
	require.ErrorIs(t, err, domain.ErrQuarantineStockBlocked)
	require.True(t, stockAt(qrn.ID).Equal(dec(3)))

	// 3. Release 1 back to staging (re-enters putaway); over-release refused.
	act := domain.QuarantineActionInput{WarehouseID: warehouseID, ProductID: productID, BatchID: batchID, Quantity: dec(1)}
	_, err = uc.ReleaseQuarantine(ctx, tenantID, userID, "admin", act)
	require.NoError(t, err)
	require.True(t, stockAt(qrn.ID).Equal(dec(2)))
	require.True(t, stockAt(stg.ID).Equal(dec(8)))
	act.Quantity = dec(5)
	_, err = uc.ReleaseQuarantine(ctx, tenantID, userID, "admin", act)
	require.ErrorIs(t, err, domain.ErrQuarantineQtyInvalid)

	// 4. Scrap the remaining 2 with notes.
	notes := "Hancur, tidak layak jual"
	act.Quantity, act.Notes = dec(2), &notes
	_, err = uc.ScrapQuarantine(ctx, tenantID, userID, "admin", act)
	require.NoError(t, err)
	require.True(t, stockAt(qrn.ID).IsZero())
	scrapLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
	require.NoError(t, err)
	require.True(t, stockAt(scrapLoc.ID).GreaterThanOrEqual(dec(2)))

	// 5. Clean receipt: QC_PASSED, no movement, no BAK.
	rc2, b2 := newPosted(4)
	det2, err := uc.SubmitQCInspection(ctx, tenantID, userID, "admin", rc2, domain.QCInspectionInput{
		InspectionMode: domain.QCModeFull, Items: []domain.QCLineInput{{BatchID: b2, CheckedQty: dec(4)}},
	})
	require.NoError(t, err)
	require.Equal(t, domain.QCStatusPassed, det2.Inspection.Status)
	require.Nil(t, det2.Inspection.BAKNumber)
	var qcMovs int
	require.NoError(t, admin.QueryRowContext(ctx, `SELECT count(*) FROM stock_movements WHERE reference_type = 'QC_INSPECTION' AND reference_id = $1`, det2.Inspection.ID).Scan(&qcMovs))
	require.Zero(t, qcMovs)

	// 6. Damage without signed BAK is rejected and leaves no trace (atomic).
	rc3, b3 := newPosted(5)
	_, err = uc.SubmitQCInspection(ctx, tenantID, userID, "admin", rc3, domain.QCInspectionInput{
		InspectionMode: domain.QCModeFull, Items: []domain.QCLineInput{{BatchID: b3, CheckedQty: dec(5), DamagedQty: dec(1), DamageReason: &why}},
	})
	require.ErrorIs(t, err, domain.ErrQCBAKDriverRequired)
	none, err := uc.GetReceiptQC(ctx, tenantID, userID, "admin", rc3)
	require.NoError(t, err)
	require.Nil(t, none)

	// 7. RLS: other tenant sees nothing.
	list, err := repo.ListQCInspections(ctx, otherTenant, warehouseID)
	require.NoError(t, err)
	require.Empty(t, list)
	own, err := uc.ListQCInspections(ctx, tenantID, userID, "admin", warehouseID)
	require.NoError(t, err)
	require.Len(t, own, 2)
	q, err := uc.ListQuarantineStock(ctx, tenantID, userID, "admin", warehouseID)
	require.NoError(t, err)
	require.Empty(t, q)
}
