package po_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	poUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/po"
)

// ---------------------------------------------------------------------------
// Mock PORepository
// ---------------------------------------------------------------------------

type mockPORepo struct {
	createFn     func(ctx context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error)
	listFn       func(ctx context.Context, tenantID uuid.UUID) ([]domain.PurchaseOrder, error)
	listPagedFn  func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error)
	getByIDFn    func(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error)
}

func (m *mockPORepo) Create(ctx context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error) {
	return m.createFn(ctx, params)
}
func (m *mockPORepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.PurchaseOrder, error) {
	return m.listFn(ctx, tenantID)
}
func (m *mockPORepo) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error) {
	return m.listPagedFn(ctx, tenantID, page, perPage)
}
func (m *mockPORepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error) {
	return m.getByIDFn(ctx, id, tenantID)
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

func TestPOUsecase_Create_Success(t *testing.T) {
	tenantID := uuid.New()
	expected := fakePO(tenantID)

	repo := &mockPORepo{
		createFn: func(_ context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error) {
			return expected, nil
		},
	}
	uc := poUC.New(repo)

	got, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: tenantID,
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   decimal.New(500_000, 0),
		Qty:      10,
		Currency: "IDR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != expected.ID {
		t.Errorf("expected PO ID %s, got %s", expected.ID, got.ID)
	}
}

func TestPOUsecase_Create_EmptyVendorID_ReturnsErrInvalidInput(t *testing.T) {
	repo := &mockPORepo{}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "",
		PONumber: "PO-001",
		Amount:   decimal.New(100, 0),
		Qty:      1,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPOUsecase_Create_EmptyPONumber_ReturnsErrInvalidInput(t *testing.T) {
	repo := &mockPORepo{}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "vendor-1",
		PONumber: "",
		Amount:   decimal.New(100, 0),
		Qty:      1,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPOUsecase_Create_ZeroQty_ReturnsErrInvalidInput(t *testing.T) {
	repo := &mockPORepo{}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   decimal.New(100, 0),
		Qty:      0,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPOUsecase_Create_DefaultsCurrencyToIDR(t *testing.T) {
	tenantID := uuid.New()
	var capturedParams domain.CreatePOParams

	repo := &mockPORepo{
		createFn: func(_ context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error) {
			capturedParams = params
			po := fakePO(tenantID)
			po.Currency = params.Currency
			return po, nil
		},
	}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: tenantID,
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   decimal.New(100, 0),
		Qty:      1,
		Currency: "", // empty — should default to IDR
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedParams.Currency != "IDR" {
		t.Errorf("expected currency IDR, got %q", capturedParams.Currency)
	}
}

func TestPOUsecase_List_Success(t *testing.T) {
	tenantID := uuid.New()
	pos := []domain.PurchaseOrder{*fakePO(tenantID), *fakePO(tenantID)}

	repo := &mockPORepo{
		listFn: func(_ context.Context, tid uuid.UUID) ([]domain.PurchaseOrder, error) {
			if tid != tenantID {
				t.Errorf("unexpected tenantID: %s", tid)
			}
			return pos, nil
		},
	}
	uc := poUC.New(repo)

	got, err := uc.List(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 POs, got %d", len(got))
	}
}

func TestPOUsecase_GetByID_Success(t *testing.T) {
	tenantID := uuid.New()
	expected := fakePO(tenantID)

	repo := &mockPORepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.PurchaseOrder, error) {
			if id != expected.ID || tid != tenantID {
				return nil, domain.ErrNotFound
			}
			return expected, nil
		},
	}
	uc := poUC.New(repo)

	got, err := uc.GetByID(context.Background(), expected.ID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != expected.ID {
		t.Errorf("expected ID %s, got %s", expected.ID, got.ID)
	}
}

func TestPOUsecase_GetByID_NotFound(t *testing.T) {
	repo := &mockPORepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PurchaseOrder, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := poUC.New(repo)

	_, err := uc.GetByID(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPOUsecase_Create_ZeroAmount_ReturnsErrInvalidInput(t *testing.T) {
	repo := &mockPORepo{}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   decimal.New(0, 0),
		Qty:      1,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for zero amount, got %v", err)
	}
}

func TestPOUsecase_Create_NegativeAmount_ReturnsErrInvalidInput(t *testing.T) {
	repo := &mockPORepo{}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   decimal.New(-100, 0),
		Qty:      1,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for negative amount, got %v", err)
	}
}

func TestPOUsecase_Create_NegativeQty_ReturnsErrInvalidInput(t *testing.T) {
	repo := &mockPORepo{}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   decimal.New(100, 0),
		Qty:      -5,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for negative qty, got %v", err)
	}
}

func TestPOUsecase_Create_RepoError_Propagates(t *testing.T) {
	repoErr := errors.New("connection timeout")
	repo := &mockPORepo{
		createFn: func(_ context.Context, _ domain.CreatePOParams) (*domain.PurchaseOrder, error) {
			return nil, repoErr
		},
	}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   decimal.New(100, 0),
		Qty:      1,
	})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error to propagate, got %v", err)
	}
}

func TestPOUsecase_Create_AmountTooLarge_ReturnsErrInvalidInput(t *testing.T) {
	repo := &mockPORepo{}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   mustDecimal("99999999999999999999"),
		Qty:      1,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for too-large amount, got %v", err)
	}
}

func TestPOUsecase_Create_AmountTooManyDecimals_ReturnsErrInvalidInput(t *testing.T) {
	repo := &mockPORepo{}
	uc := poUC.New(repo)

	_, err := uc.Create(context.Background(), domain.CreatePOParams{
		TenantID: uuid.New(),
		VendorID: "vendor-1",
		PONumber: "PO-001",
		Amount:   mustDecimal("1.23456"),
		Qty:      1,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for too many decimal places, got %v", err)
	}
}

func TestPOUsecase_List_RepoError_Propagates(t *testing.T) {
	repoErr := errors.New("db unavailable")
	repo := &mockPORepo{
		listFn: func(_ context.Context, _ uuid.UUID) ([]domain.PurchaseOrder, error) {
			return nil, repoErr
		},
	}
	uc := poUC.New(repo)

	_, err := uc.List(context.Background(), uuid.New())
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error to propagate, got %v", err)
	}
}
