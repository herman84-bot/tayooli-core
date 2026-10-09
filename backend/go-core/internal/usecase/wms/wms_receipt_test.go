package wms_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// mockWMSRepo: Stock Receipts. Mirrors the postgres semantics: post/cancel are
// all-or-nothing (validate first, then mutate), status re-checked "under lock".
// -----------------------------------------------------------------------------

func (m *mockWMSRepo) initReceipts() {
	if m.stockReceipts == nil {
		m.stockReceipts = make(map[uuid.UUID]*domain.StockReceipt)
		m.receiptItems = make(map[uuid.UUID][]domain.StockReceiptItem)
	}
}

func (m *mockWMSRepo) CreateStockReceipt(ctx context.Context, rc *domain.StockReceipt, items []domain.StockReceiptItem) error {
	m.initReceipts()
	now := time.Now().UTC()
	rc.CreatedAt, rc.UpdatedAt = now, now
	for i := range items {
		items[i].ID = uuid.New()
		items[i].TenantID = rc.TenantID
		items[i].ReceiptID = rc.ID
	}
	cp := *rc
	m.stockReceipts[rc.ID] = &cp
	m.receiptItems[rc.ID] = append([]domain.StockReceiptItem(nil), items...)
	return nil
}

func (m *mockWMSRepo) UpdateDraftStockReceipt(ctx context.Context, rc *domain.StockReceipt, items []domain.StockReceiptItem) error {
	m.initReceipts()
	cur, ok := m.stockReceipts[rc.ID]
	if !ok || cur.TenantID != rc.TenantID {
		return domain.ErrStockReceiptNotFound
	}
	if cur.Status != domain.StockReceiptStatusDraft {
		return domain.ErrStockReceiptNotDraft
	}
	cur.WarehouseID, cur.DestLocationID = rc.WarehouseID, rc.DestLocationID
	cur.ReceiptType, cur.FromName = rc.ReceiptType, rc.FromName
	cur.FromWarehouseID, cur.SourceRef, cur.TransferID = rc.FromWarehouseID, rc.SourceRef, rc.TransferID
	cur.SupplierName, cur.SupplierRef, cur.Notes = rc.SupplierName, rc.SupplierRef, rc.Notes
	for i := range items {
		items[i].ID = uuid.New()
		items[i].TenantID = rc.TenantID
		items[i].ReceiptID = rc.ID
	}
	m.receiptItems[rc.ID] = append([]domain.StockReceiptItem(nil), items...)
	return nil
}

func (m *mockWMSRepo) GetStockReceiptByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
	m.initReceipts()
	rc, ok := m.stockReceipts[id]
	if !ok || rc.TenantID != tenantID {
		return nil, nil, domain.ErrStockReceiptNotFound
	}
	cp := *rc
	items := append([]domain.StockReceiptItem(nil), m.receiptItems[id]...)
	cp.ItemCount = len(items)
	cp.TotalAcceptedQty, cp.TotalRejectedQty = decimal.Zero, decimal.Zero
	for _, it := range items {
		cp.TotalAcceptedQty = cp.TotalAcceptedQty.Add(it.AcceptedQty)
		cp.TotalRejectedQty = cp.TotalRejectedQty.Add(it.RejectedQty)
	}
	return &cp, items, nil
}

