package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// ProductCategoryRepo implements domain.ProductCategoryRepository. Like
// ProductRepo, every statement carries tenant_id and runs in a transaction
// with SET LOCAL app.current_tenant_id so the RLS policy on
// product_categories (migration 033) is enforced as a second layer.
type ProductCategoryRepo struct {
	db *sql.DB
}

func NewProductCategoryRepo(db *sql.DB) *ProductCategoryRepo {
	return &ProductCategoryRepo{db: db}
}

// withTenantTx runs fn in a tenant-scoped transaction and commits on success.
func (r *ProductCategoryRepo) withTenantTx(ctx context.Context, op string, tenantID uuid.UUID, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ProductCategoryRepo.%s: begin tx: %w", op, err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("ProductCategoryRepo.%s: set tenant: %w", op, err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ProductCategoryRepo.%s: commit: %w", op, err)
	}
	return nil
}

const listProductCategoriesSQL = `
SELECT c.id, c.tenant_id, c.name, c.created_at,
       (SELECT COUNT(*) FROM products p
         WHERE p.tenant_id = c.tenant_id AND p.category_id = c.id AND p.deleted_at IS NULL)
FROM product_categories c
WHERE c.tenant_id = $1
ORDER BY lower(c.name) ASC`

func (r *ProductCategoryRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.ProductCategory, error) {
	result := make([]domain.ProductCategory, 0)
	err := r.withTenantTx(ctx, "List", tenantID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, listProductCategoriesSQL, tenantID)
		if err != nil {
			return fmt.Errorf("ProductCategoryRepo.List: query: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var c domain.ProductCategory
			if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.CreatedAt, &c.ProductCount); err != nil {
				return fmt.Errorf("ProductCategoryRepo.List: scan: %w", err)
			}
			result = append(result, c)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("ProductCategoryRepo.List: rows: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

const getProductCategorySQL = `
SELECT id, tenant_id, name, created_at
FROM product_categories WHERE id = $1 AND tenant_id = $2`

func (r *ProductCategoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ProductCategory, error) {
	var c domain.ProductCategory
	err := r.withTenantTx(ctx, "GetByID", tenantID, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, getProductCategorySQL, id, tenantID).Scan(&c.ID, &c.TenantID, &c.Name, &c.CreatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("ProductCategoryRepo.GetByID: scan: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &c, nil
}

const createProductCategorySQL = `
INSERT INTO product_categories (id, tenant_id, name, created_at) VALUES ($1, $2, $3, $4)`

func (r *ProductCategoryRepo) Create(ctx context.Context, c *domain.ProductCategory) error {
	return r.withTenantTx(ctx, "Create", c.TenantID, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, createProductCategorySQL, c.ID, c.TenantID, c.Name, c.CreatedAt); err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}
			return fmt.Errorf("ProductCategoryRepo.Create: exec: %w", err)
		}
		return nil
	})
}

const renameProductCategorySQL = `UPDATE product_categories SET name = $1 WHERE id = $2 AND tenant_id = $3`

func (r *ProductCategoryRepo) Rename(ctx context.Context, tenantID, id uuid.UUID, name string) error {
	return r.withTenantTx(ctx, "Rename", tenantID, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, renameProductCategorySQL, name, id, tenantID)
		if err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConflict
			}
			return fmt.Errorf("ProductCategoryRepo.Rename: exec: %w", err)
		}
		return requireOneRow(res, "Rename")
	})
}

const deleteProductCategorySQL = `DELETE FROM product_categories WHERE id = $1 AND tenant_id = $2`

func (r *ProductCategoryRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.withTenantTx(ctx, "Delete", tenantID, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, deleteProductCategorySQL, id, tenantID)
		if err != nil {
			return fmt.Errorf("ProductCategoryRepo.Delete: exec: %w", err)
		}
		return requireOneRow(res, "Delete")
	})
}

func requireOneRow(res sql.Result, op string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("ProductCategoryRepo.%s: rows affected: %w", op, err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
