package wms_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// Mock WMSRepository
// -----------------------------------------------------------------------------

type mockWMSRepo struct {
	warehouses      map[uuid.UUID]domain.Warehouse
	userWarehouses  map[string][]uuid.UUID // key: "tenantID:userID" -> []warehouseID
	locations       map[uuid.UUID]domain.WarehouseLocation
	systemLocations map[string]*domain.WarehouseLocation // key: "tenantID:type"
	barcodes        map[string]domain.ProductBarcode
	skuMappings     map[string]domain.ProductSKUMapping
	products        map[string]domain.Product // key: "tenantID:sku"
	stockMovements  []domain.StockMovement
	stockLevels     map[string]decimal.Decimal // key: "tenantID:locID:prodID"
	transfers       map[uuid.UUID]*domain.StockTransfer
	transferItems   map[uuid.UUID][]domain.StockTransferItem
	deliveryOrders  map[uuid.UUID]*domain.DeliveryOrder
	doItems         map[uuid.UUID][]domain.DeliveryOrderItem
	stockOpnames    map[uuid.UUID]*domain.StockOpname
	opnameItems     map[uuid.UUID][]domain.StockOpnameItem
	stockScraps     map[uuid.UUID]*domain.StockScrap
	marketplaceBatches map[uuid.UUID]*domain.MarketplaceImportBatch
	marketplaceOrders  map[uuid.UUID]*domain.MarketplaceOrder
	stalePendingOrders []domain.MarketplaceOrder
	// Stock receipts (lazily initialised in wms_receipt_test.go)
	stockReceipts    map[uuid.UUID]*domain.StockReceipt
	receiptItems     map[uuid.UUID][]domain.StockReceiptItem
	batches          map[uuid.UUID]*domain.StockBatch
	wmsSettings      map[uuid.UUID]*domain.WMSSettings
	defaultLocations map[string]*domain.ProductDefaultLocation
	auditLogs        []domain.AuditTrailEntry
}

func newMockWMSRepo() *mockWMSRepo {
	return &mockWMSRepo{
		warehouses:         make(map[uuid.UUID]domain.Warehouse),
		userWarehouses:     make(map[string][]uuid.UUID),
		locations:          make(map[uuid.UUID]domain.WarehouseLocation),
		systemLocations:    make(map[string]*domain.WarehouseLocation),
		barcodes:           make(map[string]domain.ProductBarcode),
		skuMappings:        make(map[string]domain.ProductSKUMapping),
		products:           make(map[string]domain.Product),
		batches:            make(map[uuid.UUID]*domain.StockBatch),
		wmsSettings:        make(map[uuid.UUID]*domain.WMSSettings),
		defaultLocations:   make(map[string]*domain.ProductDefaultLocation),
		stockLevels:        make(map[string]decimal.Decimal),
		transfers:          make(map[uuid.UUID]*domain.StockTransfer),
		transferItems:      make(map[uuid.UUID][]domain.StockTransferItem),
		deliveryOrders:     make(map[uuid.UUID]*domain.DeliveryOrder),
		doItems:            make(map[uuid.UUID][]domain.DeliveryOrderItem),
		stockOpnames:       make(map[uuid.UUID]*domain.StockOpname),
		opnameItems:        make(map[uuid.UUID][]domain.StockOpnameItem),
		stockScraps:        make(map[uuid.UUID]*domain.StockScrap),
		marketplaceBatches: make(map[uuid.UUID]*domain.MarketplaceImportBatch),
		marketplaceOrders:  make(map[uuid.UUID]*domain.MarketplaceOrder),
		stockReceipts:      make(map[uuid.UUID]*domain.StockReceipt),
		receiptItems:       make(map[uuid.UUID][]domain.StockReceiptItem),
	}
}

func (m *mockWMSRepo) CreateRegional(ctx context.Context, r *domain.Regional) error {
	return nil
}

func (m *mockWMSRepo) GetRegionalByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Regional, error) {
	return nil, nil
}

func (m *mockWMSRepo) CreateWarehouse(ctx context.Context, w *domain.Warehouse) error {
	m.warehouses[w.ID] = *w
	return nil
}

func (m *mockWMSRepo) GetWarehouseByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Warehouse, error) {
	w, ok := m.warehouses[id]
	if !ok || w.TenantID != tenantID {
		return nil, domain.ErrWarehouseNotFound
	}
	return &w, nil
}

func (m *mockWMSRepo) ListWarehouses(ctx context.Context, tenantID uuid.UUID) ([]domain.Warehouse, error) {
	var list []domain.Warehouse
	for _, w := range m.warehouses {
		if w.TenantID == tenantID {
			list = append(list, w)
		}
	}
	return list, nil
}

func (m *mockWMSRepo) ListWarehousesByRegional(ctx context.Context, tenantID, regionalID uuid.UUID) ([]domain.Warehouse, error) {
	var list []domain.Warehouse
	for _, w := range m.warehouses {
		if w.TenantID == tenantID && w.RegionalID != nil && *w.RegionalID == regionalID {
			list = append(list, w)
		}
	}
	return list, nil
}

func (m *mockWMSRepo) ListWarehousesByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]domain.Warehouse, error) {
	idSet := make(map[uuid.UUID]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	var list []domain.Warehouse
	for _, w := range m.warehouses {
		if w.TenantID == tenantID && idSet[w.ID] {
			list = append(list, w)
		}
	}
	return list, nil
}

func (m *mockWMSRepo) GetUserWarehouseIDs(ctx context.Context, tenantID, userID uuid.UUID) ([]uuid.UUID, error) {
	key := fmt.Sprintf("%s:%s", tenantID, userID)
	return m.userWarehouses[key], nil
}

func (m *mockWMSRepo) AssignUserWarehouse(ctx context.Context, uw *domain.UserWarehouse) error {
	key := fmt.Sprintf("%s:%s", uw.TenantID, uw.UserID)
	m.userWarehouses[key] = append(m.userWarehouses[key], uw.WarehouseID)
	return nil
}

func (m *mockWMSRepo) CreateLocation(ctx context.Context, loc *domain.WarehouseLocation) error {
	m.locations[loc.ID] = *loc
	return nil
}

func (m *mockWMSRepo) GetLocationByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.WarehouseLocation, error) {
	loc, ok := m.locations[id]
	if !ok || loc.TenantID != tenantID {
		return nil, domain.ErrLocationNotFound
	}
	return &loc, nil
}

func (m *mockWMSRepo) GetLocationByCode(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, code string) (*domain.WarehouseLocation, error) {
	for _, loc := range m.locations {
		if loc.TenantID == tenantID && loc.Code == code {
			return &loc, nil
		}
	}
	return nil, domain.ErrLocationNotFound
}

func (m *mockWMSRepo) GetOrCreateSystemLocation(ctx context.Context, tenantID uuid.UUID, locType domain.LocationType) (*domain.WarehouseLocation, error) {
	key := fmt.Sprintf("%s:%s", tenantID, locType)
	if loc, ok := m.systemLocations[key]; ok {
		return loc, nil
	}
	newLoc := &domain.WarehouseLocation{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Code:      fmt.Sprintf("@%s", locType),
		Name:      fmt.Sprintf("Virtual %s", locType),
		Type:      locType,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	m.systemLocations[key] = newLoc
	m.locations[newLoc.ID] = *newLoc
	return newLoc, nil
}

func (m *mockWMSRepo) ListLocations(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.WarehouseLocation, error) {
	var list []domain.WarehouseLocation
	for _, loc := range m.locations {
		if loc.TenantID == tenantID {
			if warehouseID != nil {
				if loc.WarehouseID != nil && *loc.WarehouseID == *warehouseID {
					list = append(list, loc)
				}
			} else {
				list = append(list, loc)
			}
		}
	}
	return list, nil
}

func (m *mockWMSRepo) CreateBarcode(ctx context.Context, b *domain.ProductBarcode) error {
	key := fmt.Sprintf("%s:%s", b.TenantID, b.Barcode)
	m.barcodes[key] = *b
	return nil
}

func (m *mockWMSRepo) CreateSKUMapping(ctx context.Context, p *domain.ProductSKUMapping) error {
	key1 := fmt.Sprintf("%s:%s", p.TenantID, p.ExternalSKU)
	key2 := fmt.Sprintf("%s:%s:%s", p.TenantID, p.ChannelName, p.ExternalSKU)
	m.skuMappings[key1] = *p
	m.skuMappings[key2] = *p
	return nil
}

func (m *mockWMSRepo) GetSKUMapping(ctx context.Context, tenantID uuid.UUID, channelName, externalSKU string) (*domain.ProductSKUMapping, error) {
	key2 := fmt.Sprintf("%s:%s:%s", tenantID, channelName, externalSKU)
	if mapping, ok := m.skuMappings[key2]; ok {
		cp := mapping
		return &cp, nil
	}
	key1 := fmt.Sprintf("%s:%s", tenantID, externalSKU)
	if mapping, ok := m.skuMappings[key1]; ok {
		cp := mapping
		return &cp, nil
	}
	return nil, domain.ErrSKUMappingNotFound
}

func (m *mockWMSRepo) ListSKUMappings(ctx context.Context, tenantID uuid.UUID, channelName string) ([]domain.ProductSKUMapping, error) {
	seen := make(map[uuid.UUID]bool)
	var list []domain.ProductSKUMapping
	for _, mapping := range m.skuMappings {
		if mapping.TenantID == tenantID {
			if channelName == "" || strings.EqualFold(mapping.ChannelName, channelName) {
				if !seen[mapping.ID] {
					seen[mapping.ID] = true
					list = append(list, mapping)
				}
			}
		}
	}
	return list, nil
}

func (m *mockWMSRepo) ResolveBarcode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.ResolvedProduct, error) {
	// 1. Direct SKU
	skuKey := fmt.Sprintf("%s:%s", tenantID, code)
	if prod, ok := m.products[skuKey]; ok {
		return &domain.ResolvedProduct{
			ProductID:  prod.ID,
			SKU:        prod.SKU,
			Name:       prod.Name,
			Barcode:    "",
			Multiplier: decimal.NewFromInt(1),
			Source:     "SKU",
		}, nil
	}

	// 2. Physical Barcode
	barcodeKey := fmt.Sprintf("%s:%s", tenantID, code)
	if b, ok := m.barcodes[barcodeKey]; ok {
		// find product name
		var prodName, prodSKU string
		for _, p := range m.products {
			if p.ID == b.ProductID && p.TenantID == tenantID {
				prodName = p.Name
				prodSKU = p.SKU
				break
			}
		}
		return &domain.ResolvedProduct{
			ProductID:  b.ProductID,
			SKU:        prodSKU,
			Name:       prodName,
			Barcode:    b.Barcode,
			Multiplier: b.Multiplier,
			Source:     "BARCODE",
		}, nil
	}

	// 3. SKU Mapping
	mapKey := fmt.Sprintf("%s:%s", tenantID, code)
	if mp, ok := m.skuMappings[mapKey]; ok {
		var prodName, prodSKU string
		for _, p := range m.products {
			if p.ID == mp.ProductID && p.TenantID == tenantID {
				prodName = p.Name
				prodSKU = p.SKU
				break
			}
		}
		return &domain.ResolvedProduct{
			ProductID:   mp.ProductID,
			SKU:         prodSKU,
			Name:        prodName,
			ExternalSKU: mp.ExternalSKU,
			Multiplier:  mp.Multiplier,
			Source:      "MAPPING",
		}, nil
	}

	return nil, domain.ErrBarcodeNotFound
}

func (m *mockWMSRepo) CreateStockMovement(ctx context.Context, sm *domain.StockMovement) error {
	if sm.BatchID == nil {
		dummyBatch := uuid.New()
		sm.BatchID = &dummyBatch
	}
	m.stockMovements = append(m.stockMovements, *sm)
	// Update mock stock levels
	srcKey := fmt.Sprintf("%s:%s:%s", sm.TenantID, sm.SourceLocationID, sm.ProductID)
	dstKey := fmt.Sprintf("%s:%s:%s", sm.TenantID, sm.DestLocationID, sm.ProductID)
	m.stockLevels[srcKey] = m.stockLevels[srcKey].Sub(sm.Quantity)
	m.stockLevels[dstKey] = m.stockLevels[dstKey].Add(sm.Quantity)
	return nil
}

func (m *mockWMSRepo) GetStockByLocation(ctx context.Context, tenantID, locationID, productID uuid.UUID) (decimal.Decimal, error) {
	key := fmt.Sprintf("%s:%s:%s", tenantID, locationID, productID)
	return m.stockLevels[key], nil
}

func (m *mockWMSRepo) ListStockMovements(ctx context.Context, tenantID uuid.UUID, productID, locationID *uuid.UUID, limit int) ([]domain.StockMovement, error) {
	return m.stockMovements, nil
}

func (m *mockWMSRepo) ListStockSummary(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.StockSummary, error) {
	return []domain.StockSummary{}, nil
}

func (m *mockWMSRepo) DeductLocationStock(ctx context.Context, tenantID, locationID, productID uuid.UUID, qty decimal.Decimal, sm *domain.StockMovement) error {
	key := fmt.Sprintf("%s:%s:%s", tenantID, locationID, productID)
	current := m.stockLevels[key]
	if current.LessThan(qty) {
		return domain.ErrInsufficientStock
	}
	return m.CreateStockMovement(ctx, sm)
}

func (m *mockWMSRepo) CreateTransfer(ctx context.Context, t *domain.StockTransfer, items []domain.StockTransferItem) error {
	m.transfers[t.ID] = t
	m.transferItems[t.ID] = items
	return nil
}

func (m *mockWMSRepo) GetTransferByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockTransfer, []domain.StockTransferItem, error) {
	t, ok := m.transfers[id]
	if !ok || t.TenantID != tenantID {
		return nil, nil, domain.ErrTransferNotFound
	}
	return t, m.transferItems[id], nil
}

func (m *mockWMSRepo) UpdateTransferStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.TransferStatus, dispatchedAt, receivedAt *time.Time, approvedBy *uuid.UUID, rejectionReason *string) error {
	t, ok := m.transfers[id]
	if !ok || t.TenantID != tenantID {
		return domain.ErrTransferNotFound
	}
	t.Status = status
	if dispatchedAt != nil {
		t.DispatchedAt = dispatchedAt
	}
	if receivedAt != nil {
		t.ReceivedAt = receivedAt
	}
	if approvedBy != nil {
		t.ApprovedBy = approvedBy
	}
	if rejectionReason != nil {
		t.RejectionReason = rejectionReason
	}
	return nil
}

func (m *mockWMSRepo) ListTransfers(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.StockTransfer, error) {
	var list []domain.StockTransfer
	for _, t := range m.transfers {
		if t.TenantID == tenantID {
			if warehouseID != nil {
				if t.FromWarehouseID == *warehouseID || t.ToWarehouseID == *warehouseID {
					list = append(list, *t)
				}
			} else {
				list = append(list, *t)
			}
		}
	}
	return list, nil
}

func (m *mockWMSRepo) CreateDeliveryOrder(ctx context.Context, do *domain.DeliveryOrder, items []domain.DeliveryOrderItem) error {
	m.deliveryOrders[do.ID] = do
	m.doItems[do.ID] = items
	return nil
}

func (m *mockWMSRepo) GetDeliveryOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.DeliveryOrder, []domain.DeliveryOrderItem, error) {
	do, ok := m.deliveryOrders[id]
	if !ok || do.TenantID != tenantID {
		return nil, nil, domain.ErrDeliveryOrderNotFound
	}
	return do, m.doItems[id], nil
}

func (m *mockWMSRepo) UpdateDeliveryOrderStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.DeliveryOrderStatus, receivedDate *time.Time) error {
	do, ok := m.deliveryOrders[id]
	if !ok || do.TenantID != tenantID {
		return domain.ErrDeliveryOrderNotFound
	}
	do.Status = status
	if receivedDate != nil {
		do.ReceivedDate = receivedDate
	}
	return nil
}

func (m *mockWMSRepo) ListDeliveryOrders(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.DeliveryOrder, error) {
	var list []domain.DeliveryOrder
	for _, do := range m.deliveryOrders {
		if do.TenantID == tenantID {
			if warehouseID != nil {
				if do.WarehouseID == *warehouseID {
					list = append(list, *do)
				}
			} else {
				list = append(list, *do)
			}
		}
	}
	return list, nil
}

func (m *mockWMSRepo) ConfirmDeliveryOrder(ctx context.Context, tenantID, id, userID uuid.UUID) (*domain.DeliveryOrder, error) {
	do, ok := m.deliveryOrders[id]
	if !ok || do.TenantID != tenantID {
		return nil, domain.ErrDeliveryOrderNotFound
	}
	if do.Status != domain.DeliveryOrderStatusDraft {
		return nil, domain.ErrInvalidStatus
	}
	do.Status = domain.DeliveryOrderStatusConfirmed
	do.ConfirmedBy = &userID
	return do, nil
}

