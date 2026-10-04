package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*Account, error)
	GetByCode(ctx context.Context, code string, tenantID uuid.UUID) (*Account, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]*Account, error)
}
