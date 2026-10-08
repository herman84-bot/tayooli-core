package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// -----------------------------------------------------------------------------
// Stock Receipts (Barang Masuk / inbound goods receipt)
// Stored in stock_receipts / stock_receipt_items (migration 028). Stock itself is
// only ever written to the stock_movements ledger, inside the same transaction
// that changes the receipt status.
// -----------------------------------------------------------------------------

const createStockReceiptSQL = `
INSERT INTO stock_receipts (
	id, tenant_id, receipt_number, receipt_type, warehouse_id, dest_location_id,
	from_name, from_warehouse_id, source_ref, transfer_id, supplier_name, supplier_ref,
	notes, status, created_by, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

const createStockReceiptItemSQL = `
INSERT INTO stock_receipt_items (id, tenant_id, receipt_id, product_id, expected_qty, accepted_qty, rejected_qty, reject_reason, batch_number, expiry_date, batch_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

const stockReceiptColumns = `
r.id, r.tenant_id, r.receipt_number, r.receipt_type, r.warehouse_id, r.dest_location_id,
COALESCE(r.from_name, r.supplier_name, ''), r.from_warehouse_id, fw.name, r.source_ref, r.transfer_id,
COALESCE(r.supplier_name, r.from_name, ''), r.supplier_ref, r.notes,
r.status, r.created_by, r.created_at, r.updated_at, r.posted_by, r.posted_at, r.cancelled_by, r.cancelled_at, r.cancel_reason,
r.released_by, r.released_at,
COALESCE(u_cr.full_name, u_cr.email, ''),
COALESCE(u_po.full_name, u_po.email, ''),
COALESCE(u_ca.full_name, u_ca.email, ''),
COALESCE(u_re.full_name, u_re.email, ''),
(SELECT COUNT(*) FROM stock_batches b WHERE b.source_receipt_id = r.id AND b.tenant_id = r.tenant_id AND b.status = 'ON_HOLD'),
(SELECT COUNT(*) FROM stock_receipt_items i WHERE i.receipt_id = r.id AND i.tenant_id = r.tenant_id),
(SELECT COALESCE(SUM(i.accepted_qty), 0) FROM stock_receipt_items i WHERE i.receipt_id = r.id AND i.tenant_id = r.tenant_id),
(SELECT COALESCE(SUM(i.rejected_qty), 0) FROM stock_receipt_items i WHERE i.receipt_id = r.id AND i.tenant_id = r.tenant_id)`

const getStockReceiptItemsSQL = `
SELECT i.id, i.tenant_id, i.receipt_id, i.product_id, COALESCE(p.name, ''), COALESCE(p.sku, ''),
       i.expected_qty, i.accepted_qty, i.rejected_qty, i.reject_reason,
       i.batch_number, i.expiry_date, i.batch_id, i.created_at
FROM stock_receipt_items i
LEFT JOIN products p ON p.id = i.product_id AND p.tenant_id = i.tenant_id
WHERE i.receipt_id = $1 AND i.tenant_id = $2
ORDER BY i.created_at, i.id`

func scanStockReceipt(s rowScanner) (*domain.StockReceipt, error) {
	var rc domain.StockReceipt
	var fromWhID, transferID, postedBy, cancelledBy, releasedBy sql.NullString
	var fromWhName, sourceRef, supName, supRef, notes, cancelReason sql.NullString
	var postedAt, cancelledAt, releasedAt sql.NullTime
	var crName, poName, caName, reName string
	var onHoldCount int
	if err := s.Scan(
		&rc.ID, &rc.TenantID, &rc.ReceiptNumber, &rc.ReceiptType, &rc.WarehouseID, &rc.DestLocationID,
		&rc.FromName, &fromWhID, &fromWhName, &sourceRef, &transferID,
		&supName, &supRef, &notes,
		&rc.Status, &rc.CreatedBy, &rc.CreatedAt, &rc.UpdatedAt, &postedBy, &postedAt, &cancelledBy, &cancelledAt, &cancelReason,
		&releasedBy, &releasedAt,
		&crName, &poName, &caName, &reName,
		&onHoldCount,
		&rc.ItemCount, &rc.TotalAcceptedQty, &rc.TotalRejectedQty,
	); err != nil {
		return nil, err
	}
	rc.FromWarehouseID = nullUUIDToPtr(fromWhID)
	rc.FromWarehouseName = nullStringToPtr(fromWhName)
	rc.SourceRef = nullStringToPtr(sourceRef)
	rc.TransferID = nullUUIDToPtr(transferID)
	if supName.Valid {
		rc.SupplierName = supName.String
	} else {
		rc.SupplierName = rc.FromName
	}
	rc.SupplierRef = nullStringToPtr(supRef)
	rc.Notes = nullStringToPtr(notes)
	rc.PostedBy = nullUUIDToPtr(postedBy)
	rc.PostedAt = nullTimeToPtr(postedAt)
	rc.CancelledBy = nullUUIDToPtr(cancelledBy)
	rc.CancelledAt = nullTimeToPtr(cancelledAt)
	rc.CancelReason = nullStringToPtr(cancelReason)
	rc.ReleasedBy = nullUUIDToPtr(releasedBy)
	rc.ReleasedAt = nullTimeToPtr(releasedAt)
	rc.CreatedByName = crName
	rc.PostedByName = poName
	rc.CancelledByName = caName
	rc.ReleasedByName = reName
	rc.OnHoldBatchCount = onHoldCount
	return &rc, nil
}

func getStockReceiptTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID, forUpdate bool) (*domain.StockReceipt, error) {
	q := `SELECT ` + stockReceiptColumns + `
FROM stock_receipts r
LEFT JOIN warehouses fw ON fw.id = r.from_warehouse_id AND fw.tenant_id = r.tenant_id
LEFT JOIN users u_cr ON u_cr.id = r.created_by
LEFT JOIN users u_po ON u_po.id = r.posted_by
LEFT JOIN users u_ca ON u_ca.id = r.cancelled_by
LEFT JOIN users u_re ON u_re.id = r.released_by
WHERE r.id = $1 AND r.tenant_id = $2`
	if forUpdate {
		q += ` FOR UPDATE OF r`
	}
	rc, err := scanStockReceipt(tx.QueryRowContext(ctx, q, id, tenantID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrStockReceiptNotFound
		}
		return nil, err
	}
	return rc, nil
}

func getStockReceiptItemsTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID) ([]domain.StockReceiptItem, error) {
	rows, err := tx.QueryContext(ctx, getStockReceiptItemsSQL, id, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.StockReceiptItem{}
	for rows.Next() {
		var it domain.StockReceiptItem
		var expected decimal.NullDecimal
		var reason sql.NullString
		var batchNum sql.NullString
		var exp sql.NullTime
		var batchID sql.NullString
		if err := rows.Scan(&it.ID, &it.TenantID, &it.ReceiptID, &it.ProductID, &it.ProductName, &it.ProductSKU,
			&expected, &it.AcceptedQty, &it.RejectedQty, &reason,
			&batchNum, &exp, &batchID, &it.CreatedAt); err != nil {
			return nil, err
		}
		it.ExpectedQty = nullDecimalToPtr(expected)
		it.RejectReason = nullStringToPtr(reason)
		it.BatchNumber = nullStringToPtr(batchNum)
		it.ExpiryDate = nullTimeToPtr(exp)
		it.BatchID = nullUUIDToPtr(batchID)
		items = append(items, it)
	}
	return items, rows.Err()
}

func insertStockReceiptItemsTx(ctx context.Context, tx *sql.Tx, rc *domain.StockReceipt, items []domain.StockReceiptItem, now time.Time) error {
	for i := range items {
		it := &items[i]
		if it.ID == uuid.Nil {
			it.ID = uuid.New()
		}
		it.TenantID = rc.TenantID
		it.ReceiptID = rc.ID
		if it.CreatedAt.IsZero() {
			it.CreatedAt = now
		}
		if _, err := tx.ExecContext(ctx, createStockReceiptItemSQL,
			it.ID, it.TenantID, it.ReceiptID, it.ProductID, ptrToNullDecimal(it.ExpectedQty),
			it.AcceptedQty, it.RejectedQty, ptrToNullString(it.RejectReason),
			ptrToNullString(it.BatchNumber), ptrToNullTime(it.ExpiryDate), ptrToNullUUID(it.BatchID),
			it.CreatedAt); err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23503" {
				// Unknown product (or one belonging to another tenant): user error, not a 500.
				return &domain.StockReceiptValidationError{Msg: fmt.Sprintf("Baris %d: produk tidak ditemukan", i+1)}
			}
			return fmt.Errorf("item %d: %w", i, err)
		}
	}
	return nil
}