func (m *mockWMSRepo) GetAvailableStock(ctx context.Context, tenantID, warehouseID uuid.UUID, locationID *uuid.UUID, productID uuid.UUID) (decimal.Decimal, error) {
	var total decimal.Decimal
	for k, qty := range m.stockLevels {
		parts := strings.Split(k, ":")
		if len(parts) == 3 && parts[0] == tenantID.String() && parts[2] == productID.String() {
			if locationID != nil && parts[1] != locationID.String() {
				continue
			}
			total = total.Add(qty)
		}
	}
	if total.IsPositive() {
		return total, nil
	}
	return decimal.NewFromInt(1000), nil
}

func (m *mockWMSRepo) CreateStockOpname(ctx context.Context, op *domain.StockOpname) error {
	m.stockOpnames[op.ID] = op
	return nil
}

func (m *mockWMSRepo) GetStockOpnameByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockOpname, error) {
	op, ok := m.stockOpnames[id]
	if !ok || op.TenantID != tenantID {
		return nil, domain.ErrOpnameNotFound
	}
	return op, nil
}

func (m *mockWMSRepo) ListStockOpnames(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.StockOpname, error) {
	var list []domain.StockOpname
	for _, op := range m.stockOpnames {
		if op.TenantID == tenantID {
			if warehouseID != nil {
				if op.WarehouseID == *warehouseID {
					list = append(list, *op)
				}
			} else {
				list = append(list, *op)
			}
		}
	}
	return list, nil
}

func (m *mockWMSRepo) AddStockOpnameItem(ctx context.Context, item *domain.StockOpnameItem) error {
	m.opnameItems[item.OpnameID] = append(m.opnameItems[item.OpnameID], *item)
	return nil
}

func (m *mockWMSRepo) ListStockOpnameItems(ctx context.Context, tenantID, opnameID uuid.UUID) ([]domain.StockOpnameItem, error) {
	items := m.opnameItems[opnameID]
	var list []domain.StockOpnameItem
	for _, it := range items {
		if it.TenantID == tenantID {
			list = append(list, it)
		}
	}
	return list, nil
}

func (m *mockWMSRepo) UpdateStockOpnameStatus(ctx context.Context, tenantID, opnameID uuid.UUID, status domain.StockOpnameStatus, approvedBy *uuid.UUID) error {
	op, ok := m.stockOpnames[opnameID]
	if !ok || op.TenantID != tenantID {
		return domain.ErrOpnameNotFound
	}
	op.Status = status
	op.ApprovedBy = approvedBy
	return nil
}

func (m *mockWMSRepo) CreateStockScrap(ctx context.Context, scrap *domain.StockScrap) error {
	m.stockScraps[scrap.ID] = scrap
	return nil
}

func (m *mockWMSRepo) ListStockScraps(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.StockScrap, error) {
	var list []domain.StockScrap
	for _, s := range m.stockScraps {
		if s.TenantID == tenantID {
			if warehouseID != nil {
				if s.WarehouseID == *warehouseID {
					list = append(list, *s)
				}
			} else {
				list = append(list, *s)
			}
		}
	}
	return list, nil
}

func (m *mockWMSRepo) CreateMarketplaceBatch(ctx context.Context, batch *domain.MarketplaceImportBatch) error {
	cp := *batch
	m.marketplaceBatches[batch.ID] = &cp
	return nil
}

func (m *mockWMSRepo) GetMarketplaceBatchByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.MarketplaceImportBatch, error) {
	b, ok := m.marketplaceBatches[id]
	if !ok || b.TenantID != tenantID {
		return nil, domain.ErrBatchNotFound
	}
	cp := *b
	return &cp, nil
}

func (m *mockWMSRepo) ListMarketplaceBatches(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) ([]domain.MarketplaceImportBatch, error) {
	var list []domain.MarketplaceImportBatch
	for _, b := range m.marketplaceBatches {
		if b.TenantID == tenantID {
			if warehouseID == nil || b.WarehouseID == *warehouseID {
				list = append(list, *b)
			}
		}
	}
	return list, nil
}

func (m *mockWMSRepo) UpdateMarketplaceBatch(ctx context.Context, batch *domain.MarketplaceImportBatch) error {
	b, ok := m.marketplaceBatches[batch.ID]
	if !ok || b.TenantID != batch.TenantID {
		return domain.ErrBatchNotFound
	}
	cp := *batch
	m.marketplaceBatches[batch.ID] = &cp
	return nil
}

func (m *mockWMSRepo) CreateMarketplaceOrder(ctx context.Context, order *domain.MarketplaceOrder) error {
	for _, o := range m.marketplaceOrders {
		if o.TenantID == order.TenantID && o.Channel == order.Channel && o.ExternalOrderID == order.ExternalOrderID {
			return domain.ErrDuplicateMarketplaceOrder
		}
	}
	cp := *order
	cp.Items = make([]domain.MarketplaceOrderItem, len(order.Items))
	copy(cp.Items, order.Items)
	m.marketplaceOrders[order.ID] = &cp
	return nil
}

func (m *mockWMSRepo) GetMarketplaceOrderByExternalID(ctx context.Context, tenantID uuid.UUID, channel domain.MarketplaceChannel, externalID string) (*domain.MarketplaceOrder, error) {
	for _, o := range m.marketplaceOrders {
		if o.TenantID == tenantID && o.Channel == channel && o.ExternalOrderID == externalID {
			cp := *o
			cp.Items = make([]domain.MarketplaceOrderItem, len(o.Items))
			copy(cp.Items, o.Items)
			return &cp, nil
		}
	}
	return nil, domain.ErrMarketplaceOrderNotFound
}

func (m *mockWMSRepo) ListMarketplaceOrders(ctx context.Context, tenantID uuid.UUID, warehouseID, batchID *uuid.UUID, status *domain.MarketplaceOrderStatus) ([]domain.MarketplaceOrder, error) {
	var list []domain.MarketplaceOrder
	for _, o := range m.marketplaceOrders {
		if o.TenantID != tenantID {
			continue
		}
		if warehouseID != nil && o.WarehouseID != *warehouseID {
			continue
		}
		if batchID != nil && (o.BatchID == nil || *o.BatchID != *batchID) {
			continue
		}
		if status != nil && o.Status != *status {
			continue
		}
		cp := *o
		cp.Items = make([]domain.MarketplaceOrderItem, len(o.Items))
		copy(cp.Items, o.Items)
		list = append(list, cp)
	}
	return list, nil
}

func (m *mockWMSRepo) GetMarketplaceOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.MarketplaceOrder, error) {
	o, ok := m.marketplaceOrders[id]
	if !ok || o.TenantID != tenantID {
		return nil, domain.ErrMarketplaceOrderNotFound
	}
	cp := *o
	cp.Items = make([]domain.MarketplaceOrderItem, len(o.Items))
	copy(cp.Items, o.Items)
	return &cp, nil
}

func (m *mockWMSRepo) UpdateMarketplaceOrderStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.MarketplaceOrderStatus) error {
	o, ok := m.marketplaceOrders[id]
	if !ok || o.TenantID != tenantID {
		return domain.ErrMarketplaceOrderNotFound
	}
	o.Status = status
	return nil
}

func (m *mockWMSRepo) ClaimMarketplaceOrder(ctx context.Context, tenantID, id uuid.UUID, from, to domain.MarketplaceOrderStatus) (bool, error) {
	o, ok := m.marketplaceOrders[id]
	if !ok || o.TenantID != tenantID || o.Status != from {
		return false, nil
	}
	o.Status = to
	return true, nil
}

func (m *mockWMSRepo) UpdateUnmappedOrderItems(ctx context.Context, tenantID uuid.UUID, channel domain.MarketplaceChannel, externalSKU string, productID uuid.UUID) error {
	for _, o := range m.marketplaceOrders {
		if o.TenantID == tenantID && o.Channel == channel {
			for i := range o.Items {
				if o.Items[i].ExternalSKU == externalSKU && !o.Items[i].IsMapped {
					o.Items[i].ProductID = &productID
					o.Items[i].IsMapped = true
				}
			}
		}
	}
	return nil
}

func (m *mockWMSRepo) GetPendingUnmappedOrdersBySKU(ctx context.Context, tenantID uuid.UUID, channel domain.MarketplaceChannel, externalSKU string) ([]domain.MarketplaceOrder, error) {
	if m.stalePendingOrders != nil {
		// Simulates a concurrent request that read the pending list before
		// another request finished processing it.
		return m.stalePendingOrders, nil
	}
	var list []domain.MarketplaceOrder
	for _, o := range m.marketplaceOrders {
		if o.TenantID == tenantID && o.Channel == channel && o.Status == domain.MarketplaceOrderStatusUnmappedSKU {
			hasSKU := false
			for _, item := range o.Items {
				if item.ExternalSKU == externalSKU {
					hasSKU = true
					break
				}
			}
			if hasSKU {
				cp := *o
				cp.Items = make([]domain.MarketplaceOrderItem, len(o.Items))
				copy(cp.Items, o.Items)
				list = append(list, cp)
			}
		}
	}
	return list, nil
}

func (m *mockWMSRepo) GetProductBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.Product, error) {
	key := fmt.Sprintf("%s:%s", tenantID, sku)
	p, ok := m.products[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &p, nil
}

// -----------------------------------------------------------------------------
// Test Suite
// -----------------------------------------------------------------------------

func TestWarehouseAuthorizationScoping(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	regJakarta := uuid.New()
	regSurabaya := uuid.New()

	whJKT1 := uuid.New()
	whJKT2 := uuid.New()
	whSBY1 := uuid.New()

	repo.warehouses[whJKT1] = domain.Warehouse{ID: whJKT1, TenantID: tenantID, RegionalID: &regJakarta, Code: "JKT-01", Name: "Gudang Sunter"}
	repo.warehouses[whJKT2] = domain.Warehouse{ID: whJKT2, TenantID: tenantID, RegionalID: &regJakarta, Code: "JKT-02", Name: "Gudang Cakung"}
	repo.warehouses[whSBY1] = domain.Warehouse{ID: whSBY1, TenantID: tenantID, RegionalID: &regSurabaya, Code: "SBY-01", Name: "Gudang Rungkut"}

	staffUserID := uuid.New()
	regMgrUserID := uuid.New()
	adminUserID := uuid.New()

	// Assign staff only to whJKT1
	repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{UserID: staffUserID, WarehouseID: whJKT1, TenantID: tenantID})

	// Assign regional manager to whJKT1 (which belongs to regJakarta)
	repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{UserID: regMgrUserID, WarehouseID: whJKT1, TenantID: tenantID})

	t.Run("AC-1.1: Staf gudang only allowed access to assigned warehouse", func(t *testing.T) {
		err := usecase.ValidateWarehouseAccess(ctx, tenantID, staffUserID, "warehouse", whJKT1)
		assert.NoError(t, err, "staff should access assigned warehouse whJKT1")

		err = usecase.ValidateWarehouseAccess(ctx, tenantID, staffUserID, "warehouse", whJKT2)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse, "staff must be rejected when accessing unassigned whJKT2")

		err = usecase.ValidateWarehouseAccess(ctx, tenantID, staffUserID, "warehouse", whSBY1)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse, "staff must be rejected when accessing whSBY1")
	})

	t.Run("AC-1.2: Regional manager allowed access to all warehouses in their regional", func(t *testing.T) {
		err := usecase.ValidateWarehouseAccess(ctx, tenantID, regMgrUserID, "regional_manager", whJKT1)
		assert.NoError(t, err, "regional manager should access assigned whJKT1")

		err = usecase.ValidateWarehouseAccess(ctx, tenantID, regMgrUserID, "regional_manager", whJKT2)
		assert.NoError(t, err, "regional manager should access other warehouse in same regional whJKT2")

		err = usecase.ValidateWarehouseAccess(ctx, tenantID, regMgrUserID, "regional_manager", whSBY1)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse, "regional manager must be rejected for whSBY1 in different regional")
	})

	t.Run("Admin and owner have full cross-warehouse access", func(t *testing.T) {
		errAdmin := usecase.ValidateWarehouseAccess(ctx, tenantID, adminUserID, "admin", whSBY1)
		assert.NoError(t, errAdmin, "admin should access any warehouse")

		errOwner := usecase.ValidateWarehouseAccess(ctx, tenantID, adminUserID, "owner", whJKT2)
		assert.NoError(t, errOwner, "owner should access any warehouse")
	})

	t.Run("Auditor role has read-only access and cannot perform writes", func(t *testing.T) {
		auditorUserID := uuid.New()

		// Read access must succeed
		errRead := usecase.ValidateWarehouseReadAccess(ctx, tenantID, auditorUserID, "auditor", whJKT1)
		assert.NoError(t, errRead, "auditor should have read access")

		// Write access must be rejected with ErrForbidden
		errWrite := usecase.ValidateWarehouseWriteAccess(ctx, tenantID, auditorUserID, "auditor", whJKT1)
		assert.ErrorIs(t, errWrite, domain.ErrForbidden, "auditor must be rejected with ErrForbidden on write")

		// CreateLocation with auditor role -> ErrForbidden
		_, errLoc := usecase.CreateLocation(ctx, tenantID, auditorUserID, "auditor", uc.CreateLocationRequest{
			WarehouseID: &whJKT1,
			Code:        "AUD-LOC",
			Name:        "Auditor Loc",
		})
		assert.ErrorIs(t, errLoc, domain.ErrForbidden, "auditor cannot create location")

		// CreateTransfer with auditor role -> ErrForbidden
		_, errTr := usecase.CreateTransfer(ctx, tenantID, auditorUserID, "auditor", uc.CreateTransferRequest{
			FromWarehouseID: whJKT1,
			ToWarehouseID:   whJKT2,
			Items: []uc.CreateTransferItemRequest{
				{ProductID: uuid.New(), RequestedQty: decimal.NewFromInt(1)},
			},
		})
		assert.ErrorIs(t, errTr, domain.ErrForbidden, "auditor cannot create transfer")

		// CreateDeliveryOrder with auditor role -> ErrForbidden
		_, errDO := usecase.CreateDeliveryOrder(ctx, tenantID, auditorUserID, "auditor", uc.CreateDeliveryOrderRequest{
			WarehouseID: whJKT1,
			Items: []uc.CreateDeliveryOrderItemRequest{
				{ProductID: uuid.New(), Quantity: decimal.NewFromInt(1), LocationID: uuid.New()},
			},
		})
		assert.ErrorIs(t, errDO, domain.ErrForbidden, "auditor cannot create delivery order")
	})
}

