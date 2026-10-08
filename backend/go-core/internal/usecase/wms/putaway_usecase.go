package wms

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

type PutawayRequest struct {
	WarehouseID    uuid.UUID       `json:"warehouse_id"`
	ProductID      uuid.UUID       `json:"product_id"`
	BatchID        uuid.UUID       `json:"batch_id"`
	Quantity       decimal.Decimal `json:"quantity"`
	DestLocationID uuid.UUID       `json:"dest_location_id"`
	Reason         *string         `json:"reason,omitempty"`
}

type UpdateWMSSettingsRequest struct {
	RequireReleaseApproval bool `json:"require_release_approval"`
}

type SetDefaultLocationRequest struct {
	ProductID   uuid.UUID `json:"product_id"`
	WarehouseID uuid.UUID `json:"warehouse_id"`
	LocationID  uuid.UUID `json:"location_id"`
}

// -----------------------------------------------------------------------------
// 1. Putaway (sentry-wms §1.2 step 2)
// -----------------------------------------------------------------------------

// GetPutawayPending lists items in INBOUND STAGING waiting to be put away into racks.
func (u *Usecase) GetPutawayPending(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.PutawayPendingLine, error) {
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, warehouseID); err != nil {
		return nil, err
	}

	stgLoc, err := u.repo.GetOrCreateStagingLocation(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("GetPutawayPending: get staging location: %w", err)
	}

	filter := domain.BatchBalanceFilter{
		LocationIDs: []uuid.UUID{stgLoc.ID},
		WarehouseID: &warehouseID,
	}
	balances, err := u.repo.ListBatchBalances(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("GetPutawayPending: list balances: %w", err)
	}

	defaultLocs, _ := u.repo.ListDefaultLocations(ctx, tenantID, nil)
	defaultMap := make(map[uuid.UUID]domain.ProductDefaultLocation)
	for _, dl := range defaultLocs {
		if dl.WarehouseID == warehouseID {
			defaultMap[dl.ProductID] = dl
		}
	}

	var lines []domain.PutawayPendingLine
	for _, bal := range balances {
		if !bal.Quantity.IsPositive() {
			continue
		}

		line := domain.PutawayPendingLine{
			ProductID:         bal.ProductID,
			ProductName:       bal.ProductName,
			ProductSKU:        bal.ProductSKU,
			BatchID:           bal.BatchID,
			BatchNumber:       bal.BatchNumber,
			ExpiryDate:        bal.ExpiryDate,
			BatchStatus:       bal.Status,
			StagingLocationID: stgLoc.ID,
			Quantity:          bal.Quantity,
		}

		batchObj, errBatch := u.repo.GetBatchByID(ctx, tenantID, bal.BatchID)
		if errBatch == nil && batchObj != nil && batchObj.SourceReceiptID != nil {
			line.SourceReceiptID = batchObj.SourceReceiptID
			rc, _, errRc := u.repo.GetStockReceiptByID(ctx, tenantID, *batchObj.SourceReceiptID)
			if errRc == nil && rc != nil {
				line.SourceReceiptNumber = rc.ReceiptNumber
				if rc.DestLocationID != uuid.Nil {
					// Fallback suggestion from receipt dest location
					line.SuggestedLocationID = &rc.DestLocationID
					line.SuggestionSource = "RECEIPT_DEST"
					locObj, errLoc := u.repo.GetLocationByID(ctx, tenantID, rc.DestLocationID)
					if errLoc == nil && locObj != nil {
						line.SuggestedLocationCode = locObj.Code
					}
				}
			}
		}

		// Default rack overrides receipt dest location (sentry-wms §1.2 suggestion priority)
		if def, ok := defaultMap[bal.ProductID]; ok {
			defCopy := def.LocationID
			line.DefaultLocationID = &defCopy
			line.SuggestedLocationID = &defCopy
			line.SuggestedLocationCode = def.LocationCode
			line.SuggestionSource = "DEFAULT_RACK"
		}

		if line.SuggestedLocationID == nil {
			line.SuggestionSource = "UNASSIGNED"
		}

		lines = append(lines, line)
	}

	return lines, nil
}

