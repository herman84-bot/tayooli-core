package wms

// Sprint 5 WMS Usecase: Inbound Dock Scheduling, Appointments, and Pallet LPN Containerization
// ADR-014 Invariant 1 (Double-Entry Ledger per Batch), OCA/wms dock scheduling patterns.

import (
	"context"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// -----------------------------------------------------------------------------
// Inbound Dock Bays
// -----------------------------------------------------------------------------

// CreateDock validates write access on warehouse, validates request DTO, and registers a dock.
func (u *Usecase) CreateDock(ctx context.Context, tenantID, userID uuid.UUID, role string, req domain.CreateDockRequest) (*domain.InboundDock, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.CreateDock(ctx, tenantID, req)
}

// GetDock retrieves an inbound dock by ID after validating warehouse read access.
func (u *Usecase) GetDock(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID) (*domain.InboundDock, error) {
	dock, err := u.repo.GetDockByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseAccess(ctx, tenantID, userID, role, dock.WarehouseID); err != nil {
		return nil, err
	}
	return dock, nil
}

// ListDocks lists inbound docks for a warehouse after verifying warehouse read access.
func (u *Usecase) ListDocks(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID, status *domain.DockStatus) ([]domain.InboundDock, error) {
	if err := u.ValidateWarehouseAccess(ctx, tenantID, userID, role, warehouseID); err != nil {
		return nil, err
	}
	var whPtr *uuid.UUID
	if warehouseID != uuid.Nil {
		whPtr = &warehouseID
	}
	return u.repo.ListDocks(ctx, tenantID, whPtr, status)
}

// UpdateDockStatus validates request DTO, verifies warehouse write access, and transitions dock status.
func (u *Usecase) UpdateDockStatus(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID, req domain.UpdateDockStatusRequest) (*domain.InboundDock, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	dock, err := u.repo.GetDockByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, dock.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.UpdateDockStatus(ctx, tenantID, id, req)
}

// -----------------------------------------------------------------------------
// Dock Appointments
// -----------------------------------------------------------------------------

// CreateAppointment validates write access on warehouse, validates request DTO, and schedules an appointment.
func (u *Usecase) CreateAppointment(ctx context.Context, tenantID, userID uuid.UUID, role string, req domain.CreateAppointmentRequest) (*domain.DockAppointment, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.CreateAppointment(ctx, tenantID, userID, req)
}

// GetAppointment retrieves a dock appointment after validating warehouse read access.
func (u *Usecase) GetAppointment(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID) (*domain.DockAppointment, error) {
	app, err := u.repo.GetAppointmentByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseAccess(ctx, tenantID, userID, role, app.WarehouseID); err != nil {
		return nil, err
	}
	return app, nil
}

// ListAppointments lists appointments for a warehouse after verifying warehouse read access.
func (u *Usecase) ListAppointments(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID, status *domain.AppointmentStatus) ([]domain.DockAppointment, error) {
	if err := u.ValidateWarehouseAccess(ctx, tenantID, userID, role, warehouseID); err != nil {
		return nil, err
	}
	var whPtr *uuid.UUID
	if warehouseID != uuid.Nil {
		whPtr = &warehouseID
	}
	return u.repo.ListAppointments(ctx, tenantID, whPtr, status)
}

// AssignDockToAppointment validates appointment existence, validates warehouse write access, and assigns dock.
func (u *Usecase) AssignDockToAppointment(ctx context.Context, tenantID, userID uuid.UUID, role string, id, dockID uuid.UUID) (*domain.DockAppointment, error) {
	req := domain.AssignDockRequest{DockID: dockID}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	app, err := u.repo.GetAppointmentByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, app.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.AssignDockToAppointment(ctx, tenantID, id, req)
}

// UpdateAppointmentStatus validates request DTO, verifies warehouse write access, and updates status.
func (u *Usecase) UpdateAppointmentStatus(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID, req domain.UpdateAppointmentStatusRequest) (*domain.DockAppointment, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	app, err := u.repo.GetAppointmentByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, app.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.UpdateAppointmentStatus(ctx, tenantID, id, req)
}

// -----------------------------------------------------------------------------
// Stock LPNs (License Plate Numbers)
// -----------------------------------------------------------------------------

// CreateLPN validates write access on warehouse, validates request DTO, and registers a pallet container.
func (u *Usecase) CreateLPN(ctx context.Context, tenantID, userID uuid.UUID, role string, req domain.CreateLPNRequest) (*domain.StockLPN, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.CreateLPN(ctx, tenantID, userID, req)
}

// GetLPN retrieves an LPN container along with all contained items after validating warehouse read access.
func (u *Usecase) GetLPN(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID) (*domain.StockLPNDetail, error) {
	detail, err := u.repo.GetLPNByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseAccess(ctx, tenantID, userID, role, detail.LPN.WarehouseID); err != nil {
		return nil, err
	}
	return detail, nil
}

// ListLPNs lists LPN pallets for a warehouse after verifying warehouse read access.
func (u *Usecase) ListLPNs(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID, status *domain.LPNStatus) ([]domain.StockLPN, error) {
	if err := u.ValidateWarehouseAccess(ctx, tenantID, userID, role, warehouseID); err != nil {
		return nil, err
	}
	var whPtr *uuid.UUID
	if warehouseID != uuid.Nil {
		whPtr = &warehouseID
	}
	return u.repo.ListLPNs(ctx, tenantID, whPtr, nil, status)
}

// AddLPNItem packs a batch quantity into an LPN container after validating warehouse write access.
func (u *Usecase) AddLPNItem(ctx context.Context, tenantID, userID uuid.UUID, role string, lpnID uuid.UUID, req domain.AddLPNItemRequest) (*domain.StockLPNDetail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	detail, err := u.repo.GetLPNByID(ctx, tenantID, lpnID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, detail.LPN.WarehouseID); err != nil {
		return nil, err
	}
	if _, err := u.repo.AddLPNItem(ctx, tenantID, lpnID, req); err != nil {
		return nil, err
	}
	return u.repo.GetLPNByID(ctx, tenantID, lpnID)
}

// MoveLPN performs atomic forklift putaway of an LPN and all its contents to a target rack location.
// Fetches LPN first to get its WarehouseID, validates warehouse write access, then delegates to repo.
func (u *Usecase) MoveLPN(ctx context.Context, tenantID, userID uuid.UUID, role string, lpnID uuid.UUID, req domain.MoveLPNRequest) (*domain.StockLPNDetail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	detail, err := u.repo.GetLPNByID(ctx, tenantID, lpnID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, detail.LPN.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.MoveLPN(ctx, tenantID, userID, lpnID, req)
}