func TestBarcodeAndSKUResolution(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	productID := uuid.New()

	product := domain.Product{
		ID:       productID,
		TenantID: tenantID,
		SKU:      "MAS000123",
		Name:     "Indomie Goreng Original",
	}
	repo.products[fmt.Sprintf("%s:%s", tenantID, product.SKU)] = product

	// 1. Direct SKU match
	t.Run("Resolve by internal product SKU", func(t *testing.T) {
		res, err := usecase.ResolveBarcode(ctx, tenantID, "MAS000123")
		require.NoError(t, err)
		assert.Equal(t, productID, res.ProductID)
		assert.Equal(t, "MAS000123", res.SKU)
		assert.Equal(t, "Indomie Goreng Original", res.Name)
		assert.Equal(t, "SKU", res.Source)
		assert.True(t, res.Multiplier.Equal(decimal.NewFromInt(1)))
	})

	// 2. Physical barcode match
	barcode := "8998866200234"
	repo.barcodes[fmt.Sprintf("%s:%s", tenantID, barcode)] = domain.ProductBarcode{
		ID:               uuid.New(),
		TenantID:         tenantID,
		ProductID:        productID,
		Barcode:          barcode,
		BarcodeSymbology: "EAN13",
		Multiplier:       decimal.NewFromInt(24), // Carton = 24 pcs
	}

	t.Run("Resolve by physical barcode EAN-13 with multiplier", func(t *testing.T) {
		res, err := usecase.ResolveBarcode(ctx, tenantID, barcode)
		require.NoError(t, err)
		assert.Equal(t, productID, res.ProductID)
		assert.Equal(t, "Indomie Goreng Original", res.Name)
		assert.Equal(t, barcode, res.Barcode)
		assert.Equal(t, "BARCODE", res.Source)
		assert.True(t, res.Multiplier.Equal(decimal.NewFromInt(24)))
	})

	// 3. Omnichannel external SKU mapping
	extSKU := "SHOPEE-INDOMIE-GORENG-CTN"
	repo.skuMappings[fmt.Sprintf("%s:%s", tenantID, extSKU)] = domain.ProductSKUMapping{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ProductID:   productID,
		MappingType: domain.SKUMappingTypeMarketplace,
		ChannelName: "SHOPEE",
		ExternalSKU: extSKU,
		Multiplier:  decimal.NewFromInt(40), // Dus 40 pcs
	}

	t.Run("Resolve by marketplace external SKU", func(t *testing.T) {
		res, err := usecase.ResolveBarcode(ctx, tenantID, extSKU)
		require.NoError(t, err)
		assert.Equal(t, productID, res.ProductID)
		assert.Equal(t, extSKU, res.ExternalSKU)
		assert.Equal(t, "MAPPING", res.Source)
		assert.True(t, res.Multiplier.Equal(decimal.NewFromInt(40)))
	})

	t.Run("Resolve non-existent barcode returns ErrBarcodeNotFound", func(t *testing.T) {
		_, err := usecase.ResolveBarcode(ctx, tenantID, "UNKNOWN-BARCODE-999")
		assert.ErrorIs(t, err, domain.ErrBarcodeNotFound)
	})

	t.Run("Resolve empty code returns ErrInvalidInput", func(t *testing.T) {
		_, err := usecase.ResolveBarcode(ctx, tenantID, "   ")
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}

func TestStockTransferLifecycleAndNegativeStockRejection(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	adminID := uuid.New()
	approverID := uuid.New()
	whSource := uuid.New()
	whTarget := uuid.New()
	productID := uuid.New()

	locSourceID := uuid.New()
	locTargetID := uuid.New()

	repo.warehouses[whSource] = domain.Warehouse{ID: whSource, TenantID: tenantID, Code: "WHS", Name: "Warehouse Source"}
	repo.warehouses[whTarget] = domain.Warehouse{ID: whTarget, TenantID: tenantID, Code: "WHT", Name: "Warehouse Target"}
	repo.locations[locSourceID] = domain.WarehouseLocation{ID: locSourceID, TenantID: tenantID, WarehouseID: &whSource, Code: "SRC-BIN-01", Name: "Rack 01"}
	repo.locations[locTargetID] = domain.WarehouseLocation{ID: locTargetID, TenantID: tenantID, WarehouseID: &whTarget, Code: "DST-BIN-01", Name: "Rack 02"}

	// Set initial stock: 5 units available at source location
	stockKey := fmt.Sprintf("%s:%s:%s", tenantID, locSourceID, productID)
	repo.stockLevels[stockKey] = decimal.NewFromInt(5)

	// approveAndReturn submits a DRAFT transfer and approves it with a
	// different user than the requester, returning the APPROVED transfer.
	approveTransfer := func(t *testing.T, tr *domain.StockTransfer) *domain.StockTransfer {
		_, err := usecase.SubmitTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		require.NoError(t, err)
		approved, err := usecase.ApproveTransfer(ctx, tenantID, approverID, "admin", tr.ID)
		require.NoError(t, err)
		return approved
	}

	t.Run("AC-3.1 & Guardrail 2: Reject dispatch when requested qty exceeds available stock", func(t *testing.T) {
		reqTransfer := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  "TR-TEST-001",
			Items: []uc.CreateTransferItemRequest{
				{
					ProductID:        productID,
					RequestedQty:     decimal.NewFromInt(10), // Requesting 10, but only 5 in stock!
					SourceLocationID: &locSourceID,
					DestLocationID:   &locTargetID,
				},
			},
		}

		tr, err := usecase.CreateTransfer(ctx, tenantID, adminID, "admin", reqTransfer)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusDraft, tr.Status)

		approveTransfer(t, tr)

		// Attempt dispatch -> must fail with ErrInsufficientStock!
		_, err = usecase.DispatchTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInsufficientStock, "must reject dispatch due to insufficient stock")
	})

	t.Run("AC-3.1 & AC-3.2: Successful dispatch and receive with stock movements", func(t *testing.T) {
		// Top up stock to 50 units
		repo.stockLevels[stockKey] = decimal.NewFromInt(50)

		reqTransfer := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  "TR-TEST-002",
			Items: []uc.CreateTransferItemRequest{
				{
					ProductID:        productID,
					RequestedQty:     decimal.NewFromInt(10),
					SourceLocationID: &locSourceID,
					DestLocationID:   &locTargetID,
				},
			},
		}

		tr, err := usecase.CreateTransfer(ctx, tenantID, adminID, "admin", reqTransfer)
		require.NoError(t, err)

		approveTransfer(t, tr)

		// 1. Dispatch transfer: Source WH -> @TRANSIT
		dispatchedTr, err := usecase.DispatchTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusInTransit, dispatchedTr.Status)
		assert.NotNil(t, dispatchedTr.DispatchedAt)

		// Source stock should decrease from 50 to 40
		sourceStock, err := repo.GetStockByLocation(ctx, tenantID, locSourceID, productID)
		require.NoError(t, err)
		assert.True(t, sourceStock.Equal(decimal.NewFromInt(40)), "source stock should decrease by 10")

		// Destination stock should NOT increase yet
		targetStockBefore, err := repo.GetStockByLocation(ctx, tenantID, locTargetID, productID)
		require.NoError(t, err)
		assert.True(t, targetStockBefore.Equal(decimal.Zero), "target warehouse stock should not increase yet while in transit")

		// 2. Reject re-dispatch of in-transit transfer
		_, err = usecase.DispatchTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus, "cannot re-dispatch in-transit transfer")

		// 3. Receive transfer: @TRANSIT -> Target Location
		receivedTr, err := usecase.ReceiveTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusReceived, receivedTr.Status)
		assert.NotNil(t, receivedTr.ReceivedAt)

		// Target stock should now increase by 10
		targetStockAfter, err := repo.GetStockByLocation(ctx, tenantID, locTargetID, productID)
		require.NoError(t, err)
		assert.True(t, targetStockAfter.Equal(decimal.NewFromInt(10)), "target warehouse stock should increase by 10")

		// 4. Reject re-receive of already received transfer
		_, err = usecase.ReceiveTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus, "cannot re-receive already received transfer")
	})

	t.Run("Reject dispatching PENDING_APPROVAL transfer", func(t *testing.T) {
		reqTransfer := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  "TR-PENDING-001",
			Items: []uc.CreateTransferItemRequest{
				{
					ProductID:        productID,
					RequestedQty:     decimal.NewFromInt(5),
					SourceLocationID: &locSourceID,
					DestLocationID:   &locTargetID,
				},
			},
		}

		tr, err := usecase.CreateTransfer(ctx, tenantID, adminID, "admin", reqTransfer)
		require.NoError(t, err)

		// Manually set status to PENDING_APPROVAL
		tr.Status = domain.TransferStatusPendingApproval

		// DispatchTransfer must be rejected with ErrInvalidTransferStatus
		_, err = usecase.DispatchTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus, "dispatching PENDING_APPROVAL transfer must be rejected")
	})

	t.Run("Reject location spoofing across warehouses on transfer creation and dispatch", func(t *testing.T) {
		// Location belonging to whTarget used as source_location_id (which should belong to whSource)
		reqSpoofedSource := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  "TR-SPOOF-01",
			Items: []uc.CreateTransferItemRequest{
				{
					ProductID:        productID,
					RequestedQty:     decimal.NewFromInt(2),
					SourceLocationID: &locTargetID, // WRONG warehouse! Belongs to whTarget, not whSource
					DestLocationID:   &locTargetID,
				},
			},
		}
		_, err := usecase.CreateTransfer(ctx, tenantID, adminID, "admin", reqSpoofedSource)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse, "source location not in source warehouse must be rejected")

		// Location belonging to whSource used as dest_location_id (which should belong to whTarget)
		reqSpoofedDest := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  "TR-SPOOF-02",
			Items: []uc.CreateTransferItemRequest{
				{
					ProductID:        productID,
					RequestedQty:     decimal.NewFromInt(2),
					SourceLocationID: &locSourceID,
					DestLocationID:   &locSourceID, // WRONG warehouse! Belongs to whSource, not whTarget
				},
			},
		}
		_, err = usecase.CreateTransfer(ctx, tenantID, adminID, "admin", reqSpoofedDest)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse, "dest location not in dest warehouse must be rejected")

		// Transfer created normally, but item modified/spoofed before dispatch
		legitReq := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  "TR-SPOOF-03",
			Items: []uc.CreateTransferItemRequest{
				{
					ProductID:        productID,
					RequestedQty:     decimal.NewFromInt(1),
					SourceLocationID: &locSourceID,
					DestLocationID:   &locTargetID,
				},
			},
		}
		tr, err := usecase.CreateTransfer(ctx, tenantID, adminID, "admin", legitReq)
		require.NoError(t, err)

		approveTransfer(t, tr)

		// Tamper with item's source location to point to target warehouse
		items := repo.transferItems[tr.ID]
		items[0].SourceLocationID = &locTargetID
		repo.transferItems[tr.ID] = items

		_, err = usecase.DispatchTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse, "dispatch with spoofed location must be rejected")
	})

	t.Run("Auditor write attempt on transfer is rejected", func(t *testing.T) {
		auditorID := uuid.New()
		legitReq := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  "TR-AUD-01",
			Items: []uc.CreateTransferItemRequest{
				{
					ProductID:        productID,
					RequestedQty:     decimal.NewFromInt(1),
					SourceLocationID: &locSourceID,
					DestLocationID:   &locTargetID,
				},
			},
		}
		tr, err := usecase.CreateTransfer(ctx, tenantID, adminID, "admin", legitReq)
		require.NoError(t, err)

		// Auditor dispatch attempt -> ErrForbidden
		_, err = usecase.DispatchTransfer(ctx, tenantID, auditorID, "auditor", tr.ID)
		assert.ErrorIs(t, err, domain.ErrForbidden, "auditor cannot dispatch transfer")

		// Auditor receive attempt -> ErrForbidden
		_, err = usecase.ReceiveTransfer(ctx, tenantID, auditorID, "auditor", tr.ID)
		assert.ErrorIs(t, err, domain.ErrForbidden, "auditor cannot receive transfer")
	})
}

func TestStockTransferApprovalFlow(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	requesterID := uuid.New()  // "warehouse" staff who creates the transfer
	approverID := uuid.New()   // "admin" who approves/rejects
	regionalMgrID := uuid.New()
	whSource := uuid.New()
	whTarget := uuid.New()
	productID := uuid.New()

	locSourceID := uuid.New()
	locTargetID := uuid.New()

	repo.warehouses[whSource] = domain.Warehouse{ID: whSource, TenantID: tenantID, Code: "WHS", Name: "Warehouse Source"}
	repo.warehouses[whTarget] = domain.Warehouse{ID: whTarget, TenantID: tenantID, Code: "WHT", Name: "Warehouse Target"}
	repo.locations[locSourceID] = domain.WarehouseLocation{ID: locSourceID, TenantID: tenantID, WarehouseID: &whSource, Code: "SRC-BIN-01", Name: "Rack 01"}
	repo.locations[locTargetID] = domain.WarehouseLocation{ID: locTargetID, TenantID: tenantID, WarehouseID: &whTarget, Code: "DST-BIN-01", Name: "Rack 02"}
	repo.userWarehouses[fmt.Sprintf("%s:%s", tenantID, requesterID)] = []uuid.UUID{whSource}

	newDraftTransfer := func(t *testing.T, number string) *domain.StockTransfer {
		req := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  number,
			Items: []uc.CreateTransferItemRequest{
				{
					ProductID:        productID,
					RequestedQty:     decimal.NewFromInt(1),
					SourceLocationID: &locSourceID,
					DestLocationID:   &locTargetID,
				},
			},
		}
		tr, err := usecase.CreateTransfer(ctx, tenantID, requesterID, "warehouse", req)
		require.NoError(t, err)
		return tr
	}

	t.Run("Happy path: DRAFT -> PENDING_APPROVAL -> APPROVED unlocks dispatch", func(t *testing.T) {
		tr := newDraftTransfer(t, "TR-APV-001")

		submitted, err := usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusPendingApproval, submitted.Status)

		// DispatchTransfer must be rejected while still PENDING_APPROVAL
		_, err = usecase.DispatchTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus, "cannot dispatch a PENDING_APPROVAL transfer")

		approved, err := usecase.ApproveTransfer(ctx, tenantID, approverID, "admin", tr.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusApproved, approved.Status)
		require.NotNil(t, approved.ApprovedBy)
		assert.Equal(t, approverID, *approved.ApprovedBy)
	})

	t.Run("Reject submitting a transfer that is not DRAFT", func(t *testing.T) {
		tr := newDraftTransfer(t, "TR-APV-002")
		_, err := usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		require.NoError(t, err)

		_, err = usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus, "cannot re-submit a PENDING_APPROVAL transfer")
	})

	t.Run("Guardrail: warehouse staff role cannot approve or reject transfers", func(t *testing.T) {
		tr := newDraftTransfer(t, "TR-APV-003")
		_, err := usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		require.NoError(t, err)

		_, err = usecase.ApproveTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrForbidden, "warehouse staff role cannot approve")

		_, err = usecase.RejectTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID, "tidak cukup stok")
		assert.ErrorIs(t, err, domain.ErrForbidden, "warehouse staff role cannot reject")
	})

	t.Run("Guardrail: requester cannot approve or reject their own transfer even as admin", func(t *testing.T) {
		req := uc.CreateTransferRequest{
			FromWarehouseID: whSource,
			ToWarehouseID:   whTarget,
			TransferNumber:  "TR-APV-004",
			Items: []uc.CreateTransferItemRequest{
				{ProductID: productID, RequestedQty: decimal.NewFromInt(1), SourceLocationID: &locSourceID, DestLocationID: &locTargetID},
			},
		}
		// This time the requester themself holds the "admin" role.
		tr, err := usecase.CreateTransfer(ctx, tenantID, approverID, "admin", req)
		require.NoError(t, err)
		_, err = usecase.SubmitTransfer(ctx, tenantID, approverID, "admin", tr.ID)
		require.NoError(t, err)

		_, err = usecase.ApproveTransfer(ctx, tenantID, approverID, "admin", tr.ID)
		assert.ErrorIs(t, err, domain.ErrSelfApprovalForbidden, "requester cannot self-approve")

		_, err = usecase.RejectTransfer(ctx, tenantID, approverID, "admin", tr.ID, "alasan apapun")
		assert.ErrorIs(t, err, domain.ErrSelfApprovalForbidden, "requester cannot self-reject")
	})

	t.Run("Reject requires a non-empty reason and persists it", func(t *testing.T) {
		tr := newDraftTransfer(t, "TR-APV-005")
		_, err := usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		require.NoError(t, err)

		_, err = usecase.RejectTransfer(ctx, tenantID, approverID, "admin", tr.ID, "   ")
		assert.ErrorIs(t, err, domain.ErrRejectionReasonRequired, "blank rejection reason must be rejected")

		rejected, err := usecase.RejectTransfer(ctx, tenantID, approverID, "admin", tr.ID, "Stok gudang asal menipis untuk pesanan lokal")
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusRejected, rejected.Status)
		require.NotNil(t, rejected.RejectionReason)
		assert.Equal(t, "Stok gudang asal menipis untuk pesanan lokal", *rejected.RejectionReason)

		// A REJECTED transfer cannot be dispatched, approved again, or resubmitted.
		_, err = usecase.DispatchTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus)
		_, err = usecase.ApproveTransfer(ctx, tenantID, approverID, "admin", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus)
		_, err = usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus)
	})

	t.Run("Regional manager with regional access can approve", func(t *testing.T) {
		regionalID := uuid.New()
		wh := repo.warehouses[whSource]
		wh.RegionalID = &regionalID
		repo.warehouses[whSource] = wh
		repo.userWarehouses[fmt.Sprintf("%s:%s", tenantID, regionalMgrID)] = []uuid.UUID{whSource}

		tr := newDraftTransfer(t, "TR-APV-006")
		_, err := usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		require.NoError(t, err)

		approved, err := usecase.ApproveTransfer(ctx, tenantID, regionalMgrID, "regional_manager", tr.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusApproved, approved.Status)
	})

	t.Run("DispatchTransfer no longer allows jumping straight from DRAFT", func(t *testing.T) {
		tr := newDraftTransfer(t, "TR-APV-007")
		_, err := usecase.DispatchTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus, "DRAFT transfer must be submitted and approved before dispatch")
	})

	t.Run("CancelTransfer: requester can cancel a DRAFT transfer", func(t *testing.T) {
		tr := newDraftTransfer(t, "TR-CANCEL-001")

		cancelled, err := usecase.CancelTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusCancelled, cancelled.Status)
		assert.Equal(t, "TR-CANCEL-001", cancelled.TransferNumber)
	})

	t.Run("CancelTransfer: guardrail rejects anything past DRAFT", func(t *testing.T) {
		tr := newDraftTransfer(t, "TR-CANCEL-002")
		_, err := usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		require.NoError(t, err)

		_, err = usecase.CancelTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrTransferNotDraft, "PENDING_APPROVAL transfer must not be cancellable")

		// Cancelling twice must also fail (already CANCELLED, not DRAFT).
		draft := newDraftTransfer(t, "TR-CANCEL-003")
		_, err = usecase.CancelTransfer(ctx, tenantID, requesterID, "warehouse", draft.ID)
		require.NoError(t, err)
		_, err = usecase.CancelTransfer(ctx, tenantID, requesterID, "warehouse", draft.ID)
		assert.ErrorIs(t, err, domain.ErrTransferNotDraft, "already cancelled transfer must not be cancellable again")
	})

	t.Run("CancelTransfer: a cancelled transfer can never be submitted or dispatched", func(t *testing.T) {
		tr := newDraftTransfer(t, "TR-CANCEL-004")
		_, err := usecase.CancelTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		require.NoError(t, err)

		_, err = usecase.SubmitTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus)
		_, err = usecase.DispatchTransfer(ctx, tenantID, requesterID, "warehouse", tr.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidTransferStatus)
	})
}

