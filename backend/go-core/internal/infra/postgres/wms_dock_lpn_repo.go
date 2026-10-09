package postgres

// Sprint 5 WMS Postgres Repository: Inbound Docks, Appointments, Anti-Collision Dock Guard,
// Pallet LPN Containerization & Atomic Forklift Putaway (ADR-014 Invariant 1, OCA/wms dock scheduling).

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

var _ domain.WMSDockLPNRepository = (*WMSRepo)(nil)

// ============================================================================
// 1. INBOUND DOCKS
// ============================================================================

// CreateDock registers a new dock bay in a warehouse.
func (r *WMSRepo) CreateDock(ctx context.Context, tenantID uuid.UUID, req domain.CreateDockRequest) (*domain.InboundDock, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateDock: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateDock: set tenant: %w", err)
	}

	// 1. Verify warehouse belongs to tenant
	var whName string
	err = tx.QueryRowContext(ctx, `SELECT name FROM warehouses WHERE id = $1 AND tenant_id = $2`, req.WarehouseID, tenantID).Scan(&whName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("WMSRepo.CreateDock: check warehouse: %w", err)
	}

	// 2. Generate dock code if blank
	dockCode := strings.TrimSpace(req.DockCode)
	if dockCode == "" {
		var count int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM inbound_docks WHERE tenant_id = $1 AND warehouse_id = $2`, tenantID, req.WarehouseID).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CreateDock: count docks: %w", err)
		}
		dockCode = fmt.Sprintf("DOCK-%02d", count+1)
		for i := 0; i < 20; i++ {
			var exists bool
			err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM inbound_docks WHERE tenant_id = $1 AND warehouse_id = $2 AND dock_code = $3)`, tenantID, req.WarehouseID, dockCode).Scan(&exists)
			if err != nil {
				return nil, fmt.Errorf("WMSRepo.CreateDock: check dock code exists: %w", err)
			}
			if !exists {
				break
			}
			dockCode = fmt.Sprintf("DOCK-%02d", count+2+i)
		}
	}

	dockType := req.DockType
	if dockType == "" {
		dockType = domain.DockTypeInbound
	}
	maxTonnage := req.MaxTonnage
	if maxTonnage.IsZero() || maxTonnage.IsNegative() {
		maxTonnage = decimal.NewFromFloat(10.00)
	}

	dockID := uuid.New()
	now := time.Now().UTC()
	status := domain.DockStatusAvailable

	insertSQL := `
	INSERT INTO inbound_docks (id, tenant_id, warehouse_id, dock_code, dock_name, dock_type, max_tonnage, status, notes, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err = tx.ExecContext(ctx, insertSQL,
		dockID, tenantID, req.WarehouseID, dockCode, strings.TrimSpace(req.DockName),
		dockType, maxTonnage, status, ptrToNullString(req.Notes), now, now)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("WMSRepo.CreateDock: insert dock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateDock: commit: %w", err)
	}

	return &domain.InboundDock{
		ID:            dockID,
		TenantID:      tenantID,
		WarehouseID:   req.WarehouseID,
		WarehouseName: &whName,
		DockCode:      dockCode,
		DockName:      strings.TrimSpace(req.DockName),
		DockType:      dockType,
		MaxTonnage:    maxTonnage,
		Status:        status,
		Notes:         req.Notes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// GetDockByID retrieves a dock with warehouse name by ID.
func (r *WMSRepo) GetDockByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.InboundDock, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetDockByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetDockByID: set tenant: %w", err)
	}

	query := `
	SELECT d.id, d.tenant_id, d.warehouse_id, w.name, d.dock_code, d.dock_name, d.dock_type,
	       d.max_tonnage, d.status, d.notes, d.created_at, d.updated_at
	FROM inbound_docks d
	JOIN warehouses w ON w.id = d.warehouse_id AND w.tenant_id = $2
	WHERE d.id = $1 AND d.tenant_id = $2`

	var d domain.InboundDock
	var notes sql.NullString
	var whName sql.NullString
	err = tx.QueryRowContext(ctx, query, id, tenantID).Scan(
		&d.ID, &d.TenantID, &d.WarehouseID, &whName, &d.DockCode, &d.DockName, &d.DockType,
		&d.MaxTonnage, &d.Status, &notes, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDockNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetDockByID: query: %w", err)
	}
	d.WarehouseName = nullStringToPtr(whName)
	d.Notes = nullStringToPtr(notes)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetDockByID: commit: %w", err)
	}

	return &d, nil
}

// ListDocks selects docks filtered by warehouse and optional status.
func (r *WMSRepo) ListDocks(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.DockStatus) ([]domain.InboundDock, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDocks: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDocks: set tenant: %w", err)
	}

	query := `
	SELECT d.id, d.tenant_id, d.warehouse_id, w.name, d.dock_code, d.dock_name, d.dock_type,
	       d.max_tonnage, d.status, d.notes, d.created_at, d.updated_at
	FROM inbound_docks d
	JOIN warehouses w ON w.id = d.warehouse_id AND w.tenant_id = $1
	WHERE d.tenant_id = $1`

	args := []any{tenantID}
	idx := 2

	if warehouseID != nil && *warehouseID != uuid.Nil {
		query += fmt.Sprintf(" AND d.warehouse_id = $%d", idx)
		args = append(args, *warehouseID)
		idx++
	}
	if status != nil && *status != "" {
		query += fmt.Sprintf(" AND d.status = $%d", idx)
		args = append(args, *status)
		idx++
	}

	query += " ORDER BY d.dock_code ASC"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDocks: query: %w", err)
	}
	defer rows.Close()

	docks := make([]domain.InboundDock, 0)
	for rows.Next() {
		var d domain.InboundDock
		var notes sql.NullString
		var whName sql.NullString
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.WarehouseID, &whName, &d.DockCode, &d.DockName, &d.DockType,
			&d.MaxTonnage, &d.Status, &notes, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListDocks: scan: %w", err)
		}
		d.WarehouseName = nullStringToPtr(whName)
		d.Notes = nullStringToPtr(notes)
		docks = append(docks, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDocks: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDocks: commit: %w", err)
	}

	return docks, nil
}

// UpdateDockStatus updates status and notes of a dock.
func (r *WMSRepo) UpdateDockStatus(ctx context.Context, tenantID, id uuid.UUID, req domain.UpdateDockStatusRequest) (*domain.InboundDock, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateDockStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateDockStatus: set tenant: %w", err)
	}

	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM inbound_docks WHERE id = $1 AND tenant_id = $2 FOR UPDATE)`, id, tenantID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateDockStatus: lock dock: %w", err)
	}
	if !exists {
		return nil, domain.ErrDockNotFound
	}

	if req.Notes != nil {
		_, err = tx.ExecContext(ctx, `UPDATE inbound_docks SET status = $1, notes = $2, updated_at = NOW() WHERE id = $3 AND tenant_id = $4`, req.Status, *req.Notes, id, tenantID)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE inbound_docks SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`, req.Status, id, tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateDockStatus: update dock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateDockStatus: commit: %w", err)
	}

	return r.GetDockByID(ctx, tenantID, id)
}

// ============================================================================
// 2. DOCK APPOINTMENTS & INBOUND QUEUE
// ============================================================================

// CreateAppointment schedules an inbound truck arrival at the warehouse.
func (r *WMSRepo) CreateAppointment(ctx context.Context, tenantID, userID uuid.UUID, req domain.CreateAppointmentRequest) (*domain.DockAppointment, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateAppointment: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateAppointment: set tenant: %w", err)
	}

	// 1. Verify warehouse belongs to tenant
	var whName string
	err = tx.QueryRowContext(ctx, `SELECT name FROM warehouses WHERE id = $1 AND tenant_id = $2`, req.WarehouseID, tenantID).Scan(&whName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("WMSRepo.CreateAppointment: check warehouse: %w", err)
	}

	// 2. If dock_id provided, verify dock belongs to warehouse and is not occupied/maintenance
	if req.DockID != nil && *req.DockID != uuid.Nil {
		var dockWhID uuid.UUID
		var dockStatus string
		err = tx.QueryRowContext(ctx, `SELECT warehouse_id, status FROM inbound_docks WHERE id = $1 AND tenant_id = $2 FOR UPDATE`, *req.DockID, tenantID).Scan(&dockWhID, &dockStatus)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.ErrDockNotFound
			}
			return nil, fmt.Errorf("WMSRepo.CreateAppointment: check dock: %w", err)
		}
		if dockWhID != req.WarehouseID {
			return nil, domain.ErrInvalidInput
		}
		if domain.DockStatus(dockStatus) == domain.DockStatusMaintenance || domain.DockStatus(dockStatus) == domain.DockStatusOccupied {
			return nil, domain.ErrDockOccupied
		}
		var hasCollision bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM dock_appointments
				WHERE tenant_id = $1 AND dock_id = $2
				  AND status IN ('ARRIVED', 'UNLOADING')
			)`, tenantID, *req.DockID).Scan(&hasCollision)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CreateAppointment: check dock collision: %w", err)
		}
		if hasCollision {
			return nil, domain.ErrDockOccupied
		}
	}

	// 3. Generate APP-YYYYMMDD-XXXX sequence
	todayStr := time.Now().UTC().Format("20060102")
	var appNum string
	for i := 0; i < 20; i++ {
		var count int
		err := tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM dock_appointments
			WHERE tenant_id = $1 AND appointment_number LIKE $2`,
			tenantID, "APP-"+todayStr+"-%").Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CreateAppointment: count appointments: %w", err)
		}
		candidate := fmt.Sprintf("APP-%s-%04d", todayStr, count+1+i)
		var exists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM dock_appointments WHERE tenant_id = $1 AND appointment_number = $2)`,
			tenantID, candidate).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CreateAppointment: check appointment exists: %w", err)
		}
		if !exists {
			appNum = candidate
			break
		}
	}
	if appNum == "" {
		appNum = fmt.Sprintf("APP-%s-%04d", todayStr, time.Now().UnixNano()%10000)
	}

	appID := uuid.New()
	now := time.Now().UTC()
	status := domain.AppointmentStatusScheduled

	insertSQL := `
	INSERT INTO dock_appointments (
		id, tenant_id, warehouse_id, dock_id, appointment_number,
		vendor_name, vehicle_plate, driver_name, driver_phone, po_reference,
		estimated_arrival, status, notes, created_by, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5,
		$6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16
	)`
	_, err = tx.ExecContext(ctx, insertSQL,
		appID, tenantID, req.WarehouseID, ptrToNullUUID(req.DockID), appNum,
		strings.TrimSpace(req.VendorName), strings.TrimSpace(req.VehiclePlate), strings.TrimSpace(req.DriverName),
		ptrToNullString(req.DriverPhone), ptrToNullString(req.POReference),
		req.EstimatedArrival, status, ptrToNullString(req.Notes),
		ptrToNullUUID(&userID), now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("WMSRepo.CreateAppointment: insert appointment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateAppointment: commit: %w", err)
	}

	return r.GetAppointmentByID(ctx, tenantID, appID)
}

