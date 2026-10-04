package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	Address   string    `json:"address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CustomerRepository interface {
	Create(ctx context.Context, c *Customer) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]Customer, error)
}