func (r *WMSRepo) CreateStockReceipt(ctx context.Context, rc *domain.StockReceipt, items []domain.StockReceiptItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.CreateStockReceipt: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, rc.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.CreateStockReceipt: set tenant: %w", err)
	}

	if rc.ID == uuid.Nil {
		rc.ID = uuid.New()
	}
	now := time.Now().UTC()
	if rc.CreatedAt.IsZero() {
		rc.CreatedAt = now
	}
	rc.UpdatedAt = rc.CreatedAt
	if rc.Status == "" {
		rc.Status = domain.StockReceiptStatusDraft
	}
	if rc.ReceiptType == "" {
		rc.ReceiptType = domain.StockReceiptTypeProduction
	}
	if rc.FromName == "" {
		if rc.SupplierName != "" {
			rc.FromName = rc.SupplierName
		} else if rc.ReceiptType == domain.StockReceiptTypeProduction {
			rc.FromName = "Hasil Produksi"
		}
	}
	if rc.SupplierName == "" {
		rc.SupplierName = rc.FromName
	}
	if rc.SourceRef == nil && rc.SupplierRef != nil {
		rc.SourceRef = rc.SupplierRef
	}

	if _, err := tx.ExecContext(ctx, createStockReceiptSQL,
		rc.ID, rc.TenantID, rc.ReceiptNumber, rc.ReceiptType, rc.WarehouseID, rc.DestLocationID,
		rc.FromName, ptrToNullUUID(rc.FromWarehouseID), ptrToNullString(rc.SourceRef), ptrToNullUUID(rc.TransferID),
		rc.SupplierName, ptrToNullString(rc.SupplierRef),
		ptrToNullString(rc.Notes), rc.Status, rc.CreatedBy,
		rc.CreatedAt, rc.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("WMSRepo.CreateStockReceipt: duplicate receipt number: %w", domain.ErrConflict)
		}
		return fmt.Errorf("WMSRepo.CreateStockReceipt: exec header: %w", err)
	}
	if err := insertStockReceiptItemsTx(ctx, tx, rc, items, now); err != nil {
		return fmt.Errorf("WMSRepo.CreateStockReceipt: exec %w", err)
	}

	return tx.Commit()
}

