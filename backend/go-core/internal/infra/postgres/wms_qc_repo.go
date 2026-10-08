package postgres

// Sprint 2 QC & quarantine persistence. Patterns: sentry-wms 1.1 (quarantine bin is
// a non-pickable location per warehouse), ADR-014 Invariant 1 (every movement carries
// batch_id) and Invariant 3 (quarantine movement only via a QC inspection + BAK).

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

// GetOrCreateQuarantineLocation returns the warehouse QRN bin, creating it once.
func (r *WMSRepo) GetOrCreateQuarantineLocation(ctx context.Context, tenantID, warehouseID uuid.UUID) (*domain.WarehouseLocation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateQuarantineLocation: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateQuarantineLocation: set tenant: %w", err)
	}

	findSQL := `
SELECT id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at
FROM warehouse_locations
WHERE tenant_id = $1 AND warehouse_id = $2 AND type = 'QUARANTINE'
ORDER BY (code = $3) DESC, created_at ASC
LIMIT 1`
	loc, err := scanLocationRow(tx.QueryRowContext(ctx, findSQL, tenantID, warehouseID, domain.QuarantineCode))
	if err == nil {
		return loc, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateQuarantineLocation: query: %w", err)
	}
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
INSERT INTO warehouse_locations (id, tenant_id, warehouse_id, parent_id, code, barcode, name, type, is_pallet, pallet_number, max_capacity, created_at, updated_at)
VALUES ($1, $2, $3, NULL, $4, NULL, 'Area Karantina', 'QUARANTINE', FALSE, NULL, NULL, $5, $5)
ON CONFLICT (tenant_id, warehouse_id, code) DO NOTHING`,
		uuid.New(), tenantID, warehouseID, domain.QuarantineCode, now)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetOrCreateQuarantineLocation: insert: %w", err)
	}
	loc, err = scanLocationRow(tx.QueryRowContext(ctx, findSQL, tenantID, warehouseID, domain.QuarantineCode))
	if err != nil {
		// A non-quarantine location already owns code QRN in this warehouse.
		return nil, fmt.Errorf("WMSRepo.GetOrCreateQuarantineLocation: re-query (code %s taken?): %w", domain.QuarantineCode, err)
	}
	return loc, tx.Commit()
}

// batchBalanceAtTx returns the on-hand quantity of one batch at one location.
func batchBalanceAtTx(ctx context.Context, tx *sql.Tx, tenantID, locID, batchID, productID uuid.UUID) (decimal.Decimal, error) {
	var bal decimal.Decimal
	err := tx.QueryRowContext(ctx, `
SELECT COALESCE(SUM(CASE WHEN dest_location_id = $2 THEN quantity ELSE -quantity END), 0)
FROM stock_movements
WHERE tenant_id = $1 AND batch_id = $3 AND product_id = $4 AND status = 'DONE'
  AND (source_location_id = $2 OR dest_location_id = $2)`,
		tenantID, locID, batchID, productID).Scan(&bal)
	return bal, err
}

func (r *WMSRepo) insertMovementTx(ctx context.Context, tx *sql.Tx, m *domain.StockMovement) error {
	if m.BatchID == nil || *m.BatchID == uuid.Nil {
		return domain.ErrBatchRequired
	}
	_, err := tx.ExecContext(ctx, createStockMovementSQL,
		m.ID, m.TenantID, m.MovementNumber, m.ProductID,
		m.SourceLocationID, m.DestLocationID, m.Quantity,
		m.UnitCost, m.Status, m.ReferenceType, m.ReferenceID,
		ptrToNullUUID(m.ExecutedBy), ptrToNullUUID(m.BatchID), m.CreatedAt)
	if err != nil {
		return err
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
	return r.WriteAuditTx(ctx, tx, m.TenantID, m.ExecutedBy, "stock_movement", m.ID, "created", auditDetails)
}

// SubmitQCInspection records the inspection and moves damaged qty STG-IN -> QRN atomically.
func (r *WMSRepo) SubmitQCInspection(ctx context.Context, p domain.SubmitQCParams) (*domain.QCInspectionDetail, error) {
	if p.UserID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if err := setTenantLocally(ctx, tx, p.TenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: set tenant: %w", err)
	}

	// Lock receipt: concurrent submissions serialize; the unique (tenant, receipt) is the backstop.
	var status, receiptNumber string
	var warehouseID uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT status, receipt_number, warehouse_id FROM stock_receipts WHERE id = $1 AND tenant_id = $2 FOR UPDATE`,
		p.ReceiptID, p.TenantID).Scan(&status, &receiptNumber, &warehouseID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrStockReceiptNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: lock receipt: %w", err)
	}
	if domain.StockReceiptStatus(status) != domain.StockReceiptStatusPosted {
		return nil, domain.ErrQCReceiptNotPosted
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM qc_inspections WHERE tenant_id = $1 AND receipt_id = $2)`,
		p.TenantID, p.ReceiptID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: check existing: %w", err)
	}
	if exists {
		return nil, domain.ErrQCAlreadyInspected
	}

	rows, err := tx.QueryContext(ctx, `
