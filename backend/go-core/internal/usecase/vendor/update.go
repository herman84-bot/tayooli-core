package vendor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

// UpdateUsecase modifies an existing vendor and writes an audit log entry.
type UpdateUsecase struct {
	repo      domain.VendorRepository
	auditRepo domain.AuditLogRepository
}

func NewUpdateUsecase(repo domain.VendorRepository, auditRepo domain.AuditLogRepository) *UpdateUsecase {
	return &UpdateUsecase{repo: repo, auditRepo: auditRepo}
}

func (u *UpdateUsecase) Execute(ctx context.Context, params domain.UpdateVendorParams) (*domain.Vendor, error) {
	// Validate required fields and minimum length.
	nameTrimmed := strings.TrimSpace(params.Name)
	if len(nameTrimmed) < 2 {
		return nil, fmt.Errorf("name is required and must be at least 2 characters: %w", domain.ErrInvalidInput)
	}

	low := strings.ToLower(nameTrimmed)
	if low == "baru" || low == "vendor" || low == "perusahaan" || low == "pemasok" || low == "supplier" ||
		low == "customer" || low == "pelanggan" || low == "produk" || low == "barang" || low == "item" ||
		low == "(isi nama vendor)" || low == "(isi nama pelanggan)" {
		return nil, fmt.Errorf("invalid placeholder or generic name '%s': %w", params.Name, domain.ErrInvalidInput)
	}

	params.Name = nameTrimmed
	vendor, err := u.repo.Update(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("UpdateUsecase.Execute: update vendor: %w", err)
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"vendor_id": vendor.ID.String(),
		"name":      vendor.Name,
	})
	auditEntry := domain.AuditLogEntry{
		TenantID:   vendor.TenantID,
		EntityType: "vendor",
		EntityID:   vendor.ID,
		Action:     "updated",
		Details:    details,
	}
	if err := u.auditRepo.Create(ctx, auditEntry); err != nil {
		log.Error().Err(err).Str("vendor_id", vendor.ID.String()).Msg("UpdateUsecase: failed to write audit log")
	}

	return vendor, nil
}
