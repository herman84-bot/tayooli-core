package vendor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

// CreateUsecase persists a new vendor and writes an audit log entry.
type CreateUsecase struct {
	repo      domain.VendorRepository
	auditRepo domain.AuditLogRepository
}

func NewCreateUsecase(repo domain.VendorRepository, auditRepo domain.AuditLogRepository) *CreateUsecase {
	return &CreateUsecase{repo: repo, auditRepo: auditRepo}
}

func (u *CreateUsecase) Execute(ctx context.Context, params domain.CreateVendorParams) (*domain.Vendor, error) {
	// Validate required fields and minimum length.
	nameTrimmed := strings.TrimSpace(params.Name)
	if len(nameTrimmed) < 2 {
		return nil, fmt.Errorf("name is required and must be at least 2 characters: %w", domain.ErrInvalidInput)
	}

	// Reject generic junk names or placeholders
	low := strings.ToLower(nameTrimmed)
	if low == "baru" || low == "vendor" || low == "perusahaan" || low == "pemasok" || low == "supplier" ||
		low == "customer" || low == "pelanggan" || low == "produk" || low == "barang" || low == "item" ||
		low == "(isi nama vendor)" || low == "(isi nama pelanggan)" {
		return nil, fmt.Errorf("invalid placeholder or generic name '%s': %w", params.Name, domain.ErrInvalidInput)
	}

	// Application-layer case-insensitive duplicate check per tenant
	exists, err := u.repo.ExistsByName(ctx, params.TenantID, nameTrimmed)
	if err != nil {
		return nil, fmt.Errorf("CreateUsecase.Execute: check duplicate name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("vendor with name '%s' already exists: %w", nameTrimmed, domain.ErrConflict)
	}

	params.Name = nameTrimmed
	vendor, err := u.repo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("CreateUsecase.Execute: create vendor: %w", err)
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"vendor_id": vendor.ID.String(),
		"name":      vendor.Name,
		"status":    string(vendor.Status),
	})
	auditEntry := domain.AuditLogEntry{
		TenantID:   vendor.TenantID,
		EntityType: "vendor",
		EntityID:   vendor.ID,
		Action:     "created",
		Details:    details,
	}
	if err := u.auditRepo.Create(ctx, auditEntry); err != nil {
		log.Error().Err(err).Str("vendor_id", vendor.ID.String()).Msg("CreateUsecase: failed to write audit log")
	}

	return vendor, nil
}
