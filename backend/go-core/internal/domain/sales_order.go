package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SalesOrder struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	CustomerID  uuid.UUID       `json:"customer_id"`
	OrderNumber string          `json:"order_number"`
	TotalAmount decimal.Decimal `json:"total_amount"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type SalesOrderRepository interface {
	Create(ctx context.Context, so *SalesOrder) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*SalesOrder, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]SalesOrder, error)
}
