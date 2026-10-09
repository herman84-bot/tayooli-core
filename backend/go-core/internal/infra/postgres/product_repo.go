package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// ProductRepo implements domain.ProductRepository using database/sql with
// parameterized queries. Tenant isolation is enforced at two layers:
//
//  1. Application layer: every query carries WHERE tenant_id = $n.
//  2. Database layer: every operation runs inside a transaction that sets
//     SET LOCAL app.current_tenant_id, activating the RLS policies.
//
// Using transaction-scoped SET LOCAL (rather than SET SESSION) means the
// variable is automatically cleared when the transaction ends, preventing
// any accidental cross-tenant data leakage between connection pool reuse.
type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

const createProduct = `
INSERT INTO products (id, tenant_id, name, description, sku, price, cost_price, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

func (r *ProductRepo) Create(ctx context.Context, p *domain.Product) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ProductRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, p.TenantID); err != nil {
		return fmt.Errorf("ProductRepo.Create: set tenant: %w", err)
	}

	if _, err := tx.ExecContext(ctx, createProduct, p.ID, p.TenantID, p.Name, p.Description, p.SKU, p.Price, p.CostPrice, p.CreatedAt, p.UpdatedAt); err != nil {
		return fmt.Errorf("ProductRepo.Create: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ProductRepo.Create: commit: %w", err)
	}
	return nil
}

const getProductByID = `
SELECT id, tenant_id, name, COALESCE(description, ''), sku, price, cost_price, created_at, updated_at
FROM products WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

func (r *ProductRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ProductRepo.GetByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("ProductRepo.GetByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getProductByID, id, tenantID)

	var p domain.Product
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CostPrice, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("ProductRepo.GetByID: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ProductRepo.GetByID: commit: %w", err)
	}
	return &p, nil
}