// GetAppointmentByID selects appointment with dock and warehouse details.
func (r *WMSRepo) GetAppointmentByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.DockAppointment, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetAppointmentByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetAppointmentByID: set tenant: %w", err)
	}

	query := `
	SELECT a.id, a.tenant_id, a.warehouse_id, w.name, a.dock_id, d.dock_code, d.dock_name,
	       a.appointment_number, a.vendor_name, a.vehicle_plate, a.driver_name, a.driver_phone,
	       a.po_reference, a.estimated_arrival, a.actual_arrival, a.start_unloading_at, a.completed_at,
	       a.status, a.notes, a.created_by, COALESCE(u.full_name, u.email), a.created_at, a.updated_at
	FROM dock_appointments a
	JOIN warehouses w ON w.id = a.warehouse_id AND w.tenant_id = $2
	LEFT JOIN inbound_docks d ON d.id = a.dock_id AND d.tenant_id = $2
	LEFT JOIN users u ON u.id = a.created_by AND u.tenant_id = $2
	WHERE a.id = $1 AND a.tenant_id = $2`

	var a domain.DockAppointment
	var whName sql.NullString
	var dockID sql.NullString
	var dockCode sql.NullString
	var dockName sql.NullString
	var driverPhone sql.NullString
	var poRef sql.NullString
	var actualArrival sql.NullTime
	var startUnloading sql.NullTime
	var completedAt sql.NullTime
	var notes sql.NullString
	var createdBy sql.NullString
	var createdByName sql.NullString

	err = tx.QueryRowContext(ctx, query, id, tenantID).Scan(
		&a.ID, &a.TenantID, &a.WarehouseID, &whName, &dockID, &dockCode, &dockName,
		&a.AppointmentNumber, &a.VendorName, &a.VehiclePlate, &a.DriverName, &driverPhone,
		&poRef, &a.EstimatedArrival, &actualArrival, &startUnloading, &completedAt,
		&a.Status, &notes, &createdBy, &createdByName, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetAppointmentByID: query: %w", err)
	}

	a.WarehouseName = nullStringToPtr(whName)
	a.DockID = nullUUIDToPtr(dockID)
	a.DockCode = nullStringToPtr(dockCode)
	a.DockName = nullStringToPtr(dockName)
	a.DriverPhone = nullStringToPtr(driverPhone)
	a.POReference = nullStringToPtr(poRef)
	a.ActualArrival = nullTimeToPtr(actualArrival)
	a.StartUnloadingAt = nullTimeToPtr(startUnloading)
	a.CompletedAt = nullTimeToPtr(completedAt)
	a.Notes = nullStringToPtr(notes)
	a.CreatedBy = nullUUIDToPtr(createdBy)
	a.CreatedByName = nullStringToPtr(createdByName)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetAppointmentByID: commit: %w", err)
	}

	return &a, nil
}

