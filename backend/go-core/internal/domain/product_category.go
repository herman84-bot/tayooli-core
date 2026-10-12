package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ProductCategory groups products for filtering and reporting (CR-01).
// Names are unique per tenant, case-insensitively (uq_product_categories_name).
type ProductCategory struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	// ProductCount is the number of active (not deleted) products in the
	// category. Read-only; filled by List.
	ProductCount int `json:"product_count"`
}

// ProductCategoryMaxNameLen mirrors product_categories.name VARCHAR(100).
const ProductCategoryMaxNameLen = 100

type ProductCategoryRepository interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]ProductCategory, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*ProductCategory, error)
	// Create returns ErrConflict when the name already exists (case-insensitive).
	Create(ctx context.Context, c *ProductCategory) error
	// Rename returns ErrNotFound / ErrConflict.
	Rename(ctx context.Context, tenantID, id uuid.UUID, name string) error
	// Delete removes the category; its products become uncategorized
	// (FK ON DELETE SET NULL). Returns ErrNotFound when absent.
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}