func (r *WMSRepo) UpdateDraftStockReceipt(ctx context.Context, rc *domain.StockReceipt, items []domain.StockReceiptItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateDraftStockReceipt: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, rc.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdateDraftStockReceipt: set tenant: %w", err)
	}

	current, err := getStockReceiptTx(ctx, tx, rc.TenantID, rc.ID, true)
	if err != nil {
		return fmt.Errorf("WMSRepo.UpdateDraftStockReceipt: lock header: %w", err)
	}
	if current.Status != domain.StockReceiptStatusDraft {
		return domain.ErrStockReceiptNotDraft
	}

	now := time.Now().UTC()
	if rc.ReceiptType == "" {
		rc.ReceiptType = domain.StockReceiptTypeProduction
	}
	if rc.FromName == "" {
		if rc.SupplierName != "" {
			rc.FromName = rc.SupplierName
		} else if rc.ReceiptType == domain.StockReceiptTypeProduction {
			rc.FromName = "Hasil Produksi"
		}
	}
	if rc.SupplierName == "" {
		rc.SupplierName = rc.FromName
	}
	if rc.SourceRef == nil && rc.SupplierRef != nil {
		rc.SourceRef = rc.SupplierRef
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE stock_receipts
SET receipt_type = $3, warehouse_id = $4, dest_location_id = $5, from_name = $6,
    from_warehouse_id = $7, source_ref = $8, transfer_id = $9, supplier_name = $10,
    supplier_ref = $11, notes = $12, updated_at = $13
WHERE id = $1 AND tenant_id = $2 AND status = 'DRAFT'`,
		rc.ID, rc.TenantID, rc.ReceiptType, rc.WarehouseID, rc.DestLocationID, rc.FromName,
		ptrToNullUUID(rc.FromWarehouseID), ptrToNullString(rc.SourceRef), ptrToNullUUID(rc.TransferID),
		rc.SupplierName, ptrToNullString(rc.SupplierRef), ptrToNullString(rc.Notes), now); err != nil {
		return fmt.Errorf("WMSRepo.UpdateDraftStockReceipt: update header: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM stock_receipt_items WHERE receipt_id = $1 AND tenant_id = $2`, rc.ID, rc.TenantID); err != nil {
		return fmt.Errorf("WMSRepo.UpdateDraftStockReceipt: delete items: %w", err)
	}
	for i := range items {
		items[i].ID = uuid.Nil
		items[i].CreatedAt = time.Time{}
	}
	if err := insertStockReceiptItemsTx(ctx, tx, rc, items, now); err != nil {
		return fmt.Errorf("WMSRepo.UpdateDraftStockReceipt: exec %w", err)
	}

	return tx.Commit()
}

func (r *WMSRepo) GetStockReceiptByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetStockReceiptByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetStockReceiptByID: set tenant: %w", err)
	}

	rc, err := getStockReceiptTx(ctx, tx, tenantID, id, false)
	if err != nil {
		if errors.Is(err, domain.ErrStockReceiptNotFound) {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("WMSRepo.GetStockReceiptByID: scan header: %w", err)
	}
	items, err := getStockReceiptItemsTx(ctx, tx, tenantID, id)
	if err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetStockReceiptByID: items: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("WMSRepo.GetStockReceiptByID: commit: %w", err)
	}
	return rc, items, nil
}

func (r *WMSRepo) ListStockReceipts(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.StockReceiptStatus, receiptType *domain.StockReceiptType) ([]domain.StockReceipt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockReceipts: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockReceipts: set tenant: %w", err)
	}

	var statusArg, typeArg sql.NullString
	if status != nil {
		statusArg = sql.NullString{String: string(*status), Valid: true}
	}
	if receiptType != nil {
		typeArg = sql.NullString{String: string(*receiptType), Valid: true}
	}
	query := `SELECT ` + stockReceiptColumns + `
FROM stock_receipts r
LEFT JOIN warehouses fw ON fw.id = r.from_warehouse_id AND fw.tenant_id = r.tenant_id
LEFT JOIN users u_cr ON u_cr.id = r.created_by
LEFT JOIN users u_po ON u_po.id = r.posted_by
LEFT JOIN users u_ca ON u_ca.id = r.cancelled_by
LEFT JOIN users u_re ON u_re.id = r.released_by
WHERE r.tenant_id = $1
  AND ($2::uuid IS NULL OR r.warehouse_id = $2)
  AND ($3::varchar IS NULL OR r.status = $3)
  AND ($4::varchar IS NULL OR r.receipt_type = $4)
ORDER BY r.created_at DESC`

	rows, err := tx.QueryContext(ctx, query, tenantID, ptrToNullUUID(warehouseID), statusArg, typeArg)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockReceipts: query: %w", err)
	}
	defer rows.Close()

	result := []domain.StockReceipt{}
	for rows.Next() {
		rc, err := scanStockReceipt(rows)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.ListStockReceipts: scan: %w", err)
		}
		result = append(result, *rc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockReceipts: rows err: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockReceipts: commit: %w", err)
	}
	return result, nil
}

