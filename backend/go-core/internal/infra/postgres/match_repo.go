package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type MatchRepo struct {
	db *sql.DB
}

func NewMatchRepo(db *sql.DB) *MatchRepo {
	return &MatchRepo{db: db}
}

const getPOByID = `
SELECT id, tenant_id, vendor_id, po_number, amount, qty, currency, status, created_at, updated_at
FROM purchase_orders
WHERE id = $1 AND tenant_id = $2`

func (r *MatchRepo) GetPOByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("MatchRepo.GetPOByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("MatchRepo.GetPOByID: set tenant: %w", err)
	}

	var po domain.PurchaseOrder
	var status string
	err = tx.QueryRowContext(ctx, getPOByID, id, tenantID).Scan(
		&po.ID,
		&po.TenantID,
		&po.VendorID,
		&po.PONumber,
		&po.Amount,
		&po.Qty,
		&po.Currency,
		&status,
		&po.CreatedAt,
		&po.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("MatchRepo.GetPOByID: scan: %w", err)
	}
	po.Status = domain.POStatus(status)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("MatchRepo.GetPOByID: commit: %w", err)
	}
	return &po, nil
}

const getGRByPOID = `
SELECT id, tenant_id, po_id, vendor_id, received_qty, received_amount, currency, status, received_at, created_at
FROM goods_receipts
WHERE po_id = $1 AND tenant_id = $2
ORDER BY received_at DESC
LIMIT 1`

func (r *MatchRepo) GetGRByPOID(ctx context.Context, poID, tenantID uuid.UUID) (*domain.GoodsReceipt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("MatchRepo.GetGRByPOID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("MatchRepo.GetGRByPOID: set tenant: %w", err)
	}

	var gr domain.GoodsReceipt
	var status string
	err = tx.QueryRowContext(ctx, getGRByPOID, poID, tenantID).Scan(
		&gr.ID,
		&gr.TenantID,
		&gr.POID,
		&gr.VendorID,
		&gr.ReceivedQty,
		&gr.ReceivedAmount,
		&gr.Currency,
		&status,
		&gr.ReceivedAt,
		&gr.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("MatchRepo.GetGRByPOID: scan: %w", err)
	}
	gr.Status = domain.GRStatus(status)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("MatchRepo.GetGRByPOID: commit: %w", err)
	}
	return &gr, nil
}

const updateInvoiceMatchResult = `
UPDATE invoices
SET match_result = $1,
    status       = $2,
    po_id        = $3,
    updated_at   = NOW()
WHERE id = $4 AND tenant_id = $5`

func (r *MatchRepo) UpdateInvoiceMatchResult(ctx context.Context, invoiceID, tenantID uuid.UUID, poID *uuid.UUID, result domain.MatchResult, status domain.InvoiceStatus) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("MatchRepo.UpdateInvoiceMatchResult: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("MatchRepo.UpdateInvoiceMatchResult: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, updateInvoiceMatchResult, string(result), string(status), poID, invoiceID, tenantID)
	if err != nil {
		return fmt.Errorf("MatchRepo.UpdateInvoiceMatchResult: exec: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("MatchRepo.UpdateInvoiceMatchResult: rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("MatchRepo.UpdateInvoiceMatchResult: commit: %w", err)
	}
	return nil
}