func TestDeliveryOrderDispatchAndStockDeduction(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	adminID := uuid.New()
	whID := uuid.New()
	salesOrderID := uuid.New()
	productID := uuid.New()
	locID := uuid.New()

	repo.warehouses[whID] = domain.Warehouse{ID: whID, TenantID: tenantID, Code: "WH-DO", Name: "Distribution Center"}
	repo.locations[locID] = domain.WarehouseLocation{ID: locID, TenantID: tenantID, WarehouseID: &whID, Code: "BIN-OUT", Name: "Outbound Bay"}

	t.Run("Reject delivery order creation when available stock is insufficient", func(t *testing.T) {
		// Only 2 units in stock
		locKey := fmt.Sprintf("%s:%s:%s", tenantID, locID, productID)
		repo.stockLevels[locKey] = decimal.NewFromInt(2)

		req := uc.CreateDeliveryOrderRequest{
			SalesOrderID:  &salesOrderID,
			WarehouseID:   whID,
			DONumber:      "DO-FAIL-01",
			RecipientName: ptr("Customer Test"),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{
					ProductID:  productID,
					Quantity:   decimal.NewFromInt(10), // Requesting 10!
					LocationID: locID,
				},
			},
		}

		_, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", req)
		assert.ErrorIs(t, err, domain.ErrInsufficientStock)
	})

	t.Run("Reject delivery order dispatch when stock is insufficient", func(t *testing.T) {
		// Sufficient stock at creation time
		locKey := fmt.Sprintf("%s:%s:%s", tenantID, locID, productID)
		repo.stockLevels[locKey] = decimal.NewFromInt(10)

		req := uc.CreateDeliveryOrderRequest{
			SalesOrderID:   &salesOrderID,
			WarehouseID:    whID,
			DONumber:       "DO-FAIL-02",
			RecipientName:  ptr("Customer Test"),
			DriverName:     ptr("Pak Driver"),
			VehiclePlate:   ptr("B 1234 CD"),
			ExpeditionName: ptr("JNE"),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{
					ProductID:  productID,
					Quantity:   decimal.NewFromInt(10),
					LocationID: locID,
				},
			},
		}

		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", req)
		require.NoError(t, err)

		_, err = usecase.ConfirmDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		require.NoError(t, err)

		// Stock drops before dispatch
		repo.stockLevels[locKey] = decimal.NewFromInt(2)

		// Attempt dispatch -> must fail with ErrInsufficientStock
		_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		assert.ErrorIs(t, err, domain.ErrInsufficientStock)
	})

	t.Run("AC-4.1: Successful DO dispatch moves stock from location to @CUSTOMER", func(t *testing.T) {
		// Stock available: 20 units
		locKey := fmt.Sprintf("%s:%s:%s", tenantID, locID, productID)
		repo.stockLevels[locKey] = decimal.NewFromInt(20)

		req := uc.CreateDeliveryOrderRequest{
			SalesOrderID:   &salesOrderID,
			WarehouseID:    whID,
			DONumber:       "DO-SUCCESS-01",
			RecipientName:  ptr("Customer Test"),
			DriverName:     ptr("Pak Driver"),
			VehiclePlate:   ptr("B 1234 CD"),
			ExpeditionName: ptr("JNE"),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{
					ProductID:  productID,
					Quantity:   decimal.NewFromInt(8),
					LocationID: locID,
				},
			},
		}

		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", req)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryOrderStatusDraft, do.Status)

		_, err = usecase.ConfirmDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		require.NoError(t, err)

		shippedDO, err := usecase.DispatchDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryOrderStatusShipped, shippedDO.Status)

		// Stock at location should decrease by 8 (from 20 to 12)
		currentStock, err := repo.GetStockByLocation(ctx, tenantID, locID, productID)
		require.NoError(t, err)
		assert.True(t, currentStock.Equal(decimal.NewFromInt(12)), "stock should be 12 after shipping 8")

		// Re-dispatching shipped DO should return ErrDeliveryOrderNotPacked
		_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		assert.ErrorIs(t, err, domain.ErrDeliveryOrderNotPacked)
	})

	// Regression: the 13-module app has no Sales Order screen, so a direct
	// Surat Jalan (no SO) must be creatable and dispatchable.
	t.Run("Direct Surat Jalan without Sales Order is created and dispatched", func(t *testing.T) {
		locKey := fmt.Sprintf("%s:%s:%s", tenantID, locID, productID)
		repo.stockLevels[locKey] = decimal.NewFromInt(5)

		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", uc.CreateDeliveryOrderRequest{
			WarehouseID:    whID,
			DONumber:       "DO-DIRECT-01",
			RecipientName:  ptr("Customer Test"),
			DriverName:     ptr("Pak Driver"),
			VehiclePlate:   ptr("B 1234 CD"),
			ExpeditionName: ptr("JNE"),
			Items:          []uc.CreateDeliveryOrderItemRequest{{ProductID: productID, Quantity: decimal.NewFromInt(2), LocationID: locID}},
		})
		require.NoError(t, err)
		assert.Nil(t, do.SalesOrderID, "direct DO must not carry a Sales Order id")

		_, err = usecase.ConfirmDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		require.NoError(t, err)

		shipped, err := usecase.DispatchDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryOrderStatusShipped, shipped.Status)
		stock, err := repo.GetStockByLocation(ctx, tenantID, locID, productID)
		require.NoError(t, err)
		assert.True(t, stock.Equal(decimal.NewFromInt(3)), "stock should be 3 after shipping 2")
	})

	t.Run("Zero UUID Sales Order is normalised to none", func(t *testing.T) {
		zero := uuid.Nil
		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", uc.CreateDeliveryOrderRequest{
			SalesOrderID:  &zero,
			WarehouseID:   whID,
			DONumber:      "DO-DIRECT-02",
			RecipientName: ptr("Customer Test"),
			Items:         []uc.CreateDeliveryOrderItemRequest{{ProductID: productID, Quantity: decimal.NewFromInt(1), LocationID: locID}},
		})
		require.NoError(t, err)
		assert.Nil(t, do.SalesOrderID)
	})

	t.Run("Real Sales Order id is preserved", func(t *testing.T) {
		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", uc.CreateDeliveryOrderRequest{
			SalesOrderID:  &salesOrderID,
			WarehouseID:   whID,
			DONumber:      "DO-WITH-SO-01",
			RecipientName: ptr("Customer Test"),
			Items:         []uc.CreateDeliveryOrderItemRequest{{ProductID: productID, Quantity: decimal.NewFromInt(1), LocationID: locID}},
		})
		require.NoError(t, err)
		require.NotNil(t, do.SalesOrderID)
		assert.Equal(t, salesOrderID, *do.SalesOrderID)
	})

	t.Run("Force Delivery Order status to DRAFT on creation", func(t *testing.T) {
		shippedStatus := domain.DeliveryOrderStatusShipped
		req := uc.CreateDeliveryOrderRequest{
			SalesOrderID:  &salesOrderID,
			WarehouseID:   whID,
			DONumber:      "DO-FORCE-DRAFT-01",
			RecipientName: ptr("Customer Test"),
			Status:        &shippedStatus, // Attempt to create as SHIPPED
			Items: []uc.CreateDeliveryOrderItemRequest{
				{
					ProductID:  productID,
					Quantity:   decimal.NewFromInt(1),
					LocationID: locID,
				},
			},
		}

		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", req)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryOrderStatusDraft, do.Status, "delivery order must always be created in DRAFT status")
	})

	t.Run("Reject dispatching CANCELLED or RETURNED delivery orders", func(t *testing.T) {
		req := uc.CreateDeliveryOrderRequest{
			SalesOrderID:   &salesOrderID,
			WarehouseID:    whID,
			DONumber:       "DO-STATUS-TEST-01",
			RecipientName:  ptr("Customer Test"),
			DriverName:     ptr("Pak Driver"),
			VehiclePlate:   ptr("B 1234 CD"),
			ExpeditionName: ptr("JNE"),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{
					ProductID:  productID,
					Quantity:   decimal.NewFromInt(1),
					LocationID: locID,
				},
			},
		}

		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", req)
		require.NoError(t, err)

		// CANCELLED status -> rejected
		do.Status = domain.DeliveryOrderStatusCancelled
		_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		assert.ErrorIs(t, err, domain.ErrDeliveryOrderNotPacked, "dispatching CANCELLED delivery order must be rejected")

		// RETURNED status -> rejected
		do.Status = domain.DeliveryOrderStatusReturned
		_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		assert.ErrorIs(t, err, domain.ErrDeliveryOrderNotPacked, "dispatching RETURNED delivery order must be rejected")

		// DELIVERED status -> rejected
		do.Status = domain.DeliveryOrderStatusDelivered
		_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		assert.ErrorIs(t, err, domain.ErrDeliveryOrderNotPacked, "dispatching DELIVERED delivery order must be rejected")
	})

	t.Run("Reject location spoofing on delivery order create and dispatch", func(t *testing.T) {
		otherWH := uuid.New()
		repo.warehouses[otherWH] = domain.Warehouse{ID: otherWH, TenantID: tenantID, Code: "OTHER-WH", Name: "Other Warehouse"}
		otherLoc := uuid.New()
		repo.locations[otherLoc] = domain.WarehouseLocation{ID: otherLoc, TenantID: tenantID, WarehouseID: &otherWH, Code: "OTHER-LOC", Name: "Other Loc"}

		// CreateDeliveryOrder with LocationID belonging to otherWH
		reqSpoof := uc.CreateDeliveryOrderRequest{
			SalesOrderID:  &salesOrderID,
			WarehouseID:   whID,
			DONumber:      "DO-SPOOF-01",
			RecipientName: ptr("Customer Test"),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{
					ProductID:  productID,
					Quantity:   decimal.NewFromInt(1),
					LocationID: otherLoc, // Mismatched location warehouse!
				},
			},
		}
		_, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", reqSpoof)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse, "location from different warehouse must be rejected on creation")

		// Create legit DO, then tamper item location before dispatch
		legitReq := uc.CreateDeliveryOrderRequest{
			SalesOrderID:   &salesOrderID,
			WarehouseID:    whID,
			DONumber:       "DO-SPOOF-02",
			RecipientName:  ptr("Customer Test"),
			DriverName:     ptr("Pak Driver"),
			VehiclePlate:   ptr("B 1234 CD"),
			ExpeditionName: ptr("JNE"),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{
					ProductID:  productID,
					Quantity:   decimal.NewFromInt(1),
					LocationID: locID,
				},
			},
		}
		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", legitReq)
		require.NoError(t, err)

		_, err = usecase.ConfirmDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		require.NoError(t, err)

		// Tamper item location
		items := repo.doItems[do.ID]
		items[0].LocationID = otherLoc
		repo.doItems[do.ID] = items

		_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse, "dispatching DO with spoofed location must be rejected")
	})

	t.Run("Auditor cannot create or dispatch delivery order", func(t *testing.T) {
		auditorID := uuid.New()
		req := uc.CreateDeliveryOrderRequest{
			SalesOrderID:   &salesOrderID,
			WarehouseID:    whID,
			DONumber:       "DO-AUD-01",
			RecipientName:  ptr("Customer Test"),
			DriverName:     ptr("Pak Driver"),
			VehiclePlate:   ptr("B 1234 CD"),
			ExpeditionName: ptr("JNE"),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{
					ProductID:  productID,
					Quantity:   decimal.NewFromInt(1),
					LocationID: locID,
				},
			},
		}
		_, err := usecase.CreateDeliveryOrder(ctx, tenantID, auditorID, "auditor", req)
		assert.ErrorIs(t, err, domain.ErrForbidden, "auditor cannot create delivery order")

		// Create as admin
		do, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", req)
		require.NoError(t, err)

		_, err = usecase.ConfirmDeliveryOrder(ctx, tenantID, adminID, "admin", do.ID)
		require.NoError(t, err)

		// Dispatch as auditor -> rejected
		_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, auditorID, "auditor", do.ID)
		assert.ErrorIs(t, err, domain.ErrForbidden, "auditor cannot dispatch delivery order")
	})
}

