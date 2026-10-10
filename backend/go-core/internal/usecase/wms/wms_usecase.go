package wms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

type Usecase struct {
	repo domain.WMSRepository
}

func New(repo domain.WMSRepository) *Usecase {
	return &Usecase{repo: repo}
}

// -----------------------------------------------------------------------------
// Warehouse Scoping & Access Control
// -----------------------------------------------------------------------------

// ValidateWarehouseReadAccess checks whether the given user has permission to read warehouse data.
// Rules:
// 1. admin, owner, and auditor roles have tenant-wide read access across all warehouses.
// 2. warehouse role (staf gudang) only has access to warehouses assigned in user_warehouses.
// 3. regional_manager role has access to all warehouses belonging to the regional of their assigned warehouses.
func (u *Usecase) ValidateWarehouseReadAccess(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) error {
	// 0. Verify warehouse exists and belongs to this tenant
	if _, err := u.repo.GetWarehouseByID(ctx, tenantID, warehouseID); err != nil {
		return err
	}

	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	if normalizedRole == "admin" || normalizedRole == "owner" || normalizedRole == "auditor" {
		return nil
	}

	assignedWHs, err := u.repo.GetUserWarehouseIDs(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("ValidateWarehouseReadAccess: get user warehouses: %w", err)
	}

	// 1. Check explicit assignment
	for _, whID := range assignedWHs {
		if whID == warehouseID {
			return nil
		}
	}

	// 2. Check regional manager scope
	if normalizedRole == "regional_manager" {
		targetWH, err := u.repo.GetWarehouseByID(ctx, tenantID, warehouseID)
		if err != nil {
			return domain.ErrWarehouseNotFound
		}
		if targetWH.RegionalID != nil && len(assignedWHs) > 0 {
			userWHs, err := u.repo.ListWarehousesByIDs(ctx, tenantID, assignedWHs)
			if err == nil {
				for _, uwh := range userWHs {
					if uwh.RegionalID != nil && *uwh.RegionalID == *targetWH.RegionalID {
						return nil
					}
				}
			}
		}
	}

	return domain.ErrUnauthorizedWarehouse
}

// ValidateWarehouseWriteAccess checks whether the user has permission to perform write/dispatch operations.
// Rules:
// 1. auditor role is strictly read-only and is rejected with domain.ErrForbidden.
// 2. admin and owner roles have tenant-wide write access across all warehouses.
// 3. warehouse role (staf gudang) has write access only to assigned warehouses.
// 4. regional_manager role has write access to warehouses within their assigned regional.
func (u *Usecase) ValidateWarehouseWriteAccess(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) error {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	if normalizedRole == "auditor" {
		return domain.ErrForbidden
	}
	if normalizedRole != "admin" && normalizedRole != "owner" && normalizedRole != "warehouse" && normalizedRole != "regional_manager" {
		return domain.ErrForbidden
	}
	return u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, warehouseID)
}

// ValidateWarehouseAccess maintains backward compatibility, aliasing to ValidateWarehouseReadAccess.
func (u *Usecase) ValidateWarehouseAccess(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) error {
	return u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, warehouseID)
}

// validateLocationWarehouse verifies that a location exists and belongs to the expected warehouse.
func (u *Usecase) validateLocationWarehouse(ctx context.Context, tenantID, locationID, expectedWarehouseID uuid.UUID) error {
	loc, err := u.repo.GetLocationByID(ctx, tenantID, locationID)
	if err != nil {
		return err
	}
	if loc.WarehouseID == nil || *loc.WarehouseID != expectedWarehouseID {
		return domain.ErrUnauthorizedWarehouse
	}
	return nil
}

// FilterAccessibleWarehouses returns the list of warehouses accessible to the user based on role scoping.
func (u *Usecase) FilterAccessibleWarehouses(ctx context.Context, tenantID, userID uuid.UUID, role string) ([]domain.Warehouse, error) {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	allWarehouses, err := u.repo.ListWarehouses(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("FilterAccessibleWarehouses: list all: %w", err)
	}

	if normalizedRole == "admin" || normalizedRole == "owner" || normalizedRole == "auditor" {
		return allWarehouses, nil
	}

	assignedWHs, err := u.repo.GetUserWarehouseIDs(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("FilterAccessibleWarehouses: get user warehouses: %w", err)
	}

	if normalizedRole == "regional_manager" {
		// Collect all regional IDs associated with the user's assigned warehouses
		userWHs, err := u.repo.ListWarehousesByIDs(ctx, tenantID, assignedWHs)
		if err != nil {
			return nil, fmt.Errorf("FilterAccessibleWarehouses: get assigned wh details: %w", err)
		}
		regionalSet := make(map[uuid.UUID]bool)
		for _, w := range userWHs {
			if w.RegionalID != nil {
				regionalSet[*w.RegionalID] = true
			}
		}

		var filtered []domain.Warehouse
		for _, w := range allWarehouses {
			if w.RegionalID != nil && regionalSet[*w.RegionalID] {
				filtered = append(filtered, w)
			}
		}
		return filtered, nil
	}

	// Default warehouse staff: only directly assigned warehouses
	assignedSet := make(map[uuid.UUID]bool)
	for _, id := range assignedWHs {
		assignedSet[id] = true
	}
	var filtered []domain.Warehouse
	for _, w := range allWarehouses {
		if assignedSet[w.ID] {
			filtered = append(filtered, w)
		}
	}
	return filtered, nil
}

// -----------------------------------------------------------------------------
// Warehouse CRUD
// -----------------------------------------------------------------------------

type CreateWarehouseRequest struct {
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	RegionalID *uuid.UUID `json:"regional_id,omitempty"`
	Address    *string    `json:"address,omitempty"`
	IsActive   *bool      `json:"is_active,omitempty"`
}

func (u *Usecase) CreateWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, req CreateWarehouseRequest) (*domain.Warehouse, error) {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	if normalizedRole != "admin" && normalizedRole != "owner" {
		return nil, domain.ErrForbidden
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	wh := &domain.Warehouse{
		ID:         uuid.New(),
		TenantID:   tenantID,
		RegionalID: req.RegionalID,
		Code:       strings.ToUpper(strings.TrimSpace(req.Code)),
		Name:       strings.TrimSpace(req.Name),
		Address:    req.Address,
		IsActive:   isActive,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := u.repo.CreateWarehouse(ctx, wh); err != nil {
		return nil, err
	}
	return wh, nil
}

func (u *Usecase) GetWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID) (*domain.Warehouse, error) {
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, id); err != nil {
		return nil, err
	}
	return u.repo.GetWarehouseByID(ctx, tenantID, id)
}

func (u *Usecase) ListWarehouses(ctx context.Context, tenantID, userID uuid.UUID, role string) ([]domain.Warehouse, error) {
	return u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
}

func (u *Usecase) AssignUserWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, targetUserID, warehouseID uuid.UUID) error {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	if normalizedRole != "admin" && normalizedRole != "owner" {
		return domain.ErrForbidden
	}
	uw := &domain.UserWarehouse{
		UserID:      targetUserID,
		WarehouseID: warehouseID,
		TenantID:    tenantID,
		AssignedAt:  time.Now().UTC(),
	}
	return u.repo.AssignUserWarehouse(ctx, uw)
}

// -----------------------------------------------------------------------------
// Locations CRUD
// -----------------------------------------------------------------------------

type CreateLocationRequest struct {
	WarehouseID  *uuid.UUID           `json:"warehouse_id,omitempty"`
	ParentID     *uuid.UUID           `json:"parent_id,omitempty"`
	Code         string               `json:"code"`
	Barcode      *string              `json:"barcode,omitempty"`
	Name         string               `json:"name"`
	Type         *domain.LocationType `json:"type,omitempty"`
	IsPallet     bool                 `json:"is_pallet"`
	PalletNumber *string              `json:"pallet_number,omitempty"`
	MaxCapacity  *decimal.Decimal     `json:"max_capacity,omitempty"`
}

func (u *Usecase) CreateLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, req CreateLocationRequest) (*domain.WarehouseLocation, error) {
	if req.WarehouseID != nil {
		if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, *req.WarehouseID); err != nil {
			return nil, err
		}
	} else {
		// Non-warehouse (system/virtual) locations require admin/owner
		normalizedRole := strings.ToLower(strings.TrimSpace(role))
		if normalizedRole != "admin" && normalizedRole != "owner" {
			return nil, domain.ErrForbidden
		}
	}

	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		return nil, domain.ErrInvalidInput
	}

	locType := domain.LocationTypeInternal
	if req.Type != nil {
		locType = *req.Type
	}

	loc := &domain.WarehouseLocation{
		ID:           uuid.New(),
		TenantID:     tenantID,
		WarehouseID:  req.WarehouseID,
		ParentID:     req.ParentID,
		Code:         strings.TrimSpace(req.Code),
		Barcode:      req.Barcode,
		Name:         strings.TrimSpace(req.Name),
		Type:         locType,
		IsPallet:     req.IsPallet,
		PalletNumber: req.PalletNumber,
		MaxCapacity:  req.MaxCapacity,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := u.repo.CreateLocation(ctx, loc); err != nil {
		return nil, err
	}
	return loc, nil
}

// defaultReceivingLocation picks the putaway rack for transfer items without an
// explicit destination: the first INTERNAL rack of the warehouse, otherwise a
// "DEFAULT-<warehouse code>" rack that is created (or reused) on demand, so a
// warehouse with zero racks no longer blocks receiving.
func (u *Usecase) defaultReceivingLocation(ctx context.Context, tenantID, warehouseID uuid.UUID) (*domain.WarehouseLocation, error) {
	locs, err := u.repo.ListLocations(ctx, tenantID, &warehouseID)
	if err != nil {
		return nil, fmt.Errorf("defaultReceivingLocation: list: %w", err)
	}
	for i := range locs {
		if locs[i].Type == domain.LocationTypeInternal || locs[i].Type == "" {
			return &locs[i], nil
		}
	}

	wh, err := u.repo.GetWarehouseByID(ctx, tenantID, warehouseID)
	if err != nil {
		return nil, err
	}
	code := "DEFAULT-" + wh.Code
	if existing, err := u.repo.GetLocationByCode(ctx, tenantID, &warehouseID, code); err == nil {
		return existing, nil
	}
	now := time.Now().UTC()
	loc := &domain.WarehouseLocation{
		ID:          uuid.New(),
		TenantID:    tenantID,
		WarehouseID: &warehouseID,
		Code:        code,
		Name:        "Rak Default " + wh.Name,
		Type:        domain.LocationTypeInternal,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := u.repo.CreateLocation(ctx, loc); err != nil {
		return nil, fmt.Errorf("defaultReceivingLocation: create: %w", err)
	}
	return loc, nil
}

func (u *Usecase) ListLocations(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.WarehouseLocation, error) {
	if warehouseID != nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		return u.repo.ListLocations(ctx, tenantID, warehouseID)
	}

	// If no warehouse specified, return locations for warehouses user has access to
	accessibleWHs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
	if err != nil {
		return nil, fmt.Errorf("Usecase.ListLocations: %w", err)
	}

	allLocs, err := u.repo.ListLocations(ctx, tenantID, nil)
	if err != nil {
		return nil, fmt.Errorf("Usecase.ListLocations: %w", err)
	}

	whSet := make(map[uuid.UUID]struct{}, len(accessibleWHs))
	for _, wh := range accessibleWHs {
		whSet[wh.ID] = struct{}{}
	}

	filtered := make([]domain.WarehouseLocation, 0, len(allLocs))
	for _, loc := range allLocs {
		if loc.WarehouseID != nil {
			if _, ok := whSet[*loc.WarehouseID]; ok {
				filtered = append(filtered, loc)
			}
		} else {
			filtered = append(filtered, loc)
		}
	}
	return filtered, nil
}

