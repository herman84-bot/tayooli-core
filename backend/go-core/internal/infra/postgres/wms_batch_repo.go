package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

var _ domain.WMSBatchRepository = (*WMSRepo)(nil)

const stockBatchColumns = `id, tenant_id, product_id, batch_number, expiry_date, source_receipt_id, status, is_legacy, created_by, created_at`

func scanStockBatch(r rowScanner) (*domain.StockBatch, error) {
	var b domain.StockBatch
	var exp sql.NullTime
	var srcReceipt, createdBy sql.NullString
	if err := r.Scan(
		&b.ID, &b.TenantID, &b.ProductID, &b.BatchNumber,
		&exp, &srcReceipt, &b.Status, &b.IsLegacy,
		&createdBy, &b.CreatedAt,
	); err != nil {
		return nil, err
	}
	b.ExpiryDate = nullTimeToPtr(exp)
	b.SourceReceiptID = nullUUIDToPtr(srcReceipt)
	b.CreatedBy = nullUUIDToPtr(createdBy)
	return &b, nil
}

// -----------------------------------------------------------------------------
// 1. Stock Batches
// -----------------------------------------------------------------------------

func (r *WMSRepo) GetOrCreateBatch(ctx context.Context, b *domain.StockBatch) (*domain.StockBatch, error) {
	if b == nil {
		return nil, domain.ErrInvalidInput
	}
	if b.TenantID == uuid.Nil || b.ProductID == uuid.Nil || strings.TrimSpace(b.BatchNumber) == "" {
		return nil, domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateBatch: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, b.TenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateBatch: set tenant: %w", err)
	}

	batch, err := r.getOrCreateBatchTx(ctx, tx, b)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateBatch: commit: %w", err)
	}
	return batch, nil
}

