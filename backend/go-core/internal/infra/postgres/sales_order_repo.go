package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// SalesOrderRepo implements domain.SalesOrderRepository using database/sql with
// parameterized queries. Tenant isolation is enforced at two layers:
//
//  1. Application layer: every query carries WHERE tenant_id = $n.
//  2. Database layer: every operation runs inside a transaction that sets
//     SET LOCAL app.current_tenant_id, activating the RLS policies.
//
// Using transaction-scoped SET LOCAL (rather than SET SESSION) means the
// variable is automatically cleared when the transaction ends, preventing
// any accidental cross-tenant data leakage between connection pool reuse.
type SalesOrderRepo struct {
	db *sql.DB
}

func NewSalesOrderRepo(db *sql.DB) *SalesOrderRepo {
	return &SalesOrderRepo{db: db}
}

const createSalesOrder = `
INSERT INTO sales_orders (id, tenant_id, customer_id, order_number, total_amount, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

func (r *SalesOrderRepo) Create(ctx context.Context, so *domain.SalesOrder) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SalesOrderRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, so.TenantID); err != nil {
		return fmt.Errorf("SalesOrderRepo.Create: set tenant: %w", err)
	}

	if _, err := tx.ExecContext(ctx, createSalesOrder, so.ID, so.TenantID, so.CustomerID, so.OrderNumber, so.TotalAmount, so.Status, so.CreatedAt, so.UpdatedAt); err != nil {
		return fmt.Errorf("SalesOrderRepo.Create: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("SalesOrderRepo.Create: commit: %w", err)
	}
	return nil
}

const getSalesOrderByID = `
SELECT id, tenant_id, customer_id, order_number, total_amount, status, created_at, updated_at
FROM sales_orders WHERE id = $1 AND tenant_id = $2`

func (r *SalesOrderRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getSalesOrderByID, id, tenantID)

	var so domain.SalesOrder
	if err := row.Scan(&so.ID, &so.TenantID, &so.CustomerID, &so.OrderNumber, &so.TotalAmount, &so.Status, &so.CreatedAt, &so.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("SalesOrderRepo.GetByID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.GetByID: commit: %w", err)
	}
	return &so, nil
}

const listSalesOrdersByTenantPaged = `
SELECT id, tenant_id, customer_id, order_number, total_amount, status, created_at, updated_at
FROM sales_orders WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

func (r *SalesOrderRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.List: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listSalesOrdersByTenantPaged, tenantID, defaultListLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.List: query: %w", err)
	}
	defer rows.Close()

	var orders []domain.SalesOrder
	for rows.Next() {
		var so domain.SalesOrder
		if err := rows.Scan(&so.ID, &so.TenantID, &so.CustomerID, &so.OrderNumber, &so.TotalAmount, &so.Status, &so.CreatedAt, &so.UpdatedAt); err != nil {
			return nil, fmt.Errorf("SalesOrderRepo.List: scan row: %w", err)
		}
		orders = append(orders, so)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.List: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.List: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SalesOrderRepo.List: commit: %w", err)
	}
	return orders, nil
}