// ListAppointments selects appointments for warehouse filtered by optional status.
func (r *WMSRepo) ListAppointments(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.AppointmentStatus) ([]domain.DockAppointment, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAppointments: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAppointments: set tenant: %w", err)
	}

	query := `
	SELECT a.id, a.tenant_id, a.warehouse_id, w.name, a.dock_id, d.dock_code, d.dock_name,
	       a.appointment_number, a.vendor_name, a.vehicle_plate, a.driver_name, a.driver_phone,
	       a.po_reference, a.estimated_arrival, a.actual_arrival, a.start_unloading_at, a.completed_at,
	       a.status, a.notes, a.created_by, COALESCE(u.full_name, u.email), a.created_at, a.updated_at
	FROM dock_appointments a
	JOIN warehouses w ON w.id = a.warehouse_id AND w.tenant_id = $1
	LEFT JOIN inbound_docks d ON d.id = a.dock_id AND d.tenant_id = $1
	LEFT JOIN users u ON u.id = a.created_by AND u.tenant_id = $1
	WHERE a.tenant_id = $1`

	args := []any{tenantID}
	idx := 2

	if warehouseID != nil && *warehouseID != uuid.Nil {
		query += fmt.Sprintf(" AND a.warehouse_id = $%d", idx)
		args = append(args, *warehouseID)
		idx++
	}
	if status != nil && *status != "" {
		query += fmt.Sprintf(" AND a.status = $%d", idx)
		args = append(args, *status)
		idx++
	}

	query += " ORDER BY a.estimated_arrival ASC, a.created_at DESC"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAppointments: query: %w", err)
	}
	defer rows.Close()

	appointments := make([]domain.DockAppointment, 0)
	for rows.Next() {
		var a domain.DockAppointment
		var whName sql.NullString
		var dockID sql.NullString
		var dockCode sql.NullString
		var dockName sql.NullString
		var driverPhone sql.NullString
		var poRef sql.NullString
		var actualArrival sql.NullTime
		var startUnloading sql.NullTime
		var completedAt sql.NullTime
		var notes sql.NullString
		var createdBy sql.NullString
		var createdByName sql.NullString

		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.WarehouseID, &whName, &dockID, &dockCode, &dockName,
			&a.AppointmentNumber, &a.VendorName, &a.VehiclePlate, &a.DriverName, &driverPhone,
			&poRef, &a.EstimatedArrival, &actualArrival, &startUnloading, &completedAt,
			&a.Status, &notes, &createdBy, &createdByName, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListAppointments: scan: %w", err)
		}

		a.WarehouseName = nullStringToPtr(whName)
		a.DockID = nullUUIDToPtr(dockID)
		a.DockCode = nullStringToPtr(dockCode)
		a.DockName = nullStringToPtr(dockName)
		a.DriverPhone = nullStringToPtr(driverPhone)
		a.POReference = nullStringToPtr(poRef)
		a.ActualArrival = nullTimeToPtr(actualArrival)
		a.StartUnloadingAt = nullTimeToPtr(startUnloading)
		a.CompletedAt = nullTimeToPtr(completedAt)
		a.Notes = nullStringToPtr(notes)
		a.CreatedBy = nullUUIDToPtr(createdBy)
		a.CreatedByName = nullStringToPtr(createdByName)

		appointments = append(appointments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAppointments: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAppointments: commit: %w", err)
	}

	return appointments, nil
}

