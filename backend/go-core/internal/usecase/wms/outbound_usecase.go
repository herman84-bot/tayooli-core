package wms

// Sprint 3 Outbound Usecase: Picking Tasks, Pack Station Scan Verification,
// FEFO Re-pick Recommendation & Damaged Stock Isolation (Sentry-WMS §1.3, OCA §1.3).

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

func (u *Usecase) GetPickingTask(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.PickingTaskDetail, error) {
	do, _, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, do.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.GetOrCreatePickingTask(ctx, tenantID, doID)
}

func (u *Usecase) StartPickingTask(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.PickingTaskDetail, error) {
	detail, err := u.GetPickingTask(ctx, tenantID, userID, role, doID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, detail.DeliveryOrder.WarehouseID); err != nil {
		return nil, err
	}
	if detail.Task.Status == domain.PickingTaskStatusPending {
		if err := u.repo.UpdatePickingTaskStatus(ctx, tenantID, detail.Task.ID, domain.PickingTaskStatusInProgress, &userID); err != nil {
			return nil, err
		}
	}
	return u.repo.GetPickingTaskByDO(ctx, tenantID, doID)
}

func (u *Usecase) RecordPickingItem(ctx context.Context, tenantID, userID uuid.UUID, role string, doID, taskItemID uuid.UUID, pickedQty decimal.Decimal) (*domain.PickingTaskDetail, error) {
	detail, err := u.GetPickingTask(ctx, tenantID, userID, role, doID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, detail.DeliveryOrder.WarehouseID); err != nil {
		return nil, err
	}
	if err := u.repo.RecordPickingItemProgress(ctx, tenantID, taskItemID, pickedQty); err != nil {
		return nil, err
	}
	return u.repo.GetPickingTaskByDO(ctx, tenantID, doID)
}

// ReportPickingDamaged isolates damaged stock found during picking into @QUARANTINE
// and queries the next available FEFO batch to recommend an alternative pick (PDF-03/04).
func (u *Usecase) ReportPickingDamaged(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID, req domain.PickingDamagedReportRequest) (*domain.PickingDamagedReportResult, error) {
	if req.DamagedQty.LessThanOrEqual(decimal.Zero) || strings.TrimSpace(req.Reason) == "" {
		return nil, domain.ErrInvalidInput
	}
	do, _, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, do.WarehouseID); err != nil {
		return nil, err
	}

	qrn, err := u.repo.GetOrCreateQuarantineLocation(ctx, tenantID, do.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("ReportPickingDamaged: quarantine loc: %w", err)
	}

	mov := &domain.StockMovement{
		ID:               uuid.New(),
		TenantID:         tenantID,
		MovementNumber:   fmt.Sprintf("QRN-PICK-%s-%d", do.DONumber, time.Now().UnixNano()%10000),
		ProductID:        req.ProductID,
		SourceLocationID: req.SourceLocationID,
		DestLocationID:   qrn.ID,
		Quantity:         req.DamagedQty,
		Status:           domain.StockMovementStatusDone,
		ReferenceType:    "PICKING_DAMAGED",
		ReferenceID:      do.ID,
		BatchID:          &req.BatchID,
		ExecutedBy:       &userID,
		CreatedAt:        time.Now().UTC(),
	}
	if err := u.repo.DeductLocationStock(ctx, tenantID, req.SourceLocationID, req.ProductID, req.DamagedQty, mov); err != nil {
		return nil, err
	}

	// Recommend next FEFO batch from INTERNAL locations (excluding the damaged batch)
	locType := domain.LocationTypeInternal
	balances, err := u.repo.ListBatchBalances(ctx, tenantID, domain.BatchBalanceFilter{
		WarehouseID:  &do.WarehouseID,
		ProductID:    &req.ProductID,
		LocationType: &locType,
	})
	var altBatch *domain.BatchAllocation
	if err == nil {
		var validBalances []domain.BatchBalance
		for _, b := range balances {
			if b.BatchID != req.BatchID && b.Quantity.GreaterThanOrEqual(req.DamagedQty) {
				validBalances = append(validBalances, b)
			}
		}
		if allocs, aErr := domain.AllocateFEFO(validBalances, req.DamagedQty, false); aErr == nil && len(allocs) > 0 {
			altBatch = &allocs[0]
		}
	}

	return &domain.PickingDamagedReportResult{
		MovementID:       mov.ID,
		QuarantineLocID:  qrn.ID,
		DamagedQty:       req.DamagedQty,
		AlternativeBatch: altBatch,
	}, nil
}

