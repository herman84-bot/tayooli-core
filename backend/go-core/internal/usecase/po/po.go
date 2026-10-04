package po

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

var maxPOAmount, _ = decimal.NewFromString("9999999999999999.9999")

type Usecase struct {
	repo domain.PORepository
}

func New(repo domain.PORepository) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Create(ctx context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error) {
	if params.VendorID == "" || params.PONumber == "" {
		return nil, fmt.Errorf("vendor_id and po_number are required: %w", domain.ErrInvalidInput)
	}
	if len(params.VendorID) > 128 || len(params.PONumber) > 64 {
		return nil, fmt.Errorf("vendor_id or po_number exceeds maximum length: %w", domain.ErrInvalidInput)
	}
	if params.Qty <= 0 {
		return nil, fmt.Errorf("qty must be positive: %w", domain.ErrInvalidInput)
	}
	if params.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive: %w", domain.ErrInvalidInput)
	}
	if params.Amount.GreaterThan(maxPOAmount) {
		return nil, fmt.Errorf("amount exceeds maximum allowed value: %w", domain.ErrInvalidInput)
	}
	if params.Amount.Exponent() < -4 {
		return nil, fmt.Errorf("amount may not have more than 4 decimal places: %w", domain.ErrInvalidInput)
	}
	if params.Currency == "" {
		params.Currency = "IDR"
	}
	return u.repo.Create(ctx, params)
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.PurchaseOrder, error) {
	return u.repo.List(ctx, tenantID)
}

func (u *Usecase) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error) {
	return u.repo.ListPaged(ctx, tenantID, page, perPage)
}

func (u *Usecase) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error) {
	return u.repo.GetByID(ctx, id, tenantID)
}
