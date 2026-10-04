package gr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	grUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/gr"
)

// ---------------------------------------------------------------------------
// Mock GRRepository
// ---------------------------------------------------------------------------

type mockGRRepo struct {
	createFn     func(ctx context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error)
	listFn       func(ctx context.Context, tenantID uuid.UUID) ([]domain.GoodsReceipt, error)
	listPagedFn  func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error)
	getByIDFn    func(ctx context.Context, id, tenantID uuid.UUID) (*domain.GoodsReceipt, error)
}

func (m *mockGRRepo) Create(ctx context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error) {
	return m.createFn(ctx, params)
}
func (m *mockGRRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.GoodsReceipt, error) {
	return m.listFn(ctx, tenantID)
}
func (m *mockGRRepo) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error) {
	return m.listPagedFn(ctx, tenantID, page, perPage)
}
func (m *mockGRRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.GoodsReceipt, error) {
	return m.getByIDFn(ctx, id, tenantID)
}

// ---------------------------------------------------------------------------
// Mock PORepository
// ---------------------------------------------------------------------------

type mockPORepo struct {
	getByIDFn func(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error)
	err       error
}

func (m *mockPORepo) Create(_ context.Context, _ domain.CreatePOParams) (*domain.PurchaseOrder, error) {
	panic("not expected in GR tests")
}
func (m *mockPORepo) List(_ context.Context, _ uuid.UUID) ([]domain.PurchaseOrder, error) {
	panic("not expected in GR tests")
}
func (m *mockPORepo) ListPaged(_ context.Context, _ uuid.UUID, _, _ int) (*domain.PurchaseOrderListPage, error) {
	panic("not expected in GR tests")
}
func (m *mockPORepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID)
	}
	return nil, m.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic("mustDecimal: " + err.Error())
	}
	return d
}

func fakeGR(tenantID, poID uuid.UUID) *domain.GoodsReceipt {
	return &domain.GoodsReceipt{
		ID:             uuid.New(),
		TenantID:       tenantID,
		POID:           poID,
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: decimal.New(250_000, 0),
		Currency:       "IDR",
		Status:         domain.GRStatusPending,
		ReceivedAt:     time.Now(),
		CreatedAt:      time.Now(),
	}
}

func fakePO(tenantID uuid.UUID) *domain.PurchaseOrder {
	return &domain.PurchaseOrder{
		ID:        uuid.New(),
		TenantID:  tenantID,
		VendorID:  "vendor-1",
		PONumber:  "PO-001",
		Amount:    decimal.New(500_000, 0),
		Qty:       10,
		Currency:  "IDR",
		Status:    domain.POStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestGRUsecase_Create_Success(t *testing.T) {
	tenantID := uuid.New()
	po := fakePO(tenantID)
	expected := fakeGR(tenantID, po.ID)

	poRepo := &mockPORepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.PurchaseOrder, error) {
			if id != po.ID || tid != tenantID {
				return nil, domain.ErrNotFound
			}
			return po, nil
		},
	}
	grRepo := &mockGRRepo{
		createFn: func(_ context.Context, _ domain.CreateGRParams) (*domain.GoodsReceipt, error) {
			return expected, nil
		},
	}
	uc := grUC.New(grRepo, poRepo)

	got, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       tenantID,
		POID:           po.ID,
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: decimal.New(250_000, 0),
		Currency:       "IDR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != expected.ID {
		t.Errorf("expected GR ID %s, got %s", expected.ID, got.ID)
	}
}

func TestGRUsecase_Create_EmptyVendorID_ReturnsErrInvalidInput(t *testing.T) {
	poRepo := &mockPORepo{}
	grRepo := &mockGRRepo{}
	uc := grUC.New(grRepo, poRepo)

	_, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       uuid.New(),
		POID:           uuid.New(),
		VendorID:       "",
		ReceivedQty:    5,
		ReceivedAmount: decimal.New(100, 0),
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGRUsecase_Create_ZeroQty_ReturnsErrInvalidInput(t *testing.T) {
	poRepo := &mockPORepo{}
	grRepo := &mockGRRepo{}
	uc := grUC.New(grRepo, poRepo)

	_, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       uuid.New(),
		POID:           uuid.New(),
		VendorID:       "vendor-1",
		ReceivedQty:    0,
		ReceivedAmount: decimal.New(100, 0),
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGRUsecase_Create_ZeroAmount_ReturnsErrInvalidInput(t *testing.T) {
	uc := grUC.New(&mockGRRepo{}, &mockPORepo{})

	_, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       uuid.New(),
		POID:           uuid.New(),
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: decimal.Zero,
		Currency:       "IDR",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for zero amount, got %v", err)
	}
}

func TestGRUsecase_Create_NegativeAmount_ReturnsErrInvalidInput(t *testing.T) {
	uc := grUC.New(&mockGRRepo{}, &mockPORepo{})

	_, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       uuid.New(),
		POID:           uuid.New(),
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: decimal.New(-1, 0),
		Currency:       "IDR",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for negative amount, got %v", err)
	}
}