func (r *WMSRepo) getOrCreateBatchTx(ctx context.Context, tx *sql.Tx, b *domain.StockBatch) (*domain.StockBatch, error) {
	if b == nil {
		return nil, domain.ErrInvalidInput
	}
	if b.TenantID == uuid.Nil || b.ProductID == uuid.Nil || strings.TrimSpace(b.BatchNumber) == "" {
		return nil, domain.ErrInvalidInput
	}

	query := `SELECT ` + stockBatchColumns + ` FROM stock_batches WHERE tenant_id = $1 AND product_id = $2 AND batch_number = $3`
	row := tx.QueryRowContext(ctx, query, b.TenantID, b.ProductID, b.BatchNumber)
	existing, err := scanStockBatch(row)
	if err == nil {
		if existing.ExpiryDate != nil && b.ExpiryDate != nil {
			y1, m1, d1 := existing.ExpiryDate.UTC().Date()
			y2, m2, d2 := b.ExpiryDate.UTC().Date()
			if y1 != y2 || m1 != m2 || d1 != d2 {
				return nil, domain.ErrBatchExpiryMismatch
			}
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("WMSRepo.getOrCreateBatchTx: query: %w", err)
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

	insertSQL := `
INSERT INTO stock_batches (id, tenant_id, product_id, batch_number, expiry_date, source_receipt_id, status, is_legacy, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = tx.ExecContext(ctx, insertSQL,
		b.ID, b.TenantID, b.ProductID, b.BatchNumber,
		ptrToNullTime(b.ExpiryDate), ptrToNullUUID(b.SourceReceiptID),
		b.Status, b.IsLegacy, ptrToNullUUID(b.CreatedBy), b.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			row := tx.QueryRowContext(ctx, query, b.TenantID, b.ProductID, b.BatchNumber)
			existing, errScan := scanStockBatch(row)
			if errScan != nil {
				return nil, fmt.Errorf("WMSRepo.getOrCreateBatchTx: re-query after race: %w", errScan)
			}
			if existing.ExpiryDate != nil && b.ExpiryDate != nil {
				y1, m1, d1 := existing.ExpiryDate.UTC().Date()
				y2, m2, d2 := b.ExpiryDate.UTC().Date()
				if y1 != y2 || m1 != m2 || d1 != d2 {
					return nil, domain.ErrBatchExpiryMismatch
				}
			}
			return existing, nil
		}
		return nil, fmt.Errorf("WMSRepo.getOrCreateBatchTx: insert: %w", err)
	}

	return b, nil
}

func (r *WMSRepo) GetBatchByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockBatch, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetBatchByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetBatchByID: set tenant: %w", err)
	}

	query := `SELECT ` + stockBatchColumns + ` FROM stock_batches WHERE tenant_id = $1 AND id = $2`
	row := tx.QueryRowContext(ctx, query, tenantID, id)
	batch, err := scanStockBatch(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrStockBatchNotFound
		}
		return nil, fmt.Errorf("WMSRepo.GetBatchByID: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetBatchByID: commit: %w", err)
	}
	return batch, nil
}

// -----------------------------------------------------------------------------
// 2. Batch Balances (On-Hand Ledger)
// -----------------------------------------------------------------------------

func (r *WMSRepo) ListBatchBalances(ctx context.Context, tenantID uuid.UUID, f domain.BatchBalanceFilter) ([]domain.BatchBalance, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListBatchBalances: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListBatchBalances: set tenant: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(`
SELECT b.id, b.batch_number, b.expiry_date, b.status, loc.id, loc.code, sm.product_id,
       COALESCE(p.name, ''), COALESCE(p.sku, ''),
       SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty,
       b.created_at
FROM stock_movements sm
JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
LEFT JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
WHERE sm.tenant_id = $1 AND sm.status = 'DONE'`)

	args := []any{tenantID}
	argIdx := 2

	if f.ProductID != nil {
		sb.WriteString(fmt.Sprintf(" AND sm.product_id = $%d", argIdx))
		args = append(args, *f.ProductID)
		argIdx++
	}
	if f.WarehouseID != nil {
		sb.WriteString(fmt.Sprintf(" AND loc.warehouse_id = $%d", argIdx))
		args = append(args, *f.WarehouseID)
		argIdx++
	}
	if len(f.LocationIDs) > 0 {
		locStrings := make([]string, len(f.LocationIDs))
		for i, id := range f.LocationIDs {
			locStrings[i] = id.String()
		}
		sb.WriteString(fmt.Sprintf(" AND loc.id = ANY($%d::uuid[])", argIdx))
		args = append(args, pq.Array(locStrings))
		argIdx++
	}
	if f.LocationType != nil {
		sb.WriteString(fmt.Sprintf(" AND loc.type = $%d", argIdx))
		args = append(args, string(*f.LocationType))
		argIdx++
	}
	if f.BatchStatus != nil {
		sb.WriteString(fmt.Sprintf(" AND b.status = $%d", argIdx))
		args = append(args, string(*f.BatchStatus))
		argIdx++
	}

	sb.WriteString(`
GROUP BY b.id, b.batch_number, b.expiry_date, b.status, b.created_at, loc.id, loc.code, sm.product_id, p.name, p.sku
HAVING SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) > 0
ORDER BY b.expiry_date ASC NULLS LAST, b.created_at ASC, loc.code ASC, b.id ASC`)

	rows, err := tx.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListBatchBalances: query: %w", err)
	}
	defer rows.Close()

	var balances []domain.BatchBalance
	for rows.Next() {
		var bal domain.BatchBalance
		var exp sql.NullTime
		if err := rows.Scan(
			&bal.BatchID, &bal.BatchNumber, &exp, &bal.Status,
			&bal.LocationID, &bal.LocationCode, &bal.ProductID,
			&bal.ProductName, &bal.ProductSKU,
			&bal.Quantity, &bal.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListBatchBalances: scan: %w", err)
		}
		bal.ExpiryDate = nullTimeToPtr(exp)
		balances = append(balances, bal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListBatchBalances: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListBatchBalances: commit: %w", err)
	}
	return balances, nil
}

// -----------------------------------------------------------------------------
// 3. Stock Movements (Batch-Enforced)
// -----------------------------------------------------------------------------

func (r *WMSRepo) CreateStockMovements(ctx context.Context, tenantID uuid.UUID, movs []domain.StockMovement) error {
	if len(movs) == 0 {
		return nil
	}
	for _, m := range movs {
		if m.BatchID == nil || *m.BatchID == uuid.Nil {
			return domain.ErrBatchRequired
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateStockMovements: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateStockMovements: set tenant: %w", err)
	}

	now := time.Now().UTC()
	for i := range movs {
		m := &movs[i]
		if m.ID == uuid.Nil {
			m.ID = uuid.New()
		}
		if m.TenantID == uuid.Nil {
			m.TenantID = tenantID
		}
		if m.CreatedAt.IsZero() {
			m.CreatedAt = now
		}
		if m.Status == "" {
			m.Status = domain.StockMovementStatusDone
		}

		_, err := tx.ExecContext(ctx, createStockMovementSQL,
			m.ID, m.TenantID, m.MovementNumber, m.ProductID,
			m.SourceLocationID, m.DestLocationID, m.Quantity,
			m.UnitCost, m.Status, m.ReferenceType, m.ReferenceID,
			ptrToNullUUID(m.ExecutedBy), ptrToNullUUID(m.BatchID), m.CreatedAt)
		if err != nil {
			return fmt.Errorf("WMSRepo.CreateStockMovements: exec movement %d: %w", i, err)
		}

		auditDetails, _ := json.Marshal(map[string]any{
			"movement_number":    m.MovementNumber,
			"product_id":         m.ProductID,
			"source_location_id": m.SourceLocationID,
			"dest_location_id":   m.DestLocationID,
			"quantity":           m.Quantity,
			"reference_type":     m.ReferenceType,
			"reference_id":       m.ReferenceID,
			"batch_id":           m.BatchID,
		})
		if err := r.WriteAuditTx(ctx, tx, m.TenantID, m.ExecutedBy, "stock_movement", m.ID, "created", auditDetails); err != nil {
			return fmt.Errorf("WMSRepo.CreateStockMovements: audit movement %d: %w", i, err)
		}
	}

	return tx.Commit()
}

func (r *WMSRepo) ListMovementsByReference(ctx context.Context, tenantID uuid.UUID, refType string, refID uuid.UUID) ([]domain.StockMovement, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMovementsByReference: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMovementsByReference: set tenant: %w", err)
	}

	query := `
SELECT 
    sm.id, sm.tenant_id, sm.movement_number, sm.product_id,
    COALESCE(p.name, ''), COALESCE(p.sku, ''),
    sm.source_location_id, COALESCE(sl.code, ''),
    sm.dest_location_id, COALESCE(dl.code, ''),
    sm.quantity, sm.unit_cost, sm.status, sm.reference_type, sm.reference_id,
    sm.batch_id, COALESCE(b.batch_number, ''),
    sm.executed_by, COALESCE(u.full_name, u.email, ''), sm.created_at
FROM stock_movements sm
LEFT JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
LEFT JOIN warehouse_locations sl ON sl.id = sm.source_location_id AND sl.tenant_id = sm.tenant_id
LEFT JOIN warehouse_locations dl ON dl.id = sm.dest_location_id AND dl.tenant_id = sm.tenant_id
LEFT JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
LEFT JOIN users u ON u.id = sm.executed_by
WHERE sm.tenant_id = $1 AND sm.reference_type = $2 AND sm.reference_id = $3
ORDER BY sm.created_at ASC, sm.id ASC`

	rows, err := tx.QueryContext(ctx, query, tenantID, refType, refID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMovementsByReference: query: %w", err)
	}
	defer rows.Close()

	var result []domain.StockMovement
	for rows.Next() {
		var m domain.StockMovement
		var execBy, batchID sql.NullString
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.MovementNumber, &m.ProductID,
			&m.ProductName, &m.SKU,
			&m.SourceLocationID, &m.SourceLocationCode,
			&m.DestLocationID, &m.DestLocationCode,
			&m.Quantity, &m.UnitCost, &m.Status, &m.ReferenceType, &m.ReferenceID,
			&batchID, &m.BatchNumber,
			&execBy, &m.ExecutedByName, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListMovementsByReference: scan: %w", err)
		}
		m.BatchID = nullUUIDToPtr(batchID)
		m.ExecutedBy = nullUUIDToPtr(execBy)
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMovementsByReference: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListMovementsByReference: commit: %w", err)
	}
	return result, nil
}

// DeductWarehouseStock allocates FEFO across all INTERNAL racks of a warehouse.
func (r *WMSRepo) DeductWarehouseStock(ctx context.Context, tenantID, warehouseID, productID uuid.UUID, qty decimal.Decimal, mov *domain.StockMovement) error {
	if !qty.IsPositive() {
		return domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.DeductWarehouseStock: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.DeductWarehouseStock: set tenant: %w", err)
	}

	// Advisory transaction lock per (tenant, warehouse, product)
	_, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext($1::text || $2::text || $3::text))", tenantID, warehouseID, productID)
	if err != nil {
		return fmt.Errorf("WMSRepo.DeductWarehouseStock: advisory lock: %w", err)
	}

	query := `
SELECT b.id, b.batch_number, b.expiry_date, b.status, loc.id, loc.code, sm.product_id,
       COALESCE(p.name, ''), COALESCE(p.sku, ''),
       SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty,
       b.created_at
FROM stock_movements sm
JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
LEFT JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
WHERE sm.tenant_id = $1 AND sm.product_id = $2 AND loc.warehouse_id = $3 AND loc.type = 'INTERNAL' AND sm.status = 'DONE'
GROUP BY b.id, b.batch_number, b.expiry_date, b.status, b.created_at, loc.id, loc.code, sm.product_id, p.name, p.sku
HAVING SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) > 0
ORDER BY b.expiry_date ASC NULLS LAST, b.created_at ASC, loc.code ASC, b.id ASC`

	rows, err := tx.QueryContext(ctx, query, tenantID, productID, warehouseID)
	if err != nil {
		return fmt.Errorf("WMSRepo.DeductWarehouseStock: query balances: %w", err)
	}
	defer rows.Close()

	var balances []domain.BatchBalance
	for rows.Next() {
		var bal domain.BatchBalance
		var exp sql.NullTime
		if err := rows.Scan(
			&bal.BatchID, &bal.BatchNumber, &exp, &bal.Status,
			&bal.LocationID, &bal.LocationCode, &bal.ProductID,
			&bal.ProductName, &bal.ProductSKU,
			&bal.Quantity, &bal.CreatedAt,
		); err != nil {
			return fmt.Errorf("WMSRepo.DeductWarehouseStock: scan: %w", err)
		}
		bal.ExpiryDate = nullTimeToPtr(exp)
		balances = append(balances, bal)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("WMSRepo.DeductWarehouseStock: rows err: %w", err)
	}

	allocs, err := domain.AllocateFEFO(balances, qty, false)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for i, alloc := range allocs {
		m := *mov
		m.ID = uuid.New()
		m.TenantID = tenantID
		m.ProductID = productID
		m.BatchID = &alloc.BatchID
		m.SourceLocationID = alloc.LocationID
		m.Quantity = alloc.Quantity
		m.CreatedAt = now
		if m.Status == "" {
			m.Status = domain.StockMovementStatusDone
		}
		if len(allocs) > 1 {
			m.MovementNumber = fmt.Sprintf("%s-B%d", mov.MovementNumber, i+1)
		}

		_, err := tx.ExecContext(ctx, createStockMovementSQL,
			m.ID, m.TenantID, m.MovementNumber, m.ProductID,
			m.SourceLocationID, m.DestLocationID, m.Quantity,
			m.UnitCost, m.Status, m.ReferenceType, m.ReferenceID,
			ptrToNullUUID(m.ExecutedBy), ptrToNullUUID(m.BatchID), m.CreatedAt)
		if err != nil {
			return fmt.Errorf("WMSRepo.DeductWarehouseStock: insert movement %d: %w", i, err)
		}

		auditDetails, _ := json.Marshal(map[string]any{
			"movement_number":    m.MovementNumber,
			"product_id":         m.ProductID,
			"source_location_id": m.SourceLocationID,
			"dest_location_id":   m.DestLocationID,
			"quantity":           m.Quantity,
			"reference_type":     m.ReferenceType,
			"reference_id":       m.ReferenceID,
			"batch_id":           m.BatchID,
		})
		if err := r.WriteAuditTx(ctx, tx, tenantID, m.ExecutedBy, "stock_movement", m.ID, "created", auditDetails); err != nil {
			return fmt.Errorf("WMSRepo.DeductWarehouseStock: audit movement %d: %w", i, err)
		}
	}

	return tx.Commit()
}

// -----------------------------------------------------------------------------
// 4. Staging Location
// -----------------------------------------------------------------------------

func (r *WMSRepo) GetOrCreateStagingLocation(ctx context.Context, tenantID, warehouseID uuid.UUID) (*domain.WarehouseLocation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateStagingLocation: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateStagingLocation: set tenant: %w", err)
	}

	findSQL := `
SELECT id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at
FROM warehouse_locations
WHERE tenant_id = $1 AND warehouse_id = $2 AND (code = 'STG-IN' OR type = 'STAGING_INBOUND')
LIMIT 1`

	row := tx.QueryRowContext(ctx, findSQL, tenantID, warehouseID)
	loc, err := scanLocationRow(row)
	if err == nil {
		if errCommit := tx.Commit(); errCommit != nil {
			return nil, fmt.Errorf("WMSRepo.GetOrCreateStagingLocation: commit: %w", errCommit)
		}
		return loc, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateStagingLocation: query existing: %w", err)
	}

	now := time.Now().UTC()
	newLoc := domain.WarehouseLocation{
		ID:          uuid.New(),
		TenantID:    tenantID,
		WarehouseID: &warehouseID,
		Code:        domain.StagingInboundCode,
		Name:        "Inbound Staging",
		Type:        domain.LocationTypeStagingInbound,
		IsPallet:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	insertSQL := `
INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (tenant_id, warehouse_id, code) DO NOTHING`

	_, err = tx.ExecContext(ctx, insertSQL,
		newLoc.ID, newLoc.TenantID, newLoc.WarehouseID, nil,
		newLoc.Code, nil, newLoc.Name, newLoc.Type, newLoc.IsPallet,
		nil, nil, newLoc.CreatedAt, newLoc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateStagingLocation: insert: %w", err)
	}

	row = tx.QueryRowContext(ctx, findSQL, tenantID, warehouseID)
	loc, err = scanLocationRow(row)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateStagingLocation: re-query after insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateStagingLocation: commit: %w", err)
	}
	return loc, nil
}

// -----------------------------------------------------------------------------
// 5. Putaway Confirmation
// -----------------------------------------------------------------------------

func (r *WMSRepo) ConfirmPutaway(ctx context.Context, cmd domain.PutawayCommand) (*domain.StockMovement, error) {
	if cmd.ProductID == uuid.Nil || cmd.BatchID == uuid.Nil || cmd.StagingLocationID == uuid.Nil || cmd.DestLocationID == uuid.Nil || cmd.TenantID == uuid.Nil || cmd.WarehouseID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	if cmd.UserID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}
	if !cmd.Quantity.IsPositive() {
		return nil, domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, cmd.TenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: set tenant: %w", err)
	}

	// Validate dest location
	var destWhID sql.NullString
	var destType string
	err = tx.QueryRowContext(ctx, `SELECT warehouse_id, type FROM warehouse_locations WHERE id = $1 AND tenant_id = $2`, cmd.DestLocationID, cmd.TenantID).Scan(&destWhID, &destType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvalidPutawayLocation
		}
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: query dest location: %w", err)
	}
	parsedWh := nullUUIDToPtr(destWhID)
	if parsedWh == nil || *parsedWh != cmd.WarehouseID || domain.LocationType(destType) != domain.LocationTypeInternal {
		return nil, domain.ErrInvalidPutawayLocation
	}

	// Validate default rack override (trimmed reason >= 5 characters)
	if cmd.DefaultLocationID != nil && *cmd.DefaultLocationID != cmd.DestLocationID {
		if cmd.Reason == nil || len([]rune(strings.TrimSpace(*cmd.Reason))) < 5 {
			return nil, domain.ErrPutawayReasonRequired
		}
	}

	// Lock staging location stock
	if err := r.LockLocationStock(ctx, tx, cmd.TenantID, cmd.StagingLocationID, cmd.ProductID); err != nil {
		return nil, err
	}

	// Check staged batch balance
	checkSQL := `
SELECT COALESCE(
    SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE -quantity END),
    0
)
FROM stock_movements
WHERE tenant_id = $1
  AND batch_id = $3
  AND product_id = $4
  AND status = 'DONE'
  AND (source_location_id = $2 OR dest_location_id = $2)`

	var stagedBalance decimal.Decimal
	err = tx.QueryRowContext(ctx, checkSQL, cmd.TenantID, cmd.StagingLocationID, cmd.BatchID, cmd.ProductID).Scan(&stagedBalance)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: scan staged balance: %w", err)
	}
	if stagedBalance.LessThan(cmd.Quantity) {
		return nil, domain.ErrInsufficientStock
	}

	// Insert stock_movement
	now := time.Now().UTC()
	mov := domain.StockMovement{
		ID:               uuid.New(),
		TenantID:         cmd.TenantID,
		MovementNumber:   fmt.Sprintf("PUT-%d", time.Now().UnixNano()/1e6),
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

	_, err = tx.ExecContext(ctx, createStockMovementSQL,
		mov.ID, mov.TenantID, mov.MovementNumber, mov.ProductID,
		mov.SourceLocationID, mov.DestLocationID, mov.Quantity,
		mov.UnitCost, mov.Status, mov.ReferenceType, mov.ReferenceID,
		ptrToNullUUID(mov.ExecutedBy), ptrToNullUUID(mov.BatchID), mov.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: exec movement: %w", err)
	}

	// Audit log entry
	auditDetails, err := json.Marshal(map[string]any{
		"product_id":          cmd.ProductID,
		"batch_id":            cmd.BatchID,
		"staging_location_id": cmd.StagingLocationID,
		"dest_location_id":    cmd.DestLocationID,
		"quantity":            cmd.Quantity,
		"reason":              cmd.Reason,
		"movement_id":         mov.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: marshal audit details: %w", err)
	}

	if err := r.WriteAuditTx(ctx, tx, cmd.TenantID, &cmd.UserID, "putaway", mov.ID, "confirmed", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: audit log: %w", err)
	}
	if err := r.WriteAuditTx(ctx, tx, cmd.TenantID, &cmd.UserID, "stock_movement", mov.ID, "created", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: movement audit log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ConfirmPutaway: commit: %w", err)
	}
	return &mov, nil
}

// -----------------------------------------------------------------------------
// 6. Release Stock Receipt
// -----------------------------------------------------------------------------

func (r *WMSRepo) ReleaseStockReceipt(ctx context.Context, tenantID, receiptID, userID uuid.UUID) (*domain.StockReceipt, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleaseStockReceipt: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleaseStockReceipt: set tenant: %w", err)
	}

	rc, err := getStockReceiptTx(ctx, tx, tenantID, receiptID, true)
	if err != nil {
		return nil, err
	}

	var batchCount int
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stock_batches WHERE source_receipt_id = $1 AND tenant_id = $2 AND status = 'ON_HOLD'`,
		receiptID, tenantID).Scan(&batchCount)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleaseStockReceipt: count on-hold batches: %w", err)
	}
	if batchCount == 0 {
		return nil, domain.ErrReceiptNotOnHold
	}

	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx,
		`UPDATE stock_batches SET status = 'RELEASED' WHERE source_receipt_id = $1 AND tenant_id = $2 AND status = 'ON_HOLD' AND (expiry_date IS NULL OR expiry_date >= CURRENT_DATE)`,
		receiptID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleaseStockReceipt: update batches: %w", err)
	}
	releasedRows, _ := res.RowsAffected()
	if releasedRows == 0 {
		return nil, errors.New("tidak ada batch yang dapat dirilis (seluruh batch on-hold sudah kedaluwarsa)")
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE stock_receipts SET released_by = $3, released_at = $4, updated_at = $4 WHERE id = $1 AND tenant_id = $2`,
		receiptID, tenantID, userID, now)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleaseStockReceipt: update receipt: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]any{
		"released_batch_count": releasedRows,
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "stock_receipt", receiptID, "released", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleaseStockReceipt: audit log: %w", err)
	}

	rc.ReleasedBy = &userID
	rc.ReleasedAt = &now
	rc.UpdatedAt = now
	rc.OnHoldBatchCount = 0

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ReleaseStockReceipt: commit: %w", err)
	}
	return rc, nil
}

// -----------------------------------------------------------------------------
// 7. WMS Settings
// -----------------------------------------------------------------------------

func (r *WMSRepo) GetWMSSettings(ctx context.Context, tenantID uuid.UUID) (*domain.WMSSettings, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSSettings: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSSettings: set tenant: %w", err)
	}

	query := `SELECT tenant_id, require_release_approval, updated_by, updated_at FROM wms_settings WHERE tenant_id = $1`
	var s domain.WMSSettings
	var updatedBy sql.NullString
	err = tx.QueryRowContext(ctx, query, tenantID).Scan(&s.TenantID, &s.RequireReleaseApproval, &updatedBy, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if errCommit := tx.Commit(); errCommit != nil {
				return nil, fmt.Errorf("WMSRepo.GetWMSSettings: commit: %w", errCommit)
			}
			return &domain.WMSSettings{
				TenantID:               tenantID,
				RequireReleaseApproval: false,
				UpdatedAt:              time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("WMSRepo.GetWMSSettings: query: %w", err)
	}

	s.UpdatedBy = nullUUIDToPtr(updatedBy)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetWMSSettings: commit: %w", err)
	}
	return &s, nil
}

func (r *WMSRepo) UpsertWMSSettings(ctx context.Context, s *domain.WMSSettings) error {
	if s == nil || s.TenantID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpsertWMSSettings: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, s.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpsertWMSSettings: set tenant: %w", err)
	}

	now := time.Now().UTC()
	s.UpdatedAt = now

	query := `
INSERT INTO wms_settings (tenant_id, require_release_approval, updated_by, updated_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id) DO UPDATE SET
    require_release_approval = EXCLUDED.require_release_approval,
    updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at`

	_, err = tx.ExecContext(ctx, query, s.TenantID, s.RequireReleaseApproval, ptrToNullUUID(s.UpdatedBy), s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpsertWMSSettings: exec: %w", err)
	}

	return tx.Commit()
}

// -----------------------------------------------------------------------------
// 8. Product Default Locations
// -----------------------------------------------------------------------------

func (r *WMSRepo) ListDefaultLocations(ctx context.Context, tenantID uuid.UUID, productID *uuid.UUID) ([]domain.ProductDefaultLocation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDefaultLocations: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDefaultLocations: set tenant: %w", err)
	}

	query := `
SELECT pdl.tenant_id, pdl.product_id, pdl.warehouse_id, pdl.location_id,
       COALESCE(loc.code, ''), pdl.updated_by, pdl.updated_at
FROM product_default_locations pdl
LEFT JOIN warehouse_locations loc ON loc.id = pdl.location_id AND loc.tenant_id = pdl.tenant_id
WHERE pdl.tenant_id = $1
  AND ($2::uuid IS NULL OR pdl.product_id = $2)
ORDER BY pdl.updated_at DESC`

	rows, err := tx.QueryContext(ctx, query, tenantID, ptrToNullUUID(productID))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDefaultLocations: query: %w", err)
	}
	defer rows.Close()

	var result []domain.ProductDefaultLocation
	for rows.Next() {
		var d domain.ProductDefaultLocation
		var updatedBy sql.NullString
		if err := rows.Scan(
			&d.TenantID, &d.ProductID, &d.WarehouseID, &d.LocationID,
			&d.LocationCode, &updatedBy, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListDefaultLocations: scan: %w", err)
		}
		d.UpdatedBy = nullUUIDToPtr(updatedBy)
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDefaultLocations: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListDefaultLocations: commit: %w", err)
	}
	return result, nil
}

func (r *WMSRepo) SetDefaultLocation(ctx context.Context, d *domain.ProductDefaultLocation) error {
	if d == nil || d.TenantID == uuid.Nil || d.ProductID == uuid.Nil || d.WarehouseID == uuid.Nil || d.LocationID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.SetDefaultLocation: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, d.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.SetDefaultLocation: set tenant: %w", err)
	}

	var locWhID sql.NullString
	var locType string
	err = tx.QueryRowContext(ctx, `SELECT warehouse_id, type FROM warehouse_locations WHERE id = $1 AND tenant_id = $2`, d.LocationID, d.TenantID).Scan(&locWhID, &locType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrLocationNotFound
		}
		return fmt.Errorf("WMSRepo.SetDefaultLocation: validate location: %w", err)
	}
	parsedWh := nullUUIDToPtr(locWhID)
	if parsedWh == nil || *parsedWh != d.WarehouseID || domain.LocationType(locType) != domain.LocationTypeInternal {
		return domain.ErrInvalidPutawayLocation
	}

	now := time.Now().UTC()
	d.UpdatedAt = now

	query := `
INSERT INTO product_default_locations (tenant_id, product_id, warehouse_id, location_id, updated_by, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, product_id, warehouse_id) DO UPDATE SET
    location_id = EXCLUDED.location_id,
    updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at`

	_, err = tx.ExecContext(ctx, query, d.TenantID, d.ProductID, d.WarehouseID, d.LocationID, ptrToNullUUID(d.UpdatedBy), d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("WMSRepo.SetDefaultLocation: exec: %w", err)
	}

	return tx.Commit()
}

func (r *WMSRepo) DeleteDefaultLocation(ctx context.Context, tenantID, productID, warehouseID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.DeleteDefaultLocation: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("WMSRepo.DeleteDefaultLocation: set tenant: %w", err)
	}

	query := `DELETE FROM product_default_locations WHERE tenant_id = $1 AND product_id = $2 AND warehouse_id = $3`
	_, err = tx.ExecContext(ctx, query, tenantID, productID, warehouseID)
	if err != nil {
		return fmt.Errorf("WMSRepo.DeleteDefaultLocation: exec: %w", err)
	}

	return tx.Commit()
}

// -----------------------------------------------------------------------------
// 9. Traceability (TraceBatch & TraceDocument)
// -----------------------------------------------------------------------------

func (r *WMSRepo) TraceBatch(ctx context.Context, tenantID, batchID uuid.UUID) (*domain.BatchTrace, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceBatch: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceBatch: set tenant: %w", err)
	}

	// Load batch
	batchRow := tx.QueryRowContext(ctx, `SELECT `+stockBatchColumns+` FROM stock_batches WHERE tenant_id = $1 AND id = $2`, tenantID, batchID)
	batch, err := scanStockBatch(batchRow)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrStockBatchNotFound
		}
		return nil, fmt.Errorf("WMSRepo.TraceBatch: scan batch: %w", err)
	}

	// Load movements
	movQuery := `
SELECT 
    sm.id, sm.movement_number, sm.product_id,
    COALESCE(p.name, ''), COALESCE(p.sku, ''),
    b.id, b.batch_number, b.expiry_date,
    COALESCE(sl.code, ''), COALESCE(sl.type, 'INTERNAL'),
    COALESCE(dl.code, ''), COALESCE(dl.type, 'INTERNAL'),
    sm.quantity, sm.reference_type, sm.reference_id,
    COALESCE(u.full_name, u.email, ''), sm.created_at
FROM stock_movements sm
JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
LEFT JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
LEFT JOIN warehouse_locations sl ON sl.id = sm.source_location_id AND sl.tenant_id = sm.tenant_id
LEFT JOIN warehouse_locations dl ON dl.id = sm.dest_location_id AND dl.tenant_id = sm.tenant_id
LEFT JOIN users u ON u.id = sm.executed_by
WHERE sm.tenant_id = $1 AND sm.batch_id = $2
ORDER BY sm.created_at ASC, sm.id ASC`

	movRows, err := tx.QueryContext(ctx, movQuery, tenantID, batchID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceBatch: query movements: %w", err)
	}
	defer movRows.Close()

	var traceMovs []domain.TraceMovement
	totalIn := decimal.Zero
	totalOut := decimal.Zero

	isInternalOrStaging := func(t domain.LocationType) bool {
		return t == domain.LocationTypeInternal ||
			t == domain.LocationTypeStagingInbound ||
			t == domain.LocationTypeStagingOutbound
	}
	isSystemSource := func(code string, t domain.LocationType) bool {
		return strings.HasPrefix(code, "@") ||
			t == domain.LocationTypeVendor ||
			t == domain.LocationTypeProduction ||
			t == domain.LocationTypeTransit
	}
	isSystemOutDest := func(code string, t domain.LocationType) bool {
		return strings.HasPrefix(code, "@") ||
			t == domain.LocationTypeCustomer ||
			t == domain.LocationTypeScrap ||
			t == domain.LocationTypeLoss
	}

	for movRows.Next() {
		var tm domain.TraceMovement
		var exp sql.NullTime
		var srcType, dstType string
		if err := movRows.Scan(
			&tm.MovementID, &tm.MovementNumber, &tm.ProductID,
			&tm.ProductName, &tm.ProductSKU,
			&tm.BatchID, &tm.BatchNumber, &exp,
			&tm.SourceCode, &srcType,
			&tm.DestCode, &dstType,
			&tm.Quantity, &tm.ReferenceType, &tm.ReferenceID,
			&tm.ExecutedByName, &tm.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.TraceBatch: scan movement: %w", err)
		}
		tm.ExpiryDate = nullTimeToPtr(exp)
		tm.SourceType = domain.LocationType(srcType)
		tm.DestType = domain.LocationType(dstType)

		if isInternalOrStaging(tm.DestType) && isSystemSource(tm.SourceCode, tm.SourceType) {
			totalIn = totalIn.Add(tm.Quantity)
		}
		if isInternalOrStaging(tm.SourceType) && isSystemOutDest(tm.DestCode, tm.DestType) {
			totalOut = totalOut.Add(tm.Quantity)
		}

		traceMovs = append(traceMovs, tm)
	}
	if err := movRows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceBatch: movement rows err: %w", err)
	}

	// Load balances
	balQuery := `
SELECT b.id, b.batch_number, b.expiry_date, b.status, loc.id, loc.code, sm.product_id,
       SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty,
       b.created_at
FROM stock_movements sm
JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
WHERE sm.tenant_id = $1 AND sm.batch_id = $2 AND sm.status = 'DONE'
GROUP BY b.id, b.batch_number, b.expiry_date, b.status, b.created_at, loc.id, loc.code, sm.product_id
HAVING SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) > 0
ORDER BY loc.code ASC`

	balRows, err := tx.QueryContext(ctx, balQuery, tenantID, batchID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceBatch: query balances: %w", err)
	}
	defer balRows.Close()

	var balances []domain.BatchBalance
	onHand := decimal.Zero
	for balRows.Next() {
		var bal domain.BatchBalance
		var exp sql.NullTime
		if err := balRows.Scan(
			&bal.BatchID, &bal.BatchNumber, &exp, &bal.Status,
			&bal.LocationID, &bal.LocationCode, &bal.ProductID,
			&bal.Quantity, &bal.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.TraceBatch: scan balance: %w", err)
		}
		bal.ExpiryDate = nullTimeToPtr(exp)
		onHand = onHand.Add(bal.Quantity)
		balances = append(balances, bal)
	}
	if err := balRows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceBatch: balance rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceBatch: commit: %w", err)
	}

	return &domain.BatchTrace{
		Batch:     *batch,
		Movements: traceMovs,
		Balances:  balances,
		TotalIn:   totalIn,
		TotalOut:  totalOut,
		OnHand:    onHand,
	}, nil
}

func (r *WMSRepo) TraceDocument(ctx context.Context, tenantID uuid.UUID, refType string, refID uuid.UUID) (*domain.DocumentTrace, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: set tenant: %w", err)
	}

	// Resolve document number
	docNum := refID.String()
	switch refType {
	case domain.StockRefGoodsReceipt, "RECEIPT":
		var num string
		if err := tx.QueryRowContext(ctx, `SELECT receipt_number FROM stock_receipts WHERE id = $1 AND tenant_id = $2`, refID, tenantID).Scan(&num); err == nil {
			docNum = num
		}
	case domain.StockRefDeliveryOrder:
		var num string
		if err := tx.QueryRowContext(ctx, `SELECT do_number FROM delivery_orders WHERE id = $1 AND tenant_id = $2`, refID, tenantID).Scan(&num); err == nil {
			docNum = num
		}
	case domain.StockRefTransfer:
		var num string
		if err := tx.QueryRowContext(ctx, `SELECT transfer_number FROM stock_transfers WHERE id = $1 AND tenant_id = $2`, refID, tenantID).Scan(&num); err == nil {
			docNum = num
		}
	}

	// Find distinct batch IDs touched by this reference
	batchRows, err := tx.QueryContext(ctx,
		`SELECT DISTINCT batch_id FROM stock_movements WHERE tenant_id = $1 AND reference_type = $2 AND reference_id = $3 AND batch_id IS NOT NULL`,
		tenantID, refType, refID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: query batch IDs: %w", err)
	}
	defer batchRows.Close()

	var batchIDs []string
	for batchRows.Next() {
		var bid uuid.UUID
		if err := batchRows.Scan(&bid); err != nil {
			return nil, fmt.Errorf("WMSRepo.TraceDocument: scan batch ID: %w", err)
		}
		batchIDs = append(batchIDs, bid.String())
	}
	if err := batchRows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: batch rows err: %w", err)
	}

	if len(batchIDs) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("WMSRepo.TraceDocument: commit: %w", err)
		}
		return &domain.DocumentTrace{
			DocumentType:   refType,
			DocumentID:     refID,
			DocumentNumber: docNum,
			Batches:        []domain.StockBatch{},
			Movements:      []domain.TraceMovement{},
		}, nil
	}

	// Load all batches
	bQuery := `SELECT ` + stockBatchColumns + ` FROM stock_batches WHERE tenant_id = $1 AND id = ANY($2::uuid[]) ORDER BY created_at ASC`
	bRows, err := tx.QueryContext(ctx, bQuery, tenantID, pq.Array(batchIDs))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: query batches: %w", err)
	}
	defer bRows.Close()

	var batches []domain.StockBatch
	for bRows.Next() {
		b, err := scanStockBatch(bRows)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.TraceDocument: scan batch: %w", err)
		}
		batches = append(batches, *b)
	}
	if err := bRows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: batches rows err: %w", err)
	}

	// Load all movements for those batches
	movQuery := `
