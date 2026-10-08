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
	ErrPickWaveNotFound      = errors.New("gelombang pengambilan (pick wave) tidak ditemukan")
	ErrNoOrdersForWave       = errors.New("tidak ada pesanan yang cocok untuk dibuatkan pick wave")
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
	IsFreeItem       bool                  `json:"is_free_item"`
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

// ShortageTicket carries shortage details and recommended next FEFO batch (PDF-03/04).
type ShortageTicket struct {
	TaskItemID       uuid.UUID        `json:"task_item_id"`
	ProductID        uuid.UUID        `json:"product_id"`
	RequestedQty     decimal.Decimal  `json:"requested_qty"`
	PickedQty        decimal.Decimal  `json:"picked_qty"`
	ShortageQty      decimal.Decimal  `json:"shortage_qty"`
	AlternativeBatch *BatchAllocation `json:"alternative_batch,omitempty"`
}

// ShortageReportRequest carries parameters when picker encounters a shortage.
type ShortageReportRequest struct {
	TaskItemID uuid.UUID       `json:"task_item_id"`
	PickedQty  decimal.Decimal `json:"picked_qty"`
	Reason     string          `json:"reason"`
}

// PickWaveStatus represents the lifecycle of a wave picking batch.
type PickWaveStatus string

const (
	PickWaveStatusOpen       PickWaveStatus = "OPEN"
	PickWaveStatusReleased   PickWaveStatus = "RELEASED"
	PickWaveStatusInProgress PickWaveStatus = "IN_PROGRESS"
	PickWaveStatusCompleted  PickWaveStatus = "COMPLETED"
	PickWaveStatusCancelled  PickWaveStatus = "CANCELLED"
)

// PickWave represents a wave release grouping multiple orders by route/courier and type (PDF-05, OCA §1.3).
type PickWave struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	WarehouseID    uuid.UUID      `json:"warehouse_id"`
	WarehouseName  *string        `json:"warehouse_name,omitempty"`
	WaveNumber     string         `json:"wave_number"`
	OrderType      string         `json:"order_type"` // DIRECT_DO, SALES_ORDER, MARKETPLACE, TRANSFER
	ExpeditionName *string        `json:"expedition_name,omitempty"`
	RouteZone      *string        `json:"route_zone,omitempty"`
	Status         PickWaveStatus `json:"status"`
	PickerID       *uuid.UUID     `json:"picker_id,omitempty"`
	PickerName     *string        `json:"picker_name,omitempty"`
	CreatedBy      *uuid.UUID     `json:"created_by,omitempty"`
	CreatedByName  *string        `json:"created_by_name,omitempty"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	Notes          *string        `json:"notes,omitempty"`
	TotalOrders    int            `json:"total_orders"`
	TotalLines     int            `json:"total_lines"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type CreatePickWaveRequest struct {
	WarehouseID      uuid.UUID   `json:"warehouse_id"`
	OrderType        string      `json:"order_type"` // DIRECT_DO, SALES_ORDER, MARKETPLACE, TRANSFER
	ExpeditionName   *string     `json:"expedition_name,omitempty"`
	RouteZone        *string     `json:"route_zone,omitempty"`
	PickerID         *uuid.UUID  `json:"picker_id,omitempty"`
	Notes            *string     `json:"notes,omitempty"`
	DeliveryOrderIDs []uuid.UUID `json:"delivery_order_ids,omitempty"`
}

type PickWaveDetail struct {
	Wave         PickWave            `json:"wave"`
	PickingTasks []PickingTaskDetail `json:"picking_tasks"`
}

// WMSOutboundRepository defines the persistence port for Sprint 3 outbound operations.
type WMSOutboundRepository interface {
	GetOrCreatePickingTask(ctx context.Context, tenantID uuid.UUID, doID uuid.UUID) (*PickingTaskDetail, error)
	GetPickingTaskByDO(ctx context.Context, tenantID uuid.UUID, doID uuid.UUID) (*PickingTaskDetail, error)
	UpdatePickingTaskStatus(ctx context.Context, tenantID, taskID uuid.UUID, status PickingTaskStatus, pickerID *uuid.UUID) error
	RecordPickingItemProgress(ctx context.Context, tenantID, taskItemID uuid.UUID, pickedQty decimal.Decimal) error
	UpdateDOPackScan(ctx context.Context, tenantID, doID, itemID uuid.UUID, addPackedQty decimal.Decimal) (*DeliveryOrderItem, error)
	CompleteDOPacking(ctx context.Context, tenantID, doID, userID uuid.UUID, req PackCompleteRequest) (*DeliveryOrder, error)
	CreatePickWave(ctx context.Context, tenantID uuid.UUID, createdBy *uuid.UUID, req CreatePickWaveRequest) (*PickWave, error)
	ListPickWaves(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, orderType, expeditionName *string, status *PickWaveStatus) ([]PickWave, error)
	GetPickWaveByID(ctx context.Context, tenantID, waveID uuid.UUID) (*PickWaveDetail, error)
	ReleasePickWave(ctx context.Context, tenantID, waveID uuid.UUID, pickerID *uuid.UUID) (*PickWave, error)
}