func TestGRUsecase_Create_POBelongsToOtherTenant_ReturnsErrNotFound(t *testing.T) {
	myTenantID := uuid.New()
	otherTenantID := uuid.New()
	poID := uuid.New()

	// PO exists but belongs to a different tenant — poRepo returns ErrNotFound
	// for the cross-tenant lookup, preventing existence probing.
	poRepo := &mockPORepo{
		getByIDFn: func(_ context.Context, id uuid.UUID, tid uuid.UUID) (*domain.PurchaseOrder, error) {
			// Simulate RLS / application-layer tenant check:
			// the PO exists for otherTenantID but not for myTenantID.
			if tid == otherTenantID && id == poID {
				return &domain.PurchaseOrder{ID: poID, TenantID: otherTenantID}, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	grRepo := &mockGRRepo{}
	uc := grUC.New(grRepo, poRepo)

	_, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       myTenantID,
		POID:           poID,
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: decimal.New(100, 0),
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-tenant PO access, got %v", err)
	}
}

func TestGRUsecase_Create_DefaultsCurrencyToIDR(t *testing.T) {
	tenantID := uuid.New()
	po := fakePO(tenantID)
	var capturedParams domain.CreateGRParams

	poRepo := &mockPORepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PurchaseOrder, error) {
			return po, nil
		},
	}
	grRepo := &mockGRRepo{
		createFn: func(_ context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error) {
			capturedParams = params
			return fakeGR(tenantID, po.ID), nil
		},
	}
	uc := grUC.New(grRepo, poRepo)

	_, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       tenantID,
		POID:           po.ID,
		VendorID:       "vendor-1",
		ReceivedQty:    1,
		ReceivedAmount: decimal.New(100, 0),
		Currency:       "", // should default to IDR
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedParams.Currency != "IDR" {
		t.Errorf("expected currency IDR, got %q", capturedParams.Currency)
	}
}

func TestGRUsecase_List_Success(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	grs := []domain.GoodsReceipt{*fakeGR(tenantID, poID), *fakeGR(tenantID, poID)}

	poRepo := &mockPORepo{}
	grRepo := &mockGRRepo{
		listFn: func(_ context.Context, tid uuid.UUID) ([]domain.GoodsReceipt, error) {
			if tid != tenantID {
				t.Errorf("unexpected tenantID: %s", tid)
			}
			return grs, nil
		},
	}
	uc := grUC.New(grRepo, poRepo)

	got, err := uc.List(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 GRs, got %d", len(got))
	}
}

func TestGRUsecase_GetByID_Success(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	expected := fakeGR(tenantID, poID)

	poRepo := &mockPORepo{}
	grRepo := &mockGRRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.GoodsReceipt, error) {
			if id != expected.ID || tid != tenantID {
				return nil, domain.ErrNotFound
			}
			return expected, nil
		},
	}
	uc := grUC.New(grRepo, poRepo)

	got, err := uc.GetByID(context.Background(), expected.ID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != expected.ID {
		t.Errorf("expected ID %s, got %s", expected.ID, got.ID)
	}
}

func TestGRUsecase_GetByID_NotFound(t *testing.T) {
	poRepo := &mockPORepo{}
	grRepo := &mockGRRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.GoodsReceipt, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := grUC.New(grRepo, poRepo)

	_, err := uc.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGRUsecase_Create_AmountTooLarge_ReturnsErrInvalidInput(t *testing.T) {
	uc := grUC.New(&mockGRRepo{}, &mockPORepo{})

	_, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       uuid.New(),
		POID:           uuid.New(),
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: mustDecimal("99999999999999999999"),
		Currency:       "IDR",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for too-large received_amount, got %v", err)
	}
}

func TestGRUsecase_Create_AmountTooManyDecimals_ReturnsErrInvalidInput(t *testing.T) {
	uc := grUC.New(&mockGRRepo{}, &mockPORepo{})

	_, err := uc.Create(context.Background(), domain.CreateGRParams{
		TenantID:       uuid.New(),
		POID:           uuid.New(),
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: mustDecimal("1.23456"),
		Currency:       "IDR",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for too many decimal places in received_amount, got %v", err)
	}
}

func TestGRUsecase_Create_PORepoInfraError_PropagatesError(t *testing.T) {
	dbErr := errors.New("connection refused")
	uc := grUC.New(
		&mockGRRepo{},
		&mockPORepo{err: dbErr},
	)
	params := domain.CreateGRParams{
		TenantID:       uuid.New(),
		POID:           uuid.New(),
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: mustDecimal("500.00"),
		Currency:       "IDR",
	}
	_, err := uc.Create(context.Background(), params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, domain.ErrNotFound) {
		t.Errorf("infra error must NOT be mapped to ErrNotFound, got %v", err)
	}
}