// ConfirmPutaway moves stock from STAGING_INBOUND to an INTERNAL rack.
// If the destination deviates from the product default rack, a reason is required (ADR-014 Invariant 4).
func (u *Usecase) ConfirmPutaway(ctx context.Context, tenantID, userID uuid.UUID, role string, req PutawayRequest) (*domain.StockMovement, error) {
	if role == "auditor" {
		return nil, domain.ErrUnauthorized
	}
	if req.WarehouseID == uuid.Nil || req.ProductID == uuid.Nil || req.BatchID == uuid.Nil || req.DestLocationID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	if !req.Quantity.IsPositive() {
		return nil, domain.ErrInvalidInput
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}

	destLoc, err := u.repo.GetLocationByID(ctx, tenantID, req.DestLocationID)
	if err != nil {
		return nil, fmt.Errorf("ConfirmPutaway: dest location: %w", err)
	}
	if destLoc.WarehouseID == nil || *destLoc.WarehouseID != req.WarehouseID {
		return nil, domain.ErrInvalidPutawayLocation
	}
	if destLoc.Type != domain.LocationTypeInternal {
		return nil, domain.ErrInvalidPutawayLocation
	}

	stgLoc, err := u.repo.GetOrCreateStagingLocation(ctx, tenantID, req.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("ConfirmPutaway: get staging location: %w", err)
	}

	// Look up product default rack in this warehouse
	defaultLocs, _ := u.repo.ListDefaultLocations(ctx, tenantID, &req.ProductID)
	var defaultLocID *uuid.UUID
	for _, dl := range defaultLocs {
		if dl.WarehouseID == req.WarehouseID {
			locCopy := dl.LocationID
			defaultLocID = &locCopy
			break
		}
	}

	var reason *string
	if req.Reason != nil {
		r := strings.TrimSpace(*req.Reason)
		if r != "" {
			reason = &r
		}
	}

	// Invariant 4: Overriding default rack requires reason with trimmed length >= 5
	if defaultLocID != nil && *defaultLocID != req.DestLocationID {
		if reason == nil || len([]rune(*reason)) < 5 {
			return nil, domain.ErrPutawayReasonRequired
		}
	}

	// Staging batch balance check: prevent over-putaway
	stagingBalances, err := u.repo.ListBatchBalances(ctx, tenantID, domain.BatchBalanceFilter{
		LocationIDs: []uuid.UUID{stgLoc.ID},
		WarehouseID: &req.WarehouseID,
		ProductID:   &req.ProductID,
	})
	if err != nil {
		return nil, fmt.Errorf("ConfirmPutaway: list staging balances: %w", err)
	}
	var stagedQty decimal.Decimal
	foundStagedBatch := false
	for _, b := range stagingBalances {
		if b.BatchID == req.BatchID {
			stagedQty = b.Quantity
			foundStagedBatch = true
			break
		}
	}
	if !foundStagedBatch || stagedQty.LessThan(req.Quantity) {
		return nil, domain.ErrInsufficientStock
	}

	cmd := domain.PutawayCommand{
		TenantID:          tenantID,
		WarehouseID:       req.WarehouseID,
		ProductID:         req.ProductID,
		BatchID:           req.BatchID,
		Quantity:          req.Quantity,
		StagingLocationID: stgLoc.ID,
		DestLocationID:    req.DestLocationID,
		UserID:            userID,
		Reason:            reason,
		DefaultLocationID: defaultLocID,
	}

	return u.repo.ConfirmPutaway(ctx, cmd)
}

// -----------------------------------------------------------------------------
// 2. Release Approval (PDF-06)
// -----------------------------------------------------------------------------

// ReleaseStockReceipt promotes all ON_HOLD lots of a posted receipt to RELEASED.
// Only owner, admin, or regional_manager can approve release (PDF-06).
func (u *Usecase) ReleaseStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, error) {
	if role != "owner" && role != "admin" && role != "regional_manager" {
		return nil, domain.ErrUnauthorized
	}

	rc, _, err := u.repo.GetStockReceiptByID(ctx, tenantID, receiptID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, rc.WarehouseID); err != nil {
		return nil, err
	}
	if rc.Status != domain.StockReceiptStatusPosted {
		return nil, domain.ErrReceiptNotOnHold
	}

	return u.repo.ReleaseStockReceipt(ctx, tenantID, receiptID, userID)
}

