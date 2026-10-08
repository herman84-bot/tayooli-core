package wms

// Sprint 4 Outbound Usecase: Shipping Manifests, Loading Scan Verification,
// Driver Signature Handover & 8 SOP Outbound KPIs (ADR-014 Invariant 1, Master PRD §3.2, §4.1).

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// CreateShippingManifest validates write access on warehouse, validates request via req.Validate(),
// and creates the shipping manifest linking the specified DOs.
func (u *Usecase) CreateShippingManifest(ctx context.Context, tenantID, userID uuid.UUID, role string, req domain.CreateShippingManifestRequest) (*domain.ShippingManifest, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.CreateShippingManifest(ctx, tenantID, userID, req)
}

// GetShippingManifest retrieves full details of a manifest including its attached delivery orders.
func (u *Usecase) GetShippingManifest(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID) (*domain.ShippingManifestDetail, error) {
	return u.repo.GetShippingManifestByID(ctx, tenantID, id)
}

// ListShippingManifests retrieves manifests filtered by tenant, warehouse, status, or expedition.
func (u *Usecase) ListShippingManifests(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID, status *domain.ShippingManifestStatus, expeditionName *string) ([]domain.ShippingManifest, error) {
	return u.repo.ListShippingManifests(ctx, tenantID, warehouseID, status, expeditionName)
}

// ScanDOLoading validates req.Barcode not empty and marks the DO as scanned onto the truck.
func (u *Usecase) ScanDOLoading(ctx context.Context, tenantID, userID uuid.UUID, role string, manifestID uuid.UUID, req domain.LoadingScanRequest) (*domain.ShippingManifestDetail, error) {
	barcode := strings.TrimSpace(req.Barcode)
	if barcode == "" {
		return nil, domain.ErrInvalidInput
	}
	return u.repo.ScanDOLoading(ctx, tenantID, manifestID, barcode, userID)
}

// DispatchShippingManifest validates driver signature SVG and completes atomic dispatch handover.
func (u *Usecase) DispatchShippingManifest(ctx context.Context, tenantID, userID uuid.UUID, role string, manifestID uuid.UUID, req domain.DispatchShippingManifestRequest) (*domain.ShippingManifest, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return u.repo.DispatchShippingManifest(ctx, tenantID, manifestID, userID, req)
}

// GetWMSOutboundKPIs retrieves 8 operational SLA and quality metrics.
func (u *Usecase) GetWMSOutboundKPIs(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) (*domain.WMSOutboundKPISummary, error) {
	return u.repo.GetWMSOutboundKPIs(ctx, tenantID, warehouseID)
}
