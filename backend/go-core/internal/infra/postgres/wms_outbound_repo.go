package postgres

// Sprint 3 Outbound Postgres Repository: Picking Tasks, Pack Station Scan,
// Shelf Sorting & Outbound Lifecycle (ADR-014 Invariant 1, Sentry-WMS §1.3, OCA §1.3).

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

	auditDetails, _ := json.Marshal(map[string]any{
		"package_weight_kg": req.PackageWeightKg,
		"package_length_cm": req.PackageLengthCm,
		"package_width_cm":  req.PackageWidthCm,
		"package_height_cm": req.PackageHeightCm,
		"packaging_type":    req.PackagingType,
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "delivery_order", doID, "pack_completed", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.CompleteDOPacking: audit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	do, _, err := r.GetDeliveryOrderByID(ctx, tenantID, doID)
	return do, err
}

func (r *WMSRepo) CreatePickWave(ctx context.Context, tenantID uuid.UUID, createdBy *uuid.UUID, req domain.CreatePickWaveRequest) (*domain.PickWave, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreatePickWave: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreatePickWave: set tenant: %w", err)
	}

	orderType := strings.TrimSpace(req.OrderType)
	if orderType == "" {
		orderType = "DIRECT_DO"
	}

	doIDs := req.DeliveryOrderIDs
	if len(doIDs) == 0 {
		query := `
			SELECT d.id
			FROM delivery_orders d
			WHERE d.tenant_id = $1
			  AND d.warehouse_id = $2
			  AND d.order_type = $3
			  AND ($4::varchar IS NULL OR d.expedition_name = $4)
			  AND d.status IN ('DRAFT', 'CONFIRMED')
			  AND NOT EXISTS (
			      SELECT 1 FROM picking_tasks pt
			      WHERE pt.delivery_order_id = d.id AND pt.tenant_id = d.tenant_id AND pt.wave_id IS NOT NULL
			  )
			ORDER BY d.created_at ASC
			LIMIT 50`
		rows, err := tx.QueryContext(ctx, query, tenantID, req.WarehouseID, orderType, req.ExpeditionName)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CreatePickWave: find eligible DOs: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			doIDs = append(doIDs, id)
		}
		rows.Close()
	}

	if len(doIDs) == 0 {
		return nil, domain.ErrNoOrdersForWave
	}

	waveID := uuid.New()
	waveNum := fmt.Sprintf("WAVE-%s-%04d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO pick_waves
			(id, tenant_id, warehouse_id, wave_number, order_type, expedition_name, route_zone, status, picker_id, created_by, notes, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, 'OPEN', $8, $9, $10, NOW(), NOW())`,
		waveID, tenantID, req.WarehouseID, waveNum, orderType, req.ExpeditionName, req.RouteZone,
		ptrToNullUUID(req.PickerID), ptrToNullUUID(createdBy), req.Notes)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreatePickWave: insert wave: %w", err)
	}

	for _, dID := range doIDs {
		var ptID uuid.UUID
		err := tx.QueryRowContext(ctx, `SELECT id FROM picking_tasks WHERE delivery_order_id = $1 AND tenant_id = $2`, dID, tenantID).Scan(&ptID)
		if errors.Is(err, sql.ErrNoRows) {
			var doNum string
			if sErr := tx.QueryRowContext(ctx, `SELECT do_number FROM delivery_orders WHERE id = $1 AND tenant_id = $2`, dID, tenantID).Scan(&doNum); sErr == nil {
				ptID = uuid.New()
				taskNum := fmt.Sprintf("PICK-%s", doNum)
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO picking_tasks (id, tenant_id, delivery_order_id, wave_id, task_number, status, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, 'PENDING', NOW(), NOW())`,
					ptID, tenantID, dID, waveID, taskNum)
			}
		} else if err == nil {
			_, _ = tx.ExecContext(ctx, `
				UPDATE picking_tasks SET wave_id = $1, updated_at = NOW()
				WHERE id = $2 AND tenant_id = $3`, waveID, ptID, tenantID)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.getPickWaveSummary(ctx, tenantID, waveID)
}