// -----------------------------------------------------------------------------
// Barcode & SKU Resolution
// -----------------------------------------------------------------------------

func (u *Usecase) ResolveBarcode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.ResolvedProduct, error) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, domain.ErrInvalidInput
	}
	return u.repo.ResolveBarcode(ctx, tenantID, cleanCode)
}

type CreateBarcodeRequest struct {
	ProductID        uuid.UUID        `json:"product_id"`
	Barcode          string           `json:"barcode"`
	BarcodeSymbology string           `json:"barcode_symbology"`
	UOMName          string           `json:"uom_name"`
	Multiplier       *decimal.Decimal `json:"multiplier,omitempty"`
}

func (u *Usecase) CreateBarcode(ctx context.Context, tenantID uuid.UUID, req CreateBarcodeRequest) (*domain.ProductBarcode, error) {
	if req.ProductID == uuid.Nil || strings.TrimSpace(req.Barcode) == "" {
		return nil, domain.ErrInvalidInput
	}
	mult := decimal.NewFromInt(1)
	if req.Multiplier != nil && req.Multiplier.GreaterThan(decimal.Zero) {
		mult = *req.Multiplier
	}
	b := &domain.ProductBarcode{
		ID:               uuid.New(),
		TenantID:         tenantID,
		ProductID:        req.ProductID,
		Barcode:          strings.TrimSpace(req.Barcode),
		BarcodeSymbology: req.BarcodeSymbology,
		UOMName:          req.UOMName,
		Multiplier:       mult,
		CreatedAt:        time.Now().UTC(),
	}
	if err := u.repo.CreateBarcode(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

type CreateSKUMappingRequest struct {
	ProductID    uuid.UUID              `json:"product_id"`
	MappingType  domain.SKUMappingType  `json:"mapping_type"`
	ChannelName  string                 `json:"channel_name"`
	ExternalSKU  string                 `json:"external_sku"`
	ExternalName *string                `json:"external_name,omitempty"`
	Multiplier   *decimal.Decimal       `json:"multiplier,omitempty"`
}

func (u *Usecase) CreateSKUMapping(ctx context.Context, tenantID uuid.UUID, req CreateSKUMappingRequest) (*domain.ProductSKUMapping, error) {
	if req.ProductID == uuid.Nil || strings.TrimSpace(req.ExternalSKU) == "" || strings.TrimSpace(req.ChannelName) == "" {
		return nil, domain.ErrInvalidInput
	}
	mult := decimal.NewFromInt(1)
	if req.Multiplier != nil {
		if !req.Multiplier.IsPositive() || req.Multiplier.GreaterThan(decimal.NewFromInt(domain.MaxSKUMultiplier)) {
			return nil, &domain.StockReceiptValidationError{Msg: fmt.Sprintf("Multiplier wajib lebih dari 0 dan maksimal %d", domain.MaxSKUMultiplier)}
		}
		mult = *req.Multiplier
	}
	m := &domain.ProductSKUMapping{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ProductID:    req.ProductID,
		MappingType:  req.MappingType,
		ChannelName:  strings.TrimSpace(req.ChannelName),
		ExternalSKU:  strings.TrimSpace(req.ExternalSKU),
		ExternalName: req.ExternalName,
		Multiplier:   mult,
		Status:       domain.SKUMappingStatusApproved,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := u.repo.CreateSKUMapping(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Stock Transfers
// -----------------------------------------------------------------------------

type CreateTransferItemRequest struct {
	ProductID        uuid.UUID        `json:"product_id"`
	RequestedQty     decimal.Decimal  `json:"requested_qty"`
	SourceLocationID *uuid.UUID       `json:"source_location_id,omitempty"`
	DestLocationID   *uuid.UUID       `json:"dest_location_id,omitempty"`
}

type CreateTransferRequest struct {
	FromWarehouseID uuid.UUID                   `json:"from_warehouse_id"`
	ToWarehouseID   uuid.UUID                   `json:"to_warehouse_id"`
	TransferNumber  string                      `json:"transfer_number"`
	VehiclePlate    *string                     `json:"vehicle_plate,omitempty"`
	DriverName      *string                     `json:"driver_name,omitempty"`
	Notes           *string                     `json:"notes,omitempty"`
	Items           []CreateTransferItemRequest `json:"items"`
}

func (u *Usecase) CreateTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, req CreateTransferRequest) (*domain.StockTransfer, error) {
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.FromWarehouseID); err != nil {
		return nil, err
	}
	if _, err := u.repo.GetWarehouseByID(ctx, tenantID, req.ToWarehouseID); err != nil {
		return nil, err
	}
	if req.FromWarehouseID == req.ToWarehouseID {
		return nil, domain.ErrInvalidInput
	}
	if len(req.Items) == 0 {
		return nil, domain.ErrInvalidInput
	}

	transferNumber := strings.TrimSpace(req.TransferNumber)
	if transferNumber == "" {
		transferNumber = fmt.Sprintf("TR-%d", time.Now().UnixNano()/1e6)
	}

	t := &domain.StockTransfer{
		ID:              uuid.New(),
		TenantID:        tenantID,
		TransferNumber:  transferNumber,
		FromWarehouseID: req.FromWarehouseID,
		ToWarehouseID:   req.ToWarehouseID,
		Status:          domain.TransferStatusDraft,
		RequestedBy:     userID,
		VehiclePlate:    req.VehiclePlate,
		DriverName:      req.DriverName,
		Notes:           req.Notes,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	var items []domain.StockTransferItem
	for _, itemReq := range req.Items {
		if itemReq.ProductID == uuid.Nil || itemReq.RequestedQty.LessThanOrEqual(decimal.Zero) {
			return nil, domain.ErrInvalidInput
		}
		// Without a source rack the transfer can never be dispatched; reject now
		// instead of persisting a broken draft (e.g. TR-TEST-CHECK in prod).
		if itemReq.SourceLocationID == nil {
			return nil, domain.ErrSourceLocationRequired
		}
		if err := u.validateLocationWarehouse(ctx, tenantID, *itemReq.SourceLocationID, req.FromWarehouseID); err != nil {
			return nil, err
		}
		if itemReq.DestLocationID != nil {
			if err := u.validateLocationWarehouse(ctx, tenantID, *itemReq.DestLocationID, req.ToWarehouseID); err != nil {
				return nil, err
			}
		}
		items = append(items, domain.StockTransferItem{
			ID:               uuid.New(),
			TenantID:         tenantID,
			TransferID:       t.ID,
			ProductID:        itemReq.ProductID,
			RequestedQty:     itemReq.RequestedQty,
			SentQty:          decimal.Zero,
			ReceivedQty:      decimal.Zero,
			SourceLocationID: itemReq.SourceLocationID,
			DestLocationID:   itemReq.DestLocationID,
			CreatedAt:        time.Now().UTC(),
		})
	}

	if err := u.repo.CreateTransfer(ctx, t, items); err != nil {
		return nil, err
	}
	return t, nil
}

// isTransferApproverRole reports whether role is allowed to approve/reject
// stock transfers. Only tenant-wide (admin/owner) or regional (regional_manager)
// supervisory roles may act as approver — warehouse staff who request a
// transfer cannot approve their own or anyone else's transfer.
func isTransferApproverRole(role string) bool {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	return normalizedRole == "admin" || normalizedRole == "owner" || normalizedRole == "regional_manager"
}

// SubmitTransfer moves a DRAFT transfer into PENDING_APPROVAL, signaling that
// it is ready for a supervisor/manager to review. Any role with write access
// to the source warehouse (the requester themself, or another staff member of
// the same warehouse) may submit it.
func (u *Usecase) SubmitTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	t, _, err := u.repo.GetTransferByID(ctx, tenantID, transferID)
	if err != nil {
		return nil, err
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, t.FromWarehouseID); err != nil {
		return nil, err
	}

	if t.Status != domain.TransferStatusDraft {
		return nil, domain.ErrInvalidTransferStatus
	}

	newStatus := domain.TransferStatusPendingApproval
	if err := u.repo.UpdateTransferStatus(ctx, tenantID, t.ID, newStatus, nil, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("SubmitTransfer: update status: %w", err)
	}

	t.Status = newStatus
	return t, nil
}

// ApproveTransfer moves a PENDING_APPROVAL transfer into APPROVED, unlocking
// DispatchTransfer. Only admin/owner/regional_manager roles with read access
// to the source warehouse may approve, and the requester may never approve
// their own submitted transfer (segregation of duties).
func (u *Usecase) ApproveTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	t, _, err := u.repo.GetTransferByID(ctx, tenantID, transferID)
	if err != nil {
		return nil, err
	}

	if !isTransferApproverRole(role) {
		return nil, domain.ErrForbidden
	}
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, t.FromWarehouseID); err != nil {
		return nil, err
	}
	if t.RequestedBy == userID {
		return nil, domain.ErrSelfApprovalForbidden
	}

	if t.Status != domain.TransferStatusPendingApproval {
		return nil, domain.ErrInvalidTransferStatus
	}

	newStatus := domain.TransferStatusApproved
	if err := u.repo.UpdateTransferStatus(ctx, tenantID, t.ID, newStatus, nil, nil, &userID, nil); err != nil {
		return nil, fmt.Errorf("ApproveTransfer: update status: %w", err)
	}

	t.Status = newStatus
	t.ApprovedBy = &userID
	return t, nil
}

// RejectTransfer moves a PENDING_APPROVAL transfer into REJECTED with a
// mandatory reason, stopping the transfer lifecycle. Same approver rules as
// ApproveTransfer apply, including the self-approval/self-rejection guard.
func (u *Usecase) RejectTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID, reason string) (*domain.StockTransfer, error) {
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return nil, domain.ErrRejectionReasonRequired
	}

	t, _, err := u.repo.GetTransferByID(ctx, tenantID, transferID)
	if err != nil {
		return nil, err
	}

	if !isTransferApproverRole(role) {
		return nil, domain.ErrForbidden
	}
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, t.FromWarehouseID); err != nil {
		return nil, err
	}
	if t.RequestedBy == userID {
		return nil, domain.ErrSelfApprovalForbidden
	}

	if t.Status != domain.TransferStatusPendingApproval {
		return nil, domain.ErrInvalidTransferStatus
	}

	newStatus := domain.TransferStatusRejected
	if err := u.repo.UpdateTransferStatus(ctx, tenantID, t.ID, newStatus, nil, nil, &userID, &trimmedReason); err != nil {
		return nil, fmt.Errorf("RejectTransfer: update status: %w", err)
	}

	t.Status = newStatus
	t.ApprovedBy = &userID
	t.RejectionReason = &trimmedReason
	return t, nil
}

