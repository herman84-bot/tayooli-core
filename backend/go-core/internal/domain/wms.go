package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Domain errors for WMS
var (
	ErrWarehouseNotFound     = errors.New("warehouse not found")
	ErrLocationNotFound      = errors.New("location not found")
	// ErrSourceLocationRequired: a transfer item has no source rack, so stock
	// cannot be deducted from a concrete location.
	ErrSourceLocationRequired = errors.New("transfer item source location required")
	ErrInsufficientStock     = errors.New("insufficient stock at location")
	ErrBarcodeNotFound       = errors.New("barcode or external sku not found")
	ErrUnauthorizedWarehouse = errors.New("user is not authorized to access this warehouse")
	ErrInvalidTransferStatus = errors.New("invalid stock transfer status transition")
	ErrSelfApprovalForbidden = errors.New("requester cannot approve or reject their own transfer")
	ErrRejectionReasonRequired = errors.New("rejection reason is required")
	// ErrTransferNotDraft: cancel is only meaningful while the transfer is still
	// a DRAFT. Once submitted/approved/dispatched it must go through reject or
	// receive so the ledger stays consistent.
	ErrTransferNotDraft = errors.New("only draft stock transfer can be cancelled")
	ErrDeliveryOrderNotFound = errors.New("delivery order not found")
	ErrTransferNotFound      = errors.New("stock transfer not found")
	ErrOpnameNotFound        = errors.New("stock opname not found")
	ErrScrapNotFound         = errors.New("stock scrap not found")
	ErrInvalidOpnameStatus   = errors.New("invalid stock opname status transition")
	ErrDuplicateMarketplaceOrder = errors.New("duplicate marketplace order")
	ErrBatchNotFound            = errors.New("marketplace import batch not found")
	ErrMarketplaceOrderNotFound = errors.New("marketplace order not found")
	ErrSKUMappingNotFound       = errors.New("sku mapping not found")

	// Stock receipts (Barang Masuk)
	ErrStockReceiptNotFound         = errors.New("stock receipt not found")
	ErrStockReceiptNotDraft         = errors.New("stock receipt is not in DRAFT status")
	ErrStockReceiptAlreadyCancelled = errors.New("stock receipt is already cancelled")
	ErrStockReceiptStockConsumed    = errors.New("received stock has already been used and cannot be reversed")

	// Batch ledger (ADR-014 Invariant 1): every movement must carry a batch.
	ErrBatchRequired        = errors.New("stock movement batch is required")
	ErrStockBatchNotFound   = errors.New("stock batch not found")
	ErrBatchExpiryMismatch  = errors.New("batch already exists with a different expiry date")
	ErrPutawayReasonRequired = errors.New("reason is required when putaway location differs from default rack")
	ErrInvalidPutawayLocation = errors.New("putaway destination must be an internal rack in the same warehouse")
	ErrActorRequired        = errors.New("authenticated user is required")
	ErrReceiptNotOnHold     = errors.New("receipt has no batch awaiting release")
	ErrBatchOnHold          = errors.New("batch is on hold")
)

// StockBatchStatus controls whether a batch may be allocated to sales.
type StockBatchStatus string

const (
	StockBatchStatusReleased  StockBatchStatus = "RELEASED"
	StockBatchStatusAvailable StockBatchStatus = "AVAILABLE"
	StockBatchStatusOnHold    StockBatchStatus = "ON_HOLD"
)

// LegacyBatchNumber marks the batch backfilled by migration 033 for pre-batch movements.
const LegacyBatchNumber = "LEGACY"