// -----------------------------------------------------------------------------
// 3. WMS Settings
// -----------------------------------------------------------------------------

func (u *Usecase) GetWMSSettings(ctx context.Context, tenantID, userID uuid.UUID, role string) (*domain.WMSSettings, error) {
	return u.repo.GetWMSSettings(ctx, tenantID)
}

func (u *Usecase) UpdateWMSSettings(ctx context.Context, tenantID, userID uuid.UUID, role string, req UpdateWMSSettingsRequest) (*domain.WMSSettings, error) {
	if role != "owner" && role != "admin" {
		return nil, domain.ErrUnauthorized
	}
	s := &domain.WMSSettings{
		TenantID:               tenantID,
		RequireReleaseApproval: req.RequireReleaseApproval,
		UpdatedBy:              &userID,
		UpdatedAt:              time.Now().UTC(),
	}
	if err := u.repo.UpsertWMSSettings(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// -----------------------------------------------------------------------------
// 4. Product Default Locations (CR-03)
// -----------------------------------------------------------------------------

func (u *Usecase) ListDefaultLocations(ctx context.Context, tenantID, userID uuid.UUID, role string, productID *uuid.UUID) ([]domain.ProductDefaultLocation, error) {
	return u.repo.ListDefaultLocations(ctx, tenantID, productID)
}

func (u *Usecase) SetDefaultLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, req SetDefaultLocationRequest) error {
	if role == "auditor" {
		return domain.ErrUnauthorized
	}
	if req.ProductID == uuid.Nil || req.WarehouseID == uuid.Nil || req.LocationID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return err
	}

	loc, err := u.repo.GetLocationByID(ctx, tenantID, req.LocationID)
	if err != nil {
		return fmt.Errorf("SetDefaultLocation: location not found: %w", err)
	}
	if loc.WarehouseID == nil || *loc.WarehouseID != req.WarehouseID {
		return domain.ErrInvalidPutawayLocation
	}
	if loc.Type != domain.LocationTypeInternal {
		return domain.ErrInvalidPutawayLocation
	}

	def := &domain.ProductDefaultLocation{
		TenantID:    tenantID,
		ProductID:   req.ProductID,
		WarehouseID: req.WarehouseID,
		LocationID:  req.LocationID,
		UpdatedBy:   &userID,
		UpdatedAt:   time.Now().UTC(),
	}
	return u.repo.SetDefaultLocation(ctx, def)
}

func (u *Usecase) DeleteDefaultLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, productID, warehouseID uuid.UUID) error {
	if role == "auditor" {
		return domain.ErrUnauthorized
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, warehouseID); err != nil {
		return err
	}
	return u.repo.DeleteDefaultLocation(ctx, tenantID, productID, warehouseID)
}

// -----------------------------------------------------------------------------
// 5. Traceability (KO-1c)
// -----------------------------------------------------------------------------

func (u *Usecase) TraceBatch(ctx context.Context, tenantID, userID uuid.UUID, role string, batchID uuid.UUID) (*domain.BatchTrace, error) {
	return u.repo.TraceBatch(ctx, tenantID, batchID)
}

func (u *Usecase) TraceDocument(ctx context.Context, tenantID, userID uuid.UUID, role string, refType string, refID uuid.UUID) (*domain.DocumentTrace, error) {
	refType = strings.ToUpper(strings.TrimSpace(refType))
	return u.repo.TraceDocument(ctx, tenantID, refType, refID)
}

func (u *Usecase) ListAuditTrail(ctx context.Context, tenantID, userID uuid.UUID, role string, entityType string, entityID uuid.UUID) ([]domain.AuditTrailEntry, error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	return u.repo.ListAuditTrail(ctx, tenantID, entityType, entityID)
}
