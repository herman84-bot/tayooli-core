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
INSERT INTO products (id, tenant_id, name, description, sku, price, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

func (r *ProductRepo) Create(ctx context.Context, p *domain.Product) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ProductRepo.Create: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, p.TenantID); err != nil {
		return fmt.Errorf("ProductRepo.Create: set tenant: %w", err)
	}

	if _, err := tx.ExecContext(ctx, createProduct, p.ID, p.TenantID, p.Name, p.Description, p.SKU, p.Price, p.CreatedAt, p.UpdatedAt); err != nil {
		return fmt.Errorf("ProductRepo.Create: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ProductRepo.Create: commit: %w", err)
	}
	return nil
}

const getProductByID = `
SELECT id, tenant_id, name, COALESCE(description, ''), sku, price, created_at, updated_at
FROM products WHERE id = $1 AND tenant_id = $2`

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
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CreatedAt, &p.UpdatedAt); err != nil {
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
SELECT id, tenant_id, name, COALESCE(description, ''), sku, price, created_at, updated_at
FROM products WHERE tenant_id = $1
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
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CreatedAt, &p.UpdatedAt); err != nil {
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
SELECT id, tenant_id, name, COALESCE(description, ''), sku, price, created_at, updated_at
FROM products WHERE tenant_id = $1 AND id IN (%s)`, strings.Join(placeholders, ", "))

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
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CreatedAt, &p.UpdatedAt); err != nil {
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
SELECT id, tenant_id, name, COALESCE(description, ''), sku, price, created_at, updated_at
FROM products WHERE tenant_id = $1 AND LOWER(TRIM(sku)) = LOWER(TRIM($2))`

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
	if err := row.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.SKU, &p.Price, &p.CreatedAt, &p.UpdatedAt); err != nil {
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
SET name = $1, description = $2, sku = $3, price = $4, updated_at = $5
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

	res, err := tx.ExecContext(ctx, updateProduct, p.Name, p.Description, p.SKU, p.Price, p.UpdatedAt, p.ID, p.TenantID)
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

const deleteProduct = `DELETE FROM products WHERE id = $1 AND tenant_id = $2`

func (r *ProductRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ProductRepo.Delete: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("ProductRepo.Delete: set tenant: %w", err)
	}

	// Also clean up 0-quantity inventory rows if any exist
	if _, err := tx.ExecContext(ctx, `DELETE FROM inventory WHERE product_id = $1 AND tenant_id = $2`, id, tenantID); err != nil {
		return fmt.Errorf("ProductRepo.Delete: clean inventory: %w", err)
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
