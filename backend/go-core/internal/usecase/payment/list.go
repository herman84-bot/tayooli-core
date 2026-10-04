package payment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// ListUsecase retrieves payment orders for a specific tenant.
type ListUsecase struct {
	repo domain.PaymentOrderRepository
}

func NewListUsecase(repo domain.PaymentOrderRepository) *ListUsecase {
	return &ListUsecase{repo: repo}
}

func (u *ListUsecase) ExecutePaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error) {
	result, err := u.repo.ListPaged(ctx, tenantID, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("ListUsecase.ExecutePaged: %w", err)
	}
	return result, nil
}