func TestCrossTenantMultiTenancyIsolation(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantA := uuid.New()
	tenantB := uuid.New()

	userA := uuid.New()
	userB := uuid.New()

	// Seed Tenant A resources
	whA := domain.Warehouse{ID: uuid.New(), TenantID: tenantA, Code: "WHA-01", Name: "Warehouse Tenant A"}
	whA2 := domain.Warehouse{ID: uuid.New(), TenantID: tenantA, Code: "WHA-02", Name: "Warehouse Tenant A Target"}
	locA := domain.WarehouseLocation{ID: uuid.New(), TenantID: tenantA, WarehouseID: &whA.ID, Code: "LOCA-01", Name: "Bay A1"}
	prodA := domain.Product{ID: uuid.New(), TenantID: tenantA, SKU: "SKU-A-100", Name: "Product Tenant A"}

	repo.warehouses[whA.ID] = whA
	repo.warehouses[whA2.ID] = whA2
	repo.locations[locA.ID] = locA
	repo.products[fmt.Sprintf("%s:%s", tenantA, prodA.SKU)] = prodA

	// Seed Tenant B resources
	whB := domain.Warehouse{ID: uuid.New(), TenantID: tenantB, Code: "WHB-01", Name: "Warehouse Tenant B"}
	whB2 := domain.Warehouse{ID: uuid.New(), TenantID: tenantB, Code: "WHB-02", Name: "Warehouse Tenant B Target"}
	locB := domain.WarehouseLocation{ID: uuid.New(), TenantID: tenantB, WarehouseID: &whB.ID, Code: "LOCB-01", Name: "Bay B1"}
	locB2 := domain.WarehouseLocation{ID: uuid.New(), TenantID: tenantB, WarehouseID: &whB2.ID, Code: "LOCB-02", Name: "Bay B2"}
	prodB := domain.Product{ID: uuid.New(), TenantID: tenantB, SKU: "SKU-B-200", Name: "Product Tenant B"}

	repo.warehouses[whB.ID] = whB
	repo.warehouses[whB2.ID] = whB2
	repo.locations[locB.ID] = locB
	repo.locations[locB2.ID] = locB2
	repo.products[fmt.Sprintf("%s:%s", tenantB, prodB.SKU)] = prodB

	// Barcodes & SKUs for Tenant B
	barcodeB := "8991112223334"
	repo.barcodes[fmt.Sprintf("%s:%s", tenantB, barcodeB)] = domain.ProductBarcode{
		ID:               uuid.New(),
		TenantID:         tenantB,
		ProductID:        prodB.ID,
		Barcode:          barcodeB,
		BarcodeSymbology: "EAN13",
		Multiplier:       decimal.NewFromInt(1),
	}

	extSKUB := "SHOPEE-PROD-B"
	repo.skuMappings[fmt.Sprintf("%s:%s", tenantB, extSKUB)] = domain.ProductSKUMapping{
		ID:          uuid.New(),
		TenantID:    tenantB,
		ProductID:   prodB.ID,
		MappingType: domain.SKUMappingTypeMarketplace,
		ChannelName: "SHOPEE",
		ExternalSKU: extSKUB,
		Multiplier:  decimal.NewFromInt(1),
	}

	// Transfer for Tenant B
	locKeyB := fmt.Sprintf("%s:%s:%s", tenantB, locB.ID, prodB.ID)
	repo.stockLevels[locKeyB] = decimal.NewFromInt(100)

	trB, err := usecase.CreateTransfer(ctx, tenantB, userB, "admin", uc.CreateTransferRequest{
		FromWarehouseID: whB.ID,
		ToWarehouseID:   whB2.ID,
		TransferNumber:  "TR-TENANT-B-001",
		Items: []uc.CreateTransferItemRequest{
			{
				ProductID:        prodB.ID,
				RequestedQty:     decimal.NewFromInt(10),
				SourceLocationID: &locB.ID,
				DestLocationID:   &locB2.ID,
			},
		},
	})
	require.NoError(t, err)

	// Delivery Order for Tenant B
	doB, err := usecase.CreateDeliveryOrder(ctx, tenantB, userB, "admin", uc.CreateDeliveryOrderRequest{
		SalesOrderID:  ptrUUID(uuid.New()),
		WarehouseID:   whB.ID,
		DONumber:      "DO-TENANT-B-001",
		RecipientName: ptr("Customer Tenant B"),
		Items: []uc.CreateDeliveryOrderItemRequest{
			{
				ProductID:  prodB.ID,
				Quantity:   decimal.NewFromInt(5),
				LocationID: locB.ID,
			},
		},
	})
	require.NoError(t, err)

	t.Run("Tenant A cannot access or list warehouses of Tenant B", func(t *testing.T) {
		// GetWarehouse
		_, err := usecase.GetWarehouse(ctx, tenantA, userA, "admin", whB.ID)
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound, "Tenant A admin must not get Tenant B warehouse")

		// ListWarehouses
		whListA, err := usecase.ListWarehouses(ctx, tenantA, userA, "admin")
		require.NoError(t, err)
		for _, w := range whListA {
			assert.NotEqual(t, whB.ID, w.ID, "Tenant A warehouse list must not leak Tenant B warehouse")
			assert.Equal(t, tenantA, w.TenantID)
		}

		// ValidateWarehouseAccess
		err = usecase.ValidateWarehouseAccess(ctx, tenantA, userA, "admin", whB.ID)
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound, "Tenant A cannot validate access to Tenant B warehouse")
	})

	t.Run("Tenant A cannot access or list locations of Tenant B", func(t *testing.T) {
		// ListLocations tenant-wide
		locsA, err := usecase.ListLocations(ctx, tenantA, userA, "admin", nil)
		require.NoError(t, err)
		for _, l := range locsA {
			assert.NotEqual(t, locB.ID, l.ID, "Tenant A locations must not leak Tenant B location")
			assert.Equal(t, tenantA, l.TenantID)
		}

		// ListLocations specifically specifying Tenant B warehouse
		_, err = usecase.ListLocations(ctx, tenantA, userA, "admin", &whB.ID)
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound, "Tenant A cannot list locations of Tenant B warehouse")

		// CreateLocation in Tenant B warehouse
		_, err = usecase.CreateLocation(ctx, tenantA, userA, "admin", uc.CreateLocationRequest{
			WarehouseID: &whB.ID,
			Code:        "ROGUE-LOC",
			Name:        "Rogue Location",
		})
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound, "Tenant A cannot create location inside Tenant B warehouse")
	})

	t.Run("Tenant A cannot resolve barcodes, external SKUs, or product SKUs of Tenant B", func(t *testing.T) {
		// 1. Resolve internal product SKU of Tenant B
		_, err := usecase.ResolveBarcode(ctx, tenantA, prodB.SKU)
		assert.ErrorIs(t, err, domain.ErrBarcodeNotFound, "Tenant A cannot resolve Tenant B product SKU")

		// 2. Resolve physical barcode of Tenant B
		_, err = usecase.ResolveBarcode(ctx, tenantA, barcodeB)
		assert.ErrorIs(t, err, domain.ErrBarcodeNotFound, "Tenant A cannot resolve Tenant B barcode")

		// 3. Resolve marketplace SKU mapping of Tenant B
		_, err = usecase.ResolveBarcode(ctx, tenantA, extSKUB)
		assert.ErrorIs(t, err, domain.ErrBarcodeNotFound, "Tenant A cannot resolve Tenant B marketplace SKU")

		// Verify Tenant B CAN resolve its own items
		resSKU, err := usecase.ResolveBarcode(ctx, tenantB, prodB.SKU)
		require.NoError(t, err)
		assert.Equal(t, prodB.ID, resSKU.ProductID)

		resBarcode, err := usecase.ResolveBarcode(ctx, tenantB, barcodeB)
		require.NoError(t, err)
		assert.Equal(t, prodB.ID, resBarcode.ProductID)

		resMapping, err := usecase.ResolveBarcode(ctx, tenantB, extSKUB)
		require.NoError(t, err)
		assert.Equal(t, prodB.ID, resMapping.ProductID)
	})

	t.Run("Tenant A cannot access, list, dispatch, receive, or create transfers of Tenant B", func(t *testing.T) {
		// GetTransfer
		_, _, err := usecase.GetTransfer(ctx, tenantA, userA, "admin", trB.ID)
		assert.ErrorIs(t, err, domain.ErrTransferNotFound, "Tenant A cannot get Tenant B transfer")

		// ListTransfers
		transfersA, err := usecase.ListTransfers(ctx, tenantA, userA, "admin", nil)
		require.NoError(t, err)
		for _, tr := range transfersA {
			assert.NotEqual(t, trB.ID, tr.ID, "Tenant A transfers must not leak Tenant B transfer")
			assert.Equal(t, tenantA, tr.TenantID)
		}

		// DispatchTransfer
		_, err = usecase.DispatchTransfer(ctx, tenantA, userA, "admin", trB.ID)
		assert.ErrorIs(t, err, domain.ErrTransferNotFound, "Tenant A cannot dispatch Tenant B transfer")

		// ReceiveTransfer
		_, err = usecase.ReceiveTransfer(ctx, tenantA, userA, "admin", trB.ID)
		assert.ErrorIs(t, err, domain.ErrTransferNotFound, "Tenant A cannot receive Tenant B transfer")

		// CreateTransfer originating from Tenant B warehouse
		_, err = usecase.CreateTransfer(ctx, tenantA, userA, "admin", uc.CreateTransferRequest{
			FromWarehouseID: whB.ID,
			ToWarehouseID:   whA2.ID,
			TransferNumber:  "TR-ROGUE-01",
			Items: []uc.CreateTransferItemRequest{
				{ProductID: prodA.ID, RequestedQty: decimal.NewFromInt(1)},
			},
		})
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound, "Tenant A cannot create transfer from Tenant B warehouse")

		// CreateTransfer destined to Tenant B warehouse
		_, err = usecase.CreateTransfer(ctx, tenantA, userA, "admin", uc.CreateTransferRequest{
			FromWarehouseID: whA.ID,
			ToWarehouseID:   whB.ID,
			TransferNumber:  "TR-ROGUE-02",
			Items: []uc.CreateTransferItemRequest{
				{ProductID: prodA.ID, RequestedQty: decimal.NewFromInt(1)},
			},
		})
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound, "Tenant A cannot create transfer to Tenant B warehouse")
	})

	t.Run("Tenant A cannot access, list, or dispatch delivery orders of Tenant B", func(t *testing.T) {
		// GetDeliveryOrder
		_, _, err := usecase.GetDeliveryOrder(ctx, tenantA, userA, "admin", doB.ID)
		assert.ErrorIs(t, err, domain.ErrDeliveryOrderNotFound, "Tenant A cannot get Tenant B DO")

		// ListDeliveryOrders
		doListA, err := usecase.ListDeliveryOrders(ctx, tenantA, userA, "admin", nil)
		require.NoError(t, err)
		for _, do := range doListA {
			assert.NotEqual(t, doB.ID, do.ID, "Tenant A DO list must not leak Tenant B DO")
			assert.Equal(t, tenantA, do.TenantID)
		}

		// DispatchDeliveryOrder
		_, err = usecase.DispatchDeliveryOrder(ctx, tenantA, userA, "admin", doB.ID)
		assert.ErrorIs(t, err, domain.ErrDeliveryOrderNotFound, "Tenant A cannot dispatch Tenant B DO")

		// CreateDeliveryOrder in Tenant B warehouse
		_, err = usecase.CreateDeliveryOrder(ctx, tenantA, userA, "admin", uc.CreateDeliveryOrderRequest{
			SalesOrderID:  ptrUUID(uuid.New()),
			WarehouseID:   whB.ID,
			DONumber:      "DO-ROGUE-01",
			RecipientName: ptr("Customer Rogue"),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{ProductID: prodA.ID, Quantity: decimal.NewFromInt(1), LocationID: locA.ID},
			},
		})
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound, "Tenant A cannot create DO in Tenant B warehouse")
	})

	t.Run("Tenant A cannot access, list, or complete stock opnames of Tenant B", func(t *testing.T) {
		// Setup Tenant B opname
		opB, err := usecase.CreateStockOpname(ctx, tenantB, userB, "admin", uc.CreateStockOpnameRequest{
			WarehouseID: whB.ID,
		})
		require.NoError(t, err)

		// Tenant A cannot get opname of Tenant B
		_, _, err = usecase.GetStockOpname(ctx, tenantA, userA, "admin", opB.ID)
		assert.ErrorIs(t, err, domain.ErrOpnameNotFound)

		// Tenant A list does not contain Tenant B opname
		opListA, err := usecase.ListStockOpnames(ctx, tenantA, userA, "admin", nil)
		require.NoError(t, err)
		for _, o := range opListA {
			assert.NotEqual(t, opB.ID, o.ID)
		}

		// Tenant A cannot add item to Tenant B opname
		_, err = usecase.AddOpnameItem(ctx, tenantA, userA, "admin", opB.ID, uc.AddOpnameItemRequest{
			ProductID:   prodA.ID,
			LocationID:  locA.ID,
			PhysicalQty: decimal.NewFromInt(10),
		})
		assert.ErrorIs(t, err, domain.ErrOpnameNotFound)

		// Tenant A cannot complete Tenant B opname
		_, err = usecase.CompleteStockOpname(ctx, tenantA, userA, "admin", opB.ID)
		assert.ErrorIs(t, err, domain.ErrOpnameNotFound)
	})

	t.Run("Tenant A cannot list or create stock scraps for Tenant B", func(t *testing.T) {
		// Setup stock for Tenant B
		repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantB, locB.ID, prodB.ID)] = decimal.NewFromInt(20)

		scrapB, err := usecase.CreateStockScrap(ctx, tenantB, userB, "admin", uc.CreateStockScrapRequest{
			WarehouseID:      whB.ID,
			ProductID:        prodB.ID,
			SourceLocationID: locB.ID,
			Quantity:         decimal.NewFromInt(5),
			Reason:           "Tenant B scrap",
		})
		require.NoError(t, err)

		// Tenant A list does not include Tenant B scrap
		scrapListA, err := usecase.ListStockScraps(ctx, tenantA, userA, "admin", nil)
		require.NoError(t, err)
		for _, s := range scrapListA {
			assert.NotEqual(t, scrapB.ID, s.ID)
		}

		// Tenant A cannot create scrap targeting Tenant B warehouse
		_, err = usecase.CreateStockScrap(ctx, tenantA, userA, "admin", uc.CreateStockScrapRequest{
			WarehouseID:      whB.ID,
			ProductID:        prodA.ID,
			SourceLocationID: locA.ID,
			Quantity:         decimal.NewFromInt(1),
			Reason:           "Rogue scrap",
		})
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound)
	})
}

