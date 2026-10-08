package postgres

// Sprint 3 Outbound Postgres Repository: Picking Tasks, Pack Station Scan,
// Shelf Sorting & Outbound Lifecycle (ADR-014 Invariant 1, Sentry-WMS §1.3, OCA §1.3).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

// GetOrCreatePickingTask finds or creates a picking task for a given delivery order.
// Lines are assigned shelf_order based on warehouse location code ASC (snake routing / S-shape).
func (r *WMSRepo) GetOrCreatePickingTask(ctx context.Context, tenantID uuid.UUID, doID uuid.UUID) (*domain.PickingTaskDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreatePickingTask: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreatePickingTask: set tenant: %w", err)
	}

	detail, err := r.fetchPickingTaskByDOTx(ctx, tx, tenantID, doID)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return detail, nil
	}
	if !errors.Is(err, domain.ErrPickingTaskNotFound) {
		return nil, err
	}

	// Task does not exist yet. Check delivery order.
	var do domain.DeliveryOrder
	var doNumber string
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT id, do_number, status, warehouse_id
		FROM delivery_orders
		WHERE id = $1 AND tenant_id = $2`, doID, tenantID).Scan(&do.ID, &doNumber, &status, &do.WarehouseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeliveryOrderNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetOrCreatePickingTask: get DO: %w", err)
	}
	do.DONumber = doNumber
	do.Status = domain.DeliveryOrderStatus(status)

	taskID := uuid.New()
	taskNumber := fmt.Sprintf("PICK-%s", doNumber)
	now := time.Now().UTC()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO picking_tasks (id, tenant_id, delivery_order_id, task_number, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'PENDING', $5, $5)`,
		taskID, tenantID, doID, taskNumber, now)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreatePickingTask: insert task: %w", err)
	}

	// Fetch items from DO ordered by location code ASC
	rows, err := tx.QueryContext(ctx, `
		SELECT doi.product_id, COALESCE(doi.batch_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       doi.location_id, doi.quantity, wl.code, COALESCE(doi.is_free_item, false)
		FROM delivery_order_items doi
		JOIN warehouse_locations wl ON wl.id = doi.location_id AND wl.tenant_id = doi.tenant_id
		WHERE doi.delivery_order_id = $1 AND doi.tenant_id = $2
		ORDER BY wl.code ASC, doi.created_at ASC`, doID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreatePickingTask: query do items: %w", err)
	}
	defer rows.Close()

	type itemDraft struct {
		prodID, batchID, locID uuid.UUID
		qty                    decimal.Decimal
		locCode                string
		isFreeItem             bool
	}
	var drafts []itemDraft
	for rows.Next() {
		var d itemDraft
		if err := rows.Scan(&d.prodID, &d.batchID, &d.locID, &d.qty, &d.locCode, &d.isFreeItem); err != nil {
			return nil, fmt.Errorf("WMSRepo.GetOrCreatePickingTask: scan do item: %w", err)
		}
		drafts = append(drafts, d)
	}
	_ = rows.Close()

	for i, d := range drafts {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO picking_task_items
				(id, tenant_id, task_id, product_id, batch_id, source_location_id, requested_qty, picked_qty, status, shelf_order, is_free_item, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 0, 'PENDING', $8, $9, $10)`,
			uuid.New(), tenantID, taskID, d.prodID, d.batchID, d.locID, d.qty, i+1, d.isFreeItem, now)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.GetOrCreatePickingTask: insert task item %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetPickingTaskByDO(ctx, tenantID, doID)
}

func (r *WMSRepo) GetPickingTaskByDO(ctx context.Context, tenantID uuid.UUID, doID uuid.UUID) (*domain.PickingTaskDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetPickingTaskByDO: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetPickingTaskByDO: set tenant: %w", err)
	}

	detail, err := r.fetchPickingTaskByDOTx(ctx, tx, tenantID, doID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return detail, nil
}

func (r *WMSRepo) fetchPickingTaskByDOTx(ctx context.Context, tx *sql.Tx, tenantID, doID uuid.UUID) (*domain.PickingTaskDetail, error) {
	var task domain.PickingTask
	var pickerID sql.NullString
	var pickerName sql.NullString
	var startedAt, completedAt sql.NullTime
	var notes sql.NullString

	err := tx.QueryRowContext(ctx, `
		SELECT pt.id, pt.tenant_id, pt.delivery_order_id, pt.task_number, pt.status,
		       pt.picker_id, COALESCE(u.full_name, u.email, ''), pt.started_at, pt.completed_at, pt.notes,
		       pt.created_at, pt.updated_at
		FROM picking_tasks pt
		LEFT JOIN users u ON u.id = pt.picker_id AND u.tenant_id = pt.tenant_id
		WHERE pt.delivery_order_id = $1 AND pt.tenant_id = $2`, doID, tenantID).Scan(
		&task.ID, &task.TenantID, &task.DeliveryOrderID, &task.TaskNumber, &task.Status,
		&pickerID, &pickerName, &startedAt, &completedAt, &notes,
		&task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPickingTaskNotFound
		}
		return nil, fmt.Errorf("fetchPickingTaskByDOTx: task: %w", err)
	}
	task.PickerID = nullUUIDToPtr(pickerID)
	task.PickerName = nullStringToPtr(pickerName)
	task.StartedAt = nullTimeToPtr(startedAt)
	task.CompletedAt = nullTimeToPtr(completedAt)
	task.Notes = nullStringToPtr(notes)

	// Fetch items
	rows, err := tx.QueryContext(ctx, `
		SELECT pti.id, pti.tenant_id, pti.task_id, pti.product_id, COALESCE(p.name, ''), COALESCE(p.sku, ''),
		       pti.batch_id, COALESCE(sb.batch_number, ''), sb.expiry_date,
		       pti.source_location_id, COALESCE(wl.code, ''),
		       pti.requested_qty, pti.picked_qty, pti.damaged_qty, pti.status, pti.shelf_order,
		       COALESCE(pti.is_free_item, false), pti.created_at
		FROM picking_task_items pti
		LEFT JOIN products p ON p.id = pti.product_id AND p.tenant_id = pti.tenant_id
		LEFT JOIN stock_batches sb ON sb.id = pti.batch_id AND sb.tenant_id = pti.tenant_id
		LEFT JOIN warehouse_locations wl ON wl.id = pti.source_location_id AND wl.tenant_id = pti.tenant_id
		WHERE pti.task_id = $1 AND pti.tenant_id = $2
		ORDER BY pti.shelf_order ASC, wl.code ASC`, task.ID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("fetchPickingTaskByDOTx: items: %w", err)
	}
	defer rows.Close()

	var items []domain.PickingTaskItem
	for rows.Next() {
		var it domain.PickingTaskItem
		var pName, pSKU, bNum, lCode sql.NullString
		var expDate sql.NullTime
		if err := rows.Scan(
			&it.ID, &it.TenantID, &it.TaskID, &it.ProductID, &pName, &pSKU,
			&it.BatchID, &bNum, &expDate,
			&it.SourceLocationID, &lCode,
			&it.RequestedQty, &it.PickedQty, &it.DamagedQty, &it.Status, &it.ShelfOrder,
			&it.IsFreeItem, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("fetchPickingTaskByDOTx: scan item: %w", err)
		}
		it.ProductName = nullStringToPtr(pName)
		it.ProductSKU = nullStringToPtr(pSKU)
		it.BatchNumber = nullStringToPtr(bNum)
		it.ExpiryDate = nullTimeToPtr(expDate)
		it.LocationCode = nullStringToPtr(lCode)
		items = append(items, it)
	}
	_ = rows.Close()

	do, _, err := r.GetDeliveryOrderByID(ctx, tenantID, doID)
	if err != nil && !errors.Is(err, domain.ErrDeliveryOrderNotFound) {
		return nil, fmt.Errorf("fetchPickingTaskByDOTx: get DO: %w", err)
	}
	var resDO domain.DeliveryOrder
	if do != nil {
		resDO = *do
	}

	return &domain.PickingTaskDetail{
		Task:          task,
		DeliveryOrder: resDO,
		Items:         items,
	}, nil
}

func (r *WMSRepo) UpdatePickingTaskStatus(ctx context.Context, tenantID, taskID uuid.UUID, status domain.PickingTaskStatus, pickerID *uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdatePickingTaskStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdatePickingTaskStatus: set tenant: %w", err)
	}

	var startedAt, completedAt *time.Time
	now := time.Now().UTC()
	if status == domain.PickingTaskStatusInProgress {
		startedAt = &now
	} else if status == domain.PickingTaskStatusCompleted {
		completedAt = &now
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE picking_tasks
		SET status = $3,
		    picker_id = COALESCE($4, picker_id),
		    started_at = COALESCE($5, started_at),
		    completed_at = COALESCE($6, completed_at),
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`,
		taskID, tenantID, status, ptrToNullUUID(pickerID), ptrToNullTime(startedAt), ptrToNullTime(completedAt))
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdatePickingTaskStatus: exec: %w", err)
	}

	return tx.Commit()
}

