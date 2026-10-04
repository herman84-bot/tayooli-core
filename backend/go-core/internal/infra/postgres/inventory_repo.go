package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// InventoryRepo implements domain.InventoryRepository using database/sql with
// parameterized queries. Tenant isolation is enforced at two layers:
//
//  1. Application layer: every query carries WHERE tenant_id = $n.
//  2. Database layer: every operation runs inside a transaction that sets
//     SET LOCAL app.current_tenant_id, activating the RLS policies.
//
// Using transaction-scoped SET LOCAL (rather than SET SESSION) means the
// variable is automatically cleared when the transaction ends, preventing
// any accidental cross-tenant data leakage between connection pool reuse.
type InventoryRepo struct {
	db *sql.DB
}

func NewInventoryRepo(db *sql.DB) *InventoryRepo {
	return &InventoryRepo{db: db}
}

const createInventory = `
INSERT INTO inventory (id, tenant_id, product_id, quantity, warehouse_location, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

func (r *InventoryRepo) Create(ctx context.Context, i *domain.Inventory) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("InventoryRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, i.TenantID); err != nil {
		return fmt.Errorf("InventoryRepo.Create: set tenant: %w", err)
	}

	if _, err := tx.ExecContext(ctx, createInventory, i.ID, i.TenantID, i.ProductID, i.Quantity, i.WarehouseLocation, i.CreatedAt, i.UpdatedAt); err != nil {
		return fmt.Errorf("InventoryRepo.Create: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("InventoryRepo.Create: commit: %w", err)
	}
	return nil
}

const getInventoryByID = `
SELECT id, tenant_id, product_id, quantity, warehouse_location, created_at, updated_at
FROM inventory WHERE id = $1 AND tenant_id = $2`

func (r *InventoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Inventory, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getInventoryByID, id, tenantID)

	var i domain.Inventory
	if err := row.Scan(&i.ID, &i.TenantID, &i.ProductID, &i.Quantity, &i.WarehouseLocation, &i.CreatedAt, &i.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("InventoryRepo.GetByID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByID: commit: %w", err)
	}
	return &i, nil
}

const getInventoryByProductID = `
SELECT id, tenant_id, product_id, quantity, warehouse_location, created_at, updated_at
FROM inventory WHERE product_id = $1 AND tenant_id = $2`

func (r *InventoryRepo) GetByProductID(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.Inventory, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByProductID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByProductID: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, getInventoryByProductID, productID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByProductID: query: %w", err)
	}
	defer rows.Close()

	var items []domain.Inventory
	for rows.Next() {
		var i domain.Inventory
		if err := rows.Scan(&i.ID, &i.TenantID, &i.ProductID, &i.Quantity, &i.WarehouseLocation, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("InventoryRepo.GetByProductID: scan row: %w", err)
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByProductID: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByProductID: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InventoryRepo.GetByProductID: commit: %w", err)
	}
	return items, nil
}

const listInventoryByTenantPaged = `
SELECT id, tenant_id, product_id, quantity, warehouse_location, created_at, updated_at
FROM inventory WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

func (r *InventoryRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Inventory, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("InventoryRepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("InventoryRepo.List: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listInventoryByTenantPaged, tenantID, defaultListLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("InventoryRepo.List: query: %w", err)
	}
	defer rows.Close()

	var items []domain.Inventory
	for rows.Next() {
		var i domain.Inventory
		if err := rows.Scan(&i.ID, &i.TenantID, &i.ProductID, &i.Quantity, &i.WarehouseLocation, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("InventoryRepo.List: scan row: %w", err)
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("InventoryRepo.List: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("InventoryRepo.List: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("InventoryRepo.List: commit: %w", err)
	}
	return items, nil
}

const updateInventoryQuantity = `
UPDATE inventory
SET quantity = quantity + $1, updated_at = NOW()
WHERE id = $2 AND tenant_id = $3`

func (r *InventoryRepo) UpdateQuantity(ctx context.Context, tenantID, id uuid.UUID, delta float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("InventoryRepo.UpdateQuantity: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("InventoryRepo.UpdateQuantity: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, updateInventoryQuantity, delta, id, tenantID)
	if err != nil {
		return fmt.Errorf("InventoryRepo.UpdateQuantity: exec: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("InventoryRepo.UpdateQuantity: rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("InventoryRepo.UpdateQuantity: commit: %w", err)
	}
	return nil
}