// CancelTransfer discards a DRAFT transfer. Only the requester (or another user
// with write access to the source warehouse) may cancel, mirroring
// SubmitTransfer's access rule. Cancelling never moves stock, so no ledger entry
// is written — the transfer stays on record as CANCELLED for audit instead of
// being hard-deleted.
func (u *Usecase) CancelTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	t, _, err := u.repo.GetTransferByID(ctx, tenantID, transferID)
	if err != nil {
		return nil, err
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, t.FromWarehouseID); err != nil {
		return nil, err
	}

	if t.Status != domain.TransferStatusDraft {
		return nil, domain.ErrTransferNotDraft
	}

	if err := u.repo.UpdateTransferStatus(ctx, tenantID, t.ID, domain.TransferStatusCancelled, nil, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("CancelTransfer: update status: %w", err)
	}

	t.Status = domain.TransferStatusCancelled
	return t, nil
}

// DispatchTransfer validates source stock and dispatches goods to transit location.
// Moves stock: Source Location -> @TRANSIT.
func (u *Usecase) DispatchTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	t, items, err := u.repo.GetTransferByID(ctx, tenantID, transferID)
	if err != nil {
		return nil, err
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, t.FromWarehouseID); err != nil {
		return nil, err
	}

	// Only an APPROVED transfer may be dispatched. DRAFT and PENDING_APPROVAL
	// transfers must go through SubmitTransfer -> ApproveTransfer first.
	if t.Status != domain.TransferStatusApproved {
		return nil, domain.ErrInvalidTransferStatus
	}

	transitLoc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeTransit)
	if err != nil {
		return nil, fmt.Errorf("DispatchTransfer: get transit loc: %w", err)
	}

	// 1. Validate locations scoping & source location presence
	for _, item := range items {
		if item.SourceLocationID == nil {
			return nil, fmt.Errorf("%w: product %s", domain.ErrSourceLocationRequired, item.ProductID)
		}
		if err := u.validateLocationWarehouse(ctx, tenantID, *item.SourceLocationID, t.FromWarehouseID); err != nil {
			return nil, err
		}
		if item.DestLocationID != nil {
			if err := u.validateLocationWarehouse(ctx, tenantID, *item.DestLocationID, t.ToWarehouseID); err != nil {
				return nil, err
			}
		}
	}

	// 2. Execute atomic stock deduction using advisory lock and movement creation
	now := time.Now().UTC()
	for i, item := range items {
		mov := &domain.StockMovement{
			ID:               uuid.New(),
			TenantID:         tenantID,
			MovementNumber:   fmt.Sprintf("TR-DISP-%s-%d", t.TransferNumber, i+1),
			ProductID:        item.ProductID,
			SourceLocationID: *item.SourceLocationID,
			DestLocationID:   transitLoc.ID,
			Quantity:         item.RequestedQty,
			UnitCost:         decimal.Zero,
			Status:           domain.StockMovementStatusDone,
			ReferenceType:    domain.StockRefTransfer,
			ReferenceID:      t.ID,
			ExecutedBy:       &userID,
			CreatedAt:        now,
		}
		if err := u.repo.DeductLocationStock(ctx, tenantID, *item.SourceLocationID, item.ProductID, item.RequestedQty, mov); err != nil {
			return nil, err
		}
	}

	// 3. Update transfer status
	newStatus := domain.TransferStatusInTransit
	if err := u.repo.UpdateTransferStatus(ctx, tenantID, t.ID, newStatus, &now, nil, nil, nil); err != nil {
		return nil, fmt.Errorf("DispatchTransfer: update status: %w", err)
	}

	t.Status = newStatus
	t.DispatchedAt = &now
	return t, nil
}

// ReceiveTransfer confirms receipt at destination warehouse.
// Moves stock: @TRANSIT -> Target Location.
func (u *Usecase) ReceiveTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	t, items, err := u.repo.GetTransferByID(ctx, tenantID, transferID)
	if err != nil {
		return nil, err
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, t.ToWarehouseID); err != nil {
		return nil, err
	}

	if t.Status != domain.TransferStatusInTransit && t.Status != domain.TransferStatusDispatched {
		return nil, domain.ErrInvalidTransferStatus
	}

	transitLoc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeTransit)
	if err != nil {
		return nil, fmt.Errorf("ReceiveTransfer: get transit loc: %w", err)
	}

	// Pre-fetch default location if any item has no DestLocationID
	var defaultTargetLocID *uuid.UUID
	for _, item := range items {
		if item.DestLocationID != nil {
			if err := u.validateLocationWarehouse(ctx, tenantID, *item.DestLocationID, t.ToWarehouseID); err != nil {
				return nil, err
			}
		} else if defaultTargetLocID == nil {
			loc, err := u.defaultReceivingLocation(ctx, tenantID, t.ToWarehouseID)
			if err != nil {
				return nil, err
			}
			defaultTargetLocID = &loc.ID
		}
	}

	dispMovs, _ := u.repo.ListMovementsByReference(ctx, tenantID, domain.StockRefTransfer, t.ID)
	prodDispatches := make(map[uuid.UUID][]domain.StockMovement)
	for _, dm := range dispMovs {
		if dm.DestLocationID == transitLoc.ID && dm.BatchID != nil {
			prodDispatches[dm.ProductID] = append(prodDispatches[dm.ProductID], dm)
		}
	}

	now := time.Now().UTC()
	movIdx := 0
	for _, item := range items {
		targetLocID := item.DestLocationID
		if targetLocID == nil {
			targetLocID = defaultTargetLocID
		}

		dispatched := prodDispatches[item.ProductID]
		if len(dispatched) > 0 {
			for _, dm := range dispatched {
				movIdx++
				mov := &domain.StockMovement{
					ID:               uuid.New(),
					TenantID:         tenantID,
					MovementNumber:   fmt.Sprintf("TR-RECV-%s-%d", t.TransferNumber, movIdx),
					ProductID:        item.ProductID,
					BatchID:          dm.BatchID,
					SourceLocationID: transitLoc.ID,
					DestLocationID:   *targetLocID,
					Quantity:         dm.Quantity,
					UnitCost:         decimal.Zero,
					Status:           domain.StockMovementStatusDone,
					ReferenceType:    domain.StockRefTransfer,
					ReferenceID:      t.ID,
					ExecutedBy:       &userID,
					CreatedAt:        now,
				}
				if err := u.repo.CreateStockMovement(ctx, mov); err != nil {
					return nil, fmt.Errorf("ReceiveTransfer: create stock movement: %w", err)
				}
			}
		} else {
			movIdx++
			qty := item.SentQty
			if qty.IsZero() {
				qty = item.RequestedQty
			}
			batch, err := u.repo.GetOrCreateBatch(ctx, &domain.StockBatch{
				TenantID:    tenantID,
				ProductID:   item.ProductID,
				BatchNumber: fmt.Sprintf("TR-%s", t.TransferNumber),
				Status:      domain.StockBatchStatusReleased,
				CreatedBy:   &userID,
			})
			if err != nil {
				return nil, fmt.Errorf("ReceiveTransfer: get batch: %w", err)
			}
			mov := &domain.StockMovement{
				ID:               uuid.New(),
				TenantID:         tenantID,
				MovementNumber:   fmt.Sprintf("TR-RECV-%s-%d", t.TransferNumber, movIdx),
				ProductID:        item.ProductID,
				BatchID:          &batch.ID,
				SourceLocationID: transitLoc.ID,
				DestLocationID:   *targetLocID,
				Quantity:         qty,
				UnitCost:         decimal.Zero,
				Status:           domain.StockMovementStatusDone,
				ReferenceType:    domain.StockRefTransfer,
				ReferenceID:      t.ID,
				ExecutedBy:       &userID,
				CreatedAt:        now,
			}
			if err := u.repo.CreateStockMovement(ctx, mov); err != nil {
				return nil, fmt.Errorf("ReceiveTransfer: create stock movement: %w", err)
			}
		}
	}

	newStatus := domain.TransferStatusReceived
	if err := u.repo.UpdateTransferStatus(ctx, tenantID, t.ID, newStatus, nil, &now, nil, nil); err != nil {
		return nil, fmt.Errorf("ReceiveTransfer: update status: %w", err)
	}

	t.Status = newStatus
	t.ReceivedAt = &now
	return t, nil
}

func (u *Usecase) GetTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, []domain.StockTransferItem, error) {
	t, items, err := u.repo.GetTransferByID(ctx, tenantID, transferID)
	if err != nil {
		return nil, nil, err
	}
	// Check access to either source or destination warehouse
	errFrom := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, t.FromWarehouseID)
	errTo := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, t.ToWarehouseID)
	if errFrom != nil && errTo != nil {
		return nil, nil, domain.ErrUnauthorizedWarehouse
	}
	return t, items, nil
}

