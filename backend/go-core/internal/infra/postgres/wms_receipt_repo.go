package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
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
INSERT INTO stock_receipts (id, tenant_id, receipt_number, warehouse_id, dest_location_id, supplier_name, supplier_ref, notes, status, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

const createStockReceiptItemSQL = `
INSERT INTO stock_receipt_items (id, tenant_id, receipt_id, product_id, expected_qty, accepted_qty, rejected_qty, reject_reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

const stockReceiptColumns = `
r.id, r.tenant_id, r.receipt_number, r.warehouse_id, r.dest_location_id, r.supplier_name, r.supplier_ref, r.notes,
r.status, r.created_by, r.created_at, r.updated_at, r.posted_by, r.posted_at, r.cancelled_by, r.cancelled_at, r.cancel_reason,
(SELECT COUNT(*) FROM stock_receipt_items i WHERE i.receipt_id = r.id AND i.tenant_id = r.tenant_id),
(SELECT COALESCE(SUM(i.accepted_qty), 0) FROM stock_receipt_items i WHERE i.receipt_id = r.id AND i.tenant_id = r.tenant_id),
(SELECT COALESCE(SUM(i.rejected_qty), 0) FROM stock_receipt_items i WHERE i.receipt_id = r.id AND i.tenant_id = r.tenant_id)`

const getStockReceiptItemsSQL = `
SELECT i.id, i.tenant_id, i.receipt_id, i.product_id, COALESCE(p.name, ''), COALESCE(p.sku, ''),
       i.expected_qty, i.accepted_qty, i.rejected_qty, i.reject_reason, i.created_at
FROM stock_receipt_items i
LEFT JOIN products p ON p.id = i.product_id AND p.tenant_id = i.tenant_id
WHERE i.receipt_id = $1 AND i.tenant_id = $2
ORDER BY i.created_at, i.id`

func scanStockReceipt(s rowScanner) (*domain.StockReceipt, error) {
	var rc domain.StockReceipt
	var supRef, notes, postedBy, cancelledBy, cancelReason sql.NullString
	var postedAt, cancelledAt sql.NullTime
	if err := s.Scan(
		&rc.ID, &rc.TenantID, &rc.ReceiptNumber, &rc.WarehouseID, &rc.DestLocationID, &rc.SupplierName, &supRef, &notes,
		&rc.Status, &rc.CreatedBy, &rc.CreatedAt, &rc.UpdatedAt, &postedBy, &postedAt, &cancelledBy, &cancelledAt, &cancelReason,
		&rc.ItemCount, &rc.TotalAcceptedQty, &rc.TotalRejectedQty,
	); err != nil {
		return nil, err
	}
	rc.SupplierRef = nullStringToPtr(supRef)
	rc.Notes = nullStringToPtr(notes)
	rc.PostedBy = nullUUIDToPtr(postedBy)
	rc.PostedAt = nullTimeToPtr(postedAt)
	rc.CancelledBy = nullUUIDToPtr(cancelledBy)
	rc.CancelledAt = nullTimeToPtr(cancelledAt)
	rc.CancelReason = nullStringToPtr(cancelReason)
	return &rc, nil
}

func getStockReceiptTx(ctx context.Context, tx *sql.Tx, tenantID, id uuid.UUID, forUpdate bool) (*domain.StockReceipt, error) {
	q := `SELECT ` + stockReceiptColumns + ` FROM stock_receipts r WHERE r.id = $1 AND r.tenant_id = $2`
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
		if err := rows.Scan(&it.ID, &it.TenantID, &it.ReceiptID, &it.ProductID, &it.ProductName, &it.ProductSKU,
			&expected, &it.AcceptedQty, &it.RejectedQty, &reason, &it.CreatedAt); err != nil {
			return nil, err
		}
		it.ExpectedQty = nullDecimalToPtr(expected)
		it.RejectReason = nullStringToPtr(reason)
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
			it.AcceptedQty, it.RejectedQty, ptrToNullString(it.RejectReason), it.CreatedAt); err != nil {
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

	if _, err := tx.ExecContext(ctx, createStockReceiptSQL,
		rc.ID, rc.TenantID, rc.ReceiptNumber, rc.WarehouseID, rc.DestLocationID, rc.SupplierName,
		ptrToNullString(rc.SupplierRef), ptrToNullString(rc.Notes), rc.Status, rc.CreatedBy,
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
	if _, err := tx.ExecContext(ctx, `
UPDATE stock_receipts
SET warehouse_id = $3, dest_location_id = $4, supplier_name = $5, supplier_ref = $6, notes = $7, updated_at = $8
WHERE id = $1 AND tenant_id = $2`,
		rc.ID, rc.TenantID, rc.WarehouseID, rc.DestLocationID, rc.SupplierName,
		ptrToNullString(rc.SupplierRef), ptrToNullString(rc.Notes), now); err != nil {
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

func (r *WMSRepo) ListStockReceipts(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.StockReceiptStatus) ([]domain.StockReceipt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockReceipts: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListStockReceipts: set tenant: %w", err)
	}

	var statusArg sql.NullString
	if status != nil {
		statusArg = sql.NullString{String: string(*status), Valid: true}
	}
	query := `SELECT ` + stockReceiptColumns + `
FROM stock_receipts r
WHERE r.tenant_id = $1
  AND ($2::uuid IS NULL OR r.warehouse_id = $2)
  AND ($3::varchar IS NULL OR r.status = $3)
ORDER BY r.created_at DESC`

	rows, err := tx.QueryContext(ctx, query, tenantID, ptrToNullUUID(warehouseID), statusArg)
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

func insertReceiptMovementTx(ctx context.Context, tx *sql.Tx, tenantID, userID, receiptID, productID, src, dst uuid.UUID, qty decimal.Decimal, number string, now time.Time) error {
	_, err := tx.ExecContext(ctx, createStockMovementSQL,
		uuid.New(), tenantID, number, productID, src, dst, qty, decimal.Zero,
		domain.StockMovementStatusDone, domain.StockRefGoodsReceipt, receiptID,
		ptrToNullUUID(&userID), now)
	return err
}

// postedLocationTx returns the location a receipt's posted movements used,
// matched by movement_number prefix. For "GR-IN-%" the vendor side is the
// source; for "GR-REJ-%" the scrap side is the destination (vendor is source).
// wantSource selects which column to read. ok=false means no such movement.
func postedLocationTx(ctx context.Context, tx *sql.Tx, tenantID, receiptID uuid.UUID, numberLike string, wantSource bool) (uuid.UUID, bool, error) {
	col := "dest_location_id"
	if wantSource {
		col = "source_location_id"
	}
	var loc uuid.UUID
	err := tx.QueryRowContext(ctx, `
SELECT `+col+` FROM stock_movements
WHERE tenant_id = $1 AND reference_type = $2 AND reference_id = $3 AND movement_number LIKE $4
ORDER BY created_at, id LIMIT 1`,
		tenantID, domain.StockRefGoodsReceipt, receiptID, numberLike).Scan(&loc)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("WMSRepo.CancelStockReceipt: posted location: %w", err)
	}
	return loc, true, nil
}

// PostStockReceipt: DRAFT -> POSTED in a single transaction. The header row is
// locked FOR UPDATE so concurrent/double posts serialize and the second one
// sees POSTED and returns ErrStockReceiptNotDraft (no double ledger entries).
func (r *WMSRepo) PostStockReceipt(ctx context.Context, tenantID, id, userID, vendorLocID, scrapLocID uuid.UUID) (*domain.StockReceipt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: set tenant: %w", err)
	}

	rc, err := getStockReceiptTx(ctx, tx, tenantID, id, true)
	if err != nil {
		if errors.Is(err, domain.ErrStockReceiptNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: lock header: %w", err)
	}
	if rc.Status != domain.StockReceiptStatusDraft {
		return nil, domain.ErrStockReceiptNotDraft
	}
	items, err := getStockReceiptItemsTx(ctx, tx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: items: %w", err)
	}
	if len(items) == 0 {
		return nil, &domain.StockReceiptValidationError{Msg: "Penerimaan harus memiliki minimal satu barang"}
	}

	now := time.Now().UTC()
	for i, it := range items {
		if it.AcceptedQty.GreaterThan(decimal.Zero) {
			if err := insertReceiptMovementTx(ctx, tx, tenantID, userID, id, it.ProductID, vendorLocID, rc.DestLocationID,
				it.AcceptedQty, fmt.Sprintf("GR-IN-%s-%d", rc.ReceiptNumber, i+1), now); err != nil {
				return nil, fmt.Errorf("WMSRepo.PostStockReceipt: accepted movement %d: %w", i, err)
			}
		}
		if it.RejectedQty.GreaterThan(decimal.Zero) {
			if err := insertReceiptMovementTx(ctx, tx, tenantID, userID, id, it.ProductID, vendorLocID, scrapLocID,
				it.RejectedQty, fmt.Sprintf("GR-REJ-%s-%d", rc.ReceiptNumber, i+1), now); err != nil {
				return nil, fmt.Errorf("WMSRepo.PostStockReceipt: rejected movement %d: %w", i, err)
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE stock_receipts SET status = $3, posted_by = $4, posted_at = $5, updated_at = $5
WHERE id = $1 AND tenant_id = $2`, id, tenantID, domain.StockReceiptStatusPosted, userID, now); err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: update status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.PostStockReceipt: commit: %w", err)
	}
	rc.Status = domain.StockReceiptStatusPosted
	rc.PostedBy = &userID
	rc.PostedAt = &now
	rc.UpdatedAt = now
	return rc, nil
}

