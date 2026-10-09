package domain

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BatchAllocation is one slice of an outbound quantity taken from a batch at a location.
type BatchAllocation struct {
	BatchID    uuid.UUID
	LocationID uuid.UUID
	Quantity   decimal.Decimal
}

// AllocateFEFO picks batches First-Expired-First-Out (OCA §1.3: expiry_date ASC).
// Batches without expiry go last; ties fall back to oldest batch, then location code.
// ON_HOLD batches are skipped unless includeOnHold (adjustments such as scrap/opname
// may touch held stock; sales may not). It never allocates more than a balance holds
// and returns ErrInsufficientStock without a partial result when the total is short.
func AllocateFEFO(balances []BatchBalance, qty decimal.Decimal, includeOnHold bool) ([]BatchAllocation, error) {
	if !qty.IsPositive() {
		return nil, ErrInvalidInput
	}
	cands := make([]BatchBalance, 0, len(balances))
	for _, b := range balances {
		if !b.Quantity.IsPositive() {
			continue
		}
		if b.Status == StockBatchStatusOnHold && !includeOnHold {
			continue
		}
		// Outbound picking strictly restricts to RELEASED / AVAILABLE batches
		if b.Status != "" && b.Status != StockBatchStatusReleased && b.Status != StockBatchStatusAvailable && !includeOnHold {
			continue
		}
		cands = append(cands, b)
	}
	sort.SliceStable(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		switch {
		case a.ExpiryDate != nil && b.ExpiryDate == nil:
			return true
		case a.ExpiryDate == nil && b.ExpiryDate != nil:
			return false
		case a.ExpiryDate != nil && b.ExpiryDate != nil && !a.ExpiryDate.Equal(*b.ExpiryDate):
			return a.ExpiryDate.Before(*b.ExpiryDate)
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		if a.LocationCode != b.LocationCode {
			return a.LocationCode < b.LocationCode
		}
		return a.BatchID.String() < b.BatchID.String()
	})

	remaining := qty
	var out []BatchAllocation
	for _, c := range cands {
		if !remaining.IsPositive() {
			break
		}
		take := decimal.Min(c.Quantity, remaining)
		out = append(out, BatchAllocation{BatchID: c.BatchID, LocationID: c.LocationID, Quantity: take})
		remaining = remaining.Sub(take)
	}
	if remaining.IsPositive() {
		return nil, ErrInsufficientStock
	}
	return out, nil
}

// PostReceiptParams carries the system locations resolved before the post tx.
type PostReceiptParams struct {
	TenantID         uuid.UUID
	ReceiptID        uuid.UUID
	UserID           uuid.UUID
	SourceLocID      uuid.UUID // @VENDOR / @PRODUCTION / @TRANSIT
	ScrapLocID       uuid.UUID
	StagingLocID     uuid.UUID // STG-IN of the receipt's warehouse
	TransitLocID     uuid.UUID // used to mirror batches of a linked transfer
	HoldForRelease   bool      // wms_settings.require_release_approval
}

// AutoBatchNumber is the generated lot number when the receiver leaves it blank.
func AutoBatchNumber(receiptNumber string, line int) string {
	return "AUTO-" + receiptNumber + "-" + decimal.NewFromInt(int64(line)).String()
}

// BatchBalanceFilter narrows ListBatchBalances. Empty fields mean "any".
type BatchBalanceFilter struct {
	LocationIDs  []uuid.UUID
	WarehouseID  *uuid.UUID
	ProductID    *uuid.UUID
	LocationType *LocationType
	BatchStatus  *StockBatchStatus
}

// PutawayCommand moves one batch quantity from inbound staging to a rack
// (sentry-wms §1.2 step 2, POST /api/putaway/confirm).
type PutawayCommand struct {
	TenantID          uuid.UUID
	WarehouseID       uuid.UUID
	ProductID         uuid.UUID
	BatchID           uuid.UUID
	Quantity          decimal.Decimal
	StagingLocationID uuid.UUID
	DestLocationID    uuid.UUID
	UserID            uuid.UUID
	Reason            *string
	DefaultLocationID *uuid.UUID
}

// PutawayPendingLine is staged stock waiting for putaway, with a rack suggestion
// (sentry-wms GET /api/putaway/pending + /api/putaway/suggest).
type PutawayPendingLine struct {
	ProductID             uuid.UUID        `json:"product_id"`
	ProductName           string           `json:"product_name"`
	ProductSKU            string           `json:"product_sku"`
	BatchID               uuid.UUID        `json:"batch_id"`
	BatchNumber           string           `json:"batch_number"`
	ExpiryDate            *time.Time       `json:"expiry_date,omitempty"`
	BatchStatus           StockBatchStatus `json:"batch_status"`
	StagingLocationID     uuid.UUID        `json:"staging_location_id"`
	Quantity              decimal.Decimal  `json:"quantity"`
	SourceReceiptID       *uuid.UUID       `json:"source_receipt_id,omitempty"`
	SourceReceiptNumber   string           `json:"source_receipt_number,omitempty"`
	SuggestedLocationID   *uuid.UUID       `json:"suggested_location_id,omitempty"`
	SuggestedLocationCode string           `json:"suggested_location_code,omitempty"`
	SuggestionSource      string           `json:"suggestion_source,omitempty"` // DEFAULT_RACK | RECEIPT_DEST
	DefaultLocationID     *uuid.UUID       `json:"default_location_id,omitempty"`
}