func (r *WMSRepo) getPickWaveSummary(ctx context.Context, tenantID, waveID uuid.UUID) (*domain.PickWave, error) {
	var w domain.PickWave
	var whName, expName, rZone, pName, cName, notes sql.NullString
	var pID, cID sql.NullString
	var startedAt, completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT pw.id, pw.tenant_id, pw.warehouse_id, COALESCE(wh.name, ''),
		       pw.wave_number, pw.order_type, pw.expedition_name, pw.route_zone,
		       pw.status, pw.picker_id, COALESCE(u_pk.full_name, u_pk.email, ''),
		       pw.created_by, COALESCE(u_cr.full_name, u_cr.email, ''),
		       pw.started_at, pw.completed_at, pw.notes,
		       (SELECT COUNT(DISTINCT pt.id) FROM picking_tasks pt WHERE pt.wave_id = pw.id AND pt.tenant_id = pw.tenant_id),
		       (SELECT COUNT(pti.id) FROM picking_task_items pti JOIN picking_tasks pt ON pt.id = pti.task_id WHERE pt.wave_id = pw.id AND pt.tenant_id = pw.tenant_id),
		       pw.created_at, pw.updated_at
		FROM pick_waves pw
		LEFT JOIN warehouses wh ON wh.id = pw.warehouse_id AND wh.tenant_id = pw.tenant_id
		LEFT JOIN users u_pk ON u_pk.id = pw.picker_id AND u_pk.tenant_id = pw.tenant_id
		LEFT JOIN users u_cr ON u_cr.id = pw.created_by AND u_cr.tenant_id = pw.tenant_id
		WHERE pw.id = $1 AND pw.tenant_id = $2`, waveID, tenantID).Scan(
		&w.ID, &w.TenantID, &w.WarehouseID, &whName,
		&w.WaveNumber, &w.OrderType, &expName, &rZone,
		&w.Status, &pID, &pName,
		&cID, &cName,
		&startedAt, &completedAt, &notes,
		&w.TotalOrders, &w.TotalLines,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPickWaveNotFound
		}
		return nil, fmt.Errorf("WMSRepo.getPickWaveSummary: %w", err)
	}

	w.WarehouseName = nullStringToPtr(whName)
	w.ExpeditionName = nullStringToPtr(expName)
	w.RouteZone = nullStringToPtr(rZone)
	w.PickerID = nullUUIDToPtr(pID)
	w.PickerName = nullStringToPtr(pName)
	w.CreatedBy = nullUUIDToPtr(cID)
	w.CreatedByName = nullStringToPtr(cName)
	w.StartedAt = nullTimeToPtr(startedAt)
	w.CompletedAt = nullTimeToPtr(completedAt)
	w.Notes = nullStringToPtr(notes)

	return &w, nil
}

func (r *WMSRepo) ListPickWaves(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, orderType, expeditionName *string, status *domain.PickWaveStatus) ([]domain.PickWave, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListPickWaves: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListPickWaves: set tenant: %w", err)
	}

	query := `
		SELECT pw.id, pw.tenant_id, pw.warehouse_id, COALESCE(wh.name, ''),
		       pw.wave_number, pw.order_type, pw.expedition_name, pw.route_zone,
		       pw.status, pw.picker_id, COALESCE(u_pk.full_name, u_pk.email, ''),
		       pw.created_by, COALESCE(u_cr.full_name, u_cr.email, ''),
		       pw.started_at, pw.completed_at, pw.notes,
		       (SELECT COUNT(DISTINCT pt.id) FROM picking_tasks pt WHERE pt.wave_id = pw.id AND pt.tenant_id = pw.tenant_id),
		       (SELECT COUNT(pti.id) FROM picking_task_items pti JOIN picking_tasks pt ON pt.id = pti.task_id WHERE pt.wave_id = pw.id AND pt.tenant_id = pw.tenant_id),
		       pw.created_at, pw.updated_at
		FROM pick_waves pw
		LEFT JOIN warehouses wh ON wh.id = pw.warehouse_id AND wh.tenant_id = pw.tenant_id
		LEFT JOIN users u_pk ON u_pk.id = pw.picker_id AND u_pk.tenant_id = pw.tenant_id
		LEFT JOIN users u_cr ON u_cr.id = pw.created_by AND u_cr.tenant_id = pw.tenant_id
		WHERE pw.tenant_id = $1
		  AND ($2::uuid IS NULL OR pw.warehouse_id = $2)
		  AND ($3::varchar IS NULL OR pw.order_type = $3)
		  AND ($4::varchar IS NULL OR pw.expedition_name = $4)
		  AND ($5::varchar IS NULL OR pw.status = $5)
		ORDER BY pw.created_at DESC`

	var statStr *string
	if status != nil {
		s := string(*status)
		statStr = &s
	}

	rows, err := tx.QueryContext(ctx, query, tenantID, ptrToNullUUID(warehouseID), orderType, expeditionName, statStr)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListPickWaves: query: %w", err)
	}
	defer rows.Close()

	var result []domain.PickWave
	for rows.Next() {
		var w domain.PickWave
		var whName, expName, rZone, pName, cName, notes sql.NullString
		var pID, cID sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&w.ID, &w.TenantID, &w.WarehouseID, &whName,
			&w.WaveNumber, &w.OrderType, &expName, &rZone,
			&w.Status, &pID, &pName,
			&cID, &cName,
			&startedAt, &completedAt, &notes,
			&w.TotalOrders, &w.TotalLines,
			&w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListPickWaves: scan: %w", err)
		}

		w.WarehouseName = nullStringToPtr(whName)
		w.ExpeditionName = nullStringToPtr(expName)
		w.RouteZone = nullStringToPtr(rZone)
		w.PickerID = nullUUIDToPtr(pID)
		w.PickerName = nullStringToPtr(pName)
		w.CreatedBy = nullUUIDToPtr(cID)
		w.CreatedByName = nullStringToPtr(cName)
		w.StartedAt = nullTimeToPtr(startedAt)
		w.CompletedAt = nullTimeToPtr(completedAt)
		w.Notes = nullStringToPtr(notes)

		result = append(result, w)
	}

	return result, tx.Commit()
}

func (r *WMSRepo) GetPickWaveByID(ctx context.Context, tenantID, waveID uuid.UUID) (*domain.PickWaveDetail, error) {
	wave, err := r.getPickWaveSummary(ctx, tenantID, waveID)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT pt.delivery_order_id
		FROM picking_tasks pt
		WHERE pt.wave_id = $1 AND pt.tenant_id = $2
		ORDER BY pt.task_number ASC`, waveID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetPickWaveByID: find tasks: %w", err)
	}
	defer rows.Close()

	var doIDs []uuid.UUID
	for rows.Next() {
		var dID uuid.UUID
		if err := rows.Scan(&dID); err != nil {
			return nil, err
		}
		doIDs = append(doIDs, dID)
	}
	rows.Close()

	var tasks []domain.PickingTaskDetail
	for _, dID := range doIDs {
		detail, err := r.GetPickingTaskByDO(ctx, tenantID, dID)
		if err == nil && detail != nil {
			tasks = append(tasks, *detail)
		}
	}

	return &domain.PickWaveDetail{
		Wave:         *wave,
		PickingTasks: tasks,
	}, nil
}

func (r *WMSRepo) ReleasePickWave(ctx context.Context, tenantID, waveID uuid.UUID, pickerID *uuid.UUID) (*domain.PickWave, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleasePickWave: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleasePickWave: set tenant: %w", err)
	}

	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
		UPDATE pick_waves
		SET status = 'RELEASED',
		    picker_id = COALESCE($3, picker_id),
		    started_at = COALESCE(started_at, $4),
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`,
		waveID, tenantID, ptrToNullUUID(pickerID), now)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleasePickWave: update wave: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE picking_tasks
		SET status = 'IN_PROGRESS',
		    picker_id = COALESCE($3, picker_id),
		    started_at = COALESCE(started_at, $4),
		    updated_at = NOW()
		WHERE wave_id = $1 AND tenant_id = $2 AND status = 'PENDING'`,
		waveID, tenantID, ptrToNullUUID(pickerID), now)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleasePickWave: update tasks: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.getPickWaveSummary(ctx, tenantID, waveID)
}
