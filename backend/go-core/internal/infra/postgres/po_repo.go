package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type PORepo struct {
	db *sql.DB
}

func NewPORepo(db *sql.DB) *PORepo {
	return &PORepo{db: db}
}

const createPOSQL = `
INSERT INTO purchase_orders (id, tenant_id, vendor_id, po_number, amount, qty, currency)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, tenant_id, vendor_id, po_number, amount, qty, currency, status, created_at, updated_at`

func (r *PORepo) Create(ctx context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PORepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, params.TenantID); err != nil {
		return nil, fmt.Errorf("PORepo.Create: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, createPOSQL,
		uuid.New(), params.TenantID, params.VendorID, params.PONumber,
		params.Amount, params.Qty, params.Currency,
	)
	po, err := scanPORow(row)
	if err != nil {
		return nil, fmt.Errorf("PORepo.Create: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PORepo.Create: commit: %w", err)
	}
	return po, nil
}

const listPOsSQL = `
SELECT id, tenant_id, vendor_id, po_number, amount, qty, currency, status, created_at, updated_at
FROM purchase_orders
WHERE tenant_id = $1
ORDER BY created_at DESC`

func (r *PORepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.PurchaseOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PORepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("PORepo.List: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listPOsSQL, tenantID)
	if err != nil {
		return nil, fmt.Errorf("PORepo.List: query: %w", err)
	}
	defer rows.Close()

	var pos []domain.PurchaseOrder
	for rows.Next() {
		po, err := scanPO(rows)
		if err != nil {
			return nil, fmt.Errorf("PORepo.List: scan: %w", err)
		}
		pos = append(pos, *po)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PORepo.List: rows err: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("PORepo.List: close rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PORepo.List: commit: %w", err)
	}
	return pos, nil
}

const countPOsByTenant = `SELECT COUNT(*) FROM purchase_orders WHERE tenant_id = $1`

const listPOsByTenantPaged = `
SELECT id, tenant_id, vendor_id, po_number, amount, qty, currency, status, created_at, updated_at
FROM purchase_orders
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

// ListPaged returns a page of purchase orders plus the total count for the tenant.
func (r *PORepo) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PORepo.ListPaged: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("PORepo.ListPaged: set tenant: %w", err)
	}

	// Count total rows for this tenant.
	var total int
	if err := tx.QueryRowContext(ctx, countPOsByTenant, tenantID).Scan(&total); err != nil {
		return nil, fmt.Errorf("PORepo.ListPaged: count: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := tx.QueryContext(ctx, listPOsByTenantPaged, tenantID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("PORepo.ListPaged: query: %w", err)
	}
	defer rows.Close()

	var pos []domain.PurchaseOrder
	for rows.Next() {
		po, err := scanPO(rows)
		if err != nil {
			return nil, fmt.Errorf("PORepo.ListPaged: scan row: %w", err)
		}
		pos = append(pos, *po)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PORepo.ListPaged: rows err: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("PORepo.ListPaged: close rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PORepo.ListPaged: commit: %w", err)
	}

	return &domain.PurchaseOrderListPage{
		Data:    pos,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

const getPOSQL = `
SELECT id, tenant_id, vendor_id, po_number, amount, qty, currency, status, created_at, updated_at
FROM purchase_orders
WHERE id = $1 AND tenant_id = $2`

func (r *PORepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("PORepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("PORepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getPOSQL, id, tenantID)
	po, err := scanPORow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("PORepo.GetByID: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("PORepo.GetByID: commit: %w", err)
	}
	return po, nil
}

func scanPO(rows *sql.Rows) (*domain.PurchaseOrder, error) {
	var po domain.PurchaseOrder
	var status string
	if err := rows.Scan(
		&po.ID, &po.TenantID, &po.VendorID, &po.PONumber,
		&po.Amount, &po.Qty, &po.Currency, &status,
		&po.CreatedAt, &po.UpdatedAt,
	); err != nil {
		return nil, err
	}
	po.Status = domain.POStatus(status)
	return &po, nil
}

func scanPORow(row *sql.Row) (*domain.PurchaseOrder, error) {
	var po domain.PurchaseOrder
	var status string
	if err := row.Scan(
		&po.ID, &po.TenantID, &po.VendorID, &po.PONumber,
		&po.Amount, &po.Qty, &po.Currency, &status,
		&po.CreatedAt, &po.UpdatedAt,
	); err != nil {
		return nil, err
	}
	po.Status = domain.POStatus(status)
	return &po, nil
}