// AssignDockToAppointment allocates a dock with anti-collision validation.
func (r *WMSRepo) AssignDockToAppointment(ctx context.Context, tenantID, appointmentID uuid.UUID, req domain.AssignDockRequest) (*domain.DockAppointment, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: set tenant: %w", err)
	}

	// 1. Lock appointment
	var appWhID uuid.UUID
	var appStatus string
	var oldDockID sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT warehouse_id, status, dock_id
		FROM dock_appointments
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE`, appointmentID, tenantID).Scan(&appWhID, &appStatus, &oldDockID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: get appointment: %w", err)
	}
	if appStatus == string(domain.AppointmentStatusCompleted) || appStatus == string(domain.AppointmentStatusCancelled) {
		return nil, domain.ErrInvalidInput
	}

	// 2. Lock target dock
	var dockWhID uuid.UUID
	var dockStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT warehouse_id, status
		FROM inbound_docks
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE`, req.DockID, tenantID).Scan(&dockWhID, &dockStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDockNotFound
		}
		return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: get dock: %w", err)
	}
	if dockWhID != appWhID {
		return nil, domain.ErrInvalidInput
	}
	if domain.DockStatus(dockStatus) == domain.DockStatusMaintenance || domain.DockStatus(dockStatus) == domain.DockStatusOccupied {
		return nil, domain.ErrDockOccupied
	}

	// 3. Anti-Collision Guard: Check if dock is occupied by another active appointment
	var hasCollision bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM dock_appointments
			WHERE tenant_id = $1 AND dock_id = $2 AND id != $3
			  AND status IN ('ARRIVED', 'UNLOADING')
		)`, tenantID, req.DockID, appointmentID).Scan(&hasCollision)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: check collision: %w", err)
	}
	if hasCollision {
		return nil, domain.ErrDockOccupied
	}

	// 4. Update appointment dock_id
	_, err = tx.ExecContext(ctx, `
		UPDATE dock_appointments
		SET dock_id = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3`, req.DockID, appointmentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: update appointment: %w", err)
	}

	// 5. Update dock status to OCCUPIED if appointment has arrived or is unloading
	if appStatus == string(domain.AppointmentStatusArrived) || appStatus == string(domain.AppointmentStatusUnloading) {
		_, err = tx.ExecContext(ctx, `
			UPDATE inbound_docks
			SET status = $1, updated_at = NOW()
			WHERE id = $2 AND tenant_id = $3`, domain.DockStatusOccupied, req.DockID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: update dock status: %w", err)
		}
	}

	// 6. If previous dock was assigned, free it if no active appointment remains
	if oldDockID.Valid && oldDockID.String != req.DockID.String() {
		oldUUID, parseErr := uuid.Parse(oldDockID.String)
		if parseErr == nil {
			var oldStillActive bool
			err = tx.QueryRowContext(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM dock_appointments
					WHERE tenant_id = $1 AND dock_id = $2 AND id != $3
					  AND status IN ('ARRIVED', 'UNLOADING')
				)`, tenantID, oldUUID, appointmentID).Scan(&oldStillActive)
			if err != nil {
				return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: check old dock active: %w", err)
			}
			if !oldStillActive {
				_, err = tx.ExecContext(ctx, `
					UPDATE inbound_docks
					SET status = 'AVAILABLE', updated_at = NOW()
					WHERE id = $1 AND tenant_id = $2 AND status = 'OCCUPIED'`, oldUUID, tenantID)
				if err != nil {
					return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: release old dock: %w", err)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.AssignDockToAppointment: commit: %w", err)
	}

	return r.GetAppointmentByID(ctx, tenantID, appointmentID)
}