const listProductsByTenantPaged = `
SELECT id, tenant_id, name, COALESCE(description, ''), sku, price, cost_price, created_at, updated_at
FROM products WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

func (r *ProductRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Product, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ProductRepo.List: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("ProductRepo.List: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listProductsByTenantPaged, tenantID, defaultListLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("ProductRepo.List: query: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CostPrice, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ProductRepo.List: scan row: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ProductRepo.List: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("ProductRepo.List: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ProductRepo.List: commit: %w", err)
	}
	return products, nil
}

// listProductsByIDs uses a single IN (...) query to fetch the product details
// for a set of inventory rows. The placeholder list is built dynamically, but
// all values are bound as parameters — no string interpolation of user input.
func (r *ProductRepo) ListByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]domain.Product, error) {
	if len(ids) == 0 {
		return []domain.Product{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, tenantID)
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	query := fmt.Sprintf(`
SELECT id, tenant_id, name, COALESCE(description, ''), sku, price, cost_price, created_at, updated_at
FROM products WHERE tenant_id = $1 AND id IN (%s) AND deleted_at IS NULL`, strings.Join(placeholders, ", "))

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ProductRepo.ListByIDs: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("ProductRepo.ListByIDs: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ProductRepo.ListByIDs: query: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CostPrice, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ProductRepo.ListByIDs: scan row: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ProductRepo.ListByIDs: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("ProductRepo.ListByIDs: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ProductRepo.ListByIDs: commit: %w", err)
	}
	return products, nil
}

const getProductBySKU = `
SELECT id, tenant_id, name, COALESCE(description, ''), sku, price, cost_price, created_at, updated_at
FROM products WHERE tenant_id = $1 AND LOWER(TRIM(sku)) = LOWER(TRIM($2)) AND deleted_at IS NULL`

func (r *ProductRepo) GetBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.Product, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ProductRepo.GetBySKU: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("ProductRepo.GetBySKU: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getProductBySKU, tenantID, sku)
	var p domain.Product
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CostPrice, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("ProductRepo.GetBySKU: scan row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ProductRepo.GetBySKU: commit: %w", err)
	}
	return &p, nil
}

const updateProduct = `
UPDATE products
SET name = $1, description = $2, sku = $3, price = $4, updated_at = $5, cost_price = $8
WHERE id = $6 AND tenant_id = $7`

func (r *ProductRepo) Update(ctx context.Context, p *domain.Product) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ProductRepo.Update: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, p.TenantID); err != nil {
		return fmt.Errorf("ProductRepo.Update: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, updateProduct, p.Name, p.Description, p.SKU, p.Price, p.UpdatedAt, p.ID, p.TenantID, p.CostPrice)
	if err != nil {
		return fmt.Errorf("ProductRepo.Update: exec: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ProductRepo.Update: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ProductRepo.Update: commit: %w", err)
	}
	return nil
}

const hasMovementsOrStockSQL = `
SELECT (
    EXISTS (SELECT 1 FROM inventory WHERE tenant_id = $1 AND product_id = $2 AND quantity > 0)
    OR
    EXISTS (SELECT 1 FROM stock_movements WHERE tenant_id = $1 AND product_id = $2)
)`

func (r *ProductRepo) HasMovementsOrStock(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("ProductRepo.HasMovementsOrStock: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return false, fmt.Errorf("ProductRepo.HasMovementsOrStock: set tenant: %w", err)
	}

	var hasMovements bool
	if err := tx.QueryRowContext(ctx, hasMovementsOrStockSQL, tenantID, id).Scan(&hasMovements); err != nil {
		return false, fmt.Errorf("ProductRepo.HasMovementsOrStock: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("ProductRepo.HasMovementsOrStock: commit: %w", err)
	}
	return hasMovements, nil
}

// ListInventoryFromWMS aggregates qty_on_hand per product from stock_movements ledger
// (WMS single source of truth), joining product details. Only counts DONE movements
// on INTERNAL locations. Excludes soft-deleted products.
const listInventoryFromWMS = `
WITH movement_stock AS (
    SELECT 
        sm.product_id,
        SUM(CASE WHEN sm.dest_location_id = loc.id THEN sm.quantity ELSE -sm.quantity END) AS qty
    FROM stock_movements sm
    JOIN warehouse_locations loc ON (loc.id = sm.dest_location_id OR loc.id = sm.source_location_id) AND loc.tenant_id = sm.tenant_id
    WHERE sm.tenant_id = $1 AND sm.status = 'DONE' AND loc.type = 'INTERNAL'
    GROUP BY sm.product_id
)
SELECT 
    p.id,
    p.name,
    p.sku,
    p.price,
    COALESCE(ms.qty, 0) AS total_qty
FROM products p
LEFT JOIN movement_stock ms ON p.id = ms.product_id
WHERE p.tenant_id = $1 AND p.deleted_at IS NULL
ORDER BY p.name ASC`

func (r *ProductRepo) ListInventoryFromWMS(ctx context.Context, tenantID uuid.UUID) ([]domain.InventoryItemWithProduct, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ProductRepo.ListInventoryFromWMS: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("ProductRepo.ListInventoryFromWMS: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listInventoryFromWMS, tenantID)
	if err != nil {
		return nil, fmt.Errorf("ProductRepo.ListInventoryFromWMS: query: %w", err)
	}
	defer rows.Close()

	var items []domain.InventoryItemWithProduct
	for rows.Next() {
		var item domain.InventoryItemWithProduct
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.SKU, &item.Price, &item.TotalQuantity); err != nil {
			return nil, fmt.Errorf("ProductRepo.ListInventoryFromWMS: scan row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ProductRepo.ListInventoryFromWMS: rows err: %w", err)
	}

	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("ProductRepo.ListInventoryFromWMS: close rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("ProductRepo.ListInventoryFromWMS: commit: %w", err)
	}
	return items, nil
}

const deleteProduct = `UPDATE products SET deleted_at = NOW() WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

func (r *ProductRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ProductRepo.Delete: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("ProductRepo.Delete: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, deleteProduct, id, tenantID)
	if err != nil {
		return fmt.Errorf("ProductRepo.Delete: exec: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ProductRepo.Delete: rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ProductRepo.Delete: commit: %w", err)
	}
	return nil
}
