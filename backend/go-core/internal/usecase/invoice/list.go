package invoice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// ListUsecase retrieves all invoices that belong to a specific tenant.
type ListUsecase struct {
	repo domain.InvoiceRepository
}

func NewListUsecase(repo domain.InvoiceRepository) *ListUsecase {
	return &ListUsecase{repo: repo}
}

func (u *ListUsecase) Execute(ctx context.Context, tenantID uuid.UUID) ([]domain.Invoice, error) {
	invoices, err := u.repo.List(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("ListUsecase.Execute: %w", err)
	}
	return invoices, nil
}

// ExecutePaged returns a paginated result set.
func (u *ListUsecase) ExecutePaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error) {
	result, err := u.repo.ListPaged(ctx, tenantID, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("ListUsecase.ExecutePaged: %w", err)
	}
	return result, nil
}
