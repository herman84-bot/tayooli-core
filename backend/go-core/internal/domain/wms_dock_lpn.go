package domain

// Sprint 5 WMS Domain: Inbound Dock Scheduling, Appointments, and Pallet LPN Containerization
// ADR-014 Invariant 1 (Double-Entry Ledger per Batch), OCA/wms dock scheduling patterns.

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DockStatus represents operational state of an inbound dock.
type DockStatus string

const (
	DockStatusAvailable   DockStatus = "AVAILABLE"
	DockStatusOccupied    DockStatus = "OCCUPIED"
	DockStatusMaintenance DockStatus = "MAINTENANCE"
)

// DockType represents the physical purpose of the dock.
type DockType string

const (
	DockTypeInbound   DockType = "INBOUND"
	DockTypeOutbound  DockType = "OUTBOUND"
	DockTypeCrossDock DockType = "CROSS_DOCK"
)

// AppointmentStatus represents truck arrival and unloading lifecycle.
type AppointmentStatus string

const (
	AppointmentStatusScheduled AppointmentStatus = "SCHEDULED"
	AppointmentStatusArrived   AppointmentStatus = "ARRIVED"
	AppointmentStatusUnloading AppointmentStatus = "UNLOADING"
	AppointmentStatusCompleted AppointmentStatus = "COMPLETED"
	AppointmentStatusCancelled AppointmentStatus = "CANCELLED"
)

// LPNStatus represents pallet container lifecycle.
type LPNStatus string

const (
	LPNStatusStaged         LPNStatus = "STAGED"
	LPNStatusStored         LPNStatus = "STORED"
	LPNStatusPicked         LPNStatus = "PICKED"
	LPNStatusShipped        LPNStatus = "SHIPPED"
	LPNStatusDecommissioned LPNStatus = "DECOMMISSIONED"
)

// PalletType represents physical pallet container material or structure.
type PalletType string

const (
	PalletTypeWooden  PalletType = "WOODEN"
	PalletTypePlastic PalletType = "PLASTIC"
	PalletTypeMetal   PalletType = "METAL"
	PalletTypeCage    PalletType = "CAGE"
)

// Stock movement reference type for LPN mass putaway
const StockRefLPNPutaway = "PUTAWAY-LPN"

// InboundDock represents a warehouse unloading/loading bay.
type InboundDock struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	WarehouseID   uuid.UUID       `json:"warehouse_id"`
	WarehouseName *string         `json:"warehouse_name,omitempty"`
	DockCode      string          `json:"dock_code"`
	DockName      string          `json:"dock_name"`
	DockType      DockType        `json:"dock_type"`
	MaxTonnage    decimal.Decimal `json:"max_tonnage"`
	Status        DockStatus      `json:"status"`
	Notes         *string         `json:"notes,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// DockAppointment represents scheduled truck delivery slot at a warehouse dock.
type DockAppointment struct {
	ID                uuid.UUID         `json:"id"`
	TenantID          uuid.UUID         `json:"tenant_id"`
	WarehouseID       uuid.UUID         `json:"warehouse_id"`
	WarehouseName     *string           `json:"warehouse_name,omitempty"`
	DockID            *uuid.UUID        `json:"dock_id,omitempty"`
	DockCode          *string           `json:"dock_code,omitempty"`
	DockName          *string           `json:"dock_name,omitempty"`
	AppointmentNumber string            `json:"appointment_number"`
	VendorName        string            `json:"vendor_name"`
	VehiclePlate      string            `json:"vehicle_plate"`
	DriverName        string            `json:"driver_name"`
	DriverPhone       *string           `json:"driver_phone,omitempty"`
	POReference       *string           `json:"po_reference,omitempty"`
	EstimatedArrival  time.Time         `json:"estimated_arrival"`
	ActualArrival     *time.Time        `json:"actual_arrival,omitempty"`
	StartUnloadingAt  *time.Time        `json:"start_unloading_at,omitempty"`
	CompletedAt       *time.Time        `json:"completed_at,omitempty"`
	Status            AppointmentStatus `json:"status"`
	Notes             *string           `json:"notes,omitempty"`
	CreatedBy         *uuid.UUID        `json:"created_by,omitempty"`
	CreatedByName     *string           `json:"created_by_name,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// StockLPN represents a pallet container (License Plate Number).
