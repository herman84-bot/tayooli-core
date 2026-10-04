package vendor

import (
	"context"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// Facade is a convenience wrapper that satisfies the handler's
// vendorUsecase interface by delegating to individual use-case structs.
type Facade struct {
	repo     domain.VendorRepository
	create   *CreateUsecase
	list     *ListUsecase
	update   *UpdateUsecase
	delete   *DeleteUsecase
	rate     *RateUsecase
}

func New(repo domain.VendorRepository, auditRepo domain.AuditLogRepository) *Facade {
	return &Facade{
		repo:   repo,
		create: NewCreateUsecase(repo, auditRepo),
		list:   NewListUsecase(repo),
		update: NewUpdateUsecase(repo, auditRepo),
		delete: NewDeleteUsecase(repo, auditRepo),
		rate:   NewRateUsecase(repo, auditRepo),
	}
}

func (f *Facade) CreateVendor(ctx context.Context, params domain.CreateVendorParams) (*domain.Vendor, error) {
	return f.create.Execute(ctx, params)
}

func (f *Facade) ListVendors(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error) {
	return f.list.ExecutePaged(ctx, tenantID, search, page, perPage)
}

func (f *Facade) GetVendor(ctx context.Context, id, tenantID uuid.UUID) (*domain.Vendor, error) {
	return f.repo.GetByID(ctx, id, tenantID)
}

func (f *Facade) UpdateVendor(ctx context.Context, params domain.UpdateVendorParams) (*domain.Vendor, error) {
	return f.update.Execute(ctx, params)
}

func (f *Facade) DeleteVendor(ctx context.Context, id, tenantID uuid.UUID, deletedBy *uuid.UUID) error {
	return f.delete.Execute(ctx, id, tenantID, deletedBy)
}

func (f *Facade) RateVendor(ctx context.Context, params domain.AddVendorRatingParams) (*domain.VendorRating, error) {
	return f.rate.Execute(ctx, params)
}
