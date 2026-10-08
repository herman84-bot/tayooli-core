package wms_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

func (m *mockWMSRepo) GetOrCreateBatch(ctx context.Context, b *domain.StockBatch) (*domain.StockBatch, error) {
	if b == nil || b.TenantID == uuid.Nil || b.ProductID == uuid.Nil || strings.TrimSpace(b.BatchNumber) == "" {
		return nil, domain.ErrInvalidInput
	}
	if m.batches == nil {
		m.batches = make(map[uuid.UUID]*domain.StockBatch)
	}
	for _, existing := range m.batches {
		if existing.TenantID == b.TenantID && existing.ProductID == b.ProductID && existing.BatchNumber == b.BatchNumber {
			return existing, nil
		}
	}
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.Status == "" {
		b.Status = domain.StockBatchStatusReleased
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}
	cp := *b
	m.batches[b.ID] = &cp
	return &cp, nil
}

func (m *mockWMSRepo) GetBatchByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockBatch, error) {
	if m.batches == nil {
		return nil, domain.ErrStockBatchNotFound
	}
	b, ok := m.batches[id]
	if !ok || b.TenantID != tenantID {
		return nil, domain.ErrStockBatchNotFound
	}
	cp := *b
	return &cp, nil
}

func (m *mockWMSRepo) ListBatchBalances(ctx context.Context, tenantID uuid.UUID, filter domain.BatchBalanceFilter) ([]domain.BatchBalance, error) {
	var results []domain.BatchBalance
	for _, b := range m.batches {
		if b.TenantID != tenantID {
			continue
		}
		if filter.ProductID != nil && b.ProductID != *filter.ProductID {
			continue
		}
		for locID, loc := range m.locations {
			if loc.TenantID != tenantID {
				continue
			}
			key := fmt.Sprintf("%s:%s:%s", tenantID, locID, b.ProductID)
			qty := m.stockLevels[key]
			if qty.IsPositive() {
				results = append(results, domain.BatchBalance{
					BatchID:      b.ID,
					BatchNumber:  b.BatchNumber,
					ExpiryDate:   b.ExpiryDate,
					Status:       b.Status,
					LocationID:   locID,
					LocationCode: loc.Code,
					ProductID:    b.ProductID,
					Quantity:     qty,
					CreatedAt:    b.CreatedAt,
				})
			}
		}
	}
	return results, nil
}