func (m *mockWMSRepo) ListStockReceipts(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.StockReceiptStatus, receiptType *domain.StockReceiptType) ([]domain.StockReceipt, error) {
	m.initReceipts()
	var list []domain.StockReceipt
	for id, rc := range m.stockReceipts {
		if rc.TenantID != tenantID || (warehouseID != nil && rc.WarehouseID != *warehouseID) || (status != nil && rc.Status != *status) || (receiptType != nil && rc.ReceiptType != *receiptType) {
			continue
		}
		cp, _, _ := m.GetStockReceiptByID(ctx, tenantID, id)
		list = append(list, *cp)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	return list, nil
}

func (m *mockWMSRepo) PostStockReceipt(ctx context.Context, p domain.PostReceiptParams) (*domain.StockReceipt, error) {
	m.initReceipts()
	rc, ok := m.stockReceipts[p.ReceiptID]
	if !ok || rc.TenantID != p.TenantID {
		return nil, domain.ErrStockReceiptNotFound
	}
	if rc.Status != domain.StockReceiptStatusDraft {
		return nil, domain.ErrStockReceiptNotDraft
	}
	now := time.Now().UTC()
	targetStaging := p.StagingLocID
	if targetStaging == uuid.Nil {
		targetStaging = rc.DestLocationID
	}
	items := m.receiptItems[p.ReceiptID]
	for i := range items {
		it := &items[i]
		batchNum := ""
		if it.BatchNumber != nil && *it.BatchNumber != "" {
			batchNum = *it.BatchNumber
		} else {
			batchNum = domain.AutoBatchNumber(rc.ReceiptNumber, i+1)
		}
		status := domain.StockBatchStatusReleased
		if p.HoldForRelease {
			status = domain.StockBatchStatusOnHold
		}
		batchID := uuid.New()
		b := &domain.StockBatch{
			ID:              batchID,
			TenantID:        p.TenantID,
			ProductID:       it.ProductID,
			BatchNumber:     batchNum,
			ExpiryDate:      it.ExpiryDate,
			SourceReceiptID: &p.ReceiptID,
			Status:          status,
			CreatedBy:       &p.UserID,
			CreatedAt:       now,
		}
		if m.batches == nil {
			m.batches = make(map[uuid.UUID]*domain.StockBatch)
		}
		m.batches[batchID] = b
		it.BatchID = &batchID
		it.BatchNumber = &batchNum

		if it.AcceptedQty.IsPositive() {
			_ = m.CreateStockMovement(ctx, &domain.StockMovement{
				ID:               uuid.New(),
				TenantID:         p.TenantID,
				MovementNumber:   fmt.Sprintf("GR-IN-%s-%d", rc.ReceiptNumber, i+1),
				ProductID:        it.ProductID,
				BatchID:          &batchID,
				SourceLocationID: p.SourceLocID,
				DestLocationID:   targetStaging,
				Quantity:         it.AcceptedQty,
				Status:           domain.StockMovementStatusDone,
				ReferenceType:    domain.StockRefGoodsReceipt,
				ReferenceID:      p.ReceiptID,
				ExecutedBy:       &p.UserID,
				CreatedAt:        now,
			})
		}
		if it.RejectedQty.IsPositive() {
			_ = m.CreateStockMovement(ctx, &domain.StockMovement{
				ID:               uuid.New(),
				TenantID:         p.TenantID,
				MovementNumber:   fmt.Sprintf("GR-REJ-%s-%d", rc.ReceiptNumber, i+1),
				ProductID:        it.ProductID,
				BatchID:          &batchID,
				SourceLocationID: p.SourceLocID,
				DestLocationID:   p.ScrapLocID,
				Quantity:         it.RejectedQty,
				Status:           domain.StockMovementStatusDone,
				ReferenceType:    domain.StockRefGoodsReceipt,
				ReferenceID:      p.ReceiptID,
				ExecutedBy:       &p.UserID,
				CreatedAt:        now,
			})
		}
	}
	m.receiptItems[p.ReceiptID] = items
	rc.Status, rc.PostedBy, rc.PostedAt = domain.StockReceiptStatusPosted, &p.UserID, &now
	cp := *rc
	return &cp, nil
}

func (m *mockWMSRepo) CancelStockReceipt(ctx context.Context, tenantID, id, userID, vendorLocID, scrapLocID uuid.UUID, reason string) (*domain.StockReceipt, error) {
	m.initReceipts()
	rc, ok := m.stockReceipts[id]
	if !ok || rc.TenantID != tenantID {
		return nil, domain.ErrStockReceiptNotFound
	}
	if rc.Status == domain.StockReceiptStatusCancelled {
		return nil, domain.ErrStockReceiptAlreadyCancelled
	}
	now := time.Now().UTC()
	if rc.Status == domain.StockReceiptStatusPosted {
		movs, _ := m.ListMovementsByReference(ctx, tenantID, domain.StockRefGoodsReceipt, id)
		for _, sm := range movs {
			key := fmt.Sprintf("%s:%s:%s", tenantID, sm.DestLocationID, sm.ProductID)
			if m.stockLevels[key].LessThan(sm.Quantity) {
				return nil, domain.ErrStockReceiptStockConsumed
			}
		}
		for i, sm := range movs {
			revMov := sm
			revMov.ID = uuid.New()
			revMov.MovementNumber = fmt.Sprintf("GR-REV-%s-%d", rc.ReceiptNumber, i+1)
			revMov.SourceLocationID = sm.DestLocationID
			revMov.DestLocationID = sm.SourceLocationID
			revMov.ExecutedBy = &userID
			revMov.CreatedAt = now
			_ = m.CreateStockMovement(ctx, &revMov)
		}
	}
	rc.Status, rc.CancelledBy, rc.CancelledAt, rc.CancelReason = domain.StockReceiptStatusCancelled, &userID, &now, &reason
	cp := *rc
	return &cp, nil
}

// -----------------------------------------------------------------------------
// Scenario tests
// -----------------------------------------------------------------------------

func dec(i int64) decimal.Decimal { return decimal.NewFromInt(i) }
func strp(s string) *string       { return &s }

func TestStockReceiptLifecycle(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	otherWhID := uuid.New()
	staffID := uuid.New()
	outsiderID := uuid.New()

	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{ID: whID, TenantID: tenantID, Name: "Main WH", IsActive: true}))
	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{ID: otherWhID, TenantID: tenantID, Name: "Other WH", IsActive: true}))
	require.NoError(t, repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{UserID: staffID, WarehouseID: whID, TenantID: tenantID}))

	binLoc := uuid.New()
	transitLoc := uuid.New()
	otherLoc := uuid.New()
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: binLoc, TenantID: tenantID, WarehouseID: &whID, Code: "BIN-A", Type: domain.LocationTypeInternal}))
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: transitLoc, TenantID: tenantID, WarehouseID: &whID, Code: "TRANSIT", Type: domain.LocationTypeTransit}))
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: otherLoc, TenantID: tenantID, WarehouseID: &otherWhID, Code: "OTHER", Type: domain.LocationTypeInternal}))

	prodA := uuid.New()
	prodB := uuid.New()
	stock := func(loc, prod uuid.UUID) decimal.Decimal {
		s, err := repo.GetStockByLocation(ctx, tenantID, loc, prod)
		require.NoError(t, err)
		return s
	}
	validReq := func() uc.StockReceiptRequest {
		return uc.StockReceiptRequest{
			WarehouseID: whID, DestLocationID: binLoc, SupplierName: "PT Sumber Makmur", SupplierRef: strp("SJ-001"),
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodA, ExpectedQty: decPtr(12), AcceptedQty: dec(10), RejectedQty: dec(2), RejectReason: strp("Kemasan rusak")},
				{ProductID: prodB, AcceptedQty: dec(5), RejectedQty: dec(0)},
			},
		}
	}
	assertInvalid := func(t *testing.T, err error, contains string) {
		t.Helper()
		require.Error(t, err)
		var ve *domain.StockReceiptValidationError
		require.True(t, errors.As(err, &ve), "expected validation error, got %v", err)
		assert.True(t, errors.Is(err, domain.ErrInvalidInput))
		assert.Contains(t, ve.Msg, contains)
	}

	t.Run("validation errors", func(t *testing.T) {
		r := validReq()
		r.Items = nil
		_, _, err := usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", r)
		assertInvalid(t, err, "minimal satu barang")

		r = validReq()
		r.SupplierName = "   "
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", r)
		assertInvalid(t, err, "pemasok")

		r = validReq()
		r.Items[0].RejectReason = nil
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", r)
		assertInvalid(t, err, "alasan penolakan")

		r = validReq()
		r.Items[1].ProductID = prodA
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", r)
		assertInvalid(t, err, "dua kali")

		r = validReq()
		r.Items[1].AcceptedQty = dec(0)
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", r)
		assertInvalid(t, err, "lebih dari 0")

		r = validReq()
		r.Items[1].AcceptedQty = dec(-1)
		r.Items[1].RejectedQty = dec(3)
		r.Items[1].RejectReason = strp("x")
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", r)
		assertInvalid(t, err, "negatif")

		r = validReq()
		r.DestLocationID = transitLoc
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", r)
		assertInvalid(t, err, "internal")

		r = validReq()
		r.DestLocationID = otherLoc
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", r)
		assertInvalid(t, err, "gudang yang dipilih")

		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, outsiderID, "warehouse", validReq())
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)

		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "auditor", validReq())
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("post happy path creates ledger movements; double post and edit rejected", func(t *testing.T) {
		rc, items, err := usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", validReq())
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusDraft, rc.Status)
		assert.Regexp(t, `^GR-\d{8}-[0-9A-F]{8}$`, rc.ReceiptNumber)
		assert.Len(t, items, 2)
		assert.Equal(t, 2, rc.ItemCount)
		assert.True(t, rc.TotalAcceptedQty.Equal(dec(15)))
		assert.Equal(t, 0, len(repo.stockMovements), "draft must not touch the ledger")

		posted, err := usecase.PostStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusPosted, posted.Status)
		require.NotNil(t, posted.PostedBy)
		assert.Equal(t, staffID, *posted.PostedBy)

		vendor, _ := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeVendor)
		scrap, _ := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
		stgLoc, _ := repo.GetOrCreateStagingLocation(ctx, tenantID, whID)
		require.Len(t, repo.stockMovements, 3)
		type mv struct {
			src, dst, prod uuid.UUID
			qty            string
		}
		var got []mv
		for _, m := range repo.stockMovements {
			assert.Equal(t, domain.StockRefGoodsReceipt, m.ReferenceType)
			assert.Equal(t, rc.ID, m.ReferenceID)
			assert.Equal(t, domain.StockMovementStatusDone, m.Status)
			got = append(got, mv{m.SourceLocationID, m.DestLocationID, m.ProductID, m.Quantity.String()})
		}
		assert.ElementsMatch(t, []mv{
			{vendor.ID, stgLoc.ID, prodA, "10"},
			{vendor.ID, scrap.ID, prodA, "2"},
			{vendor.ID, stgLoc.ID, prodB, "5"},
		}, got)
		nums := map[string]bool{}
		for _, m := range repo.stockMovements {
			assert.False(t, nums[m.MovementNumber], "movement numbers must be unique")
			nums[m.MovementNumber] = true
		}
		assert.True(t, stock(stgLoc.ID, prodA).Equal(dec(10)))
		assert.True(t, stock(scrap.ID, prodA).Equal(dec(2)))
		assert.True(t, stock(stgLoc.ID, prodB).Equal(dec(5)))

		_, err = usecase.PostStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID)
		assert.ErrorIs(t, err, domain.ErrStockReceiptNotDraft)
		assert.Len(t, repo.stockMovements, 3, "double post must not add movements")

		_, _, err = usecase.UpdateStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID, validReq())
		assert.ErrorIs(t, err, domain.ErrStockReceiptNotDraft)

		// cancel posted -> reversal
		_, err = usecase.CancelStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID, "  ")
		assertInvalid(t, err, "Alasan pembatalan")

		cancelled, err := usecase.CancelStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID, "Salah supplier")
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusCancelled, cancelled.Status)
		require.NotNil(t, cancelled.CancelReason)
		assert.Equal(t, "Salah supplier", *cancelled.CancelReason)
		require.Len(t, repo.stockMovements, 6, "reversal appends, never deletes")
		for _, m := range repo.stockMovements[3:] {
			assert.Equal(t, vendor.ID, m.DestLocationID)
			assert.Equal(t, rc.ID, m.ReferenceID)
		}
		assert.True(t, stock(stgLoc.ID, prodA).IsZero())
		assert.True(t, stock(scrap.ID, prodA).IsZero())
		assert.True(t, stock(stgLoc.ID, prodB).IsZero())

		// terminal
		_, err = usecase.CancelStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID, "lagi")
		assert.ErrorIs(t, err, domain.ErrStockReceiptAlreadyCancelled)
		_, err = usecase.PostStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID)
		assert.ErrorIs(t, err, domain.ErrStockReceiptNotDraft)
	})

	t.Run("edit draft replaces items; cancel draft writes no movements", func(t *testing.T) {
		before := len(repo.stockMovements)
		rc, _, err := usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", validReq())
		require.NoError(t, err)

		upd := validReq()
		upd.SupplierName = "CV Baru"
		upd.Items = upd.Items[1:]
		rc2, items, err := usecase.UpdateStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID, upd)
		require.NoError(t, err)
		assert.Equal(t, "CV Baru", rc2.SupplierName)
		require.Len(t, items, 1)
		assert.Equal(t, prodB, items[0].ProductID)

		c, err := usecase.CancelStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID, "Batal kirim")
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusCancelled, c.Status)
		assert.Len(t, repo.stockMovements, before)
	})

	t.Run("cancel posted blocked when stock already used", func(t *testing.T) {
		rc, _, err := usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", validReq())
		require.NoError(t, err)
		_, err = usecase.PostStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID)
		require.NoError(t, err)

		stgLoc, _ := repo.GetOrCreateStagingLocation(ctx, tenantID, whID)
		// consume 3 of prodA from staging (e.g. putaway / consumption)
		customer, _ := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
		require.NoError(t, repo.DeductLocationStock(ctx, tenantID, stgLoc.ID, prodA, dec(3), &domain.StockMovement{
			ID: uuid.New(), TenantID: tenantID, MovementNumber: "DO-X", ProductID: prodA, SourceLocationID: stgLoc.ID,
			DestLocationID: customer.ID, Quantity: dec(3), Status: domain.StockMovementStatusDone, ReferenceType: domain.StockRefDeliveryOrder, ReferenceID: uuid.New(),
		}))
		before := len(repo.stockMovements)

		_, err = usecase.CancelStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID, "Retur")
		assert.ErrorIs(t, err, domain.ErrStockReceiptStockConsumed)
		assert.Len(t, repo.stockMovements, before, "no partial reversal")
		got, _, err := usecase.GetStockReceipt(ctx, tenantID, staffID, "warehouse", rc.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusPosted, got.Status)
	})

	t.Run("list filters by status and warehouse scope", func(t *testing.T) {
		posted := domain.StockReceiptStatusPosted
		list, err := usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", nil, &posted, nil)
		require.NoError(t, err)
		require.Len(t, list, 1)

		all, err := usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", &whID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, all, 3)

		_, err = usecase.ListStockReceipts(ctx, tenantID, outsiderID, "warehouse", &whID, nil, nil)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)

		outsiderList, err := usecase.ListStockReceipts(ctx, tenantID, outsiderID, "warehouse", nil, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, outsiderList)

		bad := domain.StockReceiptStatus("BOGUS")
		_, err = usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", nil, &bad, nil)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("not found and cross-tenant", func(t *testing.T) {
		_, err := usecase.PostStockReceipt(ctx, tenantID, staffID, "warehouse", uuid.New())
		assert.ErrorIs(t, err, domain.ErrStockReceiptNotFound)
		list, _ := usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", &whID, nil, nil)
		_, _, err = usecase.GetStockReceipt(ctx, uuid.New(), staffID, "admin", list[0].ID)
		assert.ErrorIs(t, err, domain.ErrStockReceiptNotFound)
	})

	t.Run("multi-source inbound: production output and branch transfer", func(t *testing.T) {
		// 1. Production receipt to primary warehouse
		prodReq := uc.StockReceiptRequest{
			ReceiptType:    domain.StockReceiptTypeProduction,
			WarehouseID:    whID,
			DestLocationID: binLoc,
			FromName:       "Dapur Pusat / Lini Produksi A",
			SourceRef:      strp("BATCH-202610-001"),
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodA, ExpectedQty: decPtr(50), AcceptedQty: dec(48), RejectedQty: dec(2), RejectReason: strp("Cacat cetakan")},
			},
		}
		prodRc, items, err := usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", prodReq)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptTypeProduction, prodRc.ReceiptType)
		assert.Equal(t, "Dapur Pusat / Lini Produksi A", prodRc.FromName)
		assert.Equal(t, "BATCH-202610-001", *prodRc.SourceRef)
		require.Len(t, items, 1)

		postedProd, err := usecase.PostStockReceipt(ctx, tenantID, staffID, "warehouse", prodRc.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusPosted, postedProd.Status)

		prodLoc, _ := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeProduction)
		scrapLoc, _ := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
		// Check that source location is @PRODUCTION
		stgWh, _ := repo.GetOrCreateStagingLocation(ctx, tenantID, whID)
		var foundProdAccepted, foundProdScrap bool
		for _, m := range repo.stockMovements {
			if m.ReferenceID == prodRc.ID {
				if m.DestLocationID == stgWh.ID {
					assert.Equal(t, prodLoc.ID, m.SourceLocationID)
					assert.True(t, m.Quantity.Equal(dec(48)))
					foundProdAccepted = true
				}
				if m.DestLocationID == scrapLoc.ID {
					assert.Equal(t, prodLoc.ID, m.SourceLocationID)
					assert.True(t, m.Quantity.Equal(dec(2)))
					foundProdScrap = true
				}
			}
		}
		assert.True(t, foundProdAccepted, "accepted movement must source from @PRODUCTION")
		assert.True(t, foundProdScrap, "rejected movement must source from @PRODUCTION")

		// 2. Transfer receipt validation: same warehouse rejected
		sameWhReq := uc.StockReceiptRequest{
			ReceiptType:     domain.StockReceiptTypeTransfer,
			WarehouseID:     whID,
			DestLocationID:  binLoc,
			FromWarehouseID: &whID, // Same warehouse
			FromName:        "Transfer Internal",
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodB, AcceptedQty: dec(10), RejectedQty: dec(0)},
			},
		}
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, staffID, "warehouse", sameWhReq)
		assertInvalid(t, err, "tidak boleh sama")

		// 3. Valid branch transfer receipt: from whID into otherWhID
		transferReq := uc.StockReceiptRequest{
			ReceiptType:     domain.StockReceiptTypeTransfer,
			WarehouseID:     otherWhID,
			DestLocationID:  otherLoc,
			FromWarehouseID: &whID,
			FromName:        "Gudang Pusat",
			SourceRef:       strp("TR-202610-099"),
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodB, AcceptedQty: dec(20), RejectedQty: dec(0)},
			},
		}
		transferRc, _, err := usecase.CreateStockReceipt(ctx, tenantID, staffID, "admin", transferReq)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptTypeTransfer, transferRc.ReceiptType)
		assert.Equal(t, &whID, transferRc.FromWarehouseID)

		postedTr, err := usecase.PostStockReceipt(ctx, tenantID, staffID, "admin", transferRc.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusPosted, postedTr.Status)

		transitLocObj, _ := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeTransit)
		otherStg, _ := repo.GetOrCreateStagingLocation(ctx, tenantID, otherWhID)
		var foundTransitAccepted bool
		for _, m := range repo.stockMovements {
			if m.ReferenceID == transferRc.ID {
				assert.Equal(t, transitLocObj.ID, m.SourceLocationID)
				assert.Equal(t, otherStg.ID, m.DestLocationID)
				assert.True(t, m.Quantity.Equal(dec(20)))
				foundTransitAccepted = true
			}
		}
		assert.True(t, foundTransitAccepted, "transfer receipt must source from @TRANSIT")

		// 4. Test filtering by receipt_type
		typeProd := domain.StockReceiptTypeProduction
		prodList, err := usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", &whID, nil, &typeProd)
		require.NoError(t, err)
		assert.NotEmpty(t, prodList)
		for _, item := range prodList {
			assert.Equal(t, domain.StockReceiptTypeProduction, item.ReceiptType)
		}

		typeTr := domain.StockReceiptTypeTransfer
		trList, err := usecase.ListStockReceipts(ctx, tenantID, staffID, "admin", &otherWhID, nil, &typeTr)
		require.NoError(t, err)
		assert.NotEmpty(t, trList)
		for _, item := range trList {
			assert.Equal(t, domain.StockReceiptTypeTransfer, item.ReceiptType)
		}
	})
}