SELECT product_id, batch_id, SUM(accepted_qty)
FROM stock_receipt_items
WHERE tenant_id = $1 AND receipt_id = $2 AND batch_id IS NOT NULL AND accepted_qty > 0
GROUP BY product_id, batch_id`, p.TenantID, p.ReceiptID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: load lines: %w", err)
	}
	var expected []domain.QCExpectedLine
	for rows.Next() {
		var e domain.QCExpectedLine
		if err := rows.Scan(&e.ProductID, &e.BatchID, &e.ExpectedQty); err != nil {
			rows.Close()
			return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: scan line: %w", err)
		}
		expected = append(expected, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: lines rows: %w", err)
	}

	comp, err := domain.EvaluateQC(p.Input, expected)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	inspID := uuid.New()
	var bak *string
	if comp.Damaged.IsPositive() {
		b := domain.BAKNumber(receiptNumber)
		bak = &b
	}
	in := p.Input
	_, err = tx.ExecContext(ctx, `
INSERT INTO qc_inspections (id, tenant_id, receipt_id, warehouse_id, inspection_mode, sample_qty, gross_cartons, status,
    total_checked_qty, total_passed_qty, total_damaged_qty, shortage_qty, overage_qty,
    bak_number, bak_notes, driver_name, driver_signed, notes, inspector_id, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		inspID, p.TenantID, p.ReceiptID, warehouseID, string(in.InspectionMode), decPtr(in.SampleQty), in.GrossCartons, string(comp.Status),
		comp.Checked, comp.Passed, comp.Damaged, comp.Shortage, comp.Overage,
		bak, trimmedOrNil(in.BAKNotes), trimmedOrNil(in.DriverName), in.DriverSigned, trimmedOrNil(in.Notes), p.UserID, now)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: insert inspection: %w", err)
	}

	for i := range comp.Items {
		it := &comp.Items[i]
		it.ID = uuid.New()
		_, err = tx.ExecContext(ctx, `
INSERT INTO qc_inspection_items (id, tenant_id, inspection_id, product_id, batch_id, staged_qty, checked_qty, passed_qty,
    damaged_qty, shortage_qty, overage_qty, damage_reason)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			it.ID, p.TenantID, inspID, it.ProductID, it.BatchID, it.StagedQty, it.CheckedQty, it.PassedQty,
			it.DamagedQty, it.ShortageQty, it.OverageQty, it.DamageReason)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: insert item: %w", err)
		}
		if !it.DamagedQty.IsPositive() {
			continue
		}
		if err := r.LockLocationStock(ctx, tx, p.TenantID, p.StagingLocID, it.ProductID); err != nil {
			return nil, err
		}
		staged, err := batchBalanceAtTx(ctx, tx, p.TenantID, p.StagingLocID, it.BatchID, it.ProductID)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: staged balance: %w", err)
		}
		// Part of the batch may already be put away; only staged stock can be quarantined.
		if staged.LessThan(it.DamagedQty) {
			return nil, domain.ErrQCStagedQtyChanged
		}
		batchID := it.BatchID
		mov := domain.StockMovement{
			ID:               uuid.New(),
			TenantID:         p.TenantID,
			MovementNumber:   fmt.Sprintf("QC-%s-%d", receiptNumber, i+1),
			ProductID:        it.ProductID,
			BatchID:          &batchID,
			SourceLocationID: p.StagingLocID,
			DestLocationID:   p.QuarantineLoc,
			Quantity:         it.DamagedQty,
			UnitCost:         decimal.Zero,
			Status:           domain.StockMovementStatusDone,
			ReferenceType:    domain.StockRefQC,
			ReferenceID:      inspID,
			ExecutedBy:       &p.UserID,
			CreatedAt:        now,
		}
		if err := r.insertMovementTx(ctx, tx, &mov); err != nil {
			return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: movement: %w", err)
		}
	}

	details, _ := json.Marshal(map[string]any{
		"receipt_id": p.ReceiptID, "status": comp.Status, "mode": in.InspectionMode,
		"damaged_qty": comp.Damaged, "shortage_qty": comp.Shortage, "overage_qty": comp.Overage, "bak_number": bak,
	})
	if err := r.WriteAuditTx(ctx, tx, p.TenantID, &p.UserID, "qc_inspection", inspID, "inspected", details); err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.SubmitQCInspection: commit: %w", err)
	}
	return r.GetQCInspectionByReceipt(ctx, p.TenantID, p.ReceiptID)
}

const qcInspectionSelect = `
SELECT q.id, q.tenant_id, q.receipt_id, r.receipt_number, r.supplier_name, q.warehouse_id, q.inspection_mode, q.sample_qty,
       q.gross_cartons, q.status, q.total_checked_qty, q.total_passed_qty, q.total_damaged_qty, q.shortage_qty, q.overage_qty,
       q.bak_number, q.bak_notes, q.driver_name, q.driver_signed, q.notes, q.inspector_id,
       COALESCE(u.full_name, u.email, ''), q.created_at
FROM qc_inspections q
JOIN stock_receipts r ON r.id = q.receipt_id AND r.tenant_id = q.tenant_id
LEFT JOIN users u ON u.id = q.inspector_id`

func scanQCInspection(s rowScanner) (*domain.QCInspection, error) {
	var q domain.QCInspection
	var sample decimal.NullDecimal
	var bak, bakNotes, driver, notes sql.NullString
	if err := s.Scan(&q.ID, &q.TenantID, &q.ReceiptID, &q.ReceiptNumber, &q.SupplierName, &q.WarehouseID, &q.InspectionMode, &sample,
		&q.GrossCartons, &q.Status, &q.TotalCheckedQty, &q.TotalPassedQty, &q.TotalDamagedQty, &q.ShortageQty, &q.OverageQty,
		&bak, &bakNotes, &driver, &q.DriverSigned, &notes, &q.InspectorID, &q.InspectorName, &q.CreatedAt); err != nil {
		return nil, err
	}
	if sample.Valid {
		q.SampleQty = &sample.Decimal
	}
	q.BAKNumber, q.BAKNotes, q.DriverName, q.Notes = nullStrPtr(bak), nullStrPtr(bakNotes), nullStrPtr(driver), nullStrPtr(notes)
	return &q, nil
}

// GetQCInspectionByReceipt returns (nil, nil) when the receipt has not been inspected.
func (r *WMSRepo) GetQCInspectionByReceipt(ctx context.Context, tenantID, receiptID uuid.UUID) (*domain.QCInspectionDetail, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetQCInspectionByReceipt: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetQCInspectionByReceipt: set tenant: %w", err)
	}
	q, err := scanQCInspection(tx.QueryRowContext(ctx, qcInspectionSelect+` WHERE q.tenant_id = $1 AND q.receipt_id = $2`, tenantID, receiptID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetQCInspectionByReceipt: query: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `
SELECT i.id, i.product_id, COALESCE(p.name, ''), COALESCE(p.sku, ''), i.batch_id, b.batch_number, b.expiry_date,
       i.staged_qty, i.checked_qty, i.passed_qty, i.damaged_qty, i.shortage_qty, i.overage_qty, i.damage_reason
FROM qc_inspection_items i
JOIN stock_batches b ON b.id = i.batch_id AND b.tenant_id = i.tenant_id
LEFT JOIN products p ON p.id = i.product_id AND p.tenant_id = i.tenant_id
WHERE i.tenant_id = $1 AND i.inspection_id = $2
ORDER BY p.name, b.batch_number`, tenantID, q.ID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.GetQCInspectionByReceipt: items: %w", err)
	}
	defer rows.Close()
	detail := &domain.QCInspectionDetail{Inspection: *q, Items: []domain.QCInspectionItem{}}
	for rows.Next() {
		var it domain.QCInspectionItem
		var exp sql.NullTime
		var reason sql.NullString
		if err := rows.Scan(&it.ID, &it.ProductID, &it.ProductName, &it.ProductSKU, &it.BatchID, &it.BatchNumber, &exp,
			&it.StagedQty, &it.CheckedQty, &it.PassedQty, &it.DamagedQty, &it.ShortageQty, &it.OverageQty, &reason); err != nil {
			return nil, fmt.Errorf("WMSRepo.GetQCInspectionByReceipt: scan item: %w", err)
		}
		it.ExpiryDate, it.DamageReason = nullTimeToPtr(exp), nullStrPtr(reason)
		detail.Items = append(detail.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.GetQCInspectionByReceipt: rows: %w", err)
	}
	return detail, tx.Commit()
}

func (r *WMSRepo) ListQCInspections(ctx context.Context, tenantID, warehouseID uuid.UUID) ([]domain.QCInspection, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListQCInspections: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListQCInspections: set tenant: %w", err)
	}
	rows, err := tx.QueryContext(ctx, qcInspectionSelect+` WHERE q.tenant_id = $1 AND q.warehouse_id = $2 ORDER BY q.created_at DESC LIMIT 500`, tenantID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.ListQCInspections: query: %w", err)
	}
	defer rows.Close()
	out := []domain.QCInspection{}
	for rows.Next() {
		q, err := scanQCInspection(rows)
		if err != nil {
			return nil, fmt.Errorf("WMSRepo.ListQCInspections: scan: %w", err)
		}
		out = append(out, *q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("WMSRepo.ListQCInspections: rows: %w", err)
	}
	return out, tx.Commit()
}

// MoveQuarantineStock releases (QRN -> STG-IN, re-enters putaway) or scraps (QRN -> @SCRAP).
func (r *WMSRepo) MoveQuarantineStock(ctx context.Context, p domain.QuarantineMoveParams) (*domain.StockMovement, error) {
	in := p.Input
	if p.UserID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}
	if in.ProductID == uuid.Nil || in.BatchID == uuid.Nil || !in.Quantity.IsPositive() || p.SourceLocID == uuid.Nil || p.DestLocID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveQuarantineStock: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if err := setTenantLocally(ctx, tx, p.TenantID); err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveQuarantineStock: set tenant: %w", err)
	}
	if err := r.LockLocationStock(ctx, tx, p.TenantID, p.SourceLocID, in.ProductID); err != nil {
		return nil, err
	}
	bal, err := batchBalanceAtTx(ctx, tx, p.TenantID, p.SourceLocID, in.BatchID, in.ProductID)
	if err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveQuarantineStock: balance: %w", err)
	}
	if bal.LessThan(in.Quantity) {
		return nil, domain.ErrQuarantineQtyInvalid
	}
	prefix, ref := "QRL", domain.StockRefQC
	if p.Action == "scrapped" {
		prefix, ref = "QSC", domain.StockRefScrap
	}
	batchID := in.BatchID
	mov := domain.StockMovement{
		ID:               uuid.New(),
		TenantID:         p.TenantID,
		MovementNumber:   fmt.Sprintf("%s-%s", prefix, strings.ToUpper(uuid.NewString()[:13])),
		ProductID:        in.ProductID,
		BatchID:          &batchID,
		SourceLocationID: p.SourceLocID,
		DestLocationID:   p.DestLocID,
		Quantity:         in.Quantity,
		UnitCost:         decimal.Zero,
		Status:           domain.StockMovementStatusDone,
		ReferenceType:    ref,
		ReferenceID:      in.BatchID,
		ExecutedBy:       &p.UserID,
		CreatedAt:        time.Now().UTC(),
	}
	if err := r.insertMovementTx(ctx, tx, &mov); err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveQuarantineStock: movement: %w", err)
	}
	details, _ := json.Marshal(map[string]any{
		"product_id": in.ProductID, "batch_id": in.BatchID, "quantity": in.Quantity,
		"from": p.SourceLocID, "to": p.DestLocID, "notes": trimmedOrNil(in.Notes), "movement_id": mov.ID,
	})
	if err := r.WriteAuditTx(ctx, tx, p.TenantID, &p.UserID, "quarantine", in.BatchID, p.Action, details); err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveQuarantineStock: audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("WMSRepo.MoveQuarantineStock: commit: %w", err)
	}
	return &mov, nil
}

func decPtr(d *decimal.Decimal) any {
	if d == nil {
		return nil
	}
	return *d
}

func trimmedOrNil(s *string) any {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return strings.TrimSpace(*s)
}

func nullStrPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}
