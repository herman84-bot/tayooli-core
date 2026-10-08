package domain

// Sprint 3 Outbound Domain: Picking Tasks, Pack Station Scan Verification,
// FEFO Auto-Allocation, Damaged Report & Alternative Recommendation (ADR-014 Invariant 1, Sentry-WMS §1.3, OCA §1.3).

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrPackStationIncomplete = errors.New("semua item harus dipindai 100% sebelum menyelesaikan pengemasan")
	ErrPackBarcodeMismatch   = errors.New("barcode tidak cocok dengan item pesanan ini atau item sudah selesai dikemas")
	ErrPackQtyExceeded       = errors.New("jumlah pemindaian melebihi sisa yang harus dikemas")
	ErrPickingTaskNotFound   = errors.New("dokumen picking task tidak ditemukan")
	ErrPickingItemNotFound   = errors.New("item picking tidak ditemukan")
)

type PickingTaskStatus string

const (
	PickingTaskStatusPending    PickingTaskStatus = "PENDING"
	PickingTaskStatusInProgress PickingTaskStatus = "IN_PROGRESS"
	PickingTaskStatusCompleted  PickingTaskStatus = "COMPLETED"
	PickingTaskStatusShortage   PickingTaskStatus = "SHORTAGE"
	PickingTaskStatusCancelled  PickingTaskStatus = "CANCELLED"
)

type PickingTaskItemStatus string

const (
	PickingItemStatusPending  PickingTaskItemStatus = "PENDING"
	PickingItemStatusPicked   PickingTaskItemStatus = "PICKED"
	PickingItemStatusShortage PickingTaskItemStatus = "SHORTAGE"
	PickingItemStatusDamaged  PickingTaskItemStatus = "DAMAGED"
)

// PickingTask represents picking instructions sorted by shelf order.
type PickingTask struct {
	ID              uuid.UUID         `json:"id"`
	TenantID        uuid.UUID         `json:"tenant_id"`
	DeliveryOrderID uuid.UUID         `json:"delivery_order_id"`
	TaskNumber      string            `json:"task_number"`
	Status          PickingTaskStatus `json:"status"`
	PickerID        *uuid.UUID        `json:"picker_id,omitempty"`
	PickerName      *string           `json:"picker_name,omitempty"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	Notes           *string           `json:"notes,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// PickingTaskItem represents a single shelf/batch line to be picked.
type PickingTaskItem struct {
	ID               uuid.UUID             `json:"id"`
	TenantID         uuid.UUID             `json:"tenant_id"`
	TaskID           uuid.UUID             `json:"task_id"`
	ProductID        uuid.UUID             `json:"product_id"`
	ProductName      *string               `json:"product_name,omitempty"`
	ProductSKU       *string               `json:"product_sku,omitempty"`
	BatchID          uuid.UUID             `json:"batch_id"`
	BatchNumber      *string               `json:"batch_number,omitempty"`
	ExpiryDate       *time.Time            `json:"expiry_date,omitempty"`
	SourceLocationID uuid.UUID             `json:"source_location_id"`
	LocationCode     *string               `json:"location_code,omitempty"`
	RequestedQty     decimal.Decimal       `json:"requested_qty"`
	PickedQty        decimal.Decimal       `json:"picked_qty"`
	DamagedQty       decimal.Decimal       `json:"damaged_qty"`
	Status           PickingTaskItemStatus `json:"status"`
	ShelfOrder       int                   `json:"shelf_order"`
	CreatedAt        time.Time             `json:"created_at"`
}

// PickingTaskDetail combines the header and lines for printing picking list.
type PickingTaskDetail struct {
	Task          PickingTask         `json:"task"`
	DeliveryOrder DeliveryOrder       `json:"delivery_order"`
	Items         []PickingTaskItem   `json:"items"`
}

// PackScanRequest carries interactive barcode scan data at the packing station.
type PackScanRequest struct {
	Barcode  string          `json:"barcode"`
	Quantity decimal.Decimal `json:"quantity"` // defaults to 1 if <= 0
}

// PackScanResult reports the scan outcome and order completeness.
type PackScanResult struct {
	ItemID         uuid.UUID       `json:"item_id"`
	ProductID      uuid.UUID       `json:"product_id"`
	ProductName    string          `json:"product_name"`
	ProductSKU     string          `json:"product_sku"`
	ScannedQty     decimal.Decimal `json:"scanned_qty"`
	PackedQty      decimal.Decimal `json:"packed_qty"`
	RequestedQty   decimal.Decimal `json:"requested_qty"`
	ItemCompleted  bool            `json:"item_completed"`
	OrderCompleted bool            `json:"order_completed"`
	TotalItems     int             `json:"total_items"`
	PackedItems    int             `json:"packed_items"`
}

// PackCompleteRequest finalizes packing with package dimensions & weight.
type PackCompleteRequest struct {
	PackageWeightKg *decimal.Decimal `json:"package_weight_kg,omitempty"`
	PackageLengthCm *decimal.Decimal `json:"package_length_cm,omitempty"`
	PackageWidthCm  *decimal.Decimal `json:"package_width_cm,omitempty"`
	PackageHeightCm *decimal.Decimal `json:"package_height_cm,omitempty"`
	PackagingType   *string          `json:"packaging_type,omitempty"`
}

// PickingDamagedReportRequest reports stock damaged during picking (PDF-03/04).
type PickingDamagedReportRequest struct {
	ProductID        uuid.UUID       `json:"product_id"`
	BatchID          uuid.UUID       `json:"batch_id"`
	SourceLocationID uuid.UUID       `json:"source_location_id"`
	DamagedQty       decimal.Decimal `json:"damaged_qty"`
	Reason           string          `json:"reason"`
}

// PickingDamagedReportResult contains movement to QRN and recommended alternative FEFO batch.
type PickingDamagedReportResult struct {
	MovementID       uuid.UUID        `json:"movement_id"`
	QuarantineLocID  uuid.UUID        `json:"quarantine_loc_id"`
	DamagedQty       decimal.Decimal  `json:"damaged_qty"`
	AlternativeBatch *BatchAllocation `json:"alternative_batch,omitempty"`
}

// WMSOutboundRepository defines the persistence port for Sprint 3 outbound operations.
type WMSOutboundRepository interface {
	GetOrCreatePickingTask(ctx context.Context, tenantID uuid.UUID, doID uuid.UUID) (*PickingTaskDetail, error)
	GetPickingTaskByDO(ctx context.Context, tenantID uuid.UUID, doID uuid.UUID) (*PickingTaskDetail, error)
	UpdatePickingTaskStatus(ctx context.Context, tenantID, taskID uuid.UUID, status PickingTaskStatus, pickerID *uuid.UUID) error
	RecordPickingItemProgress(ctx context.Context, tenantID, taskItemID uuid.UUID, pickedQty decimal.Decimal) error
	UpdateDOPackScan(ctx context.Context, tenantID, doID, itemID uuid.UUID, addPackedQty decimal.Decimal) (*DeliveryOrderItem, error)
	CompleteDOPacking(ctx context.Context, tenantID, doID, userID uuid.UUID, req PackCompleteRequest) (*DeliveryOrder, error)
}