func TestWMSFraudControls_F1_F5_F6(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	otherWhID := uuid.New()
	creatorID := uuid.New()
	otherUserID := uuid.New()
	adminID := uuid.New()

	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{ID: whID, TenantID: tenantID, Name: "Main WH", IsActive: true}))
	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{ID: otherWhID, TenantID: tenantID, Name: "Other WH", IsActive: true}))
	require.NoError(t, repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{UserID: creatorID, WarehouseID: whID, TenantID: tenantID}))
	require.NoError(t, repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{UserID: otherUserID, WarehouseID: whID, TenantID: tenantID}))
	require.NoError(t, repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{UserID: adminID, WarehouseID: whID, TenantID: tenantID}))

	binLoc := uuid.New()
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: binLoc, TenantID: tenantID, WarehouseID: &whID, Code: "BIN-A", Type: domain.LocationTypeInternal}))

	prodID := uuid.New()

	assertInvalid := func(t *testing.T, err error, contains string) {
		t.Helper()
		require.Error(t, err)
		var ve *domain.StockReceiptValidationError
		require.True(t, errors.As(err, &ve), "expected validation error, got %v", err)
		assert.True(t, errors.Is(err, domain.ErrInvalidInput))
		assert.Contains(t, ve.Msg, contains)
	}

	t.Run("F1: Vendor receipt reference validations", func(t *testing.T) {
		reqNoRef := uc.StockReceiptRequest{
			ReceiptType:    domain.StockReceiptTypeVendor,
			WarehouseID:    whID,
			DestLocationID: binLoc,
			SupplierName:   "PT Sumber Makmur",
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodID, AcceptedQty: dec(5), RejectedQty: dec(0)},
			},
		}
		_, _, err := usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqNoRef)
		assertInvalid(t, err, "Nomor PO atau referensi dokumen pemasok wajib diisi (minimal 4 karakter)")

		reqShortRef := reqNoRef
		reqShortRef.SourceRef = strp("PO")
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqShortRef)
		assertInvalid(t, err, "Nomor PO atau referensi dokumen pemasok wajib diisi (minimal 4 karakter)")

		reqShortSupRef := reqNoRef
		reqShortSupRef.SupplierRef = strp("ABC")
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqShortSupRef)
		assertInvalid(t, err, "Nomor PO atau referensi dokumen pemasok wajib diisi (minimal 4 karakter)")

		reqValidRef := reqNoRef
		reqValidRef.SourceRef = strp("PO-2026-001")
		rc, items, err := usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqValidRef)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusDraft, rc.Status)
		assert.NotEmpty(t, items)
	})

	t.Run("F1: Transfer receipt without FromWarehouseID returns validation error", func(t *testing.T) {
		reqTransferNoFrom := uc.StockReceiptRequest{
			ReceiptType:     domain.StockReceiptTypeTransfer,
			WarehouseID:     whID,
			DestLocationID:  binLoc,
			FromName:        "Transfer Gudang",
			FromWarehouseID: nil,
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodID, AcceptedQty: dec(5), RejectedQty: dec(0)},
			},
		}
		_, _, err := usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqTransferNoFrom)
		assertInvalid(t, err, "Gudang pengirim (asal transfer) wajib dipilih")

		nilUUID := uuid.Nil
		reqTransferNilUUID := reqTransferNoFrom
		reqTransferNilUUID.FromWarehouseID = &nilUUID
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqTransferNilUUID)
		assertInvalid(t, err, "Gudang pengirim (asal transfer) wajib dipilih")

		reqTransferValid := reqTransferNoFrom
		reqTransferValid.FromWarehouseID = &otherWhID
		rc, _, err := usecase.CreateStockReceipt(ctx, tenantID, creatorID, "admin", reqTransferValid)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusDraft, rc.Status)
	})

	t.Run("F5: AcceptedQty + RejectedQty > OrderedQty returns validation error; <= succeeds", func(t *testing.T) {
		ordered10 := dec(10)
		reqOver := uc.StockReceiptRequest{
			ReceiptType:    domain.StockReceiptTypeVendor,
			WarehouseID:    whID,
			DestLocationID: binLoc,
			SupplierName:   "PT Sumber Makmur",
			SourceRef:      strp("PO-2026-001"),
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodID, OrderedQty: &ordered10, AcceptedQty: dec(8), RejectedQty: dec(3), RejectReason: strp("Cacat")},
			},
		}
		_, _, err := usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqOver)
		assertInvalid(t, err, "total diterima + ditolak (11) melebihi jumlah dipesan (10)")

		reqOverExpected := uc.StockReceiptRequest{
			ReceiptType:    domain.StockReceiptTypeVendor,
			WarehouseID:    whID,
			DestLocationID: binLoc,
			SupplierName:   "PT Sumber Makmur",
			SourceRef:      strp("PO-2026-001"),
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodID, ExpectedQty: &ordered10, AcceptedQty: dec(11), RejectedQty: dec(0)},
			},
		}
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqOverExpected)
		assertInvalid(t, err, "total diterima + ditolak (11) melebihi jumlah dipesan (10)")

		negQty := dec(-1)
		reqNeg := reqOver
		reqNeg.Items[0].OrderedQty = &negQty
		_, _, err = usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqNeg)
		assertInvalid(t, err, "jumlah dipesan tidak boleh negatif")

		reqValid := uc.StockReceiptRequest{
			ReceiptType:    domain.StockReceiptTypeVendor,
			WarehouseID:    whID,
			DestLocationID: binLoc,
			SupplierName:   "PT Sumber Makmur",
			SourceRef:      strp("PO-2026-001"),
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodID, OrderedQty: &ordered10, AcceptedQty: dec(7), RejectedQty: dec(3), RejectReason: strp("Pecah")},
			},
		}
		rc, items, err := usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", reqValid)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusDraft, rc.Status)
		require.Len(t, items, 1)
		require.NotNil(t, items[0].OrderedQty)
		assert.True(t, items[0].OrderedQty.Equal(ordered10))
		require.NotNil(t, items[0].ExpectedQty)
		assert.True(t, items[0].ExpectedQty.Equal(ordered10))
	})

	t.Run("F6: Cancel draft receipt authorization checks", func(t *testing.T) {
		req := uc.StockReceiptRequest{
			ReceiptType:    domain.StockReceiptTypeVendor,
			WarehouseID:    whID,
			DestLocationID: binLoc,
			SupplierName:   "PT Sumber Makmur",
			SourceRef:      strp("PO-2026-001"),
			Items: []uc.StockReceiptItemRequest{
				{ProductID: prodID, AcceptedQty: dec(5), RejectedQty: dec(0)},
			},
		}
		rc, _, err := usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", req)
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusDraft, rc.Status)
		assert.Equal(t, creatorID, rc.CreatedBy)

		_, err = usecase.CancelStockReceipt(ctx, tenantID, otherUserID, "warehouse", rc.ID, "Batal oleh staf lain")
		assert.ErrorIs(t, err, domain.ErrForbidden)

		cancelledByAdmin, err := usecase.CancelStockReceipt(ctx, tenantID, adminID, "admin", rc.ID, "Batal oleh admin")
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusCancelled, cancelledByAdmin.Status)

		rc2, _, err := usecase.CreateStockReceipt(ctx, tenantID, creatorID, "warehouse", req)
		require.NoError(t, err)
		cancelledByCreator, err := usecase.CancelStockReceipt(ctx, tenantID, creatorID, "warehouse", rc2.ID, "Batal oleh pembuat")
		require.NoError(t, err)
		assert.Equal(t, domain.StockReceiptStatusCancelled, cancelledByCreator.Status)
	})
}

func decPtr(i int64) *decimal.Decimal { d := decimal.NewFromInt(i); return &d }