func TestStockOpnameDiscrepancyAndLedgerPosting(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	otherWhID := uuid.New()
	adminID := uuid.New()
	auditorID := uuid.New()
	staffID := uuid.New()

	// Setup warehouses
	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{ID: whID, TenantID: tenantID, Name: "Main WH", IsActive: true}))
	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{ID: otherWhID, TenantID: tenantID, Name: "Other WH", IsActive: true}))
	require.NoError(t, repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{UserID: staffID, WarehouseID: whID, TenantID: tenantID}))

	// Setup locations
	loc1 := uuid.New()
	loc2 := uuid.New()
	loc3 := uuid.New()
	locOther := uuid.New()
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: loc1, TenantID: tenantID, WarehouseID: &whID, Code: "LOC-01", Name: "Loc 1", Type: domain.LocationTypeInternal}))
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: loc2, TenantID: tenantID, WarehouseID: &whID, Code: "LOC-02", Name: "Loc 2", Type: domain.LocationTypeInternal}))
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: loc3, TenantID: tenantID, WarehouseID: &whID, Code: "LOC-03", Name: "Loc 3", Type: domain.LocationTypeInternal}))
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: locOther, TenantID: tenantID, WarehouseID: &otherWhID, Code: "LOC-OTHER", Name: "Loc Other", Type: domain.LocationTypeInternal}))

	// Setup products
	prod1 := uuid.New()
	prod2 := uuid.New()
	prod3 := uuid.New()

	// Initial stock levels:
	// loc1, prod1: 10
	// loc2, prod2: 20
	// loc3, prod3: 8
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, loc1, prod1)] = decimal.NewFromInt(10)
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, loc2, prod2)] = decimal.NewFromInt(20)
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, loc3, prod3)] = decimal.NewFromInt(8)

	t.Run("Create StockOpname initializes DRAFT status", func(t *testing.T) {
		op, err := usecase.CreateStockOpname(ctx, tenantID, staffID, "warehouse", uc.CreateStockOpnameRequest{
			WarehouseID: whID,
			Notes:       nil,
		})
		require.NoError(t, err)
		assert.Equal(t, domain.StockOpnameStatusDraft, op.Status)
		assert.NotEmpty(t, op.OpnameNumber)
		assert.Equal(t, whID, op.WarehouseID)
		assert.Equal(t, staffID, op.ConductedBy)
		assert.Nil(t, op.ApprovedBy)
	})

	t.Run("AddOpnameItem computes discrepancy = physical - system", func(t *testing.T) {
		op, err := usecase.CreateStockOpname(ctx, tenantID, staffID, "warehouse", uc.CreateStockOpnameRequest{
			WarehouseID: whID,
		})
		require.NoError(t, err)

		// Item 1: System=10, Physical=15 => Discrepancy = +5 (surplus)
		item1, err := usecase.AddOpnameItem(ctx, tenantID, staffID, "warehouse", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod1,
			LocationID:  loc1,
			PhysicalQty: decimal.NewFromInt(15),
			Notes:       ptr("Surplus items found"),
		})
		require.NoError(t, err)
		assert.True(t, item1.SystemQty.Equal(decimal.NewFromInt(10)))
		assert.True(t, item1.PhysicalQty.Equal(decimal.NewFromInt(15)))
		assert.True(t, item1.DiscrepancyQty.Equal(decimal.NewFromInt(5)))

		// Item 2: System=20, Physical=14 => Discrepancy = -6 (deficit)
		item2, err := usecase.AddOpnameItem(ctx, tenantID, staffID, "warehouse", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod2,
			LocationID:  loc2,
			PhysicalQty: decimal.NewFromInt(14),
			Notes:       ptr("Damaged goods missing"),
		})
		require.NoError(t, err)
		assert.True(t, item2.SystemQty.Equal(decimal.NewFromInt(20)))
		assert.True(t, item2.PhysicalQty.Equal(decimal.NewFromInt(14)))
		assert.True(t, item2.DiscrepancyQty.Equal(decimal.NewFromInt(-6)))

		// Item 3: System=8, Physical=8 => Discrepancy = 0 (exact match)
		item3, err := usecase.AddOpnameItem(ctx, tenantID, staffID, "warehouse", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod3,
			LocationID:  loc3,
			PhysicalQty: decimal.NewFromInt(8),
		})
		require.NoError(t, err)
		assert.True(t, item3.SystemQty.Equal(decimal.NewFromInt(8)))
		assert.True(t, item3.PhysicalQty.Equal(decimal.NewFromInt(8)))
		assert.True(t, item3.DiscrepancyQty.Equal(decimal.Zero))

		// Check GetStockOpname returns all 3 items
		gotOp, items, err := usecase.GetStockOpname(ctx, tenantID, staffID, "warehouse", op.ID)
		require.NoError(t, err)
		assert.Equal(t, op.ID, gotOp.ID)
		assert.Len(t, items, 3)
	})

	t.Run("CompleteStockOpname posts movements to/from @LOSS and adjusts ledger", func(t *testing.T) {
		op, err := usecase.CreateStockOpname(ctx, tenantID, staffID, "warehouse", uc.CreateStockOpnameRequest{
			WarehouseID: whID,
		})
		require.NoError(t, err)

		// Item 1: Surplus +5 (10 -> 15)
		_, err = usecase.AddOpnameItem(ctx, tenantID, staffID, "warehouse", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod1,
			LocationID:  loc1,
			PhysicalQty: decimal.NewFromInt(15),
			Notes:       ptr("Surplus found"),
		})
		require.NoError(t, err)

		// Item 2: Deficit -6 (20 -> 14)
		_, err = usecase.AddOpnameItem(ctx, tenantID, staffID, "warehouse", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod2,
			LocationID:  loc2,
			PhysicalQty: decimal.NewFromInt(14),
			Notes:       ptr("Deficit found"),
		})
		require.NoError(t, err)

		// Complete the opname by admin
		completedOp, err := usecase.CompleteStockOpname(ctx, tenantID, adminID, "admin", op.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StockOpnameStatusCompleted, completedOp.Status)
		require.NotNil(t, completedOp.ApprovedBy)
		assert.Equal(t, adminID, *completedOp.ApprovedBy)

		// Verify stock movements created
		lossLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeLoss)
		require.NoError(t, err)

		var surplusMov, deficitMov *domain.StockMovement
		for i := range repo.stockMovements {
			mov := &repo.stockMovements[i]
			if mov.ReferenceType == domain.StockRefOpname && mov.ReferenceID == op.ID {
				if mov.SourceLocationID == lossLoc.ID {
					surplusMov = mov
				} else if mov.DestLocationID == lossLoc.ID {
					deficitMov = mov
				}
			}
		}

		// Surplus check: @LOSS -> loc1, qty 5
		require.NotNil(t, surplusMov, "Surplus movement must exist")
		assert.Equal(t, lossLoc.ID, surplusMov.SourceLocationID)
		assert.Equal(t, loc1, surplusMov.DestLocationID)
		assert.Equal(t, prod1, surplusMov.ProductID)
		assert.True(t, surplusMov.Quantity.Equal(decimal.NewFromInt(5)))

		// Deficit check: loc2 -> @LOSS, qty 6
		require.NotNil(t, deficitMov, "Deficit movement must exist")
		assert.Equal(t, loc2, deficitMov.SourceLocationID)
		assert.Equal(t, lossLoc.ID, deficitMov.DestLocationID)
		assert.Equal(t, prod2, deficitMov.ProductID)
		assert.True(t, deficitMov.Quantity.Equal(decimal.NewFromInt(6)))

		// Ledger stock levels should now equal physical counts:
		// loc1: 10 + 5 = 15
		// loc2: 20 - 6 = 14
		stock1, err := repo.GetStockByLocation(ctx, tenantID, loc1, prod1)
		require.NoError(t, err)
		assert.True(t, stock1.Equal(decimal.NewFromInt(15)), "loc1 stock must be adjusted to physical count 15")

		stock2, err := repo.GetStockByLocation(ctx, tenantID, loc2, prod2)
		require.NoError(t, err)
		assert.True(t, stock2.Equal(decimal.NewFromInt(14)), "loc2 stock must be adjusted to physical count 14")
	})

	t.Run("Security: Reject adding item to completed opname", func(t *testing.T) {
		op, err := usecase.CreateStockOpname(ctx, tenantID, staffID, "warehouse", uc.CreateStockOpnameRequest{WarehouseID: whID})
		require.NoError(t, err)

		_, err = usecase.CompleteStockOpname(ctx, tenantID, adminID, "admin", op.ID)
		require.NoError(t, err)

		_, err = usecase.AddOpnameItem(ctx, tenantID, staffID, "warehouse", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod1,
			LocationID:  loc1,
			PhysicalQty: decimal.NewFromInt(12),
		})
		assert.ErrorIs(t, err, domain.ErrInvalidOpnameStatus)
	})

	t.Run("Security: Reject completing an already completed opname", func(t *testing.T) {
		op, err := usecase.CreateStockOpname(ctx, tenantID, staffID, "warehouse", uc.CreateStockOpnameRequest{WarehouseID: whID})
		require.NoError(t, err)

		_, err = usecase.CompleteStockOpname(ctx, tenantID, adminID, "admin", op.ID)
		require.NoError(t, err)

		_, err = usecase.CompleteStockOpname(ctx, tenantID, adminID, "admin", op.ID)
		assert.ErrorIs(t, err, domain.ErrInvalidOpnameStatus)
	})

	t.Run("Security: Reject adding location belonging to another warehouse", func(t *testing.T) {
		op, err := usecase.CreateStockOpname(ctx, tenantID, staffID, "warehouse", uc.CreateStockOpnameRequest{WarehouseID: whID})
		require.NoError(t, err)

		_, err = usecase.AddOpnameItem(ctx, tenantID, staffID, "warehouse", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod1,
			LocationID:  locOther, // belongs to otherWhID!
			PhysicalQty: decimal.NewFromInt(10),
		})
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)
	})

	t.Run("Security: Reject negative physical quantity", func(t *testing.T) {
		op, err := usecase.CreateStockOpname(ctx, tenantID, staffID, "warehouse", uc.CreateStockOpnameRequest{WarehouseID: whID})
		require.NoError(t, err)

		_, err = usecase.AddOpnameItem(ctx, tenantID, staffID, "warehouse", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod1,
			LocationID:  loc1,
			PhysicalQty: decimal.NewFromInt(-5),
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Security: Auditor role cannot create, add items, or complete opnames", func(t *testing.T) {
		_, err := usecase.CreateStockOpname(ctx, tenantID, auditorID, "auditor", uc.CreateStockOpnameRequest{WarehouseID: whID})
		assert.ErrorIs(t, err, domain.ErrForbidden)

		op, err := usecase.CreateStockOpname(ctx, tenantID, staffID, "warehouse", uc.CreateStockOpnameRequest{WarehouseID: whID})
		require.NoError(t, err)

		_, err = usecase.AddOpnameItem(ctx, tenantID, auditorID, "auditor", op.ID, uc.AddOpnameItemRequest{
			ProductID:   prod1,
			LocationID:  loc1,
			PhysicalQty: decimal.NewFromInt(10),
		})
		assert.ErrorIs(t, err, domain.ErrForbidden)

		_, err = usecase.CompleteStockOpname(ctx, tenantID, auditorID, "auditor", op.ID)
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestStockScrapQuarantineAndAtomicDeduction(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	otherWhID := uuid.New()
	staffID := uuid.New()
	auditorID := uuid.New()

	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{ID: whID, TenantID: tenantID, Name: "Main WH", IsActive: true}))
	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{ID: otherWhID, TenantID: tenantID, Name: "Other WH", IsActive: true}))
	require.NoError(t, repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{UserID: staffID, WarehouseID: whID, TenantID: tenantID}))

	srcLoc := uuid.New()
	scrapLoc := uuid.New()
	otherLoc := uuid.New()
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: srcLoc, TenantID: tenantID, WarehouseID: &whID, Code: "SRC-LOC", Name: "Source", Type: domain.LocationTypeInternal}))
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: scrapLoc, TenantID: tenantID, WarehouseID: &whID, Code: "SCRAP-BAY", Name: "Scrap Bay", Type: domain.LocationTypeInternal}))
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{ID: otherLoc, TenantID: tenantID, WarehouseID: &otherWhID, Code: "OTHER-LOC", Name: "Other Loc", Type: domain.LocationTypeInternal}))

	prodID := uuid.New()
	// Set initial source stock = 15
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, srcLoc, prodID)] = decimal.NewFromInt(15)

	t.Run("CreateStockScrap with physical scrap location deducts stock and logs scrap", func(t *testing.T) {
		scrap, err := usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", uc.CreateStockScrapRequest{
			WarehouseID:      whID,
			ProductID:        prodID,
			SourceLocationID: srcLoc,
			ScrapLocationID:  &scrapLoc,
			Quantity:         decimal.NewFromInt(5),
			Reason:           "Damaged packaging during handling",
		})
		require.NoError(t, err)
		assert.Equal(t, whID, scrap.WarehouseID)
		assert.Equal(t, prodID, scrap.ProductID)
		assert.Equal(t, srcLoc, scrap.SourceLocationID)
		assert.Equal(t, scrapLoc, scrap.ScrapLocationID)
		assert.True(t, scrap.Quantity.Equal(decimal.NewFromInt(5)))
		assert.Equal(t, staffID, scrap.ReportedBy)

		// Source stock should be 15 - 5 = 10
		srcStock, err := repo.GetStockByLocation(ctx, tenantID, srcLoc, prodID)
		require.NoError(t, err)
		assert.True(t, srcStock.Equal(decimal.NewFromInt(10)))

		// Scrap location stock should be 5
		scrapStock, err := repo.GetStockByLocation(ctx, tenantID, scrapLoc, prodID)
		require.NoError(t, err)
		assert.True(t, scrapStock.Equal(decimal.NewFromInt(5)))
	})

	t.Run("CreateStockScrap without scrap_location_id defaults to virtual @SCRAP", func(t *testing.T) {
		scrap, err := usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", uc.CreateStockScrapRequest{
			WarehouseID:      whID,
			ProductID:        prodID,
			SourceLocationID: srcLoc,
			ScrapLocationID:  nil,
			Quantity:         decimal.NewFromInt(4),
			Reason:           "Expired chemicals",
		})
		require.NoError(t, err)

		virtualScrapLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
		require.NoError(t, err)
		assert.Equal(t, virtualScrapLoc.ID, scrap.ScrapLocationID)

		// Source stock: 10 - 4 = 6
		srcStock, err := repo.GetStockByLocation(ctx, tenantID, srcLoc, prodID)
		require.NoError(t, err)
		assert.True(t, srcStock.Equal(decimal.NewFromInt(6)))
	})

	t.Run("Security: Reject scrap when quantity exceeds available stock", func(t *testing.T) {
		// Currently only 6 remain
		_, err := usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", uc.CreateStockScrapRequest{
			WarehouseID:      whID,
			ProductID:        prodID,
			SourceLocationID: srcLoc,
			Quantity:         decimal.NewFromInt(10),
			Reason:           "Trying to scrap more than available",
		})
		assert.ErrorIs(t, err, domain.ErrInsufficientStock)
	})

	t.Run("Security: Reject identical source and scrap location", func(t *testing.T) {
		_, err := usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", uc.CreateStockScrapRequest{
			WarehouseID:      whID,
			ProductID:        prodID,
			SourceLocationID: srcLoc,
			ScrapLocationID:  &srcLoc,
			Quantity:         decimal.NewFromInt(1),
			Reason:           "Same source and scrap",
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Security: Reject zero or negative scrap quantity", func(t *testing.T) {
		_, err := usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", uc.CreateStockScrapRequest{
			WarehouseID:      whID,
			ProductID:        prodID,
			SourceLocationID: srcLoc,
			Quantity:         decimal.Zero,
			Reason:           "Zero qty",
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Security: Reject empty reason", func(t *testing.T) {
		_, err := usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", uc.CreateStockScrapRequest{
			WarehouseID:      whID,
			ProductID:        prodID,
			SourceLocationID: srcLoc,
			Quantity:         decimal.NewFromInt(1),
			Reason:           "   ",
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("Security: Reject source location from another warehouse", func(t *testing.T) {
		_, err := usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", uc.CreateStockScrapRequest{
			WarehouseID:      whID,
			ProductID:        prodID,
			SourceLocationID: otherLoc,
			Quantity:         decimal.NewFromInt(1),
			Reason:           "Spoofed location",
		})
		assert.ErrorIs(t, err, domain.ErrUnauthorizedWarehouse)
	})

	t.Run("Security: Auditor cannot create scrap", func(t *testing.T) {
		_, err := usecase.CreateStockScrap(ctx, tenantID, auditorID, "auditor", uc.CreateStockScrapRequest{
			WarehouseID:      whID,
			ProductID:        prodID,
			SourceLocationID: srcLoc,
			Quantity:         decimal.NewFromInt(1),
			Reason:           "Auditor write attempt",
		})
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestMarketplaceSalesImportAndSKUMapping(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	staffID := uuid.New()
	auditorID := uuid.New()
	adminID := uuid.New()

	// Setup warehouse & user
	require.NoError(t, repo.CreateWarehouse(ctx, &domain.Warehouse{
		ID:       whID,
		TenantID: tenantID,
		Code:     "WH-MKT",
		Name:     "Marketplace Fulfillment WH",
		IsActive: true,
	}))
	require.NoError(t, repo.AssignUserWarehouse(ctx, &domain.UserWarehouse{
		TenantID:    tenantID,
		UserID:      staffID,
		WarehouseID: whID,
	}))

	// Setup warehouse locations: 1 internal location & 1 customer virtual location
	srcLocID := uuid.New()
	require.NoError(t, repo.CreateLocation(ctx, &domain.WarehouseLocation{
		ID:          srcLocID,
		TenantID:    tenantID,
		WarehouseID: &whID,
		Code:        "LOC-MKT-01",
		Name:        "Marketplace Shelf 1",
		Type:        domain.LocationTypeInternal,
	}))
	custLoc, err := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
	require.NoError(t, err)
	assert.Equal(t, "@CUSTOMER", custLoc.Code)

	// Setup products
	prod1ID := uuid.New()
	prod1SKU := "MAS000123"
	repo.products[fmt.Sprintf("%s:%s", tenantID, prod1SKU)] = domain.Product{
		ID:       prod1ID,
		TenantID: tenantID,
		SKU:      prod1SKU,
		Name:     "Kopi Susu Internal",
	}

	prod2ID := uuid.New()
	prod2SKU := "DIRECT-SKU-001"
	repo.products[fmt.Sprintf("%s:%s", tenantID, prod2SKU)] = domain.Product{
		ID:       prod2ID,
		TenantID: tenantID,
		SKU:      prod2SKU,
		Name:     "Direct SKU Product",
	}

	// Seed stock: 100 pcs for prod1, 5 pcs for prod2
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, srcLocID, prod1ID)] = decimal.NewFromInt(100)
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, srcLocID, prod2ID)] = decimal.NewFromInt(5)

	t.Run("AC-1: Idempotent import ignores duplicate orders", func(t *testing.T) {
		req := uc.ImportMarketplaceOrdersRequest{
			WarehouseID: whID,
			Channel:     domain.MarketplaceChannelShopee,
			FileName:    "shopee_orders.csv",
			Orders: []uc.ImportOrderRequest{
				{
					ExternalOrderID: "240909SHP001",
					TotalAmount:     decimal.NewFromInt(50000),
					Items: []uc.ImportOrderItemRequest{
						{
							ExternalSKU: prod2SKU,
							ItemName:    "Direct Product",
							Quantity:    decimal.NewFromInt(1),
							UnitPrice:   decimal.NewFromInt(50000),
							Subtotal:    decimal.NewFromInt(50000),
						},
					},
				},
			},
		}

		// First upload
		resp1, err := usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", req)
		require.NoError(t, err)
		assert.Equal(t, 1, resp1.Batch.TotalOrders)
		assert.Equal(t, 1, resp1.Batch.ProcessedOrders)
		assert.Equal(t, 0, resp1.Batch.FailedOrders)
		assert.Equal(t, domain.MarketplaceOrderStatusCompleted, resp1.Orders[0].Status)

		// Second upload of the exact same order in the same channel
		resp2, err := usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", req)
		require.NoError(t, err)
		assert.Equal(t, 1, resp2.Batch.TotalOrders)
		assert.Equal(t, 0, resp2.Batch.ProcessedOrders)
		assert.Equal(t, 1, resp2.Batch.FailedOrders) // Duplicate skipped and recorded as failed order
		assert.Equal(t, domain.MarketplaceBatchStatusFailed, resp2.Batch.Status)

		// Check stock only deducted ONCE (5 - 1 = 4)
		currentStock, err := repo.GetStockByLocation(ctx, tenantID, srcLocID, prod2ID)
		require.NoError(t, err)
		assert.True(t, currentStock.Equal(decimal.NewFromInt(4)))
	})

	var ac2OrderID uuid.UUID

	t.Run("AC-2: Multiplier resolution and stock deduction (2 Dus x 24 = 48 Pcs)", func(t *testing.T) {
		// Map KOPISUSU-DUS to prod1ID with multiplier 24
		mult24 := decimal.NewFromInt(24)
		_, err := usecase.CreateSKUMapping(ctx, tenantID, uc.CreateSKUMappingRequest{
			ProductID:   prod1ID,
			MappingType: domain.SKUMappingTypeMarketplace,
			ChannelName: "TOKOPEDIA",
			ExternalSKU: "KOPISUSU-DUS",
			Multiplier:  &mult24,
		})
		require.NoError(t, err)

		req := uc.ImportMarketplaceOrdersRequest{
			WarehouseID: whID,
			Channel:     domain.MarketplaceChannelTokopedia,
			FileName:    "tokopedia_export.csv",
			Orders: []uc.ImportOrderRequest{
				{
					ExternalOrderID: "TKP-2026-001",
					TotalAmount:     decimal.NewFromInt(240000),
					Items: []uc.ImportOrderItemRequest{
						{
							ExternalSKU: "KOPISUSU-DUS",
							ItemName:    "Kopi Susu Dus (24 pcs)",
							Quantity:    decimal.NewFromInt(2), // 2 Dus
							UnitPrice:   decimal.NewFromInt(120000),
							Subtotal:    decimal.NewFromInt(240000),
						},
					},
				},
			},
		}

		resp, err := usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", req)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Batch.ProcessedOrders)
		assert.Equal(t, domain.MarketplaceOrderStatusCompleted, resp.Orders[0].Status)
		ac2OrderID = resp.Orders[0].ID

		// Initial stock was 100. Deduct 2 * 24 = 48. Expected remaining = 52.
		stock, err := repo.GetStockByLocation(ctx, tenantID, srcLocID, prod1ID)
		require.NoError(t, err)
		assert.True(t, stock.Equal(decimal.NewFromInt(52)), "Stock should be 52 after deducting 48, got %s", stock)
	})

	t.Run("AC-3: Unmapped SKU handling and in-place resolution", func(t *testing.T) {
		req := uc.ImportMarketplaceOrdersRequest{
			WarehouseID: whID,
			Channel:     domain.MarketplaceChannelTikTok,
			FileName:    "tiktok_orders.csv",
			Orders: []uc.ImportOrderRequest{
				{
					ExternalOrderID: "TT-ORDER-888",
					TotalAmount:     decimal.NewFromInt(75000),
					Items: []uc.ImportOrderItemRequest{
						{
							ExternalSKU: "NEW-TIKTOK-VIRAL-SKU",
							ItemName:    "TikTok Viral Item",
							Quantity:    decimal.NewFromInt(2),
							UnitPrice:   decimal.NewFromInt(37500),
							Subtotal:    decimal.NewFromInt(75000),
						},
					},
				},
			},
		}

		resp, err := usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", req)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Batch.TotalOrders)
		assert.Equal(t, 1, resp.Batch.ProcessedOrders)
		assert.Equal(t, 1, resp.Batch.UnmappedSKUs)
		assert.Equal(t, domain.MarketplaceOrderStatusUnmappedSKU, resp.Orders[0].Status)
		assert.False(t, resp.Orders[0].Items[0].IsMapped)

		// Stock for prod1 should still be 52 (not yet deducted)
		stockBefore, err := repo.GetStockByLocation(ctx, tenantID, srcLocID, prod1ID)
		require.NoError(t, err)
		assert.True(t, stockBefore.Equal(decimal.NewFromInt(52)))

		// Now staff resolves the SKU mapping linking NEW-TIKTOK-VIRAL-SKU -> prod1ID with multiplier 1
		mult1 := decimal.NewFromInt(1)
		mapping, err := usecase.ResolveSKUMapping(ctx, tenantID, staffID, "warehouse", uc.CreateSKUMappingRequest{
			ProductID:   prod1ID,
			MappingType: domain.SKUMappingTypeMarketplace,
			ChannelName: "TIKTOK",
			ExternalSKU: "NEW-TIKTOK-VIRAL-SKU",
			Multiplier:  &mult1,
		})
		require.NoError(t, err)
		assert.NotNil(t, mapping)

		// Verify order was reprocessed and marked COMPLETED
		reprocessedOrder, err := repo.GetMarketplaceOrderByID(ctx, tenantID, resp.Orders[0].ID)
		require.NoError(t, err)
		assert.Equal(t, domain.MarketplaceOrderStatusCompleted, reprocessedOrder.Status)

		// Stock should now be deducted: 52 - 2 = 50
		stockAfter, err := repo.GetStockByLocation(ctx, tenantID, srcLocID, prod1ID)
		require.NoError(t, err)
		assert.True(t, stockAfter.Equal(decimal.NewFromInt(50)), "Stock should be 50 after resolving unmapped order, got %s", stockAfter)
	})

	t.Run("AC-4: WMS Stock deduction from warehouse location to @CUSTOMER with reference StockRefMarketplace", func(t *testing.T) {
		// Verify that completed marketplace orders generate StockMovement with:
		// - Source: warehouse internal location (srcLocID)
		// - Dest: @CUSTOMER virtual location (custLoc.ID)
		// - ReferenceType: StockRefMarketplace
		// - ReferenceID: order.ID
		var orderMov *domain.StockMovement
		for i := range repo.stockMovements {
			if repo.stockMovements[i].ReferenceID == ac2OrderID {
				orderMov = &repo.stockMovements[i]
				break
			}
		}
		require.NotNil(t, orderMov, "Stock movement for marketplace order must be recorded")
		assert.Equal(t, srcLocID, orderMov.SourceLocationID, "Stock deducted from primary warehouse location")
		assert.Equal(t, custLoc.ID, orderMov.DestLocationID, "Stock moved to @CUSTOMER system location")
		assert.Equal(t, "@CUSTOMER", custLoc.Code)
		assert.Equal(t, domain.StockRefMarketplace, orderMov.ReferenceType, "Reference type must be StockRefMarketplace")
		assert.Equal(t, domain.StockMovementStatusDone, orderMov.Status)
		assert.True(t, orderMov.Quantity.Equal(decimal.NewFromInt(48)), "Quantity moved must be 48 pcs (2 Dus x 24)")
	})

	t.Run("Guardrail 3: Insufficient stock flags order as STOCK_INSUFFICIENT without failing batch", func(t *testing.T) {
		// prod2 currently has 4 in stock. Order requests 10.
		req := uc.ImportMarketplaceOrdersRequest{
			WarehouseID: whID,
			Channel:     domain.MarketplaceChannelLazada,
			FileName:    "lazada_export.csv",
			Orders: []uc.ImportOrderRequest{
				{
					ExternalOrderID: "LAZ-ORDER-999",
					TotalAmount:     decimal.NewFromInt(500000),
					Items: []uc.ImportOrderItemRequest{
						{
							ExternalSKU: prod2SKU,
							ItemName:    "Direct Product",
							Quantity:    decimal.NewFromInt(10), // Needs 10, only 4 available
							UnitPrice:   decimal.NewFromInt(50000),
							Subtotal:    decimal.NewFromInt(500000),
						},
					},
				},
			},
		}

		resp, err := usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", req)
		require.NoError(t, err)
		assert.Equal(t, domain.MarketplaceBatchStatusCompleted, resp.Batch.Status)
		assert.Equal(t, 1, resp.Batch.ProcessedOrders)
		assert.Equal(t, domain.MarketplaceOrderStatusStockInsufficient, resp.Orders[0].Status)

		// Stock remains 4 (not deducted / no negative stock)
		stock, err := repo.GetStockByLocation(ctx, tenantID, srcLocID, prod2ID)
		require.NoError(t, err)
		assert.True(t, stock.Equal(decimal.NewFromInt(4)))
	})

	t.Run("Security: Auditor cannot import orders or resolve mappings", func(t *testing.T) {
		_, err := usecase.ImportMarketplaceOrders(ctx, tenantID, auditorID, "auditor", uc.ImportMarketplaceOrdersRequest{
			WarehouseID: whID,
			Channel:     domain.MarketplaceChannelShopee,
			Orders:      []uc.ImportOrderRequest{},
		})
		assert.ErrorIs(t, err, domain.ErrForbidden)

		_, err = usecase.ResolveSKUMapping(ctx, tenantID, auditorID, "auditor", uc.CreateSKUMappingRequest{
			ProductID:   prod1ID,
			ChannelName: "SHOPEE",
			ExternalSKU: "TEST",
		})
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("Multi-Tenancy: Cross-tenant isolation", func(t *testing.T) {
		tenantB := uuid.New()
		userB := uuid.New()

		// Tenant B cannot import to Tenant A's warehouse
		_, err := usecase.ImportMarketplaceOrders(ctx, tenantB, userB, "admin", uc.ImportMarketplaceOrdersRequest{
			WarehouseID: whID, // Belongs to Tenant A
			Channel:     domain.MarketplaceChannelShopee,
		})
		assert.ErrorIs(t, err, domain.ErrWarehouseNotFound)

		// Tenant B listing batches/orders sees empty results
		batchesB, err := usecase.ListMarketplaceBatches(ctx, tenantB, userB, "admin", nil)
		require.NoError(t, err)
		assert.Empty(t, batchesB)

		ordersB, err := usecase.ListMarketplaceOrders(ctx, tenantB, userB, "admin", nil, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, ordersB)

		// Tenant B cannot see Tenant A's SKU mappings
		mappingsB, err := usecase.ListSKUMappings(ctx, tenantB, "TOKOPEDIA")
		require.NoError(t, err)
		assert.Empty(t, mappingsB)

		// Tenant B cannot access Tenant A's orders by ID
		_, err = usecase.GetMarketplaceOrder(ctx, tenantB, userB, "admin", ac2OrderID)
		assert.ErrorIs(t, err, domain.ErrMarketplaceOrderNotFound)

		// Tenant B creating SKU mappings cannot view or affect Tenant A's mappings
		mult10 := decimal.NewFromInt(10)
		_, err = usecase.CreateSKUMapping(ctx, tenantB, uc.CreateSKUMappingRequest{
			ProductID:   prod1ID,
			MappingType: domain.SKUMappingTypeMarketplace,
			ChannelName: "TOKOPEDIA",
			ExternalSKU: "KOPISUSU-DUS",
			Multiplier:  &mult10,
		})
		require.NoError(t, err)

		mappingA, err := repo.GetSKUMapping(ctx, tenantID, "TOKOPEDIA", "KOPISUSU-DUS")
		require.NoError(t, err)
		assert.True(t, mappingA.Multiplier.Equal(decimal.NewFromInt(24)), "Tenant A multiplier remains unaffected by Tenant B")
	})

	t.Run("M1: qty bomb rejected before any write, valid qty passes", func(t *testing.T) {
		before, err := repo.GetStockByLocation(ctx, tenantID, srcLocID, prod2ID)
		require.NoError(t, err)
		batchesBefore, err := usecase.ListMarketplaceBatches(ctx, tenantID, adminID, "admin", nil)
		require.NoError(t, err)

		mk := func(id string, qty int64) uc.ImportMarketplaceOrdersRequest {
			return uc.ImportMarketplaceOrdersRequest{
				WarehouseID: whID, Channel: domain.MarketplaceChannelShopee,
				Orders: []uc.ImportOrderRequest{{ExternalOrderID: id, Items: []uc.ImportOrderItemRequest{
					{ExternalSKU: "DIRECT-SKU-001", ItemName: "x", Quantity: decimal.NewFromInt(qty), UnitPrice: decimal.NewFromInt(1)},
				}}},
			}
		}
		for _, q := range []int64{999999, domain.MaxMarketplaceItemQty + 1, 0, -3} {
			_, err := usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", mk(fmt.Sprintf("BOMB-%d", q), q))
			assert.ErrorIs(t, err, domain.ErrInvalidInput, "qty %d must be rejected", q)
		}
		batchesAfter, err := usecase.ListMarketplaceBatches(ctx, tenantID, adminID, "admin", nil)
		require.NoError(t, err)
		assert.Len(t, batchesAfter, len(batchesBefore), "rejected import must not create a batch")
		after, err := repo.GetStockByLocation(ctx, tenantID, srcLocID, prod2ID)
		require.NoError(t, err)
		assert.True(t, after.Equal(before), "rejected import must not move stock")

		_, err = usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", mk("OK-QTY-2", 2))
		require.NoError(t, err)

		// Batch total cap: 11 lines x 1000 = 11000 > 10000.
		big := uc.ImportMarketplaceOrdersRequest{WarehouseID: whID, Channel: domain.MarketplaceChannelShopee}
		for i := 0; i < 11; i++ {
			big.Orders = append(big.Orders, uc.ImportOrderRequest{ExternalOrderID: fmt.Sprintf("BIG-%d", i),
				Items: []uc.ImportOrderItemRequest{{ExternalSKU: "DIRECT-SKU-001", ItemName: "x", Quantity: decimal.NewFromInt(domain.MaxMarketplaceItemQty)}}})
		}
		_, err = usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", big)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("M4: resolve deducts an unmapped order only once", func(t *testing.T) {
		imp := uc.ImportMarketplaceOrdersRequest{WarehouseID: whID, Channel: domain.MarketplaceChannelLazada,
			Orders: []uc.ImportOrderRequest{{ExternalOrderID: "LZ-RACE-1", Items: []uc.ImportOrderItemRequest{
				{ExternalSKU: "LZ-RACE-SKU", ItemName: "race", Quantity: decimal.NewFromInt(3)},
			}}}}
		resp, err := usecase.ImportMarketplaceOrders(ctx, tenantID, staffID, "warehouse", imp)
		require.NoError(t, err)
		require.Equal(t, domain.MarketplaceOrderStatusUnmappedSKU, resp.Orders[0].Status)

		movesBefore := 0
		for _, mv := range repo.stockMovements {
			if mv.ReferenceID == resp.Orders[0].ID {
				movesBefore++
			}
		}
		require.Zero(t, movesBefore)

		one := decimal.NewFromInt(1)
		mreq := uc.CreateSKUMappingRequest{ProductID: prod1ID, MappingType: domain.SKUMappingTypeMarketplace, ChannelName: "LAZADA", ExternalSKU: "LZ-RACE-SKU", Multiplier: &one}
		// Both requests read the pending list before either deducts.
		stale, err := repo.GetPendingUnmappedOrdersBySKU(ctx, tenantID, domain.MarketplaceChannelLazada, "LZ-RACE-SKU")
		require.NoError(t, err)
		require.Len(t, stale, 1)
		for i := range stale[0].Items {
			stale[0].Items[i].ProductID = &prod1ID
			stale[0].Items[i].IsMapped = true
		}
		_, err = usecase.ResolveSKUMapping(ctx, tenantID, adminID, "admin", mreq)
		require.NoError(t, err)
		repo.stalePendingOrders = stale // second request still holds the old snapshot
		_, err = usecase.ResolveSKUMapping(ctx, tenantID, adminID, "admin", mreq)
		repo.stalePendingOrders = nil
		require.NoError(t, err)

		moves := 0
		for _, mv := range repo.stockMovements {
			if mv.ReferenceID == resp.Orders[0].ID {
				moves++
			}
		}
		assert.Equal(t, 1, moves, "order must be deducted exactly once")

		claimed, err := repo.ClaimMarketplaceOrder(ctx, tenantID, resp.Orders[0].ID, domain.MarketplaceOrderStatusUnmappedSKU, domain.MarketplaceOrderStatusProcessing)
		require.NoError(t, err)
		assert.False(t, claimed, "claim from a stale status must fail")
	})

	t.Run("M2: multiplier ceiling and role gate", func(t *testing.T) {
		m1000 := decimal.NewFromInt(1000)
		m6 := decimal.NewFromInt(6)
		m0 := decimal.Zero
		req := func(sku string, m *decimal.Decimal) uc.CreateSKUMappingRequest {
			return uc.CreateSKUMappingRequest{ProductID: prod1ID, MappingType: domain.SKUMappingTypeMarketplace, ChannelName: "SHOPEE", ExternalSKU: sku, Multiplier: m}
		}
		_, err := usecase.ResolveSKUMapping(ctx, tenantID, adminID, "admin", req("DRAIN-1000", &m1000))
		assert.ErrorIs(t, err, domain.ErrInvalidInput, "multiplier above ceiling rejected even for admin")
		_, err = usecase.ResolveSKUMapping(ctx, tenantID, adminID, "admin", req("ZERO-MULT", &m0))
		assert.ErrorIs(t, err, domain.ErrInvalidInput, "multiplier 0 rejected")
		_, err = usecase.ResolveSKUMapping(ctx, tenantID, staffID, "warehouse", req("PACK-6", &m6))
		assert.ErrorIs(t, err, domain.ErrForbidden, "warehouse cannot set multiplier > 1")
		got, err := usecase.ResolveSKUMapping(ctx, tenantID, adminID, "admin", req("PACK-6", &m6))
		require.NoError(t, err)
		assert.True(t, got.Multiplier.Equal(m6))
	})
}

// Regression for prod transfer TR-20261004-957: destination warehouse had zero
// racks, so receive returned 404 "location not found" (rendered as [object Object]).
func TestTransferLocationResilience(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()
	tenantID, adminID, approverID := uuid.New(), uuid.New(), uuid.New()
	whSource, whTarget, productID, locSourceID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	repo.warehouses[whSource] = domain.Warehouse{ID: whSource, TenantID: tenantID, Code: "WH0001", Name: "Gudang Cipondoh"}
	repo.warehouses[whTarget] = domain.Warehouse{ID: whTarget, TenantID: tenantID, Code: "WH-JKT", Name: "Gudang Utama Jakarta"}
	repo.locations[locSourceID] = domain.WarehouseLocation{ID: locSourceID, TenantID: tenantID, WarehouseID: &whSource, Code: "RAK-01", Name: "Rak 01", Type: domain.LocationTypeInternal}
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, locSourceID, productID)] = decimal.NewFromInt(50)

	t.Run("create rejects item without source location", func(t *testing.T) {
		_, err := usecase.CreateTransfer(ctx, tenantID, adminID, "admin", uc.CreateTransferRequest{
			FromWarehouseID: whSource, ToWarehouseID: whTarget, TransferNumber: "TR-NO-SRC",
			Items: []uc.CreateTransferItemRequest{{ProductID: productID, RequestedQty: decimal.NewFromInt(1)}},
		})
		assert.ErrorIs(t, err, domain.ErrSourceLocationRequired)
	})

	t.Run("dispatch of legacy item without source location returns ErrSourceLocationRequired", func(t *testing.T) {
		tr := &domain.StockTransfer{ID: uuid.New(), TenantID: tenantID, TransferNumber: "TR-TEST-CHECK", FromWarehouseID: whSource, ToWarehouseID: whTarget, Status: domain.TransferStatusApproved, RequestedBy: adminID}
		require.NoError(t, repo.CreateTransfer(ctx, tr, []domain.StockTransferItem{{ID: uuid.New(), TenantID: tenantID, TransferID: tr.ID, ProductID: productID, RequestedQty: decimal.NewFromInt(1)}}))
		_, err := usecase.DispatchTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		assert.ErrorIs(t, err, domain.ErrSourceLocationRequired)
	})

	t.Run("receive into warehouse with zero racks auto-creates DEFAULT location", func(t *testing.T) {
		tr, err := usecase.CreateTransfer(ctx, tenantID, adminID, "admin", uc.CreateTransferRequest{
			FromWarehouseID: whSource, ToWarehouseID: whTarget, TransferNumber: "TR-NO-RACK",
			Items: []uc.CreateTransferItemRequest{{ProductID: productID, RequestedQty: decimal.NewFromInt(4), SourceLocationID: &locSourceID}},
		})
		require.NoError(t, err)
		_, err = usecase.SubmitTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		require.NoError(t, err)
		_, err = usecase.ApproveTransfer(ctx, tenantID, approverID, "admin", tr.ID)
		require.NoError(t, err)
		_, err = usecase.DispatchTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		require.NoError(t, err)

		received, err := usecase.ReceiveTransfer(ctx, tenantID, adminID, "admin", tr.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusReceived, received.Status)

		def, err := repo.GetLocationByCode(ctx, tenantID, &whTarget, "DEFAULT-WH-JKT")
		require.NoError(t, err, "default location must be created")
		assert.Equal(t, whTarget, *def.WarehouseID)
		assert.Equal(t, domain.LocationTypeInternal, def.Type)
		stock, err := repo.GetStockByLocation(ctx, tenantID, def.ID, productID)
		require.NoError(t, err)
		assert.True(t, stock.Equal(decimal.NewFromInt(4)), "received qty lands in default location, got %s", stock)
	})
}
func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }
func ptr[T any](v T) *T { return &v }

