package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SalesInvoice struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	SalesOrderID uuid.UUID       `json:"sales_order_id"`
	InvoiceNumber string         `json:"invoice_number"`
	Amount       decimal.Decimal `json:"amount"`
	Status       string          `json:"status"`
	DueDate      time.Time       `json:"due_date"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SalesInvoiceRepository interface {
	Create(ctx context.Context, si *SalesInvoice) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*SalesInvoice, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]SalesInvoice, error)
}

type SalesInvoiceCreatedEvent struct {
	EventID        string `json:"event_id"`
	TenantID       string `json:"tenant_id"`
	SalesInvoiceID string `json:"sales_invoice_id"`
	InvoiceNumber  string `json:"invoice_number"`
	Amount         string `json:"amount"` // or number based on PRD, wait PRD says 1500.00 but string is safer for Decimal. Let's make it float64 for simplicity or keep decimal and format as decimal string?
	Currency       string `json:"currency"`
	Timestamp      string `json:"timestamp"`
}
