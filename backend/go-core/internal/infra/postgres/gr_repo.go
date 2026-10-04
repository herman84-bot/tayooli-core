package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type GRRepo struct {
	db *sql.DB
}

func NewGRRepo(db *sql.DB) *GRRepo {
	return &GRRepo{db: db}
}

const createGRSQL = `
INSERT INTO goods_receipts (id, tenant_id, po_id, vendor_id, received_qty, received_amount, currency)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, tenant_id, po_id, vendor_id, received_qty, received_amount, currency, status, received_at, created_at`

func (r *GRRepo) Create(ctx context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("GRRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, params.TenantID); err != nil {
		return nil, fmt.Errorf("GRRepo.Create: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, createGRSQL,
		uuid.New(), params.TenantID, params.POID, params.VendorID,
		params.ReceivedQty, params.ReceivedAmount, params.Currency,
	)
	gr, err := scanGRRow(row)
	if err != nil {
		return nil, fmt.Errorf("GRRepo.Create: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("GRRepo.Create: commit: %w", err)
	}
	return gr, nil
}

const listGRsSQL = `
SELECT id, tenant_id, po_id, vendor_id, received_qty, received_amount, currency, status, received_at, created_at
FROM goods_receipts
WHERE tenant_id = $1
ORDER BY received_at DESC`

func (r *GRRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.GoodsReceipt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("GRRepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("GRRepo.List: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listGRsSQL, tenantID)
	if err != nil {
		return nil, fmt.Errorf("GRRepo.List: query: %w", err)
	}
	defer rows.Close()

	var grs []domain.GoodsReceipt
	for rows.Next() {
		gr, err := scanGR(rows)
		if err != nil {
			return nil, fmt.Errorf("GRRepo.List: scan: %w", err)
		}
		grs = append(grs, *gr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GRRepo.List: rows err: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("GRRepo.List: close rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("GRRepo.List: commit: %w", err)
	}
	return grs, nil
}

const countGRsByTenant = `SELECT COUNT(*) FROM goods_receipts WHERE tenant_id = $1`

const listGRsByTenantPaged = `
SELECT id, tenant_id, po_id, vendor_id, received_qty, received_amount, currency, status, received_at, created_at
FROM goods_receipts
WHERE tenant_id = $1
ORDER BY received_at DESC
LIMIT $2 OFFSET $3`

// ListPaged returns a page of goods receipts plus the total count for the tenant.
func (r *GRRepo) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("GRRepo.ListPaged: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("GRRepo.ListPaged: set tenant: %w", err)
	}

	// Count total rows for this tenant.
	var total int
	if err := tx.QueryRowContext(ctx, countGRsByTenant, tenantID).Scan(&total); err != nil {
		return nil, fmt.Errorf("GRRepo.ListPaged: count: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := tx.QueryContext(ctx, listGRsByTenantPaged, tenantID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("GRRepo.ListPaged: query: %w", err)
	}
	defer rows.Close()

	var grs []domain.GoodsReceipt
	for rows.Next() {
		gr, err := scanGR(rows)
		if err != nil {
			return nil, fmt.Errorf("GRRepo.ListPaged: scan row: %w", err)
		}
		grs = append(grs, *gr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GRRepo.ListPaged: rows err: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("GRRepo.ListPaged: close rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("GRRepo.ListPaged: commit: %w", err)
	}

	return &domain.GoodsReceiptListPage{
		Data:    grs,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

const getGRSQL = `
SELECT id, tenant_id, po_id, vendor_id, received_qty, received_amount, currency, status, received_at, created_at
FROM goods_receipts
WHERE id = $1 AND tenant_id = $2`

func (r *GRRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.GoodsReceipt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("GRRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("GRRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getGRSQL, id, tenantID)
	gr, err := scanGRRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GRRepo.GetByID: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("GRRepo.GetByID: commit: %w", err)
	}
	return gr, nil
}

func scanGR(rows *sql.Rows) (*domain.GoodsReceipt, error) {
	var gr domain.GoodsReceipt
	var status string
	if err := rows.Scan(
		&gr.ID, &gr.TenantID, &gr.POID, &gr.VendorID,
		&gr.ReceivedQty, &gr.ReceivedAmount, &gr.Currency, &status,
		&gr.ReceivedAt, &gr.CreatedAt,
	); err != nil {
		return nil, err
	}
	gr.Status = domain.GRStatus(status)
	return &gr, nil
}

func scanGRRow(row *sql.Row) (*domain.GoodsReceipt, error) {
	var gr domain.GoodsReceipt
	var status string
	if err := row.Scan(
		&gr.ID, &gr.TenantID, &gr.POID, &gr.VendorID,
		&gr.ReceivedQty, &gr.ReceivedAmount, &gr.Currency, &status,
		&gr.ReceivedAt, &gr.CreatedAt,
	); err != nil {
		return nil, err
	}
	gr.Status = domain.GRStatus(status)
	return &gr, nil
}