SELECT 
    sm.id, sm.movement_number, sm.product_id,
    COALESCE(p.name, ''), COALESCE(p.sku, ''),
    b.id, b.batch_number, b.expiry_date,
    COALESCE(sl.code, ''), COALESCE(sl.type, 'INTERNAL'),
    COALESCE(dl.code, ''), COALESCE(dl.type, 'INTERNAL'),
    sm.quantity, sm.reference_type, sm.reference_id,
    COALESCE(u.full_name, u.email, ''), sm.created_at
FROM stock_movements sm
JOIN stock_batches b ON b.id = sm.batch_id AND b.tenant_id = sm.tenant_id
LEFT JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
LEFT JOIN warehouse_locations sl ON sl.id = sm.source_location_id AND sl.tenant_id = sm.tenant_id
LEFT JOIN warehouse_locations dl ON dl.id = sm.dest_location_id AND dl.tenant_id = sm.tenant_id
LEFT JOIN users u ON u.id = sm.executed_by
WHERE sm.tenant_id = $1 AND sm.batch_id = ANY($2::uuid[])
ORDER BY sm.created_at ASC, sm.id ASC`

	movRows, err := tx.QueryContext(ctx, movQuery, tenantID, pq.Array(batchIDs))
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: query movements: %w", err)
	}
	defer movRows.Close()

	var traceMovs []domain.TraceMovement
	for movRows.Next() {
		var tm domain.TraceMovement
		var exp sql.NullTime
		var srcType, dstType string
		if err := movRows.Scan(
			&tm.MovementID, &tm.MovementNumber, &tm.ProductID,
			&tm.ProductName, &tm.ProductSKU,
			&tm.BatchID, &tm.BatchNumber, &exp,
			&tm.SourceCode, &srcType,
			&tm.DestCode, &dstType,
			&tm.Quantity, &tm.ReferenceType, &tm.ReferenceID,
			&tm.ExecutedByName, &tm.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.TraceDocument: scan movement: %w", err)
		}
		tm.ExpiryDate = nullTimeToPtr(exp)
		tm.SourceType = domain.LocationType(srcType)
		tm.DestType = domain.LocationType(dstType)
		traceMovs = append(traceMovs, tm)
	}
	if err := movRows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: movements rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.TraceDocument: commit: %w", err)
	}

	return &domain.DocumentTrace{
		DocumentType:   refType,
		DocumentID:     refID,
		DocumentNumber: docNum,
		Batches:        batches,
		Movements:      traceMovs,
	}, nil
}

// -----------------------------------------------------------------------------
// 10. Audit Trail & Hash Chain
// -----------------------------------------------------------------------------

func (r *WMSRepo) ListAuditTrail(ctx context.Context, tenantID uuid.UUID, entityType string, entityID uuid.UUID) ([]domain.AuditTrailEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAuditTrail: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAuditTrail: set tenant: %w", err)
	}

	query := `
