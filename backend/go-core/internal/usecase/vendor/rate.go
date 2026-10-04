package vendor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

// RateUsecase records a user rating for a vendor and writes an audit log entry.
// The database trigger auto-updates avg_rating and rating_count on the vendor.
type RateUsecase struct {
	repo      domain.VendorRepository
	auditRepo domain.AuditLogRepository
}

func NewRateUsecase(repo domain.VendorRepository, auditRepo domain.AuditLogRepository) *RateUsecase {
	return &RateUsecase{repo: repo, auditRepo: auditRepo}
}

func (u *RateUsecase) Execute(ctx context.Context, params domain.AddVendorRatingParams) (*domain.VendorRating, error) {
	// Validate rating bounds.
	if params.Rating < 1 || params.Rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5: %w", domain.ErrInvalidInput)
	}

	// Verify vendor belongs to this tenant before rating.
	_, err := u.repo.GetByID(ctx, params.VendorID, params.TenantID)
	if err != nil {
		return nil, fmt.Errorf("RateUsecase.Execute: vendor not found or does not belong to tenant: %w", err)
	}

	rating, err := u.repo.AddRating(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("RateUsecase.Execute: add rating: %w", err)
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"vendor_id": rating.VendorID.String(),
		"rating":    fmt.Sprintf("%d", rating.Rating),
	})
	auditEntry := domain.AuditLogEntry{
		TenantID:   rating.TenantID,
		EntityType: "vendor",
		EntityID:   rating.VendorID,
		Action:     "rated",
		Details:    details,
		UserID:     params.RatedBy,
	}
	if err := u.auditRepo.Create(ctx, auditEntry); err != nil {
		log.Error().Err(err).Str("vendor_id", rating.VendorID.String()).Msg("RateUsecase: failed to write audit log")
	}

	return rating, nil
}
