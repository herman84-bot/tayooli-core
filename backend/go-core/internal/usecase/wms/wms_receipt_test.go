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

func (m *mockWMSRepo) ListStockReceipts(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.StockReceiptStatus) ([]domain.StockReceipt, error) {
	m.initReceipts()
	var list []domain.StockReceipt
	for id, rc := range m.stockReceipts {
		if rc.TenantID != tenantID || (warehouseID != nil && rc.WarehouseID != *warehouseID) || (status != nil && rc.Status != *status) {
			continue
		}
		cp, _, _ := m.GetStockReceiptByID(ctx, tenantID, id)
		list = append(list, *cp)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	return list, nil
}

func (m *mockWMSRepo) PostStockReceipt(ctx context.Context, tenantID, id, userID, vendorLocID, scrapLocID uuid.UUID) (*domain.StockReceipt, error) {
	m.initReceipts()
	rc, ok := m.stockReceipts[id]
	if !ok || rc.TenantID != tenantID {
		return nil, domain.ErrStockReceiptNotFound
	}
	if rc.Status != domain.StockReceiptStatusDraft {
		return nil, domain.ErrStockReceiptNotDraft
	}
	now := time.Now().UTC()
	for i, it := range m.receiptItems[id] {
		if it.AcceptedQty.IsPositive() {
			_ = m.CreateStockMovement(ctx, &domain.StockMovement{ID: uuid.New(), TenantID: tenantID, MovementNumber: fmt.Sprintf("GR-IN-%s-%d", rc.ReceiptNumber, i+1),
				ProductID: it.ProductID, SourceLocationID: vendorLocID, DestLocationID: rc.DestLocationID, Quantity: it.AcceptedQty,
				Status: domain.StockMovementStatusDone, ReferenceType: domain.StockRefGoodsReceipt, ReferenceID: id, ExecutedBy: &userID, CreatedAt: now})
		}
		if it.RejectedQty.IsPositive() {
			_ = m.CreateStockMovement(ctx, &domain.StockMovement{ID: uuid.New(), TenantID: tenantID, MovementNumber: fmt.Sprintf("GR-REJ-%s-%d", rc.ReceiptNumber, i+1),
				ProductID: it.ProductID, SourceLocationID: vendorLocID, DestLocationID: scrapLocID, Quantity: it.RejectedQty,
				Status: domain.StockMovementStatusDone, ReferenceType: domain.StockRefGoodsReceipt, ReferenceID: id, ExecutedBy: &userID, CreatedAt: now})
		}
	}
	rc.Status, rc.PostedBy, rc.PostedAt = domain.StockReceiptStatusPosted, &userID, &now
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
		items := m.receiptItems[id]
		for _, it := range items { // check everything before mutating (atomic)
			if it.AcceptedQty.IsPositive() && m.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, rc.DestLocationID, it.ProductID)].LessThan(it.AcceptedQty) {
				return nil, domain.ErrStockReceiptStockConsumed
			}
			if it.RejectedQty.IsPositive() && m.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, scrapLocID, it.ProductID)].LessThan(it.RejectedQty) {
				return nil, domain.ErrStockReceiptStockConsumed
			}
		}
		for i, it := range items {
			if it.AcceptedQty.IsPositive() {
				_ = m.CreateStockMovement(ctx, &domain.StockMovement{ID: uuid.New(), TenantID: tenantID, MovementNumber: fmt.Sprintf("GR-REV-IN-%s-%d", rc.ReceiptNumber, i+1),
					ProductID: it.ProductID, SourceLocationID: rc.DestLocationID, DestLocationID: vendorLocID, Quantity: it.AcceptedQty,
					Status: domain.StockMovementStatusDone, ReferenceType: domain.StockRefGoodsReceipt, ReferenceID: id, ExecutedBy: &userID, CreatedAt: now})
			}
			if it.RejectedQty.IsPositive() {
				_ = m.CreateStockMovement(ctx, &domain.StockMovement{ID: uuid.New(), TenantID: tenantID, MovementNumber: fmt.Sprintf("GR-REV-REJ-%s-%d", rc.ReceiptNumber, i+1),
					ProductID: it.ProductID, SourceLocationID: scrapLocID, DestLocationID: vendorLocID, Quantity: it.RejectedQty,
					Status: domain.StockMovementStatusDone, ReferenceType: domain.StockRefGoodsReceipt, ReferenceID: id, ExecutedBy: &userID, CreatedAt: now})
			}
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
			{vendor.ID, binLoc, prodA, "10"},
			{vendor.ID, scrap.ID, prodA, "2"},
			{vendor.ID, binLoc, prodB, "5"},
		}, got)
		nums := map[string]bool{}
		for _, m := range repo.stockMovements {
			assert.False(t, nums[m.MovementNumber], "movement numbers must be unique")
			nums[m.MovementNumber] = true
		}
		assert.True(t, stock(binLoc, prodA).Equal(dec(10)))
		assert.True(t, stock(scrap.ID, prodA).Equal(dec(2)))
		assert.True(t, stock(binLoc, prodB).Equal(dec(5)))

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
		assert.True(t, stock(binLoc, prodA).IsZero())
		assert.True(t, stock(scrap.ID, prodA).IsZero())
		assert.True(t, stock(binLoc, prodB).IsZero())

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

		// consume 3 of prodA from the bin (e.g. a sale / delivery)
		customer, _ := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
		require.NoError(t, repo.DeductLocationStock(ctx, tenantID, binLoc, prodA, dec(3), &domain.StockMovement{
			ID: uuid.New(), TenantID: tenantID, MovementNumber: "DO-X", ProductID: prodA, SourceLocationID: binLoc,
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
		list, err := usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", nil, &posted)
		require.NoError(t, err)
		require.Len(t, list, 1)

		all, err := usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", &whID, nil)
		require.NoError(t, err)
		assert.Len(t, all, 3)

		_, err = usecase.ListStockReceipts(ctx, tenantID, outsiderID, "warehouse", &whID, nil)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)

		outsiderList, err := usecase.ListStockReceipts(ctx, tenantID, outsiderID, "warehouse", nil, nil)
		require.NoError(t, err)
		assert.Empty(t, outsiderList)

		bad := domain.StockReceiptStatus("BOGUS")
		_, err = usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", nil, &bad)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("not found and cross-tenant", func(t *testing.T) {
		_, err := usecase.PostStockReceipt(ctx, tenantID, staffID, "warehouse", uuid.New())
		assert.ErrorIs(t, err, domain.ErrStockReceiptNotFound)
		list, _ := usecase.ListStockReceipts(ctx, tenantID, staffID, "warehouse", &whID, nil)
		_, _, err = usecase.GetStockReceipt(ctx, uuid.New(), staffID, "admin", list[0].ID)
		assert.ErrorIs(t, err, domain.ErrStockReceiptNotFound)
	})
}

func decPtr(i int64) *decimal.Decimal { d := decimal.NewFromInt(i); return &d }
