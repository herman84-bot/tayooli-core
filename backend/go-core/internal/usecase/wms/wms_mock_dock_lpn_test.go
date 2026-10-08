package wms_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

// Mock methods for WMSDockLPNRepository on mockWMSRepo

func (m *mockWMSRepo) CreateDock(ctx context.Context, tenantID uuid.UUID, req domain.CreateDockRequest) (*domain.InboundDock, error) {
	now := time.Now().UTC()
	whName := "Mock Warehouse"
	return &domain.InboundDock{
		ID:            uuid.New(),
		TenantID:      tenantID,
		WarehouseID:   req.WarehouseID,
		WarehouseName: &whName,
		DockCode:      req.DockCode,
		DockName:      req.DockName,
		DockType:      req.DockType,
		MaxTonnage:    req.MaxTonnage,
		Status:        domain.DockStatusAvailable,
		Notes:         req.Notes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (m *mockWMSRepo) GetDockByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.InboundDock, error) {
	now := time.Now().UTC()
	whName := "Mock Warehouse"
	return &domain.InboundDock{
		ID:            id,
		TenantID:      tenantID,
		WarehouseID:   uuid.New(),
		WarehouseName: &whName,
		DockCode:      "DOCK-01",
		DockName:      "Mock Dock",
		DockType:      domain.DockTypeInbound,
		MaxTonnage:    decimal.NewFromFloat(10.0),
		Status:        domain.DockStatusAvailable,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (m *mockWMSRepo) ListDocks(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.DockStatus) ([]domain.InboundDock, error) {
	return []domain.InboundDock{}, nil
}

func (m *mockWMSRepo) UpdateDockStatus(ctx context.Context, tenantID, id uuid.UUID, req domain.UpdateDockStatusRequest) (*domain.InboundDock, error) {
	dock, err := m.GetDockByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	dock.Status = req.Status
	dock.Notes = req.Notes
	return dock, nil
}

func (m *mockWMSRepo) CreateAppointment(ctx context.Context, tenantID, userID uuid.UUID, req domain.CreateAppointmentRequest) (*domain.DockAppointment, error) {
	now := time.Now().UTC()
	return &domain.DockAppointment{
		ID:                uuid.New(),
		TenantID:          tenantID,
		WarehouseID:       req.WarehouseID,
		DockID:            req.DockID,
		AppointmentNumber: "APP-MOCK-001",
		VendorName:        req.VendorName,
		VehiclePlate:      req.VehiclePlate,
		DriverName:        req.DriverName,
		DriverPhone:       req.DriverPhone,
		POReference:       req.POReference,
		EstimatedArrival:  req.EstimatedArrival,
		Status:            domain.AppointmentStatusScheduled,
		Notes:             req.Notes,
		CreatedBy:         &userID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

func (m *mockWMSRepo) GetAppointmentByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.DockAppointment, error) {
	now := time.Now().UTC()
	return &domain.DockAppointment{
		ID:                id,
		TenantID:          tenantID,
		WarehouseID:       uuid.New(),
		AppointmentNumber: "APP-MOCK-001",
		VendorName:        "Mock Vendor",
		VehiclePlate:      "B 1234 CD",
		DriverName:        "Driver",
		EstimatedArrival:  now,
		Status:            domain.AppointmentStatusScheduled,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

func (m *mockWMSRepo) ListAppointments(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.AppointmentStatus) ([]domain.DockAppointment, error) {
	return []domain.DockAppointment{}, nil
}

func (m *mockWMSRepo) AssignDockToAppointment(ctx context.Context, tenantID, appointmentID uuid.UUID, req domain.AssignDockRequest) (*domain.DockAppointment, error) {
	app, err := m.GetAppointmentByID(ctx, tenantID, appointmentID)
	if err != nil {
		return nil, err
	}
	app.DockID = &req.DockID
	return app, nil
}

func (m *mockWMSRepo) UpdateAppointmentStatus(ctx context.Context, tenantID, appointmentID uuid.UUID, req domain.UpdateAppointmentStatusRequest) (*domain.DockAppointment, error) {
	app, err := m.GetAppointmentByID(ctx, tenantID, appointmentID)
	if err != nil {
		return nil, err
	}
	app.Status = req.Status
	app.Notes = req.Notes
	return app, nil
}

func (m *mockWMSRepo) CreateLPN(ctx context.Context, tenantID, userID uuid.UUID, req domain.CreateLPNRequest) (*domain.StockLPN, error) {
	now := time.Now().UTC()
	return &domain.StockLPN{
		ID:            uuid.New(),
		TenantID:      tenantID,
		WarehouseID:   req.WarehouseID,
		LPNCode:       "LPN-MOCK-001",
		LocationID:    req.LocationID,
		PalletType:    req.PalletType,
		Status:        domain.LPNStatusStaged,
		MaxWeightKg:   req.MaxWeightKg,
		TotalWeightKg: decimal.Zero,
		Notes:         req.Notes,
		CreatedBy:     &userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (m *mockWMSRepo) GetLPNByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockLPNDetail, error) {
	now := time.Now().UTC()
	return &domain.StockLPNDetail{
		LPN: domain.StockLPN{
			ID:            id,
			TenantID:      tenantID,
			WarehouseID:   uuid.New(),
			LPNCode:       "LPN-MOCK-001",
			LocationID:    uuid.New(),
			PalletType:    domain.PalletTypeWooden,
			Status:        domain.LPNStatusStaged,
			MaxWeightKg:   decimal.NewFromFloat(1000),
			TotalWeightKg: decimal.Zero,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Items: []domain.StockLPNItem{},
	}, nil
}

func (m *mockWMSRepo) ListLPNs(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, locationID *uuid.UUID, status *domain.LPNStatus) ([]domain.StockLPN, error) {
	return []domain.StockLPN{}, nil
}

func (m *mockWMSRepo) AddLPNItem(ctx context.Context, tenantID, lpnID uuid.UUID, req domain.AddLPNItemRequest) (*domain.StockLPNItem, error) {
	now := time.Now().UTC()
	return &domain.StockLPNItem{
		ID:        uuid.New(),
		TenantID:  tenantID,
		LPNID:     lpnID,
		ProductID: req.ProductID,
		BatchID:   req.BatchID,
		Quantity:  req.Quantity,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (m *mockWMSRepo) MoveLPN(ctx context.Context, tenantID, userID, lpnID uuid.UUID, req domain.MoveLPNRequest) (*domain.StockLPNDetail, error) {
	detail, err := m.GetLPNByID(ctx, tenantID, lpnID)
	if err != nil {
		return nil, err
	}
	detail.LPN.LocationID = req.TargetLocationID
	detail.LPN.Status = domain.LPNStatusStored
	return detail, nil
}