func isDateBeforeToday(t *time.Time, now time.Time) bool {
	if t == nil {
		return false
	}
	y1, m1, d1 := t.UTC().Date()
	y2, m2, d2 := now.UTC().Date()
	if y1 < y2 {
		return true
	}
	if y1 > y2 {
		return false
	}
	if m1 < m2 {
		return true
	}
	if m1 > m2 {
		return false
	}
	return d1 < d2
}

func (r *WMSRepo) insertReceiptMovementTx(ctx context.Context, tx *sql.Tx, tenantID, userID, receiptID, productID, batchID, src, dst uuid.UUID, qty decimal.Decimal, number string, now time.Time) error {
	movID := uuid.New()
	_, err := tx.ExecContext(ctx, createStockMovementSQL,
		movID, tenantID, number, productID, src, dst, qty, decimal.Zero,
		domain.StockMovementStatusDone, domain.StockRefGoodsReceipt, receiptID,
		ptrToNullUUID(&userID), ptrToNullUUID(&batchID), now)
	if err != nil {
		return err
	}
	auditDetails, _ := json.Marshal(map[string]any{
		"movement_number":    number,
		"product_id":         productID,
		"source_location_id": src,
		"dest_location_id":   dst,
		"quantity":           qty,
		"reference_type":     domain.StockRefGoodsReceipt,
		"reference_id":       receiptID,
		"batch_id":           batchID,
	})
	return r.WriteAuditTx(ctx, tx, tenantID, &userID, "stock_movement", movID, "created", auditDetails)
}

