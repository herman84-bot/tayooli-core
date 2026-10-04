package vendor

import (
	"context"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// ListUsecase retrieves vendors for a specific tenant with optional search.
type ListUsecase struct {
	repo domain.VendorRepository
}

func NewListUsecase(repo domain.VendorRepository) *ListUsecase {
	return &ListUsecase{repo: repo}
}

func (u *ListUsecase) ExecutePaged(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error) {
	return u.repo.ListPaged(ctx, tenantID, search, page, perPage)
}
