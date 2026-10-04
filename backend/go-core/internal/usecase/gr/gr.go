package gr

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

var maxGRAmount, _ = decimal.NewFromString("9999999999999999.9999")

type Usecase struct {
	repo   domain.GRRepository
	poRepo domain.PORepository
}

func New(repo domain.GRRepository, poRepo domain.PORepository) *Usecase {
	return &Usecase{repo: repo, poRepo: poRepo}
}

func (u *Usecase) Create(ctx context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error) {
	if params.VendorID == "" {
		return nil, fmt.Errorf("vendor_id is required: %w", domain.ErrInvalidInput)
	}
	if len(params.VendorID) > 128 {
		return nil, fmt.Errorf("vendor_id exceeds maximum length: %w", domain.ErrInvalidInput)
	}
	if params.ReceivedQty <= 0 {
		return nil, fmt.Errorf("received_qty must be positive: %w", domain.ErrInvalidInput)
	}
	if params.ReceivedAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("received_amount must be positive: %w", domain.ErrInvalidInput)
	}
	if params.ReceivedAmount.GreaterThan(maxGRAmount) {
		return nil, fmt.Errorf("received_amount exceeds maximum allowed value: %w", domain.ErrInvalidInput)
	}
	if params.ReceivedAmount.Exponent() < -4 {
		return nil, fmt.Errorf("received_amount may not have more than 4 decimal places: %w", domain.ErrInvalidInput)
	}
	if params.Currency == "" {
		params.Currency = "IDR"
	}
	// Security: verify PO exists and belongs to same tenant.
	// Returns ErrNotFound regardless of whether PO is truly absent or belongs
	// to another tenant — prevents cross-tenant existence probing.
	if _, err := u.poRepo.GetByID(ctx, params.POID, params.TenantID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("GRUsecase.Create: verify PO ownership: %w", err)
	}
	return u.repo.Create(ctx, params)
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.GoodsReceipt, error) {
	return u.repo.List(ctx, tenantID)
}

func (u *Usecase) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error) {
	return u.repo.ListPaged(ctx, tenantID, page, perPage)
}

func (u *Usecase) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.GoodsReceipt, error) {
	return u.repo.GetByID(ctx, id, tenantID)
}