// PostStockReceipt: DRAFT -> POSTED in a single transaction. The header row is
// locked FOR UPDATE so concurrent/double posts serialize and the second one
// sees POSTED and returns ErrStockReceiptNotDraft (no double ledger entries).
// Adopted patterns: sentry-wms §1.2 (inbound to staging), OCA §1.3 (lot + expiry).
func (r *WMSRepo) PostStockReceipt(ctx context.Context, p domain.PostReceiptParams) (*domain.StockReceipt, error) {
	if p.UserID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}
	if p.ReceiptID == uuid.Nil || p.TenantID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, p.TenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: set tenant: %w", err)
	}

	rc, err := getStockReceiptTx(ctx, tx, p.TenantID, p.ReceiptID, true)
	if err != nil {
		if errors.Is(err, domain.ErrStockReceiptNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: lock header: %w", err)
	}
	if rc.Status != domain.StockReceiptStatusDraft {
		return nil, domain.ErrStockReceiptNotDraft
	}
	items, err := getStockReceiptItemsTx(ctx, tx, p.TenantID, p.ReceiptID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: items: %w", err)
	}
	if len(items) == 0 {
		return nil, &domain.StockReceiptValidationError{Msg: "Penerimaan harus memiliki minimal satu barang"}
	}

	targetStagingLoc := p.StagingLocID
	if targetStagingLoc == uuid.Nil {
		targetStagingLoc = rc.DestLocationID
	}

	now := time.Now().UTC()
	for i, it := range items {
		batchNum := ""
		if it.BatchNumber != nil && strings.TrimSpace(*it.BatchNumber) != "" {
			batchNum = strings.TrimSpace(*it.BatchNumber)
		} else {
			batchNum = domain.AutoBatchNumber(rc.ReceiptNumber, i+1)
		}
		batchStatus := domain.StockBatchStatusReleased
		if p.HoldForRelease || (it.ExpiryDate != nil && isDateBeforeToday(it.ExpiryDate, now)) {
			batchStatus = domain.StockBatchStatusOnHold
		}

		batchObj := &domain.StockBatch{
			ID:              uuid.New(),
			TenantID:        p.TenantID,
			ProductID:       it.ProductID,
			BatchNumber:     batchNum,
			ExpiryDate:      it.ExpiryDate,
			SourceReceiptID: &rc.ID,
			Status:          batchStatus,
			CreatedBy:       &p.UserID,
			CreatedAt:       now,
		}
		batch, err := r.getOrCreateBatchTx(ctx, tx, batchObj)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.PostStockReceipt: batch line %d: %w", i+1, err)
		}

		if _, err := tx.ExecContext(ctx, `
UPDATE stock_receipt_items
SET batch_id = $1, batch_number = $2, expiry_date = $3
WHERE id = $4 AND tenant_id = $5`,
			batch.ID, batch.BatchNumber, ptrToNullTime(batch.ExpiryDate), it.ID, p.TenantID); err != nil {
			return nil, fmt.Errorf("WMSRepo.PostStockReceipt: update item batch: %w", err)
		}

		if it.AcceptedQty.GreaterThan(decimal.Zero) {
			if err := r.insertReceiptMovementTx(ctx, tx, p.TenantID, p.UserID, p.ReceiptID, it.ProductID, batch.ID, p.SourceLocID, targetStagingLoc,
				it.AcceptedQty, fmt.Sprintf("GR-IN-%s-%d", rc.ReceiptNumber, i+1), now); err != nil {
				return nil, fmt.Errorf("WMSRepo.PostStockReceipt: accepted movement %d: %w", i, err)
			}
		}
		if it.RejectedQty.GreaterThan(decimal.Zero) {
			if err := r.insertReceiptMovementTx(ctx, tx, p.TenantID, p.UserID, p.ReceiptID, it.ProductID, batch.ID, p.SourceLocID, p.ScrapLocID,
				it.RejectedQty, fmt.Sprintf("GR-REJ-%s-%d", rc.ReceiptNumber, i+1), now); err != nil {
				return nil, fmt.Errorf("WMSRepo.PostStockReceipt: rejected movement %d: %w", i, err)
			}
		}
	}

	if rc.TransferID != nil {
		_, _ = tx.ExecContext(ctx, `
UPDATE stock_transfers
SET status = 'RECEIVED', received_at = $3, updated_at = $3
WHERE id = $1 AND tenant_id = $2 AND status IN ('DISPATCHED', 'IN_TRANSIT')`, *rc.TransferID, p.TenantID, now)
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE stock_receipts SET status = $3, posted_by = $4, posted_at = $5, updated_at = $5
WHERE id = $1 AND tenant_id = $2`, p.ReceiptID, p.TenantID, domain.StockReceiptStatusPosted, p.UserID, now); err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: update status: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]any{
		"receipt_number": rc.ReceiptNumber,
		"receipt_type":   rc.ReceiptType,
		"warehouse_id":   rc.WarehouseID,
		"staging_loc_id": targetStagingLoc,
		"item_count":     len(items),
		"held":           p.HoldForRelease,
	})
	if err := r.WriteAuditTx(ctx, tx, p.TenantID, &p.UserID, "stock_receipt", p.ReceiptID, "posted", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: audit: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: commit: %w", err)
	}
	rc.Status = domain.StockReceiptStatusPosted
	rc.PostedBy = &p.UserID
	rc.PostedAt = &now
	rc.UpdatedAt = now
	return rc, nil
}

// CancelStockReceipt: DRAFT -> CANCELLED, or POSTED -> CANCELLED with reversing
// movements, all in one transaction. Movements are never deleted.
func (r *WMSRepo) CancelStockReceipt(ctx context.Context, tenantID, id, userID, vendorLocID, scrapLocID uuid.UUID, reason string) (*domain.StockReceipt, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: set tenant: %w", err)
	}

	rc, err := getStockReceiptTx(ctx, tx, tenantID, id, true)
	if err != nil {
		if errors.Is(err, domain.ErrStockReceiptNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: lock header: %w", err)
	}
	if rc.Status == domain.StockReceiptStatusCancelled {
		return nil, domain.ErrStockReceiptAlreadyCancelled
	}

	now := time.Now().UTC()
	if rc.Status == domain.StockReceiptStatusPosted {
		type origMovement struct {
			productID  uuid.UUID
			batchID    uuid.UUID
			sourceID   uuid.UUID
			destID     uuid.UUID
			qty        decimal.Decimal
			movNumber  string
		}
		rows, err := tx.QueryContext(ctx, `
SELECT product_id, batch_id, source_location_id, dest_location_id, quantity, movement_number
FROM stock_movements
WHERE tenant_id = $1 AND reference_type = $2 AND reference_id = $3 AND status = 'DONE' AND movement_number LIKE 'GR-%'
ORDER BY created_at, id`, tenantID, domain.StockRefGoodsReceipt, id)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: query movements: %w", err)
		}
		defer rows.Close()

		var origs []origMovement
		type lockKey struct{ loc, prod uuid.UUID }
		var keys []lockKey
		for rows.Next() {
			var o origMovement
			if err := rows.Scan(&o.productID, &o.batchID, &o.sourceID, &o.destID, &o.qty, &o.movNumber); err != nil {
				return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: scan movement: %w", err)
			}
			origs = append(origs, o)
			keys = append(keys, lockKey{o.destID, o.productID})
		}
		_ = rows.Close()

		sort.Slice(keys, func(a, b int) bool {
			if keys[a].loc != keys[b].loc {
				return keys[a].loc.String() < keys[b].loc.String()
			}
			return keys[a].prod.String() < keys[b].prod.String()
		})
		for _, k := range keys {
			if err := r.LockLocationStock(ctx, tx, tenantID, k.loc, k.prod); err != nil {
				return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: %w", err)
			}
		}

		for _, o := range origs {
			var cur decimal.Decimal
			calcSQL := `
SELECT COALESCE(
    SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE -quantity END),
    0
)
FROM stock_movements
WHERE tenant_id = $1 AND product_id = $3 AND batch_id = $4
  AND (source_location_id = $2 OR dest_location_id = $2)
  AND status = 'DONE'`
			if err := tx.QueryRowContext(ctx, calcSQL, tenantID, o.destID, o.productID, o.batchID).Scan(&cur); err != nil {
				return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: scan batch stock: %w", err)
			}
			if cur.LessThan(o.qty) {
				return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: batch %s at location %s has %s, needs %s: %w",
					o.batchID, o.destID, cur.String(), o.qty.String(), domain.ErrStockReceiptStockConsumed)
			}
		}

		for i, o := range origs {
			revNumber := fmt.Sprintf("GR-REV-%s-%d", rc.ReceiptNumber, i+1)
			if err := r.insertReceiptMovementTx(ctx, tx, tenantID, userID, id, o.productID, o.batchID, o.destID, o.sourceID,
				o.qty, revNumber, now); err != nil {
				return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: reverse movement %d: %w", i, err)
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE stock_receipts SET status = $3, cancelled_by = $4, cancelled_at = $5, cancel_reason = $6, updated_at = $5
WHERE id = $1 AND tenant_id = $2`, id, tenantID, domain.StockReceiptStatusCancelled, userID, now, reason); err != nil {
		return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: update status: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]any{
		"receipt_number": rc.ReceiptNumber,
		"reason":         reason,
		"prior_status":   rc.Status,
	})
	if err := r.WriteAuditTx(ctx, tx, tenantID, &userID, "stock_receipt", id, "cancelled", auditDetails); err != nil {
		return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: audit: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: commit: %w", err)
	}
	rc.Status = domain.StockReceiptStatusCancelled
	rc.CancelledBy = &userID
	rc.CancelledAt = &now
	rc.CancelReason = &reason
	rc.UpdatedAt = now
	return rc, nil
}