SELECT a.id, a.entity_type, a.entity_id, a.action, a.user_id,
       COALESCE(u.full_name, u.email, 'Sistem'), a.details, a.created_at
FROM audit_logs a
LEFT JOIN users u ON u.id = a.user_id
WHERE a.tenant_id = $1 AND a.entity_type = $2 AND a.entity_id = $3
ORDER BY a.created_at ASC, a.id ASC`

	rows, err := tx.QueryContext(ctx, query, tenantID, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAuditTrail: query: %w", err)
	}
	defer rows.Close()

	var result []domain.AuditTrailEntry
	for rows.Next() {
		var entry domain.AuditTrailEntry
		var userID sql.NullString
		var details []byte
		if err := rows.Scan(
			&entry.ID, &entry.EntityType, &entry.EntityID, &entry.Action, &userID,
			&entry.UserName, &details, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("WMSRepo.ListAuditTrail: scan: %w", err)
		}
		entry.UserID = nullUUIDToPtr(userID)
		if len(details) > 0 {
			entry.Details = json.RawMessage(details)
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAuditTrail: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListAuditTrail: commit: %w", err)
	}
	return result, nil
}

// WriteAuditTx appends an entry to audit_logs maintaining the SHA-256 hash chain per tenant.
func (r *WMSRepo) WriteAuditTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, userID *uuid.UUID, entityType string, entityID uuid.UUID, action string, details json.RawMessage) error {
	var prevHash sql.NullString
	queryLock := `SELECT current_hash FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE`
	err := tx.QueryRowContext(ctx, queryLock, tenantID).Scan(&prevHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("WriteAuditTx: query prev_hash: %w", err)
	}

	pHash := ""
	if prevHash.Valid {
		pHash = prevHash.String
	}

	now := time.Now().UTC()
	// current_hash = sha256(prev_hash + tenantID + entityType + entityID + action + details + timestamp)
	h := sha256.New()
	h.Write([]byte(pHash))
	h.Write([]byte(tenantID.String()))
	h.Write([]byte(entityType))
	h.Write([]byte(entityID.String()))
	h.Write([]byte(action))
	h.Write(details)
	h.Write([]byte(now.Format(time.RFC3339Nano)))
	currentHash := hex.EncodeToString(h.Sum(nil))

	insertSQL := `
INSERT INTO audit_logs (id, tenant_id, user_id, entity_type, entity_id, action, details, prev_hash, current_hash, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	var detailsArg any
	if len(details) > 0 {
		detailsArg = details
	}

	_, err = tx.ExecContext(ctx, insertSQL,
		uuid.New(), tenantID, ptrToNullUUID(userID), entityType, entityID, action,
		detailsArg, pHash, currentHash, now)
	if err != nil {
		return fmt.Errorf("WriteAuditTx: insert audit log: %w", err)
	}
	return nil
}
