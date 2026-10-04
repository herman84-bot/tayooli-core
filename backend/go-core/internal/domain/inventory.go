package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Inventory struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	ProductID         uuid.UUID `json:"product_id"`
	Quantity          float64   `json:"quantity"`
	WarehouseLocation string    `json:"warehouse_location,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type InventoryRepository interface {
	Create(ctx context.Context, i *Inventory) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Inventory, error)
	GetByProductID(ctx context.Context, tenantID, productID uuid.UUID) ([]Inventory, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]Inventory, error)
	UpdateQuantity(ctx context.Context, tenantID, id uuid.UUID, delta float64) error
}
