package domain

// Sprint 4 Outbound Domain: Shipping Manifests, Loading Scan, Courier Multi-DO Consolidation,
// Driver Signature Canvas, and Outbound Operational KPIs (ADR-014 Invariant 1, Master PRD §3.2, §4.1).

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ShippingManifestStatus represents the lifecycle of a shipping manifest.
type ShippingManifestStatus string

const (
	ShippingManifestStatusStaged     ShippingManifestStatus = "STAGED"
	ShippingManifestStatusLoaded     ShippingManifestStatus = "LOADED"
	ShippingManifestStatusDispatched ShippingManifestStatus = "DISPATCHED"
	ShippingManifestStatusCancelled  ShippingManifestStatus = "CANCELLED"
)

// DeliveryOrderStatusStaged marks a delivery order that is consolidated and staged into a shipping manifest.
const DeliveryOrderStatusStaged DeliveryOrderStatus = "STAGED"

// ShippingManifest represents courier multi-DO shipping manifest header.
type ShippingManifest struct {
	ID                 uuid.UUID              `json:"id"`
	TenantID           uuid.UUID              `json:"tenant_id"`
	WarehouseID        uuid.UUID              `json:"warehouse_id"`
	WarehouseName      *string                `json:"warehouse_name,omitempty"`
	ManifestNumber     string                 `json:"manifest_number"`
	ExpeditionName     string                 `json:"expedition_name"`
	DriverName         string                 `json:"driver_name"`
	VehiclePlate       string                 `json:"vehicle_plate"`
	DriverPhone        *string                `json:"driver_phone,omitempty"`
	TotalPackages      int                    `json:"total_packages"`
	TotalWeightKg      decimal.Decimal        `json:"total_weight_kg"`
	Status             ShippingManifestStatus `json:"status"`
	DriverSignatureSVG *string                `json:"driver_signature_svg,omitempty"`
	Notes              *string                `json:"notes,omitempty"`
	CreatedBy          *uuid.UUID             `json:"created_by,omitempty"`
	CreatedByName      *string                `json:"created_by_name,omitempty"`
	DispatchedBy       *uuid.UUID             `json:"dispatched_by,omitempty"`
	DispatchedByName   *string                `json:"dispatched_by_name,omitempty"`
	DispatchedAt       *time.Time             `json:"dispatched_at,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// ShippingManifestItem represents a delivery order attached to the manifest.
type ShippingManifestItem struct {
	DeliveryOrderID uuid.UUID        `json:"delivery_order_id"`
	DONumber        string           `json:"do_number"`
	CustomerName    string           `json:"customer_name"`
	DestinationCity string           `json:"destination_city"`
	PackageWeightKg *decimal.Decimal `json:"package_weight_kg,omitempty"`
	PackagingType   *string          `json:"packaging_type,omitempty"`
	Scanned         bool             `json:"scanned"`
	ScannedAt       *time.Time       `json:"scanned_at,omitempty"`
	ScannedByName   *string          `json:"scanned_by_name,omitempty"`
}

// ShippingManifestDetail combines the manifest header and its delivery order items.
type ShippingManifestDetail struct {
	Manifest ShippingManifest       `json:"manifest"`
	Items    []ShippingManifestItem `json:"items"`
}

// CreateShippingManifestRequest carries parameters for creating a new courier manifest.
type CreateShippingManifestRequest struct {
	WarehouseID      uuid.UUID   `json:"warehouse_id"`
	ExpeditionName   string      `json:"expedition_name"`
	DriverName       string      `json:"driver_name"`
	VehiclePlate     string      `json:"vehicle_plate"`
	DriverPhone      *string     `json:"driver_phone,omitempty"`
	DeliveryOrderIDs []uuid.UUID `json:"delivery_order_ids"`
	Notes            *string     `json:"notes,omitempty"`
}

// Validate checks required fields of CreateShippingManifestRequest.
func (r *CreateShippingManifestRequest) Validate() error {
	if r.WarehouseID == uuid.Nil {
		return ErrInvalidInput
	}
	if strings.TrimSpace(r.ExpeditionName) == "" || strings.TrimSpace(r.DriverName) == "" || strings.TrimSpace(r.VehiclePlate) == "" {
		return ErrInvalidInput
	}
	if len(r.DeliveryOrderIDs) == 0 {
		return ErrManifestEmpty
	}
	return nil
}

// LoadingScanRequest carries barcode scanned during truck loading.
type LoadingScanRequest struct {
	Barcode string `json:"barcode"`
}

// DispatchShippingManifestRequest carries driver signature and notes for dispatch handover.
type DispatchShippingManifestRequest struct {
	DriverSignatureSVG string  `json:"driver_signature_svg"`
	Notes              *string `json:"notes,omitempty"`
}

// Validate checks whether driver signature SVG is populated.
func (r *DispatchShippingManifestRequest) Validate() error {
	if len(strings.TrimSpace(r.DriverSignatureSVG)) <= 10 {
		return ErrManifestSignatureRequired
	}
	return nil
}

// WMSOutboundKPISummary carries 8 operational SLA and quality metrics.
type WMSOutboundKPISummary struct {
	DockToStockAvgMinutes   float64 `json:"dock_to_stock_avg_minutes"`
	ReceivingAccuracyPct    float64 `json:"receiving_accuracy_pct"`
	POCompliancePct         float64 `json:"po_compliance_pct"`
	BacklogInboundCount     int     `json:"backlog_inbound_count"`
	OrderToDispatchAvgHours float64 `json:"order_to_dispatch_avg_hours"`
	PickingAccuracyPct      float64 `json:"picking_accuracy_pct"`
	OnTimeShipmentPct       float64 `json:"on_time_shipment_pct"`
	BacklogOutboundCount    int     `json:"backlog_outbound_count"`
}

// Domain sentinel errors for shipping manifests.
var (
	ErrManifestNotFound          = errors.New("shipping manifest tidak ditemukan")
	ErrInvalidManifestStatus     = errors.New("status manifest tidak valid untuk operasi ini")
	ErrManifestSignatureRequired = errors.New("tanda tangan sopir ekspedisi wajib diisi")
	ErrDOMisload                 = errors.New("nomor Surat Jalan tidak terdaftar dalam manifest ini atau ekspedisi berbeda")
	ErrManifestEmpty             = errors.New("manifest minimal harus memuat 1 Surat Jalan")
)

// WMSManifestRepository defines persistence operations for shipping manifests.
type WMSManifestRepository interface {
	CreateShippingManifest(ctx context.Context, tenantID, userID uuid.UUID, req CreateShippingManifestRequest) (*ShippingManifest, error)
	GetShippingManifestByID(ctx context.Context, tenantID, id uuid.UUID) (*ShippingManifestDetail, error)
	ListShippingManifests(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *ShippingManifestStatus, expeditionName *string) ([]ShippingManifest, error)
	ScanDOLoading(ctx context.Context, tenantID, manifestID uuid.UUID, barcode string, userID uuid.UUID) (*ShippingManifestDetail, error)
	DispatchShippingManifest(ctx context.Context, tenantID, manifestID, userID uuid.UUID, req DispatchShippingManifestRequest) (*ShippingManifest, error)
	GetWMSOutboundKPIs(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) (*WMSOutboundKPISummary, error)
}