// StockBatch is a lot of one product (KO-1). Expiry drives FEFO (OCA §1.3).
type StockBatch struct {
	ID              uuid.UUID        `json:"id"`
	TenantID        uuid.UUID        `json:"tenant_id"`
	ProductID       uuid.UUID        `json:"product_id"`
	BatchNumber     string           `json:"batch_number"`
	ExpiryDate      *time.Time       `json:"expiry_date,omitempty"`
	SourceReceiptID *uuid.UUID       `json:"source_receipt_id,omitempty"`
	Status          StockBatchStatus `json:"status"`
	IsLegacy        bool             `json:"is_legacy"`
	CreatedBy       *uuid.UUID       `json:"created_by,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
}

// BatchBalance is the on-hand quantity of one batch at one location.
type BatchBalance struct {
	BatchID      uuid.UUID        `json:"batch_id"`
	BatchNumber  string           `json:"batch_number"`
	ExpiryDate   *time.Time       `json:"expiry_date,omitempty"`
	Status       StockBatchStatus `json:"status"`
	LocationID   uuid.UUID        `json:"location_id"`
	LocationCode string           `json:"location_code"`
	ProductID    uuid.UUID        `json:"product_id"`
	ProductName  string           `json:"product_name,omitempty"`
	ProductSKU   string           `json:"product_sku,omitempty"`
	Quantity     decimal.Decimal  `json:"quantity"`
	CreatedAt    time.Time        `json:"-"`
}

// StockReceiptValidationError carries a user-facing (Indonesian) validation message.
// It unwraps to ErrInvalidInput so generic callers still treat it as a 400.
type StockReceiptValidationError struct {
	Msg string
}

func (e *StockReceiptValidationError) Error() string { return e.Msg }
func (e *StockReceiptValidationError) Unwrap() error { return ErrInvalidInput }

// StockReceiptStatus represents inbound goods receipt lifecycle.
type StockReceiptStatus string

const (
	StockReceiptStatusDraft     StockReceiptStatus = "DRAFT"
	StockReceiptStatusPosted    StockReceiptStatus = "POSTED"
	StockReceiptStatusCancelled StockReceiptStatus = "CANCELLED"
)

// LocationType represents the nature of a warehouse location.
type LocationType string

const (
	LocationTypeInternal   LocationType = "INTERNAL"
	LocationTypeVendor     LocationType = "VENDOR"
	LocationTypeCustomer   LocationType = "CUSTOMER"
	LocationTypeTransit    LocationType = "TRANSIT"
	LocationTypeLoss       LocationType = "LOSS"
	LocationTypeScrap      LocationType = "SCRAP"
	LocationTypeProduction LocationType = "PRODUCTION"
	// Staging bins (sentry-wms §1.1): received goods wait here; never pickable for orders.
	LocationTypeStagingInbound  LocationType = "STAGING_INBOUND"
	LocationTypeStagingOutbound LocationType = "STAGING_OUTBOUND"
	LocationTypeQuarantine      LocationType = "QUARANTINE"
)

// StagingInboundCode is the per-warehouse inbound staging bin code.
const StagingInboundCode = "STG-IN"

// SKUMappingType represents the type of external mapping.
type SKUMappingType string

const (
	SKUMappingTypeCustomer    SKUMappingType = "CUSTOMER"
	SKUMappingTypeMarketplace SKUMappingType = "MARKETPLACE"
	SKUMappingTypeVendor      SKUMappingType = "VENDOR"
)

// StockMovementStatus represents ledger movement finality.
type StockMovementStatus string

const (
	StockMovementStatusPending   StockMovementStatus = "PENDING"
	StockMovementStatusDone      StockMovementStatus = "DONE"
	StockMovementStatusCancelled StockMovementStatus = "CANCELLED"
)

// TransferStatus represents inter-warehouse transfer lifecycle.
type TransferStatus string

const (
	TransferStatusDraft           TransferStatus = "DRAFT"
	TransferStatusPendingApproval TransferStatus = "PENDING_APPROVAL"
	TransferStatusApproved        TransferStatus = "APPROVED"
	TransferStatusDispatched      TransferStatus = "DISPATCHED"
	TransferStatusInTransit       TransferStatus = "IN_TRANSIT"
	TransferStatusReceived        TransferStatus = "RECEIVED"
	TransferStatusRejected        TransferStatus = "REJECTED"
	// TransferStatusCancelled: a DRAFT transfer discarded by its requester. No
	// stock ever moved, so nothing is written to the ledger.
	TransferStatusCancelled TransferStatus = "CANCELLED"
)

// DeliveryOrderStatus represents delivery order lifecycle.
type DeliveryOrderStatus string

const (
	DeliveryOrderStatusDraft     DeliveryOrderStatus = "DRAFT"
	DeliveryOrderStatusConfirmed DeliveryOrderStatus = "CONFIRMED"
	DeliveryOrderStatusPicked    DeliveryOrderStatus = "PICKED"
	DeliveryOrderStatusPacked    DeliveryOrderStatus = "PACKED"
	DeliveryOrderStatusShipped   DeliveryOrderStatus = "SHIPPED"
	DeliveryOrderStatusDelivered DeliveryOrderStatus = "DELIVERED"
	DeliveryOrderStatusReturned  DeliveryOrderStatus = "RETURNED"
	DeliveryOrderStatusCancelled DeliveryOrderStatus = "CANCELLED"
)

// StockOpnameStatus represents physical inventory counting lifecycle.
type StockOpnameStatus string

const (
	StockOpnameStatusDraft      StockOpnameStatus = "DRAFT"
	StockOpnameStatusInProgress StockOpnameStatus = "IN_PROGRESS"
	StockOpnameStatusCompleted  StockOpnameStatus = "COMPLETED"
	StockOpnameStatusCancelled  StockOpnameStatus = "CANCELLED"
)

// Reference types for stock movements
const (
	StockRefTransfer      = "TRANSFER"
	StockRefDeliveryOrder = "DELIVERY_ORDER"
	StockRefGoodsReceipt  = "GOODS_RECEIPT"
	StockRefAdjustment    = "ADJUSTMENT"
	StockRefOpname        = "OPNAME"
	StockRefScrap         = "SCRAP"
	StockRefMarketplace   = "MARKETPLACE"
	StockRefPutaway       = "PUTAWAY"
	StockRefPOS           = "POS_SALE"
)

// MarketplaceChannel represents supported e-commerce channels.
type MarketplaceChannel string

const (
	MarketplaceChannelShopee    MarketplaceChannel = "SHOPEE"
	MarketplaceChannelTokopedia MarketplaceChannel = "TOKOPEDIA"
	MarketplaceChannelTikTok    MarketplaceChannel = "TIKTOK"
	MarketplaceChannelLazada    MarketplaceChannel = "LAZADA"
	MarketplaceChannelBlibli    MarketplaceChannel = "BLIBLI"
	MarketplaceChannelOther     MarketplaceChannel = "OTHER"
)

// MarketplaceBatchStatus represents import session lifecycle.
type MarketplaceBatchStatus string

const (
	MarketplaceBatchStatusPending    MarketplaceBatchStatus = "PENDING"
	MarketplaceBatchStatusProcessing MarketplaceBatchStatus = "PROCESSING"
	MarketplaceBatchStatusCompleted  MarketplaceBatchStatus = "COMPLETED"
	MarketplaceBatchStatusFailed     MarketplaceBatchStatus = "FAILED"
)

// MarketplaceOrderStatus represents normalized marketplace order state.
type MarketplaceOrderStatus string

const (
	MarketplaceOrderStatusPending           MarketplaceOrderStatus = "PENDING"
	MarketplaceOrderStatusProcessing        MarketplaceOrderStatus = "PROCESSING"
	MarketplaceOrderStatusCompleted         MarketplaceOrderStatus = "COMPLETED"
	MarketplaceOrderStatusFailed            MarketplaceOrderStatus = "FAILED"
	MarketplaceOrderStatusUnmappedSKU       MarketplaceOrderStatus = "UNMAPPED_SKU"
	MarketplaceOrderStatusStockInsufficient MarketplaceOrderStatus = "STOCK_INSUFFICIENT"
)

// Regional represents an administrative or operational region grouping multiple warehouses.
type Regional struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Warehouse represents a physical or operational warehouse entity.
type Warehouse struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	RegionalID *uuid.UUID `json:"regional_id,omitempty"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Address    *string    `json:"address,omitempty"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// UserWarehouse maps warehouse operators/staff to assigned warehouses.
type UserWarehouse struct {
	UserID      uuid.UUID `json:"user_id"`
	WarehouseID uuid.UUID `json:"warehouse_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	AssignedAt  time.Time `json:"assigned_at"`
}