// UpdateAppointmentStatus progresses the status of an appointment.
func (r *WMSRepo) UpdateAppointmentStatus(ctx context.Context, tenantID, appointmentID uuid.UUID, req domain.UpdateAppointmentStatusRequest) (*domain.DockAppointment, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: set tenant: %w", err)
	}

	var dockID sql.NullString
	var currentStatus string
	var actualArrival sql.NullTime
	var startUnloading sql.NullTime
	var completedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT dock_id, status, actual_arrival, start_unloading_at, completed_at
		FROM dock_appointments
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE`, appointmentID, tenantID).Scan(&dockID, &currentStatus, &actualArrival, &startUnloading, &completedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAppointmentNotFound
		}
		return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: lock appointment: %w", err)
	}
	if currentStatus == string(domain.AppointmentStatusCompleted) || currentStatus == string(domain.AppointmentStatusCancelled) {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now().UTC()
	actArr := actualArrival
	startUnl := startUnloading
	compAt := completedAt

	switch req.Status {
	case domain.AppointmentStatusArrived:
		if !actArr.Valid {
			actArr = sql.NullTime{Time: now, Valid: true}
		}
		if dockID.Valid && strings.TrimSpace(dockID.String) != "" {
			dUUID, err := uuid.Parse(dockID.String)
			if err == nil && dUUID != uuid.Nil {
				var dockStatus string
				err = tx.QueryRowContext(ctx, `
					SELECT status FROM inbound_docks
					WHERE id = $1 AND tenant_id = $2
					FOR UPDATE`, dUUID, tenantID).Scan(&dockStatus)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return nil, domain.ErrDockNotFound
					}
					return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: check dock status: %w", err)
				}
				if domain.DockStatus(dockStatus) == domain.DockStatusMaintenance {
					return nil, domain.ErrDockOccupied
				}

				var collisionCount int
				err = tx.QueryRowContext(ctx, `
					SELECT COUNT(*) FROM dock_appointments
					WHERE tenant_id = $1 AND dock_id = $2 AND id != $3
					  AND status IN ('ARRIVED', 'UNLOADING')`,
					tenantID, dUUID, appointmentID).Scan(&collisionCount)
				if err != nil {
					return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: check collision: %w", err)
				}
				if collisionCount > 0 {
					return nil, domain.ErrDockOccupied
				}

				_, err = tx.ExecContext(ctx, `
					UPDATE inbound_docks SET status = 'OCCUPIED', updated_at = NOW()
					WHERE id = $1 AND tenant_id = $2`, dUUID, tenantID)
				if err != nil {
					return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: set dock occupied: %w", err)
				}
			}
		}

	case domain.AppointmentStatusUnloading:
		if !dockID.Valid || strings.TrimSpace(dockID.String) == "" {
			return nil, domain.ErrInvalidInput
		}
		dUUID, err := uuid.Parse(dockID.String)
		if err != nil || dUUID == uuid.Nil {
			return nil, domain.ErrInvalidInput
		}

		var dockStatus string
		err = tx.QueryRowContext(ctx, `
			SELECT status FROM inbound_docks
			WHERE id = $1 AND tenant_id = $2
			FOR UPDATE`, dUUID, tenantID).Scan(&dockStatus)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.ErrDockNotFound
			}
			return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: check dock status: %w", err)
		}
		if domain.DockStatus(dockStatus) == domain.DockStatusMaintenance {
			return nil, domain.ErrDockOccupied
		}

		var collisionCount int
		err = tx.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM dock_appointments
			WHERE tenant_id = $1 AND dock_id = $2 AND id != $3
			  AND status IN ('ARRIVED', 'UNLOADING')`,
			tenantID, dUUID, appointmentID).Scan(&collisionCount)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: check collision: %w", err)
		}
		if collisionCount > 0 {
			return nil, domain.ErrDockOccupied
		}

		if !actArr.Valid {
			actArr = sql.NullTime{Time: now, Valid: true}
		}
		if !startUnl.Valid {
			startUnl = sql.NullTime{Time: now, Valid: true}
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE inbound_docks SET status = 'OCCUPIED', updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2`, dUUID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: set dock occupied: %w", err)
		}

	case domain.AppointmentStatusCompleted, domain.AppointmentStatusCancelled:
		if req.Status == domain.AppointmentStatusCompleted && !compAt.Valid {
			compAt = sql.NullTime{Time: now, Valid: true}
		}
		if dockID.Valid && strings.TrimSpace(dockID.String) != "" {
			dUUID, err := uuid.Parse(dockID.String)
			if err == nil && dUUID != uuid.Nil {
				var activeCount int
				err = tx.QueryRowContext(ctx, `
					SELECT COUNT(*) FROM dock_appointments
					WHERE tenant_id = $1 AND dock_id = $2 AND id != $3 AND status IN ('ARRIVED', 'UNLOADING')`,
					tenantID, dUUID, appointmentID).Scan(&activeCount)
				if err != nil {
					return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: check active appointments: %w", err)
				}
				if activeCount == 0 {
					_, err = tx.ExecContext(ctx, `
						UPDATE inbound_docks SET status = 'AVAILABLE', updated_at = NOW()
						WHERE id = $1 AND tenant_id = $2 AND status = 'OCCUPIED'`, dUUID, tenantID)
					if err != nil {
						return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: release dock: %w", err)
					}
				}
			}
		}
	}

	updateSQL := `
	UPDATE dock_appointments
	SET status = $1,
	    actual_arrival = $2,
	    start_unloading_at = $3,
	    completed_at = $4,
	    notes = COALESCE($5, notes),
	    updated_at = NOW()
	WHERE id = $6 AND tenant_id = $7`

	_, err = tx.ExecContext(ctx, updateSQL,
		req.Status, actArr, startUnl, compAt, ptrToNullString(req.Notes), appointmentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.UpdateAppointmentStatus: commit: %w", err)
	}

	return r.GetAppointmentByID(ctx, tenantID, appointmentID)
}

// ============================================================================
// 3. PALLET LPN CONTAINERIZATION
// ============================================================================

// CreateLPN registers a new pallet container.
func (r *WMSRepo) CreateLPN(ctx context.Context, tenantID, userID uuid.UUID, req domain.CreateLPNRequest) (*domain.StockLPN, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateLPN: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateLPN: set tenant: %w", err)
	}

	// 1. Verify warehouse belongs to tenant
	var whName string
	err = tx.QueryRowContext(ctx, `SELECT name FROM warehouses WHERE id = $1 AND tenant_id = $2`, req.WarehouseID, tenantID).Scan(&whName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("WMSRepo.CreateLPN: check warehouse: %w", err)
	}

	// 2. Verify location belongs to tenant and warehouse
	var locWhID sql.NullString
	var locCode, locName string
	err = tx.QueryRowContext(ctx, `
		SELECT warehouse_id, code, name
		FROM warehouse_locations
		WHERE id = $1 AND tenant_id = $2`, req.LocationID, tenantID).Scan(&locWhID, &locCode, &locName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrLocationNotFound
		}
		return nil, fmt.Errorf("WMSRepo.CreateLPN: check location: %w", err)
	}
	if !locWhID.Valid || locWhID.String != req.WarehouseID.String() {
		return nil, domain.ErrInvalidInput
	}

	// 3. Generate LPN-YYYYMMDD-XXXX sequence if blank
	lpnCode := strings.TrimSpace(req.LPNCode)
	if lpnCode == "" {
		todayStr := time.Now().UTC().Format("20060102")
		for i := 0; i < 20; i++ {
			var count int
			err := tx.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM stock_lpns
				WHERE tenant_id = $1 AND warehouse_id = $2 AND lpn_code LIKE $3`,
				tenantID, req.WarehouseID, "LPN-"+todayStr+"-%").Scan(&count)
			if err != nil {
				return nil, fmt.Errorf("WMSRepo.CreateLPN: count lpns: %w", err)
			}
			candidate := fmt.Sprintf("LPN-%s-%04d", todayStr, count+1+i)
			var exists bool
			err = tx.QueryRowContext(ctx, `
				SELECT EXISTS(SELECT 1 FROM stock_lpns WHERE tenant_id = $1 AND warehouse_id = $2 AND lpn_code = $3)`,
				tenantID, req.WarehouseID, candidate).Scan(&exists)
			if err != nil {
				return nil, fmt.Errorf("WMSRepo.CreateLPN: check lpn exists: %w", err)
			}
			if !exists {
				lpnCode = candidate
				break
			}
		}
		if lpnCode == "" {
			lpnCode = fmt.Sprintf("LPN-%s-%04d", todayStr, time.Now().UnixNano()%10000)
		}
	}

	palletType := req.PalletType
	if palletType == "" {
		palletType = domain.PalletTypeWooden
	}
	maxWeight := req.MaxWeightKg
	if maxWeight.IsZero() || maxWeight.IsNegative() {
		maxWeight = decimal.NewFromFloat(1000.00)
	}

	lpnID := uuid.New()
	now := time.Now().UTC()
	status := domain.LPNStatusStaged
	totalWeight := decimal.Zero

	insertSQL := `
	INSERT INTO stock_lpns (
		id, tenant_id, warehouse_id, lpn_code, location_id, pallet_type,
		status, max_weight_kg, total_weight_kg, notes, created_by, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
	)`
	_, err = tx.ExecContext(ctx, insertSQL,
		lpnID, tenantID, req.WarehouseID, lpnCode, req.LocationID, palletType,
		status, maxWeight, totalWeight, ptrToNullString(req.Notes),
		ptrToNullUUID(&userID), now, now)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("WMSRepo.CreateLPN: insert lpn: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.CreateLPN: commit: %w", err)
	}

	return &domain.StockLPN{
		ID:            lpnID,
		TenantID:      tenantID,
		WarehouseID:   req.WarehouseID,
		WarehouseName: &whName,
		LPNCode:       lpnCode,
		LocationID:    req.LocationID,
		LocationCode:  &locCode,
		LocationName:  &locName,
		PalletType:    palletType,
		Status:        status,
		MaxWeightKg:   maxWeight,
		TotalWeightKg: totalWeight,
		Notes:         req.Notes,
		CreatedBy:     &userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// GetLPNByID loads LPN and all contained product batch items.
func (r *WMSRepo) GetLPNByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockLPNDetail, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLPNByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLPNByID: set tenant: %w", err)
	}

	queryHeader := `
	SELECT l.id, l.tenant_id, l.warehouse_id, w.name, l.lpn_code, l.location_id,
	       loc.code, loc.name, l.pallet_type, l.status, l.max_weight_kg, l.total_weight_kg,
	       l.notes, l.created_by, COALESCE(u.full_name, u.email), l.created_at, l.updated_at
	FROM stock_lpns l
	JOIN warehouses w ON w.id = l.warehouse_id AND w.tenant_id = $2
	JOIN warehouse_locations loc ON loc.id = l.location_id AND loc.tenant_id = $2
	LEFT JOIN users u ON u.id = l.created_by AND u.tenant_id = $2
	WHERE l.id = $1 AND l.tenant_id = $2`

	var lpn domain.StockLPN
	var whName sql.NullString
	var locCode sql.NullString
	var locName sql.NullString
	var notes sql.NullString
	var createdBy sql.NullString
	var createdByName sql.NullString

	err = tx.QueryRowContext(ctx, queryHeader, id, tenantID).Scan(
		&lpn.ID, &lpn.TenantID, &lpn.WarehouseID, &whName, &lpn.LPNCode, &lpn.LocationID,
		&locCode, &locName, &lpn.PalletType, &lpn.Status, &lpn.MaxWeightKg, &lpn.TotalWeightKg,
		&notes, &createdBy, &createdByName, &lpn.CreatedAt, &lpn.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrLPNNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetLPNByID: query header: %w", err)
	}

	lpn.WarehouseName = nullStringToPtr(whName)
	lpn.LocationCode = nullStringToPtr(locCode)
	lpn.LocationName = nullStringToPtr(locName)
	lpn.Notes = nullStringToPtr(notes)
	lpn.CreatedBy = nullUUIDToPtr(createdBy)
	lpn.CreatedByName = nullStringToPtr(createdByName)

	queryItems := `
	SELECT i.id, i.tenant_id, i.lpn_id, i.product_id, p.name, p.sku,
	       i.batch_id, b.batch_number, b.expiry_date, i.quantity, i.created_at, i.updated_at
	FROM stock_lpn_items i
	JOIN products p ON p.id = i.product_id AND p.tenant_id = $2
	JOIN stock_batches b ON b.id = i.batch_id AND b.tenant_id = $2
	WHERE i.lpn_id = $1 AND i.tenant_id = $2
	ORDER BY i.created_at ASC`

	rows, err := tx.QueryContext(ctx, queryItems, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLPNByID: query items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.StockLPNItem, 0)
	for rows.Next() {
		var item domain.StockLPNItem
		var prodName sql.NullString
		var prodSKU sql.NullString
		var batchNum sql.NullString
		var expiryDate sql.NullTime

		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.LPNID, &item.ProductID, &prodName, &prodSKU,
			&item.BatchID, &batchNum, &expiryDate, &item.Quantity, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.GetLPNByID: scan item: %w", err)
		}

		item.ProductName = nullStringToPtr(prodName)
		item.ProductSKU = nullStringToPtr(prodSKU)
		item.BatchNumber = nullStringToPtr(batchNum)
		item.ExpiryDate = nullTimeToPtr(expiryDate)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLPNByID: items err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetLPNByID: commit: %w", err)
	}

	return &domain.StockLPNDetail{
		LPN:   lpn,
		Items: items,
	}, nil
}

