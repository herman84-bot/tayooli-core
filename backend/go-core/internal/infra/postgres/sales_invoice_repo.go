package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// SalesInvoiceRepo implements domain.SalesInvoiceRepository using database/sql with
// parameterized queries. Tenant isolation is enforced at two layers:
//
//  1. Application layer: every query carries WHERE tenant_id = $n.
//  2. Database layer: every operation runs inside a transaction that sets
//     SET LOCAL app.current_tenant_id, activating the RLS policies.
//
// Using transaction-scoped SET LOCAL (rather than SET SESSION) means the
// variable is automatically cleared when the transaction ends, preventing
// any accidental cross-tenant data leakage between connection pool reuse.
type SalesInvoiceRepo struct {
	db *sql.DB
}

func NewSalesInvoiceRepo(db *sql.DB) *SalesInvoiceRepo {
	return &SalesInvoiceRepo{db: db}
}

const createSalesInvoice = `
INSERT INTO sales_invoices (id, tenant_id, sales_order_id, invoice_number, amount, status, due_date, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

func (r *SalesInvoiceRepo) Create(ctx context.Context, si *domain.SalesInvoice) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SalesInvoiceRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, si.TenantID); err != nil {
		return fmt.Errorf("SalesInvoiceRepo.Create: set tenant: %w", err)
	}

	if _, err := tx.ExecContext(ctx, createSalesInvoice, si.ID, si.TenantID, si.SalesOrderID, si.InvoiceNumber, si.Amount, si.Status, si.DueDate, si.CreatedAt, si.UpdatedAt); err != nil {
		return fmt.Errorf("SalesInvoiceRepo.Create: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("SalesInvoiceRepo.Create: commit: %w", err)
	}
	return nil
}

const getSalesInvoiceByID = `
SELECT id, tenant_id, sales_order_id, invoice_number, amount, status, due_date, created_at, updated_at
FROM sales_invoices WHERE id = $1 AND tenant_id = $2`

func (r *SalesInvoiceRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesInvoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getSalesInvoiceByID, id, tenantID)

	var si domain.SalesInvoice
	if err := row.Scan(&si.ID, &si.TenantID, &si.SalesOrderID, &si.InvoiceNumber, &si.Amount, &si.Status, &si.DueDate, &si.CreatedAt, &si.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("SalesInvoiceRepo.GetByID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.GetByID: commit: %w", err)
	}
	return &si, nil
}

const listSalesInvoicesByTenantPaged = `
SELECT id, tenant_id, sales_order_id, invoice_number, amount, status, due_date, created_at, updated_at
FROM sales_invoices WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

func (r *SalesInvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesInvoice, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.List: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listSalesInvoicesByTenantPaged, tenantID, defaultListLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.List: query: %w", err)
	}
	defer rows.Close()

	var invoices []domain.SalesInvoice
	for rows.Next() {
		var si domain.SalesInvoice
		if err := rows.Scan(&si.ID, &si.TenantID, &si.SalesOrderID, &si.InvoiceNumber, &si.Amount, &si.Status, &si.DueDate, &si.CreatedAt, &si.UpdatedAt); err != nil {
			return nil, fmt.Errorf("SalesInvoiceRepo.List: scan row: %w", err)
		}
		invoices = append(invoices, si)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.List: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.List: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SalesInvoiceRepo.List: commit: %w", err)
	}
	return invoices, nil
}
