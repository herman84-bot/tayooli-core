package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	SKU         string    `json:"sku"`
	Price       float64   `json:"price"`
	// CostPrice is the purchase/standard cost used as unit_cost on stock
	// movements (M6). 0 means not set.
	CostPrice float64 `json:"cost_price"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// InventoryItemWithProduct is returned by ListInventoryFromWMS repo method.
// Aggregates WMS stock per product with product master data.
type InventoryItemWithProduct struct {
	ProductID     uuid.UUID
	ProductName   string
	SKU           string
	Price         float64
	TotalQuantity float64
}

type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Product, error)
	GetBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*Product, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]Product, error)
	// ListByIDs returns the products whose IDs are in the given set — used by
	// the inventory listing to enrich inventory rows with product details.
	ListByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	HasMovementsOrStock(ctx context.Context, tenantID, id uuid.UUID) (bool, error)
	// ListInventoryFromWMS aggregates qty per product from stock_movements WMS ledger.
	// Returns product name, SKU, price, and total qty from DONE movements to INTERNAL locations.
	ListInventoryFromWMS(ctx context.Context, tenantID uuid.UUID) ([]InventoryItemWithProduct, error)
}