func (r *WMSRepo) RecordPickingItemProgress(ctx context.Context, tenantID, taskItemID uuid.UUID, pickedQty decimal.Decimal) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.RecordPickingItemProgress: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.RecordPickingItemProgress: set tenant: %w", err)
	}

	var reqQty decimal.Decimal
	err = tx.QueryRowContext(ctx, `
		SELECT requested_qty FROM picking_task_items WHERE id = $1 AND tenant_id = $2`,
		taskItemID, tenantID).Scan(&reqQty)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrPickingItemNotFound
		}
		return fmt.Errorf("WMSRepo.RecordPickingItemProgress: find item: %w", err)
	}

	newStatus := domain.PickingItemStatusPicked
	if pickedQty.LessThan(reqQty) {
		newStatus = domain.PickingItemStatusShortage
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE picking_task_items
		SET picked_qty = $3, status = $4
		WHERE id = $1 AND tenant_id = $2`,
		taskItemID, tenantID, pickedQty, newStatus)
	if err != nil {
		return fmt.Errorf("WMSRepo.RecordPickingItemProgress: update item: %w", err)
	}

	return tx.Commit()
}

// UpdateDOPackScan increments packed_qty on a delivery_order_item.
func (r *WMSRepo) UpdateDOPackScan(ctx context.Context, tenantID, doID, itemID uuid.UUID, addPackedQty decimal.Decimal) (*domain.DeliveryOrderItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateDOPackScan: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateDOPackScan: set tenant: %w", err)
	}

	var it domain.DeliveryOrderItem
	var bID, pName, pSKU, lCode sql.NullString
	var bNum sql.NullString
	var expDate sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT doi.id, doi.tenant_id, doi.delivery_order_id, doi.product_id, doi.quantity, doi.location_id,
		       doi.batch_id, doi.is_free_item, doi.packed_qty, doi.created_at,
		       COALESCE(p.name, ''), COALESCE(p.sku, ''), COALESCE(wl.code, ''),
		       COALESCE(sb.batch_number, ''), sb.expiry_date
		FROM delivery_order_items doi
		LEFT JOIN products p ON p.id = doi.product_id AND p.tenant_id = doi.tenant_id
		LEFT JOIN warehouse_locations wl ON wl.id = doi.location_id AND wl.tenant_id = doi.tenant_id
		LEFT JOIN stock_batches sb ON sb.id = doi.batch_id AND sb.tenant_id = doi.tenant_id
		WHERE doi.id = $1 AND doi.delivery_order_id = $2 AND doi.tenant_id = $3
		FOR UPDATE OF doi`, itemID, doID, tenantID).Scan(
		&it.ID, &it.TenantID, &it.DeliveryOrderID, &it.ProductID, &it.Quantity, &it.LocationID,
		&bID, &it.IsFreeItem, &it.PackedQty, &it.CreatedAt,
		&pName, &pSKU, &lCode, &bNum, &expDate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeliveryOrderNotFound
		}
		return nil, fmt.Errorf("WMSRepo.UpdateDOPackScan: select item: %w", err)
	}
	it.BatchID = nullUUIDToPtr(bID)
	it.ProductName = nullStringToPtr(pName)
	it.ProductSKU = nullStringToPtr(pSKU)
	it.LocationCode = nullStringToPtr(lCode)
	it.BatchNumber = nullStringToPtr(bNum)
	it.ExpiryDate = nullTimeToPtr(expDate)

	nextPacked := it.PackedQty.Add(addPackedQty)
	if nextPacked.GreaterThan(it.Quantity) {
		return nil, domain.ErrPackQtyExceeded
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE delivery_order_items
		SET packed_qty = $3
		WHERE id = $1 AND tenant_id = $2`, itemID, tenantID, nextPacked)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateDOPackScan: update item: %w", err)
	}

	it.PackedQty = nextPacked
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &it, nil
}

// CompleteDOPacking transitions a DO to PACKED, records dimensions/weight and actor.
func (r *WMSRepo) CompleteDOPacking(ctx context.Context, tenantID, doID, userID uuid.UUID, req domain.PackCompleteRequest) (*domain.DeliveryOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CompleteDOPacking: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.CompleteDOPacking: set tenant: %w", err)
	}

	// Verify all items are 100% packed
	var uncompletedCount int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM delivery_order_items
		WHERE delivery_order_id = $1 AND tenant_id = $2 AND packed_qty < quantity`,
		doID, tenantID).Scan(&uncompletedCount)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CompleteDOPacking: check uncompleted items: %w", err)
	}
	if uncompletedCount > 0 {
		return nil, domain.ErrPackStationIncomplete
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE delivery_orders
		SET status = 'PACKED',
		    packed_by = $3,
		    package_weight_kg = COALESCE($4, package_weight_kg),
		    package_length_cm = COALESCE($5, package_length_cm),
		    package_width_cm  = COALESCE($6, package_width_cm),
		    package_height_cm = COALESCE($7, package_height_cm),
		    packaging_type    = COALESCE($8, packaging_type),
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`,
		doID, tenantID, userID,
		decPtr(req.PackageWeightKg), decPtr(req.PackageLengthCm),
		decPtr(req.PackageWidthCm), decPtr(req.PackageHeightCm),
		req.PackagingType)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CompleteDOPacking: update DO: %w", err)
	}

	// Also mark picking_task as COMPLETED
	_, _ = tx.ExecContext(ctx, `
		UPDATE picking_tasks
		SET status = 'COMPLETED', completed_at = NOW(), updated_at = NOW()
		WHERE delivery_order_id = $1 AND tenant_id = $2 AND status != 'COMPLETED'`,
		doID, tenantID)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	do, _, err := r.GetDeliveryOrderByID(ctx, tenantID, doID)
	return do, err
}