func TestWMSFraudControls_F2_Dispatch(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	locID := uuid.New()
	productID := uuid.New()
	creatorID := uuid.New()
	dispatcherID := uuid.New()

	repo.warehouses[whID] = domain.Warehouse{ID: whID, TenantID: tenantID, Code: "WH-F2", Name: "F2 Warehouse", IsActive: true}
	repo.locations[locID] = domain.WarehouseLocation{ID: locID, TenantID: tenantID, WarehouseID: &whID, Code: "LOC-F2", Name: "Rack F2", Type: domain.LocationTypeInternal}
	repo.userWarehouses[fmt.Sprintf("%s:%s", tenantID, creatorID)] = []uuid.UUID{whID}
	repo.userWarehouses[fmt.Sprintf("%s:%s", tenantID, dispatcherID)] = []uuid.UUID{whID}

	locKey := fmt.Sprintf("%s:%s:%s", tenantID, locID, productID)
	repo.stockLevels[locKey] = decimal.NewFromInt(100)

	// 1. Dispatch from DRAFT returns ErrDeliveryOrderNotPacked
	req := uc.CreateDeliveryOrderRequest{
		WarehouseID:    whID,
		DONumber:       "DO-F2-01",
		RecipientName:  ptr("PT Sukses"),
		DriverName:     ptr("Pak Supir"),
		VehiclePlate:   ptr("B 1234 ABC"),
		ExpeditionName: ptr("JNE"),
		Items: []uc.CreateDeliveryOrderItemRequest{
			{ProductID: productID, Quantity: decimal.NewFromInt(5), LocationID: locID},
		},
	}
	do, err := usecase.CreateDeliveryOrder(ctx, tenantID, creatorID, "warehouse", req)
	require.NoError(t, err)
	assert.Equal(t, domain.DeliveryOrderStatusDraft, do.Status)

	_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, dispatcherID, "warehouse", do.ID)
	assert.ErrorIs(t, err, domain.ErrDeliveryOrderNotPacked, "dispatch from DRAFT must return ErrDeliveryOrderNotPacked")

	// 2. Dispatch without driver/plate/expedition returns ErrDeliveryOrderIncompleteShip
	// Move DO to CONFIRMED
	repo.deliveryOrders[do.ID].Status = domain.DeliveryOrderStatusConfirmed
	repo.deliveryOrders[do.ID].DriverName = nil
	_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, dispatcherID, "warehouse", do.ID)
	assert.ErrorIs(t, err, domain.ErrDeliveryOrderIncompleteShip, "dispatch without driver name must return ErrDeliveryOrderIncompleteShip")

	repo.deliveryOrders[do.ID].DriverName = ptr("Pak Supir")
	repo.deliveryOrders[do.ID].VehiclePlate = nil
	_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, dispatcherID, "warehouse", do.ID)
	assert.ErrorIs(t, err, domain.ErrDeliveryOrderIncompleteShip, "dispatch without vehicle plate must return ErrDeliveryOrderIncompleteShip")

	repo.deliveryOrders[do.ID].VehiclePlate = ptr("B 1234 ABC")
	repo.deliveryOrders[do.ID].ExpeditionName = nil
	_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, dispatcherID, "warehouse", do.ID)
	assert.ErrorIs(t, err, domain.ErrDeliveryOrderIncompleteShip, "dispatch without expedition name must return ErrDeliveryOrderIncompleteShip")

	// 3. Dispatch by same user with role warehouse returns ErrSelfApprovalForbidden
	repo.deliveryOrders[do.ID].ExpeditionName = ptr("JNE")
	_, err = usecase.DispatchDeliveryOrder(ctx, tenantID, creatorID, "warehouse", do.ID)
	assert.ErrorIs(t, err, domain.ErrSelfApprovalForbidden, "dispatch by creator with role warehouse must return ErrSelfApprovalForbidden")

	// 4. Dispatch from PACKED with shipping details by different user succeeds
	repo.deliveryOrders[do.ID].Status = domain.DeliveryOrderStatusPacked
	repo.deliveryOrders[do.ID].PackedBy = ptr(uuid.New())
	dispatched, err := usecase.DispatchDeliveryOrder(ctx, tenantID, dispatcherID, "warehouse", do.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.DeliveryOrderStatusShipped, dispatched.Status)
}