// ListLPNs selects LPNs for warehouse.
func (r *WMSRepo) ListLPNs(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, locationID *uuid.UUID, status *domain.LPNStatus) ([]domain.StockLPN, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLPNs: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLPNs: set tenant: %w", err)
	}

	query := `
	SELECT l.id, l.tenant_id, l.warehouse_id, w.name, l.lpn_code, l.location_id,
	       loc.code, loc.name, l.pallet_type, l.status, l.max_weight_kg, l.total_weight_kg,
	       l.notes, l.created_by, COALESCE(u.full_name, u.email), l.created_at, l.updated_at
	FROM stock_lpns l
	JOIN warehouses w ON w.id = l.warehouse_id AND w.tenant_id = $1
	JOIN warehouse_locations loc ON loc.id = l.location_id AND loc.tenant_id = $1
	LEFT JOIN users u ON u.id = l.created_by AND u.tenant_id = $1
	WHERE l.tenant_id = $1`

	args := []any{tenantID}
	idx := 2

	if warehouseID != nil && *warehouseID != uuid.Nil {
		query += fmt.Sprintf(" AND l.warehouse_id = $%d", idx)
		args = append(args, *warehouseID)
		idx++
	}
	if locationID != nil && *locationID != uuid.Nil {
		query += fmt.Sprintf(" AND l.location_id = $%d", idx)
		args = append(args, *locationID)
		idx++
	}
	if status != nil && *status != "" {
		query += fmt.Sprintf(" AND l.status = $%d", idx)
		args = append(args, *status)
		idx++
	}

	query += " ORDER BY l.created_at DESC"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLPNs: query: %w", err)
	}
	defer rows.Close()

	lpns := make([]domain.StockLPN, 0)
	for rows.Next() {
		var lpn domain.StockLPN
		var whName sql.NullString
		var locCode sql.NullString
		var locName sql.NullString
		var notes sql.NullString
		var createdBy sql.NullString
		var createdByName sql.NullString

		if err := rows.Scan(
			&lpn.ID, &lpn.TenantID, &lpn.WarehouseID, &whName, &lpn.LPNCode, &lpn.LocationID,
			&locCode, &locName, &lpn.PalletType, &lpn.Status, &lpn.MaxWeightKg, &lpn.TotalWeightKg,
			&notes, &createdBy, &createdByName, &lpn.CreatedAt, &lpn.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListLPNs: scan: %w", err)
		}

		lpn.WarehouseName = nullStringToPtr(whName)
		lpn.LocationCode = nullStringToPtr(locCode)
		lpn.LocationName = nullStringToPtr(locName)
		lpn.Notes = nullStringToPtr(notes)
		lpn.CreatedBy = nullUUIDToPtr(createdBy)
		lpn.CreatedByName = nullStringToPtr(createdByName)
		lpns = append(lpns, lpn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLPNs: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListLPNs: commit: %w", err)
	}

	return lpns, nil
}