func (u *Usecase) ListTransfers(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockTransfer, error) {
	if warehouseID != nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		return u.repo.ListTransfers(ctx, tenantID, warehouseID)
	}

	// Filter by accessible warehouses
	accessibleWHs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
	if err != nil {
		return nil, err
	}
	whSet := make(map[uuid.UUID]bool)
	for _, w := range accessibleWHs {
		whSet[w.ID] = true
	}

	allTransfers, err := u.repo.ListTransfers(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	var result []domain.StockTransfer
	for _, t := range allTransfers {
		if whSet[t.FromWarehouseID] || whSet[t.ToWarehouseID] {
			result = append(result, t)
		}
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// Delivery Orders (Surat Jalan)
// -----------------------------------------------------------------------------

type CreateDeliveryOrderItemRequest struct {
	ProductID  uuid.UUID       `json:"product_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	LocationID uuid.UUID       `json:"location_id"`
	BatchID    *uuid.UUID      `json:"batch_id,omitempty"`
	IsFreeItem bool            `json:"is_free_item,omitempty"`
}

type CreateDeliveryOrderRequest struct {
	SalesOrderID   *uuid.UUID                       `json:"sales_order_id,omitempty"` // optional
	CustomerID     *uuid.UUID                       `json:"customer_id,omitempty"`    // CR-02a
	WarehouseID    uuid.UUID                        `json:"warehouse_id"`
	DONumber       string                           `json:"do_number"`
	OrderType      *string                          `json:"order_type,omitempty"`
	Status         *domain.DeliveryOrderStatus      `json:"status,omitempty"`
	ExpeditionName *string                          `json:"expedition_name,omitempty"`
	TrackingNumber *string                          `json:"tracking_number,omitempty"`
	DriverName     *string                          `json:"driver_name,omitempty"`
	VehiclePlate   *string                          `json:"vehicle_plate,omitempty"`
	RecipientName  *string                          `json:"recipient_name,omitempty"`
	Items          []CreateDeliveryOrderItemRequest `json:"items"`
}

func (u *Usecase) CreateDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, req CreateDeliveryOrderRequest) (*domain.DeliveryOrder, error) {
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}
	if len(req.Items) == 0 {
		return nil, domain.ErrInvalidInput
	}

	recipientName := ""
	if req.RecipientName != nil {
		recipientName = strings.TrimSpace(*req.RecipientName)
	}
	if (req.CustomerID == nil || *req.CustomerID == uuid.Nil) && recipientName == "" {
		return nil, domain.ErrInvalidInput
	}

	type prodLocKey struct {
		prod uuid.UUID
		loc  uuid.UUID
	}
	requestedTotals := make(map[prodLocKey]decimal.Decimal, len(req.Items))
	// Validate items and location scoping
	for _, itemReq := range req.Items {
		if itemReq.ProductID == uuid.Nil || !itemReq.Quantity.IsPositive() {
			return nil, domain.ErrInvalidInput
		}
		if itemReq.LocationID != uuid.Nil {
			if err := u.validateLocationWarehouse(ctx, tenantID, itemReq.LocationID, req.WarehouseID); err != nil {
				return nil, err
			}
		}
		key := prodLocKey{prod: itemReq.ProductID, loc: itemReq.LocationID}
		requestedTotals[key] = requestedTotals[key].Add(itemReq.Quantity)
	}

	// Verify available stock (on_hand - allocated - quarantine) per product/location
	for key, totalReqQty := range requestedTotals {
		var locPtr *uuid.UUID
		if key.loc != uuid.Nil {
			locPtr = &key.loc
		}
		avail, err := u.repo.GetAvailableStock(ctx, tenantID, req.WarehouseID, locPtr, key.prod)
		if err != nil {
			return nil, fmt.Errorf("check available stock: %w", err)
		}
		if avail.LessThan(totalReqQty) {
			return nil, &domain.InsufficientStockError{
				Available: avail,
				Requested: totalReqQty,
				Msg:       fmt.Sprintf("stok tidak mencukupi (tersedia: %s, diminta: %s)", avail.String(), totalReqQty.String()),
			}
		}
	}

	doNumber := strings.TrimSpace(req.DONumber)
	if doNumber == "" {
		doNumber = fmt.Sprintf("DO-%d", time.Now().UnixNano()/1e6)
	}
	// The zero UUID is not a real Sales Order; treat it as "no SO".
	if req.SalesOrderID != nil && *req.SalesOrderID == uuid.Nil {
		req.SalesOrderID = nil
	}

	orderType := "DIRECT_DO"
	if req.OrderType != nil && strings.TrimSpace(*req.OrderType) != "" {
		orderType = strings.TrimSpace(*req.OrderType)
	}

	// Force status to DRAFT on creation - do NOT accept SHIPPED or DELIVERED from request
	status := domain.DeliveryOrderStatusDraft

	do := &domain.DeliveryOrder{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SalesOrderID:   req.SalesOrderID,
		CustomerID:     req.CustomerID,
		CreatedBy:      &userID,
		WarehouseID:    req.WarehouseID,
		DONumber:       doNumber,
		OrderType:      orderType,
		Status:         status,
		ExpeditionName: req.ExpeditionName,
		TrackingNumber: req.TrackingNumber,
		DriverName:     req.DriverName,
		VehiclePlate:   req.VehiclePlate,
		RecipientName:  req.RecipientName,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	var items []domain.DeliveryOrderItem
	for _, itemReq := range req.Items {
		if itemReq.BatchID != nil && *itemReq.BatchID != uuid.Nil && itemReq.LocationID != uuid.Nil {
			batch, err := u.repo.GetBatchByID(ctx, tenantID, *itemReq.BatchID)
			if err != nil {
				return nil, err
			}
			if batch.Status != domain.StockBatchStatusReleased {
				return nil, domain.ErrBatchOnHold
			}
			// Specific batch + location provided
			items = append(items, domain.DeliveryOrderItem{
				ID:              uuid.New(),
				TenantID:        tenantID,
				DeliveryOrderID: do.ID,
				ProductID:       itemReq.ProductID,
				Quantity:        itemReq.Quantity,
				LocationID:      itemReq.LocationID,
				BatchID:         itemReq.BatchID,
				IsFreeItem:      itemReq.IsFreeItem,
				CreatedAt:       time.Now().UTC(),
			})
			continue
		}

		// Auto FEFO allocation (BE-06): query INTERNAL rack batch balances and pick earliest expiry
		locTypeInternal := domain.LocationTypeInternal
		statusReleased := domain.StockBatchStatusReleased
		filter := domain.BatchBalanceFilter{
			WarehouseID:  &req.WarehouseID,
			ProductID:    &itemReq.ProductID,
			LocationType: &locTypeInternal,
			BatchStatus:  &statusReleased,
		}
		if itemReq.LocationID != uuid.Nil {
			filter.LocationIDs = []uuid.UUID{itemReq.LocationID}
		}
		balances, err := u.repo.ListBatchBalances(ctx, tenantID, filter)
		if err != nil {
			return nil, fmt.Errorf("auto FEFO allocation: list balances: %w", err)
		}
		allocs, err := domain.AllocateFEFO(balances, itemReq.Quantity, false)
		if err == nil && len(allocs) > 0 {
			for _, al := range allocs {
				bID := al.BatchID
				items = append(items, domain.DeliveryOrderItem{
					ID:              uuid.New(),
					TenantID:        tenantID,
					DeliveryOrderID: do.ID,
					ProductID:       itemReq.ProductID,
					Quantity:        al.Quantity,
					LocationID:      al.LocationID,
					BatchID:         &bID,
					IsFreeItem:      itemReq.IsFreeItem,
					CreatedAt:       time.Now().UTC(),
				})
			}
		} else {
			// Fallback when batches are not yet initialized (only if explicit location is provided)
			if itemReq.LocationID == uuid.Nil {
				return nil, &domain.InsufficientStockError{
					Available: decimal.Zero,
					Requested: itemReq.Quantity,
					Msg:       fmt.Sprintf("stok batch tidak ditemukan untuk alokasi otomatis produk %s", itemReq.ProductID),
				}
			}
			items = append(items, domain.DeliveryOrderItem{
				ID:              uuid.New(),
				TenantID:        tenantID,
				DeliveryOrderID: do.ID,
				ProductID:       itemReq.ProductID,
				Quantity:        itemReq.Quantity,
				LocationID:      itemReq.LocationID,
				BatchID:         nil,
				IsFreeItem:      itemReq.IsFreeItem,
				CreatedAt:       time.Now().UTC(),
			})
		}
	}

	if err := u.repo.CreateDeliveryOrder(ctx, do, items); err != nil {
		return nil, err
	}

	// Also generate initial Picking Task automatically so warehouse crew can pick
	_, _ = u.repo.GetOrCreatePickingTask(ctx, tenantID, do.ID)

	return do, nil
}

// ConfirmDeliveryOrder transitions a DRAFT Delivery Order to CONFIRMED.
func (u *Usecase) ConfirmDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, error) {
	do, _, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, do.WarehouseID); err != nil {
		return nil, err
	}
	if do.Status != domain.DeliveryOrderStatusDraft {
		return nil, domain.ErrInvalidStatus
	}
	return u.repo.ConfirmDeliveryOrder(ctx, tenantID, doID, userID)
}

// CancelDeliveryOrder cancels a DRAFT Delivery Order and releases allocated stock.
func (u *Usecase) CancelDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, error) {
	do, _, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, err
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, do.WarehouseID); err != nil {
		return nil, err
	}

	if do.Status != domain.DeliveryOrderStatusDraft {
		return nil, domain.ErrDeliveryOrderNotDraft
	}

	if err := u.repo.CancelDeliveryOrder(ctx, tenantID, do.ID, userID); err != nil {
		return nil, fmt.Errorf("CancelDeliveryOrder: update status: %w", err)
	}

	do.Status = domain.DeliveryOrderStatusCancelled
	return do, nil
}

// DispatchDeliveryOrder transitions a DO to SHIPPED and deducts inventory from rack to @CUSTOMER.
func (u *Usecase) DispatchDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, error) {
	do, items, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, err
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, do.WarehouseID); err != nil {
		return nil, err
	}

	// F3 Status check: Dispatch hanya dari PACKED (atau CONFIRMED jika RequirePickPack disabled).
	// DRAFT tidak boleh langsung dispatch (exfiltration blocker).
	settings, _ := u.repo.GetWMSSettings(ctx, tenantID)
	requirePickPack := false
	if settings != nil {
		requirePickPack = settings.RequirePickPack
	}
	if requirePickPack {
		if do.Status != domain.DeliveryOrderStatusPacked || do.PackedBy == nil {
			return nil, domain.ErrDeliveryOrderNotPacked
		}
	} else {
		// jika picking/packing optional, izinkan CONFIRMED atau PACKED, TAPI BLOCK DRAFT
		if do.Status == domain.DeliveryOrderStatusDraft {
			return nil, domain.ErrDeliveryOrderNotPacked
		}
		if do.Status != domain.DeliveryOrderStatusPacked && do.Status != domain.DeliveryOrderStatusConfirmed {
			return nil, domain.ErrDeliveryOrderNotPacked
		}
	}

	// F2 Mandatory shipping details
	if do.DriverName == nil || strings.TrimSpace(*do.DriverName) == "" ||
		do.VehiclePlate == nil || strings.TrimSpace(*do.VehiclePlate) == "" ||
		do.ExpeditionName == nil || strings.TrimSpace(*do.ExpeditionName) == "" {
		return nil, domain.ErrDeliveryOrderIncompleteShip
	}

	// F2 Segregation of duties
	if role != "admin" && role != "owner" {
		if do.CreatedBy != nil && *do.CreatedBy == userID {
			return nil, domain.ErrSelfApprovalForbidden
		}
	}

	// Verify location warehouse scoping for each item
	for _, item := range items {
		if err := u.validateLocationWarehouse(ctx, tenantID, item.LocationID, do.WarehouseID); err != nil {
			return nil, err
		}
	}

	custLoc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
	if err != nil {
		return nil, fmt.Errorf("DispatchDeliveryOrder: get customer loc: %w", err)
	}

	// Execute atomic stock deduction using advisory lock and movement creation
	now := time.Now().UTC()
	for i, item := range items {
		mov := &domain.StockMovement{
			ID:               uuid.New(),
			TenantID:         tenantID,
			MovementNumber:   fmt.Sprintf("DO-SHIP-%s-%d", do.DONumber, i+1),
			ProductID:        item.ProductID,
			SourceLocationID: item.LocationID,
			DestLocationID:   custLoc.ID,
			Quantity:         item.Quantity,
			UnitCost:         decimal.Zero,
			Status:           domain.StockMovementStatusDone,
			ReferenceType:    domain.StockRefDeliveryOrder,
			ReferenceID:      do.ID,
			BatchID:          item.BatchID, // KO-1: batch preserved on dispatch
			ExecutedBy:       &userID,
			CreatedAt:        now,
		}
		if err := u.repo.DeductLocationStock(ctx, tenantID, item.LocationID, item.ProductID, item.Quantity, mov); err != nil {
			return nil, err
		}
	}

	newStatus := domain.DeliveryOrderStatusShipped
	if err := u.repo.UpdateDeliveryOrderStatus(ctx, tenantID, do.ID, newStatus, nil); err != nil {
		return nil, fmt.Errorf("DispatchDeliveryOrder: update status: %w", err)
	}

	do.Status = newStatus
	do.DispatchedBy = &userID
	return do, nil
}

func (u *Usecase) GetDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, []domain.DeliveryOrderItem, error) {
	do, items, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, nil, err
	}
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, do.WarehouseID); err != nil {
		return nil, nil, err
	}
	return do, items, nil
}

func (u *Usecase) ListDeliveryOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.DeliveryOrder, error) {
	if warehouseID != nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		return u.repo.ListDeliveryOrders(ctx, tenantID, warehouseID)
	}

	accessibleWHs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
	if err != nil {
		return nil, err
	}
	whSet := make(map[uuid.UUID]bool)
	for _, w := range accessibleWHs {
		whSet[w.ID] = true
	}

	allOrders, err := u.repo.ListDeliveryOrders(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	var result []domain.DeliveryOrder
	for _, do := range allOrders {
		if whSet[do.WarehouseID] {
			result = append(result, do)
		}
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// Stock Opname (Physical Inventory Counting)
// -----------------------------------------------------------------------------

type CreateStockOpnameRequest struct {
	WarehouseID  uuid.UUID `json:"warehouse_id"`
	OpnameNumber string    `json:"opname_number,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
}

type AddOpnameItemRequest struct {
	ProductID   uuid.UUID       `json:"product_id"`
	LocationID  uuid.UUID       `json:"location_id"`
	PhysicalQty decimal.Decimal `json:"physical_qty"`
	Notes       *string         `json:"notes,omitempty"`
}

func (u *Usecase) CreateStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, req CreateStockOpnameRequest) (*domain.StockOpname, error) {
	if req.WarehouseID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}

	opnameNumber := strings.TrimSpace(req.OpnameNumber)
	if opnameNumber == "" {
		opnameNumber = fmt.Sprintf("OPN-%d", time.Now().UnixNano()/1e6)
	}

	now := time.Now().UTC()
	op := &domain.StockOpname{
		ID:           uuid.New(),
		TenantID:     tenantID,
		WarehouseID:  req.WarehouseID,
		OpnameNumber: opnameNumber,
		Status:       domain.StockOpnameStatusDraft,
		ConductedBy:  userID,
		Notes:        req.Notes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := u.repo.CreateStockOpname(ctx, op); err != nil {
		return nil, fmt.Errorf("CreateStockOpname: %w", err)
	}
	return op, nil
}

func (u *Usecase) AddOpnameItem(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID, req AddOpnameItemRequest) (*domain.StockOpnameItem, error) {
	if opnameID == uuid.Nil || req.ProductID == uuid.Nil || req.LocationID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	if req.PhysicalQty.IsNegative() {
		return nil, domain.ErrInvalidInput
	}

	op, err := u.repo.GetStockOpnameByID(ctx, tenantID, opnameID)
	if err != nil {
		return nil, err
	}

	if op.Status != domain.StockOpnameStatusDraft && op.Status != domain.StockOpnameStatusInProgress {
		return nil, domain.ErrInvalidOpnameStatus
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, op.WarehouseID); err != nil {
		return nil, err
	}

	if err := u.validateLocationWarehouse(ctx, tenantID, req.LocationID, op.WarehouseID); err != nil {
		return nil, err
	}

	systemQty, err := u.repo.GetStockByLocation(ctx, tenantID, req.LocationID, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("AddOpnameItem: get system stock: %w", err)
	}

	discrepancyQty := req.PhysicalQty.Sub(systemQty)
	item := &domain.StockOpnameItem{
		ID:             uuid.New(),
		OpnameID:       opnameID,
		TenantID:       tenantID,
		ProductID:      req.ProductID,
		LocationID:     req.LocationID,
		SystemQty:      systemQty,
		PhysicalQty:    req.PhysicalQty,
		DiscrepancyQty: discrepancyQty,
		Notes:          req.Notes,
		CreatedAt:      time.Now().UTC(),
	}

	if err := u.repo.AddStockOpnameItem(ctx, item); err != nil {
		return nil, fmt.Errorf("AddOpnameItem: persist: %w", err)
	}
	return item, nil
}

func (u *Usecase) CompleteStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID) (*domain.StockOpname, error) {
	if opnameID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	op, err := u.repo.GetStockOpnameByID(ctx, tenantID, opnameID)
	if err != nil {
		return nil, err
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, op.WarehouseID); err != nil {
		return nil, err
	}

	if op.Status != domain.StockOpnameStatusDraft &&
		op.Status != domain.StockOpnameStatusInProgress &&
		op.Status != domain.StockOpnameStatusPendingApproval {
		return nil, domain.ErrInvalidOpnameStatus
	}

	items, err := u.repo.ListStockOpnameItems(ctx, tenantID, opnameID)
	if err != nil {
		return nil, fmt.Errorf("CompleteStockOpname: list items: %w", err)
	}

	// Discrepancy reason check
	for _, item := range items {
		if !item.DiscrepancyQty.IsZero() {
			itemHasNotes := item.Notes != nil && strings.TrimSpace(*item.Notes) != ""
			opnameHasNotes := op.Notes != nil && strings.TrimSpace(*op.Notes) != ""
			if !itemHasNotes && !opnameHasNotes {
				return nil, domain.ErrInvalidInput
			}
		}
	}

	// Approver check
	isApprover := role == "admin" || role == "owner" || role == "regional_manager"
	if !isApprover {
		if op.Status == domain.StockOpnameStatusDraft || op.Status == domain.StockOpnameStatusInProgress {
			// Transition to PENDING_APPROVAL without writing ledger movements
			if err := u.repo.UpdateStockOpnameStatus(ctx, tenantID, op.ID, domain.StockOpnameStatusPendingApproval, nil); err != nil {
				return nil, err
			}
			op.Status = domain.StockOpnameStatusPendingApproval
			return op, nil
		}
		return nil, domain.ErrForbidden
	}

	// Caller IS an approver:
	// Check segregation of duties: approver cannot be the one who conducted the opname!
	if op.ConductedBy == userID {
		return nil, domain.ErrSelfApprovalForbidden
	}

	lossLoc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeLoss)
	if err != nil {
		return nil, fmt.Errorf("CompleteStockOpname: get loss location: %w", err)
	}

	now := time.Now().UTC()
	for i, item := range items {
		if item.DiscrepancyQty.IsZero() {
			continue
		}
		if item.DiscrepancyQty.GreaterThan(decimal.Zero) {
			// Surplus recovery: movement from @LOSS to item.LocationID.
			// Found stock has no inbound lot, so it gets an opname lot
			// (ADR-014 Invariant 1: every movement carries a batch).
			batch, err := u.repo.GetOrCreateBatch(ctx, &domain.StockBatch{
				TenantID:    tenantID,
				ProductID:   item.ProductID,
				BatchNumber: fmt.Sprintf("OPN-%s", op.OpnameNumber),
				Status:      domain.StockBatchStatusReleased,
				CreatedBy:   &userID,
			})
			if err != nil {
				return nil, fmt.Errorf("CompleteStockOpname: surplus batch: %w", err)
			}
			mov := &domain.StockMovement{
				BatchID:          &batch.ID,
				ID:               uuid.New(),
				TenantID:         tenantID,
				MovementNumber:   fmt.Sprintf("OPN-SURPLUS-%s-%d", op.OpnameNumber, i+1),
				ProductID:        item.ProductID,
				SourceLocationID: lossLoc.ID,
				DestLocationID:   item.LocationID,
				Quantity:         item.DiscrepancyQty,
				UnitCost:         decimal.Zero,
				Status:           domain.StockMovementStatusDone,
				ReferenceType:    domain.StockRefOpname,
				ReferenceID:      op.ID,
				ExecutedBy:       &userID,
				CreatedAt:        now,
			}
			if err := u.repo.CreateStockMovement(ctx, mov); err != nil {
				return nil, fmt.Errorf("CompleteStockOpname: create surplus movement: %w", err)
			}
		} else {
			// Deficit loss adjustment: movement from item.LocationID to @LOSS
			absQty := item.DiscrepancyQty.Abs()
			mov := &domain.StockMovement{
				ID:               uuid.New(),
				TenantID:         tenantID,
				MovementNumber:   fmt.Sprintf("OPN-LOSS-%s-%d", op.OpnameNumber, i+1),
				ProductID:        item.ProductID,
				SourceLocationID: item.LocationID,
				DestLocationID:   lossLoc.ID,
				Quantity:         absQty,
				UnitCost:         decimal.Zero,
				Status:           domain.StockMovementStatusDone,
				ReferenceType:    domain.StockRefOpname,
				ReferenceID:      op.ID,
				ExecutedBy:       &userID,
				CreatedAt:        now,
			}
			// DeductLocationStock allocates the loss across on-hand batches
			// (FEFO, ON_HOLD included for opname) so each movement has batch_id.
			if err := u.repo.DeductLocationStock(ctx, tenantID, item.LocationID, item.ProductID, absQty, mov); err != nil {
				return nil, fmt.Errorf("CompleteStockOpname: create loss movement: %w", err)
			}
		}
	}

	if err := u.repo.UpdateStockOpnameStatus(ctx, tenantID, op.ID, domain.StockOpnameStatusCompleted, &userID); err != nil {
		return nil, fmt.Errorf("CompleteStockOpname: update status: %w", err)
	}

	op.Status = domain.StockOpnameStatusCompleted
	op.ApprovedBy = &userID
	op.UpdatedAt = now
	return op, nil
}