func (m *mockWMSRepo) CreateStockMovements(ctx context.Context, tenantID uuid.UUID, movs []domain.StockMovement) error {
	for i := range movs {
		if err := m.CreateStockMovement(ctx, &movs[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockWMSRepo) ListMovementsByReference(ctx context.Context, tenantID uuid.UUID, refType string, refID uuid.UUID) ([]domain.StockMovement, error) {
	var list []domain.StockMovement
	for _, sm := range m.stockMovements {
		if sm.TenantID == tenantID && sm.ReferenceType == refType && sm.ReferenceID == refID {
			list = append(list, sm)
		}
	}
	return list, nil
}

func (m *mockWMSRepo) DeductWarehouseStock(ctx context.Context, tenantID, warehouseID, productID uuid.UUID, qty decimal.Decimal, mov *domain.StockMovement) error {
	var targetLoc *domain.WarehouseLocation
	for _, loc := range m.locations {
		if loc.TenantID == tenantID && loc.WarehouseID != nil && *loc.WarehouseID == warehouseID && loc.Type == domain.LocationTypeInternal {
			key := fmt.Sprintf("%s:%s:%s", tenantID, loc.ID, productID)
			if m.stockLevels[key].GreaterThanOrEqual(qty) {
				locCopy := loc
				targetLoc = &locCopy
				break
			}
		}
	}
	if targetLoc == nil {
		return domain.ErrInsufficientStock
	}
	mov.SourceLocationID = targetLoc.ID
	return m.DeductLocationStock(ctx, tenantID, targetLoc.ID, productID, qty, mov)
}

func (m *mockWMSRepo) GetOrCreateStagingLocation(ctx context.Context, tenantID, warehouseID uuid.UUID) (*domain.WarehouseLocation, error) {
	for _, loc := range m.locations {
		if loc.TenantID == tenantID && loc.WarehouseID != nil && *loc.WarehouseID == warehouseID && (loc.Code == domain.StagingInboundCode || loc.Type == domain.LocationTypeStagingInbound) {
			cp := loc
			return &cp, nil
		}
	}
	now := time.Now().UTC()
	stg := domain.WarehouseLocation{
		ID:          uuid.New(),
		TenantID:    tenantID,
		WarehouseID: &warehouseID,
		Code:        domain.StagingInboundCode,
		Name:        "Inbound Staging",
		Type:        domain.LocationTypeStagingInbound,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.locations[stg.ID] = stg
	return &stg, nil
}

func (m *mockWMSRepo) ConfirmPutaway(ctx context.Context, cmd domain.PutawayCommand) (*domain.StockMovement, error) {
	if cmd.ProductID == uuid.Nil || cmd.BatchID == uuid.Nil || cmd.StagingLocationID == uuid.Nil || cmd.DestLocationID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	if cmd.UserID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}
	if !cmd.Quantity.IsPositive() {
		return nil, domain.ErrInvalidInput
	}
	stgKey := fmt.Sprintf("%s:%s:%s", cmd.TenantID, cmd.StagingLocationID, cmd.ProductID)
	if m.stockLevels[stgKey].LessThan(cmd.Quantity) {
		return nil, domain.ErrInsufficientStock
	}
	now := time.Now().UTC()
	mov := &domain.StockMovement{
		ID:               uuid.New(),
		TenantID:         cmd.TenantID,
		MovementNumber:   fmt.Sprintf("PUTAWAY-%d", time.Now().UnixNano()),
		ProductID:        cmd.ProductID,
		BatchID:          &cmd.BatchID,
		SourceLocationID: cmd.StagingLocationID,
		DestLocationID:   cmd.DestLocationID,
		Quantity:         cmd.Quantity,
		UnitCost:         decimal.Zero,
		Status:           domain.StockMovementStatusDone,
		ReferenceType:    domain.StockRefPutaway,
		ReferenceID:      cmd.BatchID,
		ExecutedBy:       &cmd.UserID,
		CreatedAt:        now,
	}
	if err := m.CreateStockMovement(ctx, mov); err != nil {
		return nil, err
	}
	return mov, nil
}

func (m *mockWMSRepo) ReleaseStockReceipt(ctx context.Context, tenantID, receiptID, userID uuid.UUID) (*domain.StockReceipt, error) {
	m.initReceipts()
	rc, ok := m.stockReceipts[receiptID]
	if !ok || rc.TenantID != tenantID {
		return nil, domain.ErrStockReceiptNotFound
	}
	now := time.Now().UTC()
	rc.ReleasedBy = &userID
	rc.ReleasedAt = &now
	for _, b := range m.batches {
		if b.TenantID == tenantID && b.SourceReceiptID != nil && *b.SourceReceiptID == receiptID {
			b.Status = domain.StockBatchStatusReleased
		}
	}
	cp := *rc
	return &cp, nil
}

func (m *mockWMSRepo) GetWMSSettings(ctx context.Context, tenantID uuid.UUID) (*domain.WMSSettings, error) {
	if m.wmsSettings == nil {
		m.wmsSettings = make(map[uuid.UUID]*domain.WMSSettings)
	}
	s, ok := m.wmsSettings[tenantID]
	if !ok {
		return &domain.WMSSettings{TenantID: tenantID, RequireReleaseApproval: false, UpdatedAt: time.Now().UTC()}, nil
	}
	cp := *s
	return &cp, nil
}

func (m *mockWMSRepo) UpsertWMSSettings(ctx context.Context, s *domain.WMSSettings) error {
	if s == nil || s.TenantID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	if m.wmsSettings == nil {
		m.wmsSettings = make(map[uuid.UUID]*domain.WMSSettings)
	}
	s.UpdatedAt = time.Now().UTC()
	cp := *s
	m.wmsSettings[s.TenantID] = &cp
	return nil
}

func (m *mockWMSRepo) ListDefaultLocations(ctx context.Context, tenantID uuid.UUID, productID *uuid.UUID) ([]domain.ProductDefaultLocation, error) {
	var list []domain.ProductDefaultLocation
	for _, def := range m.defaultLocations {
		if def.TenantID != tenantID {
			continue
		}
		if productID != nil && def.ProductID != *productID {
			continue
		}
		list = append(list, *def)
	}
	return list, nil
}

func (m *mockWMSRepo) SetDefaultLocation(ctx context.Context, def *domain.ProductDefaultLocation) error {
	if def == nil || def.TenantID == uuid.Nil || def.ProductID == uuid.Nil || def.WarehouseID == uuid.Nil || def.LocationID == uuid.Nil {
		return domain.ErrInvalidInput
	}
	if m.defaultLocations == nil {
		m.defaultLocations = make(map[string]*domain.ProductDefaultLocation)
	}
	key := fmt.Sprintf("%s:%s:%s", def.TenantID, def.ProductID, def.WarehouseID)
	cp := *def
	m.defaultLocations[key] = &cp
	return nil
}

func (m *mockWMSRepo) DeleteDefaultLocation(ctx context.Context, tenantID, productID, warehouseID uuid.UUID) error {
	if m.defaultLocations == nil {
		return nil
	}
	key := fmt.Sprintf("%s:%s:%s", tenantID, productID, warehouseID)
	delete(m.defaultLocations, key)
	return nil
}

func (m *mockWMSRepo) TraceBatch(ctx context.Context, tenantID, batchID uuid.UUID) (*domain.BatchTrace, error) {
	b, err := m.GetBatchByID(ctx, tenantID, batchID)
	if err != nil {
		return nil, err
	}
	trace := &domain.BatchTrace{Batch: *b}
	for _, sm := range m.stockMovements {
		if sm.TenantID == tenantID && sm.BatchID != nil && *sm.BatchID == batchID {
			trace.Movements = append(trace.Movements, domain.TraceMovement{
				MovementID:     sm.ID,
				MovementNumber: sm.MovementNumber,
				ReferenceType:  sm.ReferenceType,
				ReferenceID:    sm.ReferenceID,
				Quantity:       sm.Quantity,
				CreatedAt:      sm.CreatedAt,
			})
		}
	}
	return trace, nil
}

func (m *mockWMSRepo) TraceDocument(ctx context.Context, tenantID uuid.UUID, refType string, refID uuid.UUID) (*domain.DocumentTrace, error) {
	trace := &domain.DocumentTrace{
		DocumentType: refType,
		DocumentID:   refID,
	}
	for _, sm := range m.stockMovements {
		if sm.TenantID == tenantID && sm.ReferenceType == refType && sm.ReferenceID == refID {
			trace.Movements = append(trace.Movements, domain.TraceMovement{
				MovementID:     sm.ID,
				MovementNumber: sm.MovementNumber,
				ReferenceType:  sm.ReferenceType,
				ReferenceID:    sm.ReferenceID,
				Quantity:       sm.Quantity,
				CreatedAt:      sm.CreatedAt,
			})
		}
	}
	return trace, nil
}

func (m *mockWMSRepo) ListAuditTrail(ctx context.Context, tenantID uuid.UUID, entityType string, entityID uuid.UUID) ([]domain.AuditTrailEntry, error) {
	var entries []domain.AuditTrailEntry
	for _, a := range m.auditLogs {
		if a.EntityType == entityType && a.EntityID == entityID {
			entries = append(entries, a)
		}
	}
	return entries, nil
}