// WarehouseLocation represents a zone, rack, bin, pallet or virtual location.
type WarehouseLocation struct {
	ID           uuid.UUID        `json:"id"`
	TenantID     uuid.UUID        `json:"tenant_id"`
	WarehouseID  *uuid.UUID       `json:"warehouse_id,omitempty"`
	ParentID     *uuid.UUID       `json:"parent_id,omitempty"`
	Code         string           `json:"code"`
	Barcode      *string          `json:"barcode,omitempty"`
	Name         string           `json:"name"`
	Type         LocationType     `json:"type"`
	IsPallet     bool             `json:"is_pallet"`
	PalletNumber *string          `json:"pallet_number,omitempty"`
	MaxCapacity  *decimal.Decimal `json:"max_capacity,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// ProductBarcode represents physical barcode labels associated with a product.
type ProductBarcode struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	ProductID        uuid.UUID       `json:"product_id"`
	Barcode          string          `json:"barcode"`
	BarcodeSymbology string          `json:"barcode_symbology"`
	UOMName          string          `json:"uom_name"`
	Multiplier       decimal.Decimal `json:"multiplier"`
	CreatedAt        time.Time       `json:"created_at"`
}

// ProductSKUMapping maps external channel or customer SKUs to internal product.
type ProductSKUMapping struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	ProductID    uuid.UUID       `json:"product_id"`
	MappingType  SKUMappingType  `json:"mapping_type"`
	ChannelName  string          `json:"channel_name"`
	ExternalSKU  string          `json:"external_sku"`
	ExternalName *string         `json:"external_name,omitempty"`
	Multiplier   decimal.Decimal `json:"multiplier"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// StockMovement represents an immutable ledger entry of physical or virtual stock transfer.
type StockMovement struct {
	ID                 uuid.UUID           `json:"id"`
	TenantID           uuid.UUID           `json:"tenant_id"`
	MovementNumber     string              `json:"movement_number"`
	ProductID          uuid.UUID           `json:"product_id"`
	ProductName        string              `json:"product_name,omitempty"`
	SKU                string              `json:"sku,omitempty"`
	SourceLocationID   uuid.UUID           `json:"source_location_id"`
	SourceLocationCode string              `json:"source_location_code,omitempty"`
	DestLocationID     uuid.UUID           `json:"dest_location_id"`
	DestLocationCode   string              `json:"dest_location_code,omitempty"`
	Quantity           decimal.Decimal     `json:"quantity"`
	UnitCost           decimal.Decimal     `json:"unit_cost"`
	Status             StockMovementStatus `json:"status"`
	ReferenceType      string              `json:"reference_type"`
	ReferenceID        uuid.UUID           `json:"reference_id"`
	BatchID            *uuid.UUID          `json:"batch_id,omitempty"`
	BatchNumber        string              `json:"batch_number,omitempty"`
	ExecutedBy         *uuid.UUID          `json:"executed_by,omitempty"`
	ExecutedByName     string              `json:"executed_by_name,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
}

type StockSummary struct {
	ProductID     uuid.UUID       `json:"product_id"`
	SKU           string          `json:"sku"`
	ProductName   string          `json:"product_name"`
	WarehouseID   *uuid.UUID      `json:"warehouse_id,omitempty"`
	WarehouseName string          `json:"warehouse_name,omitempty"`
	LocationID    *uuid.UUID      `json:"location_id,omitempty"`
	LocationCode  string          `json:"location_code,omitempty"`
	Quantity      decimal.Decimal `json:"quantity"`
	AllocatedQty  decimal.Decimal `json:"allocated_qty"`
	AvailableQty  decimal.Decimal `json:"available_qty"`
}

// InsufficientStockError carries available and requested quantities for stock failures.
type InsufficientStockError struct {
	Available decimal.Decimal
	Requested decimal.Decimal
	Msg       string
}

func (e *InsufficientStockError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return fmt.Sprintf("stok tidak mencukupi (tersedia: %s, diminta: %s)", e.Available.String(), e.Requested.String())
}

func (e *InsufficientStockError) Unwrap() error {
	return ErrInsufficientStock
}

// StockTransfer represents inter-warehouse transfer header.
type StockTransfer struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	TransferNumber  string         `json:"transfer_number"`
	FromWarehouseID uuid.UUID      `json:"from_warehouse_id"`
	ToWarehouseID   uuid.UUID      `json:"to_warehouse_id"`
	Status          TransferStatus `json:"status"`
	RequestedBy     uuid.UUID      `json:"requested_by"`
	ApprovedBy      *uuid.UUID     `json:"approved_by,omitempty"`
	VehiclePlate    *string        `json:"vehicle_plate,omitempty"`
	DriverName      *string        `json:"driver_name,omitempty"`
	DispatchedAt    *time.Time     `json:"dispatched_at,omitempty"`
	ReceivedAt      *time.Time     `json:"received_at,omitempty"`
	Notes           *string        `json:"notes,omitempty"`
	RejectionReason *string        `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// StockTransferItem represents line items in a stock transfer.
type StockTransferItem struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	TransferID       uuid.UUID       `json:"transfer_id"`
	ProductID        uuid.UUID       `json:"product_id"`
	RequestedQty     decimal.Decimal `json:"requested_qty"`
	SentQty          decimal.Decimal `json:"sent_qty"`
	ReceivedQty      decimal.Decimal `json:"received_qty"`
	SourceLocationID *uuid.UUID      `json:"source_location_id,omitempty"`
	DestLocationID   *uuid.UUID      `json:"dest_location_id,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// DeliveryOrder represents outbound delivery order (Surat Jalan).
type DeliveryOrder struct {
	ID               uuid.UUID           `json:"id"`
	TenantID         uuid.UUID           `json:"tenant_id"`
	SalesOrderID     *uuid.UUID          `json:"sales_order_id"` // nil = direct Surat Jalan (no SO)
	WarehouseID      uuid.UUID           `json:"warehouse_id"`
	DONumber         string              `json:"do_number"`
	Status           DeliveryOrderStatus `json:"status"`
	CustomerID       *uuid.UUID          `json:"customer_id,omitempty"`
	CustomerName     *string             `json:"customer_name,omitempty"`
	CreatedBy        *uuid.UUID          `json:"created_by,omitempty"`
	CreatedByName    *string             `json:"created_by_name,omitempty"`
	ConfirmedBy      *uuid.UUID          `json:"confirmed_by,omitempty"`
	ConfirmedByName  *string             `json:"confirmed_by_name,omitempty"`
	PackedBy         *uuid.UUID          `json:"packed_by,omitempty"`
	PackedByName     *string             `json:"packed_by_name,omitempty"`
	DispatchedBy     *uuid.UUID          `json:"dispatched_by,omitempty"`
	DispatchedByName *string             `json:"dispatched_by_name,omitempty"`
	PackageWeightKg  *decimal.Decimal    `json:"package_weight_kg,omitempty"`
	PackageLengthCm  *decimal.Decimal    `json:"package_length_cm,omitempty"`
	PackageWidthCm   *decimal.Decimal    `json:"package_width_cm,omitempty"`
	PackageHeightCm  *decimal.Decimal    `json:"package_height_cm,omitempty"`
	PackagingType    *string             `json:"packaging_type,omitempty"`
	OrderType        string              `json:"order_type"`
	ExpeditionName   *string             `json:"expedition_name,omitempty"`
	TrackingNumber   *string             `json:"tracking_number,omitempty"`
	DriverName       *string             `json:"driver_name,omitempty"`
	VehiclePlate     *string             `json:"vehicle_plate,omitempty"`
	RecipientName    *string             `json:"recipient_name,omitempty"`
	ReceivedDate     *time.Time          `json:"received_date,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// DeliveryOrderItem represents individual line items dispatched in a delivery order.
type DeliveryOrderItem struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	DeliveryOrderID uuid.UUID       `json:"delivery_order_id"`
	ProductID       uuid.UUID       `json:"product_id"`
	Quantity        decimal.Decimal `json:"quantity"`
	LocationID      uuid.UUID       `json:"location_id"`
	BatchID         *uuid.UUID      `json:"batch_id,omitempty"`
	BatchNumber     *string         `json:"batch_number,omitempty"`
	ExpiryDate      *time.Time      `json:"expiry_date,omitempty"`
	IsFreeItem      bool            `json:"is_free_item"`
	PackedQty       decimal.Decimal `json:"packed_qty"`
	CreatedAt       time.Time       `json:"created_at"`
	ProductName     *string         `json:"product_name,omitempty"`
	ProductSKU      *string         `json:"product_sku,omitempty"`
	LocationCode    *string         `json:"location_code,omitempty"`
}

// StockOpname represents a physical inventory counting header.
type StockOpname struct {
	ID           uuid.UUID         `json:"id"`
	TenantID     uuid.UUID         `json:"tenant_id"`
	WarehouseID  uuid.UUID         `json:"warehouse_id"`
	OpnameNumber string            `json:"opname_number"`
	Status       StockOpnameStatus `json:"status"`
	ConductedBy  uuid.UUID         `json:"conducted_by"`
	ApprovedBy   *uuid.UUID        `json:"approved_by,omitempty"`
	Notes        *string           `json:"notes,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// StockOpnameItem represents physical count item line & discrepancies.
type StockOpnameItem struct {
	ID             uuid.UUID       `json:"id"`
	OpnameID       uuid.UUID       `json:"opname_id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	ProductID      uuid.UUID       `json:"product_id"`
	LocationID     uuid.UUID       `json:"location_id"`
	SystemQty      decimal.Decimal `json:"system_qty"`
	PhysicalQty    decimal.Decimal `json:"physical_qty"`
	DiscrepancyQty decimal.Decimal `json:"discrepancy_qty"`
	Notes          *string         `json:"notes,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// StockScrap represents damaged goods & scrap quarantine ledger entry.
type StockScrap struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	ScrapNumber        string          `json:"scrap_number"`
	WarehouseID        uuid.UUID       `json:"warehouse_id"`
	ProductID          uuid.UUID       `json:"product_id"`
	SourceLocationID   uuid.UUID       `json:"source_location_id"`
	ScrapLocationID    uuid.UUID       `json:"scrap_location_id"`
	Quantity           decimal.Decimal `json:"quantity"`
	Reason             string          `json:"reason"`
	ReportedBy         uuid.UUID       `json:"reported_by"`
	CreatedAt          time.Time       `json:"created_at"`
}

// StockReceiptType identifies where goods originated: PRODUCTION, TRANSFER, or VENDOR.
type StockReceiptType string

const (
	StockReceiptTypeProduction StockReceiptType = "PRODUCTION" // Hasil Produksi (Dapur / Pabrik / Workshop)
	StockReceiptTypeTransfer   StockReceiptType = "TRANSFER"   // Kiriman Transfer Antar-Gudang
	StockReceiptTypeVendor     StockReceiptType = "VENDOR"     // Pembelian dari Pemasok Luar
)

// StockReceipt represents an inbound goods receipt header (Barang Masuk).
// ItemCount / TotalAcceptedQty / TotalRejectedQty are computed aggregates.
type StockReceipt struct {
	ID                uuid.UUID          `json:"id"`
	TenantID          uuid.UUID          `json:"tenant_id"`
	ReceiptNumber     string             `json:"receipt_number"`
	ReceiptType       StockReceiptType   `json:"receipt_type"`
	WarehouseID       uuid.UUID          `json:"warehouse_id"`
	DestLocationID    uuid.UUID          `json:"dest_location_id"`
	FromName          string             `json:"from_name"` // Universal source display name (Produksi / Gudang Asal / Supplier)
	FromWarehouseID   *uuid.UUID         `json:"from_warehouse_id,omitempty"`
	FromWarehouseName *string            `json:"from_warehouse_name,omitempty"`
	SourceRef         *string            `json:"source_ref,omitempty"` // No. Batch / No. SPK / No. Transfer / Surat Jalan
	TransferID        *uuid.UUID         `json:"transfer_id,omitempty"`
	SupplierName      string             `json:"supplier_name,omitempty"` // Backward-compatible alias of FromName
	SupplierRef       *string            `json:"supplier_ref,omitempty"`  // Backward-compatible alias of SourceRef
	Notes             *string            `json:"notes,omitempty"`
	Status            StockReceiptStatus `json:"status"`
	CreatedBy         uuid.UUID          `json:"created_by"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
	PostedBy          *uuid.UUID         `json:"posted_by,omitempty"`
	PostedAt          *time.Time         `json:"posted_at,omitempty"`
	CancelledBy       *uuid.UUID         `json:"cancelled_by,omitempty"`
	CancelledAt       *time.Time         `json:"cancelled_at,omitempty"`
	CancelReason      *string            `json:"cancel_reason,omitempty"`
	ReleasedBy        *uuid.UUID         `json:"released_by,omitempty"`
	ReleasedAt        *time.Time         `json:"released_at,omitempty"`
	CreatedByName     string             `json:"created_by_name,omitempty"`
	PostedByName      string             `json:"posted_by_name,omitempty"`
	CancelledByName   string             `json:"cancelled_by_name,omitempty"`
	ReleasedByName    string             `json:"released_by_name,omitempty"`
	OnHoldBatchCount  int                `json:"on_hold_batch_count"`
	ItemCount         int                `json:"item_count"`
	TotalAcceptedQty  decimal.Decimal    `json:"total_accepted_qty"`
	TotalRejectedQty  decimal.Decimal    `json:"total_rejected_qty"`
}

// StockReceiptItem represents a received product line.
type StockReceiptItem struct {
	ID           uuid.UUID        `json:"id"`
	TenantID     uuid.UUID        `json:"tenant_id"`
	ReceiptID    uuid.UUID        `json:"receipt_id"`
	ProductID    uuid.UUID        `json:"product_id"`
	ProductName  string           `json:"product_name"`
	ProductSKU   string           `json:"product_sku"`
	ExpectedQty  *decimal.Decimal `json:"expected_qty,omitempty"`
	AcceptedQty  decimal.Decimal  `json:"accepted_qty"`
	RejectedQty  decimal.Decimal  `json:"rejected_qty"`
	RejectReason *string          `json:"reject_reason,omitempty"`
	// Batch/lot (KO-1). Empty BatchNumber on post -> AUTO-<GR>-<line>.
	BatchNumber *string    `json:"batch_number,omitempty"`
	ExpiryDate  *time.Time `json:"expiry_date,omitempty"`
	BatchID     *uuid.UUID `json:"batch_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ResolvedProduct represents the resolved master product from a barcode, SKU, or external mapping.
type ResolvedProduct struct {
	ProductID   uuid.UUID       `json:"product_id"`
	SKU         string          `json:"sku"`
	Name        string          `json:"name"`
	Barcode     string          `json:"barcode"`
	ExternalSKU string          `json:"external_sku,omitempty"`
	Multiplier  decimal.Decimal `json:"multiplier"`
	Source      string          `json:"source"` // "SKU", "BARCODE", "MAPPING"
}

// MarketplaceImportBatch represents an ingestion session for marketplace order imports.
type MarketplaceImportBatch struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	BatchNumber     string                 `json:"batch_number"`
	Channel         MarketplaceChannel     `json:"channel"`
	WarehouseID     uuid.UUID              `json:"warehouse_id"`
	FileName        string                 `json:"file_name"`
	TotalOrders     int                    `json:"total_orders"`
	ProcessedOrders int                    `json:"processed_orders"`
	FailedOrders    int                    `json:"failed_orders"`
	UnmappedSKUs    int                    `json:"unmapped_skus"`
	Status          MarketplaceBatchStatus `json:"status"`
	UploadedBy      uuid.UUID              `json:"uploaded_by"`
	CreatedAt       time.Time              `json:"created_at"`
}

// MarketplaceOrder represents a canonical normalized sales order imported from a marketplace channel.
type MarketplaceOrder struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	BatchID         *uuid.UUID             `json:"batch_id,omitempty"`
	WarehouseID     uuid.UUID              `json:"warehouse_id"`
	Channel         MarketplaceChannel     `json:"channel"`
	ExternalOrderID string                 `json:"external_order_id"`
	OrderDate       time.Time              `json:"order_date"`
	CustomerName    *string                `json:"customer_name,omitempty"`
	CustomerPhone   *string                `json:"customer_phone,omitempty"`
	ShippingAddress *string                `json:"shipping_address,omitempty"`
	Courier         *string                `json:"courier,omitempty"`
	TrackingNumber  *string                `json:"tracking_number,omitempty"`
	TotalAmount     decimal.Decimal        `json:"total_amount"`
	ShippingFee     decimal.Decimal        `json:"shipping_fee"`
	MarketplaceFee  decimal.Decimal        `json:"marketplace_fee"`
	NetAmount       decimal.Decimal        `json:"net_amount"`
	Status          MarketplaceOrderStatus `json:"status"`
	SalesOrderID    *uuid.UUID             `json:"sales_order_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	Items           []MarketplaceOrderItem `json:"items,omitempty"`
}

// MarketplaceOrderItem represents a single line item within a marketplace sales order.
type MarketplaceOrderItem struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	OrderID     uuid.UUID       `json:"order_id"`
	ExternalSKU string          `json:"external_sku"`
	ProductID   *uuid.UUID      `json:"product_id,omitempty"`
	ItemName    string          `json:"item_name"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Subtotal    decimal.Decimal `json:"subtotal"`
	IsMapped    bool            `json:"is_mapped"`
}

// WMSRepository defines database operations for WMS entities.
type WMSRepository interface {
	WMSBatchRepository
	WMSQCRepository
	WMSOutboundRepository
	WMSManifestRepository
	WMSDockLPNRepository

	// Regional
	CreateRegional(ctx context.Context, r *Regional) error
	GetRegionalByID(ctx context.Context, tenantID, id uuid.UUID) (*Regional, error)

	// Warehouses
	CreateWarehouse(ctx context.Context, w *Warehouse) error
	GetWarehouseByID(ctx context.Context, tenantID, id uuid.UUID) (*Warehouse, error)
	ListWarehouses(ctx context.Context, tenantID uuid.UUID) ([]Warehouse, error)
	ListWarehousesByRegional(ctx context.Context, tenantID, regionalID uuid.UUID) ([]Warehouse, error)
	ListWarehousesByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]Warehouse, error)
	GetUserWarehouseIDs(ctx context.Context, tenantID, userID uuid.UUID) ([]uuid.UUID, error)
	AssignUserWarehouse(ctx context.Context, uw *UserWarehouse) error

	// Locations
	CreateLocation(ctx context.Context, loc *WarehouseLocation) error
	GetLocationByID(ctx context.Context, tenantID, id uuid.UUID) (*WarehouseLocation, error)
	GetLocationByCode(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, code string) (*WarehouseLocation, error)
	GetOrCreateSystemLocation(ctx context.Context, tenantID uuid.UUID, locType LocationType) (*WarehouseLocation, error)
	ListLocations(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]WarehouseLocation, error)

	// Barcodes & Mappings
	CreateBarcode(ctx context.Context, b *ProductBarcode) error
	CreateSKUMapping(ctx context.Context, m *ProductSKUMapping) error
	ResolveBarcode(ctx context.Context, tenantID uuid.UUID, code string) (*ResolvedProduct, error)

	// Stock movements & Ledger
	CreateStockMovement(ctx context.Context, m *StockMovement) error
	GetStockByLocation(ctx context.Context, tenantID, locationID, productID uuid.UUID) (decimal.Decimal, error)
	ListStockMovements(ctx context.Context, tenantID uuid.UUID, productID, locationID *uuid.UUID, limit int) ([]StockMovement, error)
	ListStockSummary(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]StockSummary, error)
	DeductLocationStock(ctx context.Context, tenantID, locationID, productID uuid.UUID, qty decimal.Decimal, mov *StockMovement) error

	// Stock Transfers
	CreateTransfer(ctx context.Context, t *StockTransfer, items []StockTransferItem) error
	GetTransferByID(ctx context.Context, tenantID, id uuid.UUID) (*StockTransfer, []StockTransferItem, error)
	UpdateTransferStatus(ctx context.Context, tenantID, id uuid.UUID, status TransferStatus, dispatchedAt, receivedAt *time.Time, approvedBy *uuid.UUID, rejectionReason *string) error
	ListTransfers(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]StockTransfer, error)

	// Delivery Orders
	CreateDeliveryOrder(ctx context.Context, do *DeliveryOrder, items []DeliveryOrderItem) error
	GetDeliveryOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*DeliveryOrder, []DeliveryOrderItem, error)
	ConfirmDeliveryOrder(ctx context.Context, tenantID, id, userID uuid.UUID) (*DeliveryOrder, error)
	UpdateDeliveryOrderStatus(ctx context.Context, tenantID, id uuid.UUID, status DeliveryOrderStatus, receivedDate *time.Time) error
	ListDeliveryOrders(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]DeliveryOrder, error)
	GetAvailableStock(ctx context.Context, tenantID, warehouseID uuid.UUID, locationID *uuid.UUID, productID uuid.UUID) (decimal.Decimal, error)

	// Stock Opname
	CreateStockOpname(ctx context.Context, op *StockOpname) error
	GetStockOpnameByID(ctx context.Context, tenantID, id uuid.UUID) (*StockOpname, error)
	ListStockOpnames(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]StockOpname, error)
	AddStockOpnameItem(ctx context.Context, item *StockOpnameItem) error
	ListStockOpnameItems(ctx context.Context, tenantID, opnameID uuid.UUID) ([]StockOpnameItem, error)
	UpdateStockOpnameStatus(ctx context.Context, tenantID, opnameID uuid.UUID, status StockOpnameStatus, approvedBy *uuid.UUID) error

	// Stock Scrap
	CreateStockScrap(ctx context.Context, scrap *StockScrap) error
	ListStockScraps(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]StockScrap, error)

	// Stock Receipts (Barang Masuk)
	CreateStockReceipt(ctx context.Context, rc *StockReceipt, items []StockReceiptItem) error
	// UpdateDraftStockReceipt replaces header fields and all items in one tx; ErrStockReceiptNotDraft if not DRAFT.
	UpdateDraftStockReceipt(ctx context.Context, rc *StockReceipt, items []StockReceiptItem) error
	GetStockReceiptByID(ctx context.Context, tenantID, id uuid.UUID) (*StockReceipt, []StockReceiptItem, error)
	ListStockReceipts(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *StockReceiptStatus, receiptType *StockReceiptType) ([]StockReceipt, error)
	// PostStockReceipt atomically creates batches, writes ledger movements into
	// inbound staging (sentry-wms §1.2 step 1), writes the audit row and sets POSTED.
	PostStockReceipt(ctx context.Context, p PostReceiptParams) (*StockReceipt, error)
	// CancelStockReceipt atomically cancels a DRAFT, or reverses a POSTED receipt's movements.
	CancelStockReceipt(ctx context.Context, tenantID, id, userID, defaultSourceLocID, scrapLocID uuid.UUID, reason string) (*StockReceipt, error)

	// Marketplace Sales Orders & SKU Mappings
	CreateMarketplaceBatch(ctx context.Context, batch *MarketplaceImportBatch) error
	GetMarketplaceBatchByID(ctx context.Context, tenantID, id uuid.UUID) (*MarketplaceImportBatch, error)
	ListMarketplaceBatches(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]MarketplaceImportBatch, error)
	UpdateMarketplaceBatch(ctx context.Context, batch *MarketplaceImportBatch) error
	CreateMarketplaceOrder(ctx context.Context, order *MarketplaceOrder) error
	GetMarketplaceOrderByExternalID(ctx context.Context, tenantID uuid.UUID, channel MarketplaceChannel, externalID string) (*MarketplaceOrder, error)
	ListMarketplaceOrders(ctx context.Context, tenantID uuid.UUID, warehouseID, batchID *uuid.UUID, status *MarketplaceOrderStatus) ([]MarketplaceOrder, error)
	GetMarketplaceOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*MarketplaceOrder, error)
	UpdateMarketplaceOrderStatus(ctx context.Context, tenantID, id uuid.UUID, status MarketplaceOrderStatus) error
	GetSKUMapping(ctx context.Context, tenantID uuid.UUID, channelName, externalSKU string) (*ProductSKUMapping, error)
	ListSKUMappings(ctx context.Context, tenantID uuid.UUID, channelName string) ([]ProductSKUMapping, error)
	UpdateUnmappedOrderItems(ctx context.Context, tenantID uuid.UUID, channel MarketplaceChannel, externalSKU string, productID uuid.UUID) error
	GetPendingUnmappedOrdersBySKU(ctx context.Context, tenantID uuid.UUID, channel MarketplaceChannel, externalSKU string) ([]MarketplaceOrder, error)
	GetProductBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*Product, error)
}