// ScanPackStationItem validates scanned barcode against DO line items and increments packed_qty (BE-07).
func (u *Usecase) ScanPackStationItem(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID, req domain.PackScanRequest) (*domain.PackScanResult, error) {
	scanQty := req.Quantity
	if scanQty.LessThanOrEqual(decimal.Zero) {
		scanQty = decimal.NewFromInt(1)
	}
	barcode := strings.TrimSpace(req.Barcode)
	if barcode == "" {
		return nil, domain.ErrInvalidInput
	}

	do, items, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, do.WarehouseID); err != nil {
		return nil, err
	}

	// Try resolving barcode to a product
	resolved, _ := u.repo.ResolveBarcode(ctx, tenantID, barcode)

	var targetItem *domain.DeliveryOrderItem
	for i := range items {
		it := &items[i]
		if it.PackedQty.GreaterThanOrEqual(it.Quantity) {
			continue // Already fully packed
		}
		// Match by: resolved product id OR SKU OR product name OR batch number
		match := false
		if resolved != nil && resolved.ProductID == it.ProductID {
			match = true
		} else if it.ProductSKU != nil && strings.EqualFold(*it.ProductSKU, barcode) {
			match = true
		} else if it.BatchNumber != nil && strings.EqualFold(*it.BatchNumber, barcode) {
			match = true
		} else if it.ProductName != nil && strings.EqualFold(*it.ProductName, barcode) {
			match = true
		}

		if match {
			targetItem = it
			break
		}
	}

	if targetItem == nil {
		return nil, domain.ErrPackBarcodeMismatch
	}

	// Increment packed qty on item
	updatedItem, err := u.repo.UpdateDOPackScan(ctx, tenantID, doID, targetItem.ID, scanQty)
	if err != nil {
		return nil, err
	}

	// Count overall DO completeness
	_, allItems, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, err
	}
	packedCount := 0
	for _, it := range allItems {
		if it.PackedQty.GreaterThanOrEqual(it.Quantity) {
			packedCount++
		}
	}
	orderCompleted := (packedCount == len(allItems))

	pName, pSKU := "", ""
	if updatedItem.ProductName != nil {
		pName = *updatedItem.ProductName
	}
	if updatedItem.ProductSKU != nil {
		pSKU = *updatedItem.ProductSKU
	}

	return &domain.PackScanResult{
		ItemID:         updatedItem.ID,
		ProductID:      updatedItem.ProductID,
		ProductName:    pName,
		ProductSKU:     pSKU,
		ScannedQty:     scanQty,
		PackedQty:      updatedItem.PackedQty,
		RequestedQty:   updatedItem.Quantity,
		ItemCompleted:  updatedItem.PackedQty.GreaterThanOrEqual(updatedItem.Quantity),
		OrderCompleted: orderCompleted,
		TotalItems:     len(allItems),
		PackedItems:    packedCount,
	}, nil
}

// CompletePackStation confirms 100% packing match and transitions DO to PACKED.
func (u *Usecase) CompletePackStation(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID, req domain.PackCompleteRequest) (*domain.DeliveryOrder, error) {
	do, _, err := u.repo.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, do.WarehouseID); err != nil {
		return nil, err
	}

	packedDO, err := u.repo.CompleteDOPacking(ctx, tenantID, doID, userID, req)
	if err != nil {
		return nil, err
	}
	return packedDO, nil
}