// WMSSettings holds tenant-level warehouse rules (PDF-06 release approval).
type WMSSettings struct {
	TenantID               uuid.UUID  `json:"tenant_id"`
	RequireReleaseApproval bool       `json:"require_release_approval"`
	RequirePickPack        bool       `json:"require_pick_pack"`
	UpdatedBy              *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// ProductDefaultLocation is the default rack of a product in one warehouse (CR-03).
type ProductDefaultLocation struct {
	TenantID     uuid.UUID  `json:"tenant_id"`
	ProductID    uuid.UUID  `json:"product_id"`
	WarehouseID  uuid.UUID  `json:"warehouse_id"`
	LocationID   uuid.UUID  `json:"location_id"`
	LocationCode string     `json:"location_code,omitempty"`
	UpdatedBy    *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TraceMovement is one ledger row enriched for traceability views (KO-1c).
type TraceMovement struct {
	MovementID      uuid.UUID       `json:"movement_id"`
	MovementNumber  string          `json:"movement_number"`
	ProductID       uuid.UUID       `json:"product_id"`
	ProductName     string          `json:"product_name"`
	ProductSKU      string          `json:"product_sku"`
	BatchID         uuid.UUID       `json:"batch_id"`
	BatchNumber     string          `json:"batch_number"`
	ExpiryDate      *time.Time      `json:"expiry_date,omitempty"`
	SourceCode      string          `json:"source_location_code"`
	SourceType      LocationType    `json:"source_location_type"`
	DestCode        string          `json:"dest_location_code"`
	DestType        LocationType    `json:"dest_location_type"`
	Quantity        decimal.Decimal `json:"quantity"`
	ReferenceType   string          `json:"reference_type"`
	ReferenceID     uuid.UUID       `json:"reference_id"`
	ReferenceNumber string          `json:"reference_number,omitempty"`
	Counterparty    string          `json:"counterparty,omitempty"` // supplier / recipient
	ExecutedByName  string          `json:"executed_by_name,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// BatchTrace answers "where did this batch come from and where did it go".
type BatchTrace struct {
	Batch     StockBatch      `json:"batch"`
	Movements []TraceMovement `json:"movements"`
	Balances  []BatchBalance  `json:"balances"`
	TotalIn   decimal.Decimal `json:"total_in"`
	TotalOut  decimal.Decimal `json:"total_out"`
	OnHand    decimal.Decimal `json:"on_hand"`
}

// DocumentTrace lists the batches a document touched and every movement of those batches.
type DocumentTrace struct {
	DocumentType   string          `json:"document_type"`
	DocumentID     uuid.UUID       `json:"document_id"`
	DocumentNumber string          `json:"document_number"`
	Batches        []StockBatch    `json:"batches"`
	Movements      []TraceMovement `json:"movements"`
}

// AuditTrailEntry is an audit_logs row with the actor's display name.
type AuditTrailEntry struct {
	ID         uuid.UUID       `json:"id"`
	EntityType string          `json:"entity_type"`
	EntityID   uuid.UUID       `json:"entity_id"`
	Action     string          `json:"action"`
	UserID     *uuid.UUID      `json:"user_id,omitempty"`
	UserName   string          `json:"user_name"`
	Details    json.RawMessage `json:"details,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// WMSBatchRepository holds Sprint 1 batch/staging/putaway persistence. It is embedded
// in WMSRepository so the existing single WMS repo keeps one implementation.
type WMSBatchRepository interface {
	GetOrCreateBatch(ctx context.Context, b *StockBatch) (*StockBatch, error)
	GetBatchByID(ctx context.Context, tenantID, id uuid.UUID) (*StockBatch, error)
	ListBatchBalances(ctx context.Context, tenantID uuid.UUID, f BatchBalanceFilter) ([]BatchBalance, error)
	// CreateStockMovements writes several movements in one transaction (all or nothing).
	CreateStockMovements(ctx context.Context, tenantID uuid.UUID, movs []StockMovement) error
	ListMovementsByReference(ctx context.Context, tenantID uuid.UUID, refType string, refID uuid.UUID) ([]StockMovement, error)
	// DeductWarehouseStock allocates FEFO across all INTERNAL racks of a warehouse.
	DeductWarehouseStock(ctx context.Context, tenantID, warehouseID, productID uuid.UUID, qty decimal.Decimal, mov *StockMovement) error
	GetOrCreateStagingLocation(ctx context.Context, tenantID, warehouseID uuid.UUID) (*WarehouseLocation, error)
	ConfirmPutaway(ctx context.Context, cmd PutawayCommand) (*StockMovement, error)
	ReleaseStockReceipt(ctx context.Context, tenantID, receiptID, userID uuid.UUID) (*StockReceipt, error)
	GetWMSSettings(ctx context.Context, tenantID uuid.UUID) (*WMSSettings, error)
	UpsertWMSSettings(ctx context.Context, s *WMSSettings) error
	ListDefaultLocations(ctx context.Context, tenantID uuid.UUID, productID *uuid.UUID) ([]ProductDefaultLocation, error)
	SetDefaultLocation(ctx context.Context, d *ProductDefaultLocation) error
	DeleteDefaultLocation(ctx context.Context, tenantID, productID, warehouseID uuid.UUID) error
	TraceBatch(ctx context.Context, tenantID, batchID uuid.UUID) (*BatchTrace, error)
	TraceDocument(ctx context.Context, tenantID uuid.UUID, refType string, refID uuid.UUID) (*DocumentTrace, error)
	ListAuditTrail(ctx context.Context, tenantID uuid.UUID, entityType string, entityID uuid.UUID) ([]AuditTrailEntry, error)
}