// CancelStockReceipt: DRAFT -> CANCELLED, or POSTED -> CANCELLED with reversing
// movements, all in one transaction. Movements are never deleted.
func (r *WMSRepo) CancelStockReceipt(ctx context.Context, tenantID, id, userID, vendorLocID, scrapLocID uuid.UUID, reason string) (*domain.StockReceipt, error) {
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
		items, err := getStockReceiptItemsTx(ctx, tx, tenantID, id)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: items: %w", err)
		}

		// Reverse against the exact locations used at post time, read from this
		// receipt's own ledger rows. Re-resolving @VENDOR/@SCRAP could return a
		// different row if duplicate virtual locations exist (warehouse_id NULL
		// is not deduplicated by the UNIQUE constraint).
		if loc, ok, err := postedLocationTx(ctx, tx, tenantID, id, "GR-IN-%", true); err != nil {
			return nil, err
		} else if ok {
			vendorLocID = loc
		}
		if loc, ok, err := postedLocationTx(ctx, tx, tenantID, id, "GR-REJ-%", false); err != nil {
			return nil, err
		} else if ok {
			scrapLocID = loc
		}

		// Collect (location, product) pairs to lock; sort for deterministic lock order (deadlock avoidance).
		type lockKey struct{ loc, prod uuid.UUID }
		var keys []lockKey
		for _, it := range items {
			if it.AcceptedQty.GreaterThan(decimal.Zero) {
				keys = append(keys, lockKey{rc.DestLocationID, it.ProductID})
			}
			if it.RejectedQty.GreaterThan(decimal.Zero) {
				keys = append(keys, lockKey{scrapLocID, it.ProductID})
			}
		}
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

		checkStock := func(loc, prod uuid.UUID, need decimal.Decimal) error {
			var cur decimal.Decimal
			if err := tx.QueryRowContext(ctx, deductStockCalcSQL, tenantID, loc, prod).Scan(&cur); err != nil {
				return fmt.Errorf("WMSRepo.CancelStockReceipt: scan stock: %w", err)
			}
			if cur.LessThan(need) {
				return fmt.Errorf("WMSRepo.CancelStockReceipt: product %s at location %s has %s, needs %s: %w",
					prod, loc, cur.String(), need.String(), domain.ErrStockReceiptStockConsumed)
			}
			return nil
		}
		for _, it := range items {
			if it.AcceptedQty.GreaterThan(decimal.Zero) {
				if err := checkStock(rc.DestLocationID, it.ProductID, it.AcceptedQty); err != nil {
					return nil, err
				}
			}
			if it.RejectedQty.GreaterThan(decimal.Zero) {
				if err := checkStock(scrapLocID, it.ProductID, it.RejectedQty); err != nil {
					return nil, err
				}
			}
		}

		for i, it := range items {
			if it.AcceptedQty.GreaterThan(decimal.Zero) {
				if err := insertReceiptMovementTx(ctx, tx, tenantID, userID, id, it.ProductID, rc.DestLocationID, vendorLocID,
					it.AcceptedQty, fmt.Sprintf("GR-REV-IN-%s-%d", rc.ReceiptNumber, i+1), now); err != nil {
					return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: reverse accepted %d: %w", i, err)
				}
			}
			if it.RejectedQty.GreaterThan(decimal.Zero) {
				if err := insertReceiptMovementTx(ctx, tx, tenantID, userID, id, it.ProductID, scrapLocID, vendorLocID,
					it.RejectedQty, fmt.Sprintf("GR-REV-REJ-%s-%d", rc.ReceiptNumber, i+1), now); err != nil {
					return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: reverse rejected %d: %w", i, err)
				}
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE stock_receipts SET status = $3, cancelled_by = $4, cancelled_at = $5, cancel_reason = $6, updated_at = $5
WHERE id = $1 AND tenant_id = $2`, id, tenantID, domain.StockReceiptStatusCancelled, userID, now, reason); err != nil {
		return nil, fmt.Errorf("WMSRepo.CancelStockReceipt: update status: %w", err)
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