type StockLPN struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	WarehouseID   uuid.UUID       `json:"warehouse_id"`
	WarehouseName *string         `json:"warehouse_name,omitempty"`
	LPNCode       string          `json:"lpn_code"`
	LocationID    uuid.UUID       `json:"location_id"`
	LocationCode  *string         `json:"location_code,omitempty"`
	LocationName  *string         `json:"location_name,omitempty"`
	PalletType    PalletType      `json:"pallet_type"`
	Status        LPNStatus       `json:"status"`
	MaxWeightKg   decimal.Decimal `json:"max_weight_kg"`
	TotalWeightKg decimal.Decimal `json:"total_weight_kg"`
	Notes         *string         `json:"notes,omitempty"`
	CreatedBy     *uuid.UUID      `json:"created_by,omitempty"`
	CreatedByName *string         `json:"created_by_name,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// StockLPNItem represents an item lot contained within an LPN.
type StockLPNItem struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	LPNID       uuid.UUID       `json:"lpn_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName *string         `json:"product_name,omitempty"`
	ProductSKU  *string         `json:"product_sku,omitempty"`
	BatchID     uuid.UUID       `json:"batch_id"`
	BatchNumber *string         `json:"batch_number,omitempty"`
	ExpiryDate  *time.Time      `json:"expiry_date,omitempty"`
	Quantity    decimal.Decimal `json:"quantity"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// StockLPNDetail combines the LPN pallet header with its contents.
type StockLPNDetail struct {
	LPN   StockLPN       `json:"lpn"`
	Items []StockLPNItem `json:"items"`
}

// Request DTOs

// CreateDockRequest carries parameters for creating a new dock bay.
type CreateDockRequest struct {
	WarehouseID uuid.UUID       `json:"warehouse_id"`
	DockCode    string          `json:"dock_code"` // Optional; generated if blank
	DockName    string          `json:"dock_name"`
	DockType    DockType        `json:"dock_type"`
	MaxTonnage  decimal.Decimal `json:"max_tonnage"`
	Notes       *string         `json:"notes,omitempty"`
}

// Validate checks CreateDockRequest fields.
func (r *CreateDockRequest) Validate() error {
	if r.WarehouseID == uuid.Nil {
		return ErrInvalidInput
	}
	if strings.TrimSpace(r.DockName) == "" {
		return ErrInvalidInput
	}
	if r.DockType != "" && r.DockType != DockTypeInbound && r.DockType != DockTypeOutbound && r.DockType != DockTypeCrossDock {
		return ErrInvalidInput
	}
	if r.MaxTonnage.IsNegative() {
		return ErrInvalidInput
	}
	return nil
}

// UpdateDockStatusRequest carries status change for an inbound dock.
type UpdateDockStatusRequest struct {
	Status DockStatus `json:"status"`
	Notes  *string    `json:"notes,omitempty"`
}

// Validate checks UpdateDockStatusRequest fields.
func (r *UpdateDockStatusRequest) Validate() error {
	if r.Status != DockStatusAvailable && r.Status != DockStatusOccupied && r.Status != DockStatusMaintenance {
		return ErrInvalidStatus
	}
	return nil
}

// CreateAppointmentRequest carries parameters for scheduling a truck arrival.
type CreateAppointmentRequest struct {
	WarehouseID      uuid.UUID  `json:"warehouse_id"`
	DockID           *uuid.UUID `json:"dock_id,omitempty"`
	VendorName       string     `json:"vendor_name"`
	VehiclePlate     string     `json:"vehicle_plate"`
	DriverName       string     `json:"driver_name"`
	DriverPhone      *string    `json:"driver_phone,omitempty"`
	POReference      *string    `json:"po_reference,omitempty"`
	EstimatedArrival time.Time  `json:"estimated_arrival"`
	Notes            *string    `json:"notes,omitempty"`
}

// Validate checks CreateAppointmentRequest fields.
func (r *CreateAppointmentRequest) Validate() error {
	if r.WarehouseID == uuid.Nil {
		return ErrInvalidInput
	}
	if strings.TrimSpace(r.VendorName) == "" || strings.TrimSpace(r.VehiclePlate) == "" || strings.TrimSpace(r.DriverName) == "" {
		return ErrInvalidInput
	}
	if r.EstimatedArrival.IsZero() {
		return ErrInvalidInput
	}
	return nil
}

// AssignDockRequest carries dock assignment for an appointment.
type AssignDockRequest struct {
	DockID uuid.UUID `json:"dock_id"`
}

// Validate checks AssignDockRequest fields.
func (r *AssignDockRequest) Validate() error {
	if r.DockID == uuid.Nil {
		return ErrInvalidInput
	}
	return nil
}

// UpdateAppointmentStatusRequest carries appointment status progression.
type UpdateAppointmentStatusRequest struct {
	Status AppointmentStatus `json:"status"`
	Notes  *string           `json:"notes,omitempty"`
}

// Validate checks UpdateAppointmentStatusRequest fields.
func (r *UpdateAppointmentStatusRequest) Validate() error {
	if r.Status != AppointmentStatusScheduled &&
		r.Status != AppointmentStatusArrived &&
		r.Status != AppointmentStatusUnloading &&
		r.Status != AppointmentStatusCompleted &&
		r.Status != AppointmentStatusCancelled {
		return ErrInvalidStatus
	}
	return nil
}

// CreateLPNRequest carries parameters for registering a pallet container.
type CreateLPNRequest struct {
	WarehouseID uuid.UUID       `json:"warehouse_id"`
	LocationID  uuid.UUID       `json:"location_id"`
	LPNCode     string          `json:"lpn_code"` // Optional; generated if blank
	PalletType  PalletType      `json:"pallet_type"`
	MaxWeightKg decimal.Decimal `json:"max_weight_kg"`
	Notes       *string         `json:"notes,omitempty"`
}

// Validate checks CreateLPNRequest fields.
func (r *CreateLPNRequest) Validate() error {
	if r.WarehouseID == uuid.Nil || r.LocationID == uuid.Nil {
		return ErrInvalidInput
	}
	if r.PalletType != "" &&
		r.PalletType != PalletTypeWooden &&
		r.PalletType != PalletTypePlastic &&
		r.PalletType != PalletTypeMetal &&
		r.PalletType != PalletTypeCage {
		return ErrInvalidInput
	}
	if r.MaxWeightKg.IsNegative() {
		return ErrInvalidInput
	}
	return nil
}

// AddLPNItemRequest carries parameters for packing a batch quantity into an LPN.
type AddLPNItemRequest struct {
	ProductID uuid.UUID       `json:"product_id"`
	BatchID   uuid.UUID       `json:"batch_id"`
	Quantity  decimal.Decimal `json:"quantity"`
}

// Validate checks AddLPNItemRequest fields.
func (r *AddLPNItemRequest) Validate() error {
	if r.ProductID == uuid.Nil || r.BatchID == uuid.Nil {
		return ErrInvalidInput
	}
	if !r.Quantity.IsPositive() {
		return ErrInvalidInput
	}
	return nil
}

// MoveLPNRequest carries destination location for atomic forklift putaway.
type MoveLPNRequest struct {
	TargetLocationID uuid.UUID `json:"target_location_id"`
	Notes            *string   `json:"notes,omitempty"`
}

// Validate checks MoveLPNRequest fields.
func (r *MoveLPNRequest) Validate() error {
	if r.TargetLocationID == uuid.Nil {
		return ErrInvalidInput
	}
	return nil
}

// Domain sentinel errors
var (
	ErrDockNotFound        = errors.New("inbound dock not found")
	ErrDockOccupied        = errors.New("dock is currently occupied or undergoing unloading")
	ErrAppointmentNotFound = errors.New("dock appointment not found")
	ErrLPNNotFound         = errors.New("stock lpn not found")
	ErrLPNEmpty            = errors.New("cannot move empty lpn with no stock items")
	ErrInvalidLocationType = errors.New("target location must be an internal rack location")
)

// WMSDockLPNRepository defines persistence operations for docks, appointments, and LPNs.
type WMSDockLPNRepository interface {
	CreateDock(ctx context.Context, tenantID uuid.UUID, req CreateDockRequest) (*InboundDock, error)
	GetDockByID(ctx context.Context, tenantID, id uuid.UUID) (*InboundDock, error)
	ListDocks(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *DockStatus) ([]InboundDock, error)
	UpdateDockStatus(ctx context.Context, tenantID, id uuid.UUID, req UpdateDockStatusRequest) (*InboundDock, error)

	CreateAppointment(ctx context.Context, tenantID, userID uuid.UUID, req CreateAppointmentRequest) (*DockAppointment, error)
	GetAppointmentByID(ctx context.Context, tenantID, id uuid.UUID) (*DockAppointment, error)
	ListAppointments(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *AppointmentStatus) ([]DockAppointment, error)
	AssignDockToAppointment(ctx context.Context, tenantID, appointmentID uuid.UUID, req AssignDockRequest) (*DockAppointment, error)
	UpdateAppointmentStatus(ctx context.Context, tenantID, appointmentID uuid.UUID, req UpdateAppointmentStatusRequest) (*DockAppointment, error)

	CreateLPN(ctx context.Context, tenantID, userID uuid.UUID, req CreateLPNRequest) (*StockLPN, error)
	GetLPNByID(ctx context.Context, tenantID, id uuid.UUID) (*StockLPNDetail, error)
	ListLPNs(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, locationID *uuid.UUID, status *LPNStatus) ([]StockLPN, error)
	AddLPNItem(ctx context.Context, tenantID, lpnID uuid.UUID, req AddLPNItemRequest) (*StockLPNItem, error)
	MoveLPN(ctx context.Context, tenantID, userID, lpnID uuid.UUID, req MoveLPNRequest) (*StockLPNDetail, error)
}
