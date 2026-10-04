package vendor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

// DeleteUsecase soft-deletes a vendor (status -> inactive) and writes an audit log entry.
type DeleteUsecase struct {
	repo      domain.VendorRepository
	auditRepo domain.AuditLogRepository
}

func NewDeleteUsecase(repo domain.VendorRepository, auditRepo domain.AuditLogRepository) *DeleteUsecase {
	return &DeleteUsecase{repo: repo, auditRepo: auditRepo}
}

func (u *DeleteUsecase) Execute(ctx context.Context, id, tenantID uuid.UUID, deletedBy *uuid.UUID) error {
	err := u.repo.SoftDelete(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("DeleteUsecase.Execute: soft delete vendor: %w", err)
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"vendor_id": id.String(),
		"new_status": "inactive",
	})
	auditEntry := domain.AuditLogEntry{
		TenantID:   tenantID,
		EntityType: "vendor",
		EntityID:   id,
		Action:     "deleted",
		Details:    details,
		UserID:     deletedBy,
	}
	if err := u.auditRepo.Create(ctx, auditEntry); err != nil {
		log.Error().Err(err).Str("vendor_id", id.String()).Msg("DeleteUsecase: failed to write audit log")
	}

	return nil
}
