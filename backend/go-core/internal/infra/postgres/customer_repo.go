package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// CustomerRepo implements domain.CustomerRepository using database/sql with
// parameterized queries. Tenant isolation is enforced at two layers:
//
//  1. Application layer: every query carries WHERE tenant_id = $n.
//  2. Database layer: every operation runs inside a transaction that sets
//     SET LOCAL app.current_tenant_id, activating the RLS policies.
//
// Using transaction-scoped SET LOCAL (rather than SET SESSION) means the
// variable is automatically cleared when the transaction ends, preventing
// any accidental cross-tenant data leakage between connection pool reuse.
type CustomerRepo struct {
	db *sql.DB
}

func NewCustomerRepo(db *sql.DB) *CustomerRepo {
	return &CustomerRepo{db: db}
}

const createCustomer = `
INSERT INTO customers (id, tenant_id, name, email, phone, address, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

func (r *CustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("CustomerRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, c.TenantID); err != nil {
		return fmt.Errorf("CustomerRepo.Create: set tenant: %w", err)
	}

	if _, err := tx.ExecContext(ctx, createCustomer, c.ID, c.TenantID, c.Name, c.Email, c.Phone, c.Address, c.CreatedAt, c.UpdatedAt); err != nil {
		return fmt.Errorf("CustomerRepo.Create: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("CustomerRepo.Create: commit: %w", err)
	}
	return nil
}

const getCustomerByID = `
SELECT id, tenant_id, name, email, phone, address, created_at, updated_at
FROM customers WHERE id = $1 AND tenant_id = $2`

func (r *CustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("CustomerRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("CustomerRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getCustomerByID, id, tenantID)

	var c domain.Customer
	if err := row.Scan(&c.ID, &c.TenantID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("CustomerRepo.GetByID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("CustomerRepo.GetByID: commit: %w", err)
	}
	return &c, nil
}

const defaultListLimit = 1000

const listCustomersByTenantPaged = `
SELECT id, tenant_id, name, email, phone, address, created_at, updated_at
FROM customers WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

func (r *CustomerRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Customer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("CustomerRepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("CustomerRepo.List: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listCustomersByTenantPaged, tenantID, defaultListLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("CustomerRepo.List: query: %w", err)
	}
	defer rows.Close()

	var customers []domain.Customer
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("CustomerRepo.List: scan row: %w", err)
		}
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CustomerRepo.List: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("CustomerRepo.List: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("CustomerRepo.List: commit: %w", err)
	}
	return customers, nil
}