// AddLPNItem inserts or updates an item lot inside an LPN, recalculating total_weight_kg.
func (r *WMSRepo) AddLPNItem(ctx context.Context, tenantID, lpnID uuid.UUID, req domain.AddLPNItemRequest) (*domain.StockLPNItem, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.AddLPNItem: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.AddLPNItem: set tenant: %w", err)
	}

	// 1. Lock LPN
	var lpnStatus string
	err = tx.QueryRowContext(ctx, `SELECT status FROM stock_lpns WHERE id = $1 AND tenant_id = $2 FOR UPDATE`, lpnID, tenantID).Scan(&lpnStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrLPNNotFound
		}
		return nil, fmt.Errorf("WMSRepo.AddLPNItem: lock lpn: %w", err)
	}
	if domain.LPNStatus(lpnStatus) == domain.LPNStatusShipped || domain.LPNStatus(lpnStatus) == domain.LPNStatusDecommissioned {
		return nil, domain.ErrInvalidInput
	}

	// 2. Verify product
	var prodName, prodSKU string
	err = tx.QueryRowContext(ctx, `SELECT name, sku FROM products WHERE id = $1 AND tenant_id = $2`, req.ProductID, tenantID).Scan(&prodName, &prodSKU)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("WMSRepo.AddLPNItem: get product: %w", err)
	}

	// 3. Verify batch belongs to product
	var batchProdID uuid.UUID
	var batchNum string
	var expiryDate sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT product_id, batch_number, expiry_date
		FROM stock_batches
		WHERE id = $1 AND tenant_id = $2`, req.BatchID, tenantID).Scan(&batchProdID, &batchNum, &expiryDate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrStockBatchNotFound
		}
		return nil, fmt.Errorf("WMSRepo.AddLPNItem: get batch: %w", err)
	}
	if batchProdID != req.ProductID {
		return nil, domain.ErrInvalidInput
	}

	// 4. Insert or update LPN item
	itemID := uuid.New()
	var finalQty decimal.Decimal
	var createdAt, updatedAt time.Time

	upsertSQL := `
	INSERT INTO stock_lpn_items (
		id, tenant_id, lpn_id, product_id, batch_id, quantity, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, NOW(), NOW()
	)
	ON CONFLICT (tenant_id, lpn_id, batch_id)
	DO UPDATE SET
		quantity = stock_lpn_items.quantity + EXCLUDED.quantity,
		updated_at = NOW()
	RETURNING id, quantity, created_at, updated_at`

	err = tx.QueryRowContext(ctx, upsertSQL,
		itemID, tenantID, lpnID, req.ProductID, req.BatchID, req.Quantity).Scan(
		&itemID, &finalQty, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.AddLPNItem: upsert item: %w", err)
	}

	// 5. Recalculate LPN total_weight_kg
	_, err = tx.ExecContext(ctx, `
		UPDATE stock_lpns
		SET total_weight_kg = (
			SELECT COALESCE(SUM(quantity), 0)
			FROM stock_lpn_items
			WHERE lpn_id = $1 AND tenant_id = $2
		), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`, lpnID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.AddLPNItem: recalculate total weight: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.AddLPNItem: commit: %w", err)
	}

	return &domain.StockLPNItem{
		ID:          itemID,
		TenantID:    tenantID,
		LPNID:       lpnID,
		ProductID:   req.ProductID,
		ProductName: &prodName,
		ProductSKU:  &prodSKU,
		BatchID:     req.BatchID,
		BatchNumber: &batchNum,
		ExpiryDate:  nullTimeToPtr(expiryDate),
		Quantity:    finalQty,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

// MoveLPN performs atomic forklift putaway of an LPN and all its contents to a target rack location.
func (r *WMSRepo) MoveLPN(ctx context.Context, tenantID, userID, lpnID uuid.UUID, req domain.MoveLPNRequest) (*domain.StockLPNDetail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveLPN: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveLPN: set tenant: %w", err)
	}

	// 1. Lock LPN FOR UPDATE
	var lpn domain.StockLPN
	var createdBy sql.NullString
	var notes sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT id, warehouse_id, lpn_code, location_id, pallet_type, status, max_weight_kg, total_weight_kg, notes, created_by, created_at, updated_at
		FROM stock_lpns
		WHERE id = $1 AND tenant_id = $2
		FOR UPDATE`, lpnID, tenantID).Scan(
		&lpn.ID, &lpn.WarehouseID, &lpn.LPNCode, &lpn.LocationID, &lpn.PalletType,
		&lpn.Status, &lpn.MaxWeightKg, &lpn.TotalWeightKg, &notes, &createdBy, &lpn.CreatedAt, &lpn.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrLPNNotFound
		}
		return nil, fmt.Errorf("WMSRepo.MoveLPN: lock lpn: %w", err)
	}
	if lpn.Status == domain.LPNStatusShipped || lpn.Status == domain.LPNStatusDecommissioned {
		return nil, domain.ErrInvalidInput
	}
	if req.TargetLocationID == lpn.LocationID {
		return nil, domain.ErrInvalidInput
	}
	lpn.TenantID = tenantID
	lpn.Notes = nullStringToPtr(notes)
	lpn.CreatedBy = nullUUIDToPtr(createdBy)

	// 2. Validate target location exists, belongs to tenant and warehouse, and is INTERNAL / RACK
	var destWhID sql.NullString
	var destType string
	err = tx.QueryRowContext(ctx, `
		SELECT warehouse_id, type
		FROM warehouse_locations
		WHERE id = $1 AND tenant_id = $2`, req.TargetLocationID, tenantID).Scan(&destWhID, &destType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrLocationNotFound
		}
		return nil, fmt.Errorf("WMSRepo.MoveLPN: query dest location: %w", err)
	}
	if !destWhID.Valid || destWhID.String != lpn.WarehouseID.String() {
		return nil, domain.ErrInvalidInput
	}
	if domain.LocationType(destType) != domain.LocationTypeInternal && destType != "RACK" {
		return nil, domain.ErrInvalidLocationType
	}

	// 3. Load items from stock_lpn_items
	rows, err := tx.QueryContext(ctx, `
		SELECT id, product_id, batch_id, quantity
		FROM stock_lpn_items
		WHERE lpn_id = $1 AND tenant_id = $2
		ORDER BY created_at ASC
		FOR UPDATE`, lpnID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveLPN: query lpn items: %w", err)
	}
	defer rows.Close()

	type itemRow struct {
		id        uuid.UUID
		productID uuid.UUID
		batchID   uuid.UUID
		quantity  decimal.Decimal
	}
	var items []itemRow
	for rows.Next() {
		var it itemRow
		if err := rows.Scan(&it.id, &it.productID, &it.batchID, &it.quantity); err != nil {
			return nil, fmt.Errorf("WMSRepo.MoveLPN: scan item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveLPN: rows err: %w", err)
	}
	_ = rows.Close()

	if len(items) == 0 {
		return nil, domain.ErrLPNEmpty
	}

	// 4. For each item: Create StockMovement preserving batch_id (ADR-014 Invariant 1)
	now := time.Now().UTC()
	for i, it := range items {
		// Advisory lock per location stock
		if err := r.LockLocationStock(ctx, tx, tenantID, lpn.LocationID, it.productID); err != nil {
			return nil, err
		}

		// Verify available batch stock at source location
		var currentStock decimal.Decimal
		batchCalcSQL := `
		SELECT COALESCE(
			SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE -quantity END),
			0
		)
		FROM stock_movements
		WHERE tenant_id = $1 AND product_id = $3 AND batch_id = $4
		  AND (source_location_id = $2 OR dest_location_id = $2)
		  AND status = 'DONE'`
		if err := tx.QueryRowContext(ctx, batchCalcSQL, tenantID, lpn.LocationID, it.productID, it.batchID).Scan(&currentStock); err != nil {
			return nil, fmt.Errorf("WMSRepo.MoveLPN: scan batch stock: %w", err)
		}
		if currentStock.LessThan(it.quantity) {
			return nil, domain.ErrInsufficientStock
		}

		movID := uuid.New()
		movNumber := fmt.Sprintf("PUTAWAY-LPN-%s-%d", lpn.LPNCode, i+1)
		_, err = tx.ExecContext(ctx, createStockMovementSQL,
			movID, tenantID, movNumber, it.productID,
			lpn.LocationID, req.TargetLocationID, it.quantity,
			decimal.Zero, domain.StockMovementStatusDone,
			domain.StockRefLPNPutaway, lpnID,
			ptrToNullUUID(&userID), ptrToNullUUID(&it.batchID), now)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.MoveLPN: create stock movement: %w", err)
		}

		auditDetails, _ := json.Marshal(map[string]any{
			"movement_number":    movNumber,
			"product_id":         it.productID,
			"source_location_id": lpn.LocationID,
			"dest_location_id":   req.TargetLocationID,
			"quantity":           it.quantity,
			"reference_type":     domain.StockRefLPNPutaway,
			"reference_id":       lpnID,
			"batch_id":           it.batchID,
		})
		if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "stock_movement", movID, "created", auditDetails); err != nil {
			return nil, fmt.Errorf("WMSRepo.MoveLPN: audit movement: %w", err)
		}
	}

	// 5. Update stock_lpns location_id and status to STORED
	_, err = tx.ExecContext(ctx, `
		UPDATE stock_lpns
		SET location_id = $1, status = 'STORED', notes = COALESCE($2, notes), updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4`, req.TargetLocationID, ptrToNullString(req.Notes), lpnID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveLPN: update lpn: %w", err)
	}

	// 6. Write audit log for LPN move
	lpnAuditDetails, _ := json.Marshal(map[string]any{
		"lpn_id":             lpnID,
		"lpn_code":           lpn.LPNCode,
		"source_location_id": lpn.LocationID,
		"dest_location_id":   req.TargetLocationID,
		"item_count":         len(items),
		"notes":              req.Notes,
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "stock_lpn", lpnID, "moved", lpnAuditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveLPN: audit lpn: %w", err)
	}

	// 7. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveLPN: commit: %w", err)
	}

	return r.GetLPNByID(ctx, tenantID, lpnID)
}