func TestWMSFraudControls_F3_Opname(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	locID := uuid.New()
	prodID := uuid.New()
	conductorID := uuid.New()
	adminID := uuid.New()

	repo.warehouses[whID] = domain.Warehouse{ID: whID, TenantID: tenantID, Code: "WH-F3", Name: "F3 Warehouse", IsActive: true}
	repo.locations[locID] = domain.WarehouseLocation{ID: locID, TenantID: tenantID, WarehouseID: &whID, Code: "LOC-F3", Name: "Rack F3", Type: domain.LocationTypeInternal}
	repo.userWarehouses[fmt.Sprintf("%s:%s", tenantID, conductorID)] = []uuid.UUID{whID}
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, locID, prodID)] = decimal.NewFromInt(20)

	// 1. Complete opname by warehouse non-approver transitions to PENDING_APPROVAL (no movements)
	op1, err := usecase.CreateStockOpname(ctx, tenantID, conductorID, "warehouse", uc.CreateStockOpnameRequest{WarehouseID: whID})
	require.NoError(t, err)
	_, err = usecase.AddOpnameItem(ctx, tenantID, conductorID, "warehouse", op1.ID, uc.AddOpnameItemRequest{
		ProductID:   prodID,
		LocationID:  locID,
		PhysicalQty: decimal.NewFromInt(25),
		Notes:       ptr("Surplus found on top shelf"),
	})
	require.NoError(t, err)

	movementsBefore := len(repo.stockMovements)
	pendingOp, err := usecase.CompleteStockOpname(ctx, tenantID, conductorID, "warehouse", op1.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StockOpnameStatusPendingApproval, pendingOp.Status)
	assert.Equal(t, movementsBefore, len(repo.stockMovements), "no movements should be posted when transitioning to PENDING_APPROVAL")

	// 2. Complete opname by conductor with admin role returns ErrSelfApprovalForbidden
	op2, err := usecase.CreateStockOpname(ctx, tenantID, adminID, "admin", uc.CreateStockOpnameRequest{WarehouseID: whID})
	require.NoError(t, err)
	_, err = usecase.AddOpnameItem(ctx, tenantID, adminID, "admin", op2.ID, uc.AddOpnameItemRequest{
		ProductID:   prodID,
		LocationID:  locID,
		PhysicalQty: decimal.NewFromInt(18),
		Notes:       ptr("Slight shortage"),
	})
	require.NoError(t, err)

	_, err = usecase.CompleteStockOpname(ctx, tenantID, adminID, "admin", op2.ID)
	assert.ErrorIs(t, err, domain.ErrSelfApprovalForbidden, "conductor cannot self-approve their own opname")

	// 3. Complete opname with discrepancy and no notes returns ErrInvalidInput
	op3, err := usecase.CreateStockOpname(ctx, tenantID, conductorID, "warehouse", uc.CreateStockOpnameRequest{WarehouseID: whID})
	require.NoError(t, err)
	_, err = usecase.AddOpnameItem(ctx, tenantID, conductorID, "warehouse", op3.ID, uc.AddOpnameItemRequest{
		ProductID:   prodID,
		LocationID:  locID,
		PhysicalQty: decimal.NewFromInt(15), // Discrepancy = -5, notes = nil
	})
	require.NoError(t, err)

	_, err = usecase.CompleteStockOpname(ctx, tenantID, adminID, "admin", op3.ID)
	assert.ErrorIs(t, err, domain.ErrInvalidInput, "opname with discrepancy and no notes must return ErrInvalidInput")

	// 4. Complete opname by different admin with valid notes completes and posts movements
	op4, err := usecase.CreateStockOpname(ctx, tenantID, conductorID, "warehouse", uc.CreateStockOpnameRequest{WarehouseID: whID})
	require.NoError(t, err)
	_, err = usecase.AddOpnameItem(ctx, tenantID, conductorID, "warehouse", op4.ID, uc.AddOpnameItemRequest{
		ProductID:   prodID,
		LocationID:  locID,
		PhysicalQty: decimal.NewFromInt(15),
		Notes:       ptr("Damaged items removed during physical check"),
	})
	require.NoError(t, err)

	completedOp, err := usecase.CompleteStockOpname(ctx, tenantID, adminID, "admin", op4.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StockOpnameStatusCompleted, completedOp.Status)
	assert.Equal(t, &adminID, completedOp.ApprovedBy)
	assert.Greater(t, len(repo.stockMovements), movementsBefore, "movements must be posted upon opname completion")
}

func TestWMSFraudControls_F3_Scrap(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	locID := uuid.New()
	prodID := uuid.New()
	staffID := uuid.New()
	adminID := uuid.New()

	repo.warehouses[whID] = domain.Warehouse{ID: whID, TenantID: tenantID, Code: "WH-F3-S", Name: "Scrap WH", IsActive: true}
	repo.locations[locID] = domain.WarehouseLocation{ID: locID, TenantID: tenantID, WarehouseID: &whID, Code: "LOC-SCRAP", Name: "Rack Scrap", Type: domain.LocationTypeInternal}
	repo.userWarehouses[fmt.Sprintf("%s:%s", tenantID, staffID)] = []uuid.UUID{whID}
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, locID, prodID)] = decimal.NewFromInt(100)

	// 1. Scrap with reason < 10 chars returns ErrInvalidInput
	reqShortReason := uc.CreateStockScrapRequest{
		WarehouseID:      whID,
		ProductID:        prodID,
		SourceLocationID: locID,
		Quantity:         decimal.NewFromInt(5),
		Reason:           "short",
	}
	_, err := usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", reqShortReason)
	assert.ErrorIs(t, err, domain.ErrInvalidInput, "reason < 10 characters must return ErrInvalidInput")

	// 2. Scrap > 10 units by role warehouse returns ErrScrapApprovalRequired
	reqThresholdExceeded := uc.CreateStockScrapRequest{
		WarehouseID:      whID,
		ProductID:        prodID,
		SourceLocationID: locID,
		Quantity:         decimal.NewFromInt(15), // > 10
		Reason:           "Water leakage spoiled whole carton of goods",
	}
	_, err = usecase.CreateStockScrap(ctx, tenantID, staffID, "warehouse", reqThresholdExceeded)
	assert.ErrorIs(t, err, domain.ErrScrapApprovalRequired, "scrap > 10 units by warehouse role must require approval")

	// 3. Scrap > 10 units by admin succeeds
	scrap, err := usecase.CreateStockScrap(ctx, tenantID, adminID, "admin", reqThresholdExceeded)
	require.NoError(t, err)
	assert.NotNil(t, scrap.ApprovedBy)
	assert.Equal(t, adminID, *scrap.ApprovedBy)
}

func TestWMSFraudControls_F4_Putaway_OnHold(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	adminID := uuid.New()
	prodID := uuid.New()
	batchID := uuid.New()

	repo.warehouses[whID] = domain.Warehouse{ID: whID, TenantID: tenantID, Code: "WH-F4", Name: "Putaway WH", IsActive: true}
	destLoc := uuid.New()
	repo.locations[destLoc] = domain.WarehouseLocation{
		ID:          destLoc,
		TenantID:    tenantID,
		WarehouseID: &whID,
		Code:        "RACK-F4",
		Type:        domain.LocationTypeInternal,
	}

	stgLoc, err := repo.GetOrCreateStagingLocation(ctx, tenantID, whID)
	require.NoError(t, err)

	b := &domain.StockBatch{
		ID:          batchID,
		TenantID:    tenantID,
		ProductID:   prodID,
		BatchNumber: "LOT-HOLD-001",
		Status:      domain.StockBatchStatusOnHold,
		CreatedAt:   time.Now().UTC(),
	}
	_, err = repo.GetOrCreateBatch(ctx, b)
	require.NoError(t, err)

	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, stgLoc.ID, prodID)] = decimal.NewFromInt(50)

	// ConfirmPutaway for batch with status ON_HOLD returns ErrBatchOnHold
	req := uc.PutawayRequest{
		WarehouseID:    whID,
		ProductID:      prodID,
		BatchID:        batchID,
		Quantity:       decimal.NewFromInt(10),
		DestLocationID: destLoc,
		Reason:         ptr("Rack selection"),
	}
	_, err = usecase.ConfirmPutaway(ctx, tenantID, adminID, "admin", req)
	assert.ErrorIs(t, err, domain.ErrBatchOnHold, "putaway for ON_HOLD batch must return ErrBatchOnHold")
}

func TestWMSFraudControls_F6_DO_CustomerRecipient(t *testing.T) {
	repo := newMockWMSRepo()
	usecase := uc.New(repo)
	ctx := context.Background()

	tenantID := uuid.New()
	whID := uuid.New()
	locID := uuid.New()
	prodID := uuid.New()
	adminID := uuid.New()

	repo.warehouses[whID] = domain.Warehouse{ID: whID, TenantID: tenantID, Code: "WH-F6", Name: "DO Recipient WH", IsActive: true}
	repo.locations[locID] = domain.WarehouseLocation{ID: locID, TenantID: tenantID, WarehouseID: &whID, Code: "LOC-F6", Name: "Rack F6", Type: domain.LocationTypeInternal}
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, locID, prodID)] = decimal.NewFromInt(50)

	// 1. Neither CustomerID nor RecipientName provided -> ErrInvalidInput
	reqBothNil := uc.CreateDeliveryOrderRequest{
		WarehouseID: whID,
		DONumber:    "DO-F6-01",
		Items: []uc.CreateDeliveryOrderItemRequest{
			{ProductID: prodID, Quantity: decimal.NewFromInt(5), LocationID: locID},
		},
	}
	_, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", reqBothNil)
	assert.ErrorIs(t, err, domain.ErrInvalidInput, "missing both CustomerID and RecipientName must return ErrInvalidInput")

	// 2. Whitespace-only RecipientName and nil CustomerID -> ErrInvalidInput
	reqWhitespace := uc.CreateDeliveryOrderRequest{
		WarehouseID:   whID,
		DONumber:      "DO-F6-02",
		RecipientName: ptr("   "),
		Items: []uc.CreateDeliveryOrderItemRequest{
			{ProductID: prodID, Quantity: decimal.NewFromInt(5), LocationID: locID},
		},
	}
	_, err = usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", reqWhitespace)
	assert.ErrorIs(t, err, domain.ErrInvalidInput, "whitespace RecipientName must return ErrInvalidInput")

	// 3. Valid RecipientName provided -> Succeeds
	reqValidRecipient := uc.CreateDeliveryOrderRequest{
		WarehouseID:   whID,
		DONumber:      "DO-F6-03",
		RecipientName: ptr("Toko Sumber Rejeki"),
		Items: []uc.CreateDeliveryOrderItemRequest{
			{ProductID: prodID, Quantity: decimal.NewFromInt(5), LocationID: locID},
		},
	}
	doRecipient, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", reqValidRecipient)
	require.NoError(t, err)
	assert.NotNil(t, doRecipient)

	// 4. Valid CustomerID provided with nil RecipientName -> Succeeds
	custID := uuid.New()
	reqValidCustomer := uc.CreateDeliveryOrderRequest{
		WarehouseID: whID,
		DONumber:    "DO-F6-04",
		CustomerID:  &custID,
		Items: []uc.CreateDeliveryOrderItemRequest{
			{ProductID: prodID, Quantity: decimal.NewFromInt(5), LocationID: locID},
		},
	}
	doCustomer, err := usecase.CreateDeliveryOrder(ctx, tenantID, adminID, "admin", reqValidCustomer)
	require.NoError(t, err)
	assert.NotNil(t, doCustomer)
}