func (u *Usecase) GetStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID) (*domain.StockOpname, []domain.StockOpnameItem, error) {
	if opnameID == uuid.Nil {
		return nil, nil, domain.ErrInvalidInput
	}
	op, err := u.repo.GetStockOpnameByID(ctx, tenantID, opnameID)
	if err != nil {
		return nil, nil, err
	}
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, op.WarehouseID); err != nil {
		return nil, nil, err
	}
	items, err := u.repo.ListStockOpnameItems(ctx, tenantID, opnameID)
	if err != nil {
		return nil, nil, fmt.Errorf("GetStockOpname: list items: %w", err)
	}
	return op, items, nil
}

func (u *Usecase) ListStockOpnames(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockOpname, error) {
	if warehouseID != nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		return u.repo.ListStockOpnames(ctx, tenantID, warehouseID)
	}

	accessibleWHs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
	if err != nil {
		return nil, err
	}
	whSet := make(map[uuid.UUID]bool)
	for _, w := range accessibleWHs {
		whSet[w.ID] = true
	}

	allOpnames, err := u.repo.ListStockOpnames(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	var result []domain.StockOpname
	for _, op := range allOpnames {
		if whSet[op.WarehouseID] {
			result = append(result, op)
		}
	}
	if result == nil {
		result = []domain.StockOpname{}
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// Stock Scrap (Damaged Goods & Scrap Quarantine)
// -----------------------------------------------------------------------------

type CreateStockScrapRequest struct {
	WarehouseID      uuid.UUID       `json:"warehouse_id"`
	ProductID        uuid.UUID       `json:"product_id"`
	SourceLocationID uuid.UUID       `json:"source_location_id"`
	ScrapLocationID  *uuid.UUID      `json:"scrap_location_id,omitempty"`
	Quantity         decimal.Decimal `json:"quantity"`
	Reason           string          `json:"reason"`
	ScrapNumber      string          `json:"scrap_number,omitempty"`
}

func (u *Usecase) CreateStockScrap(ctx context.Context, tenantID, userID uuid.UUID, role string, req CreateStockScrapRequest) (*domain.StockScrap, error) {
	if req.WarehouseID == uuid.Nil || req.ProductID == uuid.Nil || req.SourceLocationID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	if req.Quantity.LessThanOrEqual(decimal.Zero) {
		return nil, domain.ErrInvalidInput
	}
	reason := strings.TrimSpace(req.Reason)
	if len(reason) < 10 {
		return nil, domain.ErrInvalidInput
	}

	const scrapApprovalThreshold = 10
	if req.Quantity.GreaterThan(decimal.NewFromInt(scrapApprovalThreshold)) {
		if role != "admin" && role != "owner" {
			return nil, domain.ErrScrapApprovalRequired
		}
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}

	if err := u.validateLocationWarehouse(ctx, tenantID, req.SourceLocationID, req.WarehouseID); err != nil {
		return nil, err
	}

	var scrapLocationID uuid.UUID
	if req.ScrapLocationID == nil || *req.ScrapLocationID == uuid.Nil {
		scrapLoc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
		if err != nil {
			return nil, fmt.Errorf("CreateStockScrap: get virtual scrap loc: %w", err)
		}
		scrapLocationID = scrapLoc.ID
	} else {
		scrapLocationID = *req.ScrapLocationID
		targetLoc, err := u.repo.GetLocationByID(ctx, tenantID, scrapLocationID)
		if err != nil {
			return nil, err
		}
		if targetLoc.Type != domain.LocationTypeScrap {
			if targetLoc.WarehouseID == nil || *targetLoc.WarehouseID != req.WarehouseID {
				return nil, domain.ErrUnauthorizedWarehouse
			}
		}
	}

	if req.SourceLocationID == scrapLocationID {
		return nil, domain.ErrInvalidInput
	}

	scrapNumber := strings.TrimSpace(req.ScrapNumber)
	if scrapNumber == "" {
		scrapNumber = fmt.Sprintf("SCRAP-%d", time.Now().UnixNano()/1e6)
	}

	scrapID := uuid.New()
	now := time.Now().UTC()
	mov := &domain.StockMovement{
		ID:               uuid.New(),
		TenantID:         tenantID,
		MovementNumber:   fmt.Sprintf("SCRAP-MOV-%s", scrapNumber),
		ProductID:        req.ProductID,
		SourceLocationID: req.SourceLocationID,
		DestLocationID:   scrapLocationID,
		Quantity:         req.Quantity,
		UnitCost:         decimal.Zero,
		Status:           domain.StockMovementStatusDone,
		ReferenceType:    domain.StockRefScrap,
		ReferenceID:      scrapID,
		ExecutedBy:       &userID,
		CreatedAt:        now,
	}

	if err := u.repo.DeductLocationStock(ctx, tenantID, req.SourceLocationID, req.ProductID, req.Quantity, mov); err != nil {
		return nil, err
	}

	var approvedBy *uuid.UUID
	if role == "admin" || role == "owner" {
		approvedBy = &userID
	}

	scrap := &domain.StockScrap{
		ID:                 scrapID,
		TenantID:           tenantID,
		ScrapNumber:        scrapNumber,
		WarehouseID:        req.WarehouseID,
		ProductID:          req.ProductID,
		SourceLocationID:   req.SourceLocationID,
		ScrapLocationID:    scrapLocationID,
		Quantity:           req.Quantity,
		Reason:             reason,
		ReportedBy:         userID,
		ApprovedBy:         approvedBy,
		CreatedAt:          now,
	}

	if err := u.repo.CreateStockScrap(ctx, scrap); err != nil {
		return nil, fmt.Errorf("CreateStockScrap: save scrap: %w", err)
	}
	return scrap, nil
}

func (u *Usecase) ListStockScraps(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockScrap, error) {
	if warehouseID != nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		return u.repo.ListStockScraps(ctx, tenantID, warehouseID)
	}

	accessibleWHs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
	if err != nil {
		return nil, err
	}
	whSet := make(map[uuid.UUID]bool)
	for _, w := range accessibleWHs {
		whSet[w.ID] = true
	}

	allScraps, err := u.repo.ListStockScraps(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	var result []domain.StockScrap
	for _, scrap := range allScraps {
		if whSet[scrap.WarehouseID] {
			result = append(result, scrap)
		}
	}
	if result == nil {
		result = []domain.StockScrap{}
	}
	return result, nil
}

// -----------------------------------------------------------------------------
// Marketplace Sales Orders & SKU Mappings
// -----------------------------------------------------------------------------

type ImportOrderItemRequest struct {
	ExternalSKU string          `json:"external_sku"`
	ItemName    string          `json:"item_name"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Subtotal    decimal.Decimal `json:"subtotal"`
}

type ImportOrderRequest struct {
	ExternalOrderID string                   `json:"external_order_id"`
	OrderDate       time.Time                `json:"order_date"`
	CustomerName    *string                  `json:"customer_name,omitempty"`
	CustomerPhone   *string                  `json:"customer_phone,omitempty"`
	ShippingAddress *string                  `json:"shipping_address,omitempty"`
	Courier         *string                  `json:"courier,omitempty"`
	TrackingNumber  *string                  `json:"tracking_number,omitempty"`
	TotalAmount     decimal.Decimal          `json:"total_amount"`
	ShippingFee     decimal.Decimal          `json:"shipping_fee"`
	MarketplaceFee  decimal.Decimal          `json:"marketplace_fee"`
	NetAmount       decimal.Decimal          `json:"net_amount"`
	Items           []ImportOrderItemRequest `json:"items"`
}

type ImportMarketplaceOrdersRequest struct {
	WarehouseID      uuid.UUID                 `json:"warehouse_id"`
	SourceLocationID uuid.UUID                 `json:"source_location_id"`
	Channel          domain.MarketplaceChannel `json:"channel"`
	FileName         string                    `json:"file_name"`
	Orders           []ImportOrderRequest      `json:"orders"`
}

type ImportMarketplaceOrdersResponse struct {
	Batch  *domain.MarketplaceImportBatch `json:"batch"`
	Orders []domain.MarketplaceOrder      `json:"orders"`
}

// deductOrderStock moves every line of a fully mapped order from the order's
// explicit source rack to @CUSTOMER. Quantity per line is
// item.Quantity * item.Multiplier, where Multiplier is the snapshot taken
// when the line was mapped (M3), never the mapping's current value.
func (u *Usecase) deductOrderStock(ctx context.Context, tenantID, userID uuid.UUID, order *domain.MarketplaceOrder) error {
	if len(order.Items) == 0 {
		return nil
	}
	// M5: no implicit rack. Orders without a source rack (legacy rows from
	// before migration 040) must not guess one.
	if order.SourceLocationID == nil || *order.SourceLocationID == uuid.Nil {
		return &domain.StockReceiptValidationError{Msg: "Pesanan marketplace belum memiliki rak sumber pemotongan stok"}
	}
	srcLoc, err := u.repo.GetLocationByID(ctx, tenantID, *order.SourceLocationID)
	if err != nil {
		return fmt.Errorf("deductOrderStock: source location: %w", err)
	}

	custLoc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
	if err != nil {
		return fmt.Errorf("deductOrderStock: get customer location: %w", err)
	}

	for _, item := range order.Items {
		if item.ProductID == nil {
			return fmt.Errorf("deductOrderStock: product id is nil")
		}
		mult := decimal.NewFromInt(1)
		if item.Multiplier != nil && item.Multiplier.IsPositive() {
			mult = *item.Multiplier
		}

		deductQty := item.Quantity.Mul(mult)
		// M6: record the real cost of goods leaving stock. The movement is
		// per base unit (deductQty), so unit_cost is the base unit cost.
		unitCost, err := u.repo.GetProductCostPrice(ctx, tenantID, *item.ProductID)
		if err != nil {
			return fmt.Errorf("deductOrderStock: cost price: %w", err)
		}
		mov := &domain.StockMovement{
			ID:               uuid.New(),
			TenantID:         tenantID,
			MovementNumber:   fmt.Sprintf("MKT-MOV-%s", uuid.New().String()[:8]),
			ProductID:        *item.ProductID,
			SourceLocationID: srcLoc.ID,
			DestLocationID:   custLoc.ID,
			Quantity:         deductQty,
			UnitCost:         unitCost,
			Status:           domain.StockMovementStatusDone,
			ReferenceType:    domain.StockRefMarketplace,
			ReferenceID:      order.ID,
			ExecutedBy:       &userID,
			CreatedAt:        time.Now().UTC(),
		}

		if err := u.repo.DeductLocationStock(ctx, tenantID, srcLoc.ID, *item.ProductID, deductQty, mov); err != nil {
			return err
		}
	}

	return nil
}

// resolveMarketplaceSourceLocation validates the rack chosen at import (M5):
// it must exist, belong to the import warehouse and be an INTERNAL rack.
func (u *Usecase) resolveMarketplaceSourceLocation(ctx context.Context, tenantID, warehouseID, locationID uuid.UUID) (*domain.WarehouseLocation, error) {
	if locationID == uuid.Nil {
		return nil, &domain.StockReceiptValidationError{Msg: "Rak sumber pemotongan stok wajib dipilih"}
	}
	loc, err := u.repo.GetLocationByID(ctx, tenantID, locationID)
	if err != nil {
		if errors.Is(err, domain.ErrLocationNotFound) {
			return nil, &domain.StockReceiptValidationError{Msg: "Rak sumber tidak ditemukan"}
		}
		return nil, fmt.Errorf("resolveMarketplaceSourceLocation: %w", err)
	}
	if loc.WarehouseID == nil || *loc.WarehouseID != warehouseID || loc.Type != domain.LocationTypeInternal {
		return nil, &domain.StockReceiptValidationError{Msg: "Rak sumber harus rak internal di gudang yang dipilih"}
	}
	return loc, nil
}
func (u *Usecase) ImportMarketplaceOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, req ImportMarketplaceOrdersRequest) (*ImportMarketplaceOrdersResponse, error) {
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}

	ch := domain.MarketplaceChannel(strings.ToUpper(strings.TrimSpace(string(req.Channel))))
	if !ch.IsValid() {
		return nil, &domain.StockReceiptValidationError{Msg: fmt.Sprintf("Channel %q tidak valid. Pilihan: SHOPEE, TOKOPEDIA, TIKTOK, LAZADA, BLIBLI, OTHER", req.Channel)}
	}

	// M9: row/batch order count limits.
	if len(req.Orders) == 0 {
		return nil, &domain.StockReceiptValidationError{Msg: "Tidak ada pesanan untuk diimpor"}
	}
	if len(req.Orders) > domain.MaxMarketplaceBatchOrders {
		return nil, &domain.StockReceiptValidationError{Msg: fmt.Sprintf("Jumlah pesanan (%d) melebihi batas maksimal %d per import", len(req.Orders), domain.MaxMarketplaceBatchOrders)}
	}

	// M1: validate every line before anything is written, so a bad file
	// creates no batch, no orders and no movements.
	maxItem := decimal.NewFromInt(domain.MaxMarketplaceItemQty)
	batchQty := decimal.Zero
	for _, o := range req.Orders {
		for _, it := range o.Items {
			if !it.Quantity.IsPositive() || it.Quantity.GreaterThan(maxItem) {
				return nil, &domain.StockReceiptValidationError{Msg: fmt.Sprintf(
					"Pesanan %s SKU %s: jumlah %s tidak valid (wajib 1-%d)",
					strings.TrimSpace(o.ExternalOrderID), strings.TrimSpace(it.ExternalSKU), it.Quantity.String(), domain.MaxMarketplaceItemQty)}
			}
			// M10: marketplace products are tracked in piece units (PCS), quantity must be an integer.
			if !it.Quantity.Equal(it.Quantity.Floor()) {
				return nil, &domain.StockReceiptValidationError{Msg: fmt.Sprintf(
					"Pesanan %s SKU %s: kuantitas %s tidak valid (harus bilangan bulat untuk PCS)",
					strings.TrimSpace(o.ExternalOrderID), strings.TrimSpace(it.ExternalSKU), it.Quantity.String())}
			}
			batchQty = batchQty.Add(it.Quantity)
		}
	}
	if batchQty.GreaterThan(decimal.NewFromInt(domain.MaxMarketplaceBatchQty)) {
		return nil, &domain.StockReceiptValidationError{Msg: fmt.Sprintf(
			"Total jumlah %s melebihi batas %d per import", batchQty.String(), domain.MaxMarketplaceBatchQty)}
	}

	srcLoc, err := u.resolveMarketplaceSourceLocation(ctx, tenantID, req.WarehouseID, req.SourceLocationID)
	if err != nil {
		return nil, err
	}

	batchNum := fmt.Sprintf("BATCH-MKT-%s-%d", ch, time.Now().UTC().UnixNano())
	fileName := strings.TrimSpace(req.FileName)
	if fileName == "" {
		fileName = "manual_import"
	}

	batch := &domain.MarketplaceImportBatch{
		ID:              uuid.New(),
		TenantID:        tenantID,
		BatchNumber:     batchNum,
		Channel:         ch,
		WarehouseID:     req.WarehouseID,
		FileName:        fileName,
		TotalOrders:     len(req.Orders),
		ProcessedOrders: 0,
		FailedOrders:    0,
		UnmappedSKUs:    0,
		Status:          domain.MarketplaceBatchStatusProcessing,
		UploadedBy:      userID,
		CreatedAt:       time.Now().UTC(),
	}

	if err := u.repo.CreateMarketplaceBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("ImportMarketplaceOrders: create batch: %w", err)
	}

	createdOrders := make([]domain.MarketplaceOrder, 0, len(req.Orders))

	for _, orderReq := range req.Orders {
		extOrderID := strings.TrimSpace(orderReq.ExternalOrderID)
		if extOrderID == "" {
			batch.FailedOrders++
			continue
		}

		// Idempotency check: duplicate order check
		existing, err := u.repo.GetMarketplaceOrderByExternalID(ctx, tenantID, ch, extOrderID)
		if err == nil && existing != nil {
			batch.FailedOrders++
			continue
		}

		orderID := uuid.New()
		now := time.Now().UTC()
		orderDate := orderReq.OrderDate
		if orderDate.IsZero() {
			orderDate = now
		}

		order := domain.MarketplaceOrder{
			ID:              orderID,
			TenantID:        tenantID,
			BatchID:         &batch.ID,
			WarehouseID:      req.WarehouseID,
			SourceLocationID: &srcLoc.ID,
			Channel:          ch,
			ExternalOrderID:  extOrderID,
			OrderDate:       orderDate,
			CustomerName:    orderReq.CustomerName,
			CustomerPhone:   orderReq.CustomerPhone,
			ShippingAddress: orderReq.ShippingAddress,
			Courier:         orderReq.Courier,
			TrackingNumber:  orderReq.TrackingNumber,
			TotalAmount:     orderReq.TotalAmount,
			ShippingFee:     orderReq.ShippingFee,
			MarketplaceFee:  orderReq.MarketplaceFee,
			NetAmount:       orderReq.NetAmount,
			Status:          domain.MarketplaceOrderStatusCompleted,
			CreatedAt:       now,
			Items:           make([]domain.MarketplaceOrderItem, 0, len(orderReq.Items)),
		}

		hasUnmapped := false
		calcSubtotal := decimal.Zero

		for _, itmReq := range orderReq.Items {
			if itmReq.Quantity.LessThanOrEqual(decimal.Zero) {
				return nil, fmt.Errorf("%w: item %q quantity must be greater than zero", domain.ErrInvalidInput, itmReq.ItemName)
			}
			extSKU := strings.TrimSpace(itmReq.ExternalSKU)
			item := domain.MarketplaceOrderItem{
				ID:          uuid.New(),
				TenantID:    tenantID,
				OrderID:     orderID,
				ExternalSKU: extSKU,
				ItemName:    strings.TrimSpace(itmReq.ItemName),
				Quantity:    itmReq.Quantity,
				UnitPrice:   itmReq.UnitPrice,
				Subtotal:    itmReq.Subtotal,
				IsMapped:    false,
			}
			if item.Subtotal.IsZero() && !item.Quantity.IsZero() && !item.UnitPrice.IsZero() {
				item.Subtotal = item.Quantity.Mul(item.UnitPrice)
			}
			calcSubtotal = calcSubtotal.Add(item.Subtotal)

			// 1. Check SKU mapping table
			mapping, err := u.repo.GetSKUMapping(ctx, tenantID, string(ch), extSKU)
			if err == nil && mapping != nil {
				item.ProductID = &mapping.ProductID
				item.IsMapped = true
				mult := mapping.Multiplier
				if mult.LessThanOrEqual(decimal.Zero) {
					mult = decimal.NewFromInt(1)
				}
				item.Multiplier = &mult
			} else {
				// 2. Lookup master product by SKU matching external_sku
				prod, err := u.repo.GetProductBySKU(ctx, tenantID, extSKU)
				if err == nil && prod != nil {
					item.ProductID = &prod.ID
					item.IsMapped = true
					one := decimal.NewFromInt(1)
					item.Multiplier = &one
				} else {
					// 3. Unmapped SKU
					item.IsMapped = false
					hasUnmapped = true
					batch.UnmappedSKUs++
				}
			}

			order.Items = append(order.Items, item)
		}

		if order.TotalAmount.IsZero() && !calcSubtotal.IsZero() {
			order.TotalAmount = calcSubtotal
		}
		// M8: amounts must be non-negative and a supplied net must reconcile
		// with total + shipping - marketplace fee (1 rupiah rounding tolerance).
		expectedNet := order.TotalAmount.Add(order.ShippingFee).Sub(order.MarketplaceFee)
		if order.TotalAmount.IsNegative() || order.ShippingFee.IsNegative() || order.MarketplaceFee.IsNegative() || expectedNet.IsNegative() {
			return nil, &domain.StockReceiptValidationError{Msg: fmt.Sprintf("Pesanan %s: nominal tidak boleh negatif", extOrderID)}
		}
		if order.NetAmount.IsZero() {
			order.NetAmount = expectedNet
		} else if order.NetAmount.Sub(expectedNet).Abs().GreaterThan(decimal.NewFromInt(1)) {
			return nil, &domain.StockReceiptValidationError{Msg: fmt.Sprintf(
				"Pesanan %s: net_amount %s tidak sesuai total + ongkir - biaya marketplace (%s)", extOrderID, order.NetAmount.String(), expectedNet.String())}
		}

		if hasUnmapped {
			order.Status = domain.MarketplaceOrderStatusUnmappedSKU
			if err := u.repo.CreateMarketplaceOrder(ctx, &order); err != nil {
				if errors.Is(err, domain.ErrDuplicateMarketplaceOrder) {
					batch.FailedOrders++
					continue
				}
				return nil, fmt.Errorf("ImportMarketplaceOrders: create order: %w", err)
			}
		} else {
			// M7: mapped orders wait for batch approval; no stock moves here.
			order.Status = domain.MarketplaceOrderStatusPending
			if err := u.repo.CreateMarketplaceOrder(ctx, &order); err != nil {
				if errors.Is(err, domain.ErrDuplicateMarketplaceOrder) {
					batch.FailedOrders++
					continue
				}
				return nil, fmt.Errorf("ImportMarketplaceOrders: create order: %w", err)
			}
		}

		batch.ProcessedOrders++
		createdOrders = append(createdOrders, order)
	}

	if batch.ProcessedOrders == 0 && batch.FailedOrders > 0 {
		batch.Status = domain.MarketplaceBatchStatusFailed
	} else {
		batch.Status = domain.MarketplaceBatchStatusPendingApproval
	}

	if err := u.repo.UpdateMarketplaceBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("ImportMarketplaceOrders: update batch: %w", err)
	}

	return &ImportMarketplaceOrdersResponse{
		Batch:  batch,
		Orders: createdOrders,
	}, nil
}

func (u *Usecase) ResolveSKUMapping(ctx context.Context, tenantID, userID uuid.UUID, role string, req CreateSKUMappingRequest) (*domain.ProductSKUMapping, error) {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	if normalizedRole == "auditor" {
		return nil, domain.ErrForbidden
	}
	if normalizedRole != "admin" && normalizedRole != "owner" && normalizedRole != "warehouse" && normalizedRole != "regional_manager" {
		return nil, domain.ErrForbidden
	}

	if req.ProductID == uuid.Nil || strings.TrimSpace(req.ExternalSKU) == "" || strings.TrimSpace(req.ChannelName) == "" {
		return nil, domain.ErrInvalidInput
	}
	// M2: a multiplier > 1 multiplies every future deduction, so only
	// owner/admin may set it. Warehouse staff can only map 1:1.
	if req.Multiplier != nil && req.Multiplier.GreaterThan(decimal.NewFromInt(1)) &&
		normalizedRole != "admin" && normalizedRole != "owner" {
		return nil, domain.ErrForbidden
	}

	// 1. Create / update mapping
	mapping, err := u.CreateSKUMapping(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	channel := domain.MarketplaceChannel(strings.ToUpper(strings.TrimSpace(req.ChannelName)))
	extSKU := strings.TrimSpace(req.ExternalSKU)

	// 2. Update unmapped order items in DB
	if err := u.repo.UpdateUnmappedOrderItems(ctx, tenantID, channel, extSKU, req.ProductID, mapping.Multiplier); err != nil {
		return nil, fmt.Errorf("ResolveSKUMapping: update items: %w", err)
	}

	// 3. Find pending unmapped orders containing this SKU
	pendingOrders, err := u.repo.GetPendingUnmappedOrdersBySKU(ctx, tenantID, channel, extSKU)
	if err != nil {
		return nil, fmt.Errorf("ResolveSKUMapping: get pending orders: %w", err)
	}

	// 4. For affected orders where all items are now mapped, deduct stock and complete order
	for _, order := range pendingOrders {
		allMapped := true
		for _, itm := range order.Items {
			if !itm.IsMapped {
				allMapped = false
				break
			}
		}

		if allMapped {
			// M4: claim the order first; a concurrent resolve that loses the
			// claim must not deduct a second time.
			// M7: the resolved order joins the approval queue instead of
			// deducting immediately.
			claimed, errClaim := u.repo.ClaimMarketplaceOrder(ctx, tenantID, order.ID, domain.MarketplaceOrderStatusUnmappedSKU, domain.MarketplaceOrderStatusPending)
			if errClaim != nil {
				return nil, fmt.Errorf("ResolveSKUMapping: claim order: %w", errClaim)
			}
			if !claimed || order.BatchID == nil {
				continue
			}
			// A batch that was already approved must be approved again for
			// this newly deductible order.
			if _, errReopen := u.repo.ClaimMarketplaceBatch(ctx, tenantID, *order.BatchID, domain.MarketplaceBatchStatusDeducted, domain.MarketplaceBatchStatusPendingApproval, nil); errReopen != nil {
				return nil, fmt.Errorf("ResolveSKUMapping: reopen batch: %w", errReopen)
			}
		}
	}

	return mapping, nil
}

// ApproveMarketplaceBatch deducts stock for every PENDING order of a batch
// (M7). Only owner/admin may approve and never the uploader (segregation of
// duties). The PENDING_APPROVAL->APPROVED claim makes concurrent approvals
// deduct once.
func (u *Usecase) ApproveMarketplaceBatch(ctx context.Context, tenantID, userID uuid.UUID, role string, batchID uuid.UUID) (*domain.MarketplaceImportBatch, error) {
	batch, err := u.authorizeMarketplaceBatchDecision(ctx, tenantID, userID, role, batchID)
	if err != nil {
		return nil, err
	}
	claimed, err := u.repo.ClaimMarketplaceBatch(ctx, tenantID, batchID, domain.MarketplaceBatchStatusPendingApproval, domain.MarketplaceBatchStatusApproved, &userID)
	if err != nil {
		return nil, fmt.Errorf("ApproveMarketplaceBatch: claim: %w", err)
	}
	if !claimed {
		return nil, &domain.StockReceiptValidationError{Msg: "Batch tidak menunggu persetujuan (sudah diproses atau ditolak)"}
	}

	pending := domain.MarketplaceOrderStatusPending
	orders, err := u.repo.ListMarketplaceOrders(ctx, tenantID, nil, &batchID, &pending)
	if err != nil {
		return nil, fmt.Errorf("ApproveMarketplaceBatch: list orders: %w", err)
	}
	for i := range orders {
		order := orders[i]
		ok, errClaim := u.repo.ClaimMarketplaceOrder(ctx, tenantID, order.ID, domain.MarketplaceOrderStatusPending, domain.MarketplaceOrderStatusProcessing)
		if errClaim != nil {
			return nil, fmt.Errorf("ApproveMarketplaceBatch: claim order: %w", errClaim)
		}
		if !ok {
			continue
		}
		next := domain.MarketplaceOrderStatusCompleted
		if errStock := u.deductOrderStock(ctx, tenantID, userID, &order); errStock != nil {
			next = domain.MarketplaceOrderStatusStockInsufficient
		} else if _, errSO := u.repo.EnsureMarketplaceSalesOrder(ctx, tenantID, &order); errSO != nil {
			// M8: stock already left; surface the error so the batch stays
			// APPROVED (visible) instead of silently missing its sales order.
			return nil, fmt.Errorf("ApproveMarketplaceBatch: sales order: %w", errSO)
		}
		if errUpd := u.repo.UpdateMarketplaceOrderStatus(ctx, tenantID, order.ID, next); errUpd != nil {
			return nil, fmt.Errorf("ApproveMarketplaceBatch: update order: %w", errUpd)
		}
	}

	if _, err := u.repo.ClaimMarketplaceBatch(ctx, tenantID, batchID, domain.MarketplaceBatchStatusApproved, domain.MarketplaceBatchStatusDeducted, nil); err != nil {
		return nil, fmt.Errorf("ApproveMarketplaceBatch: finish: %w", err)
	}
	return u.repo.GetMarketplaceBatchByID(ctx, tenantID, batch.ID)
}

// RejectMarketplaceBatch closes a batch without moving stock (M7).
func (u *Usecase) RejectMarketplaceBatch(ctx context.Context, tenantID, userID uuid.UUID, role string, batchID uuid.UUID) (*domain.MarketplaceImportBatch, error) {
	if _, err := u.authorizeMarketplaceBatchDecision(ctx, tenantID, userID, role, batchID); err != nil {
		return nil, err
	}
	claimed, err := u.repo.ClaimMarketplaceBatch(ctx, tenantID, batchID, domain.MarketplaceBatchStatusPendingApproval, domain.MarketplaceBatchStatusRejected, &userID)
	if err != nil {
		return nil, fmt.Errorf("RejectMarketplaceBatch: claim: %w", err)
	}
	if !claimed {
		return nil, &domain.StockReceiptValidationError{Msg: "Batch tidak menunggu persetujuan (sudah diproses atau ditolak)"}
	}
	pending := domain.MarketplaceOrderStatusPending
	orders, err := u.repo.ListMarketplaceOrders(ctx, tenantID, nil, &batchID, &pending)
	if err != nil {
		return nil, fmt.Errorf("RejectMarketplaceBatch: list orders: %w", err)
	}
	for _, o := range orders {
		if _, err := u.repo.ClaimMarketplaceOrder(ctx, tenantID, o.ID, domain.MarketplaceOrderStatusPending, domain.MarketplaceOrderStatusFailed); err != nil {
			return nil, fmt.Errorf("RejectMarketplaceBatch: fail order: %w", err)
		}
	}
	return u.repo.GetMarketplaceBatchByID(ctx, tenantID, batchID)
}

func (u *Usecase) authorizeMarketplaceBatchDecision(ctx context.Context, tenantID, userID uuid.UUID, role string, batchID uuid.UUID) (*domain.MarketplaceImportBatch, error) {
	r := strings.ToLower(strings.TrimSpace(role))
	if r != "admin" && r != "owner" {
		return nil, domain.ErrForbidden
	}
	batch, err := u.repo.GetMarketplaceBatchByID(ctx, tenantID, batchID)
	if err != nil {
		return nil, err
	}
	if batch.UploadedBy == userID {
		return nil, domain.ErrMarketplaceSelfApproval
	}
	return batch, nil
}

func (u *Usecase) ListMarketplaceBatches(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.MarketplaceImportBatch, error) {
	if warehouseID != nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		return u.repo.ListMarketplaceBatches(ctx, tenantID, warehouseID)
	}

	accessibleWHs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
	if err != nil {
		return nil, err
	}
	whSet := make(map[uuid.UUID]bool)
	for _, w := range accessibleWHs {
		whSet[w.ID] = true
	}

	allBatches, err := u.repo.ListMarketplaceBatches(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	res := make([]domain.MarketplaceImportBatch, 0)
	for _, b := range allBatches {
		if whSet[b.WarehouseID] {
			res = append(res, b)
		}
	}
	return res, nil
}

func (u *Usecase) ListMarketplaceOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID, batchID *uuid.UUID, status *domain.MarketplaceOrderStatus) ([]domain.MarketplaceOrder, error) {
	if warehouseID != nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		return u.repo.ListMarketplaceOrders(ctx, tenantID, warehouseID, batchID, status)
	}

	accessibleWHs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
	if err != nil {
		return nil, err
	}
	whSet := make(map[uuid.UUID]bool)
	for _, w := range accessibleWHs {
		whSet[w.ID] = true
	}

	allOrders, err := u.repo.ListMarketplaceOrders(ctx, tenantID, nil, batchID, status)
	if err != nil {
		return nil, err
	}

	res := make([]domain.MarketplaceOrder, 0)
	for _, o := range allOrders {
		if whSet[o.WarehouseID] {
			res = append(res, o)
		}
	}
	return res, nil
}

func (u *Usecase) GetMarketplaceOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, orderID uuid.UUID) (*domain.MarketplaceOrder, error) {
	order, err := u.repo.GetMarketplaceOrderByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, order.WarehouseID); err != nil {
		return nil, err
	}
	return order, nil
}

func (u *Usecase) ListSKUMappings(ctx context.Context, tenantID uuid.UUID, channelName string) ([]domain.ProductSKUMapping, error) {
	return u.repo.ListSKUMappings(ctx, tenantID, channelName)
}

func (u *Usecase) ListStockMovements(ctx context.Context, tenantID, userID uuid.UUID, role string, productID, locationID *uuid.UUID, limit int) ([]domain.StockMovement, error) {
	return u.repo.ListStockMovements(ctx, tenantID, productID, locationID, limit)
}

func (u *Usecase) ListStockSummary(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockSummary, error) {
	return u.repo.ListStockSummary(ctx, tenantID, warehouseID)
}
