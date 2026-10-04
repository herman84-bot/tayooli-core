package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PaymentOrderStatus represents the lifecycle of a payment order.
// Allowed transitions:
//
//	draft -> approved (approve usecase)
//	draft -> rejected (reject usecase)
//	approved -> paid   (pay usecase, also marks invoice as paid)
type PaymentOrderStatus string

const (
	PaymentOrderStatusDraft           PaymentOrderStatus = "draft"
	PaymentOrderStatusPendingApproval PaymentOrderStatus = "pending_approval"
	PaymentOrderStatusApproved        PaymentOrderStatus = "approved"
	PaymentOrderStatusPaid            PaymentOrderStatus = "paid"
	PaymentOrderStatusRejected        PaymentOrderStatus = "rejected"
	PaymentOrderStatusCancelled       PaymentOrderStatus = "cancelled"
)

// PaymentOrder links an approved invoice to a payment execution.
type PaymentOrder struct {
	ID             uuid.UUID          `json:"id"`
	TenantID       uuid.UUID          `json:"tenant_id"`
	InvoiceID      uuid.UUID          `json:"invoice_id"`
	Amount         decimal.Decimal    `json:"amount"`
	Currency       string             `json:"currency"`
	PaymentMethod  string             `json:"payment_method"`
	ReferenceNumber *string           `json:"reference_number,omitempty"`
	Status         PaymentOrderStatus `json:"status"`
	Notes          *string            `json:"notes,omitempty"`
	CreatedBy      *uuid.UUID         `json:"created_by,omitempty"`
	ApprovedBy     *uuid.UUID         `json:"approved_by,omitempty"`
	PaidAt         *time.Time         `json:"paid_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// CreatePaymentOrderParams are the inputs for creating a new payment order.
type CreatePaymentOrderParams struct {
	TenantID        uuid.UUID
	InvoiceID       uuid.UUID
	Amount          decimal.Decimal
	Currency        string
	PaymentMethod   string
	ReferenceNumber *string
	Notes           *string
	CreatedBy       *uuid.UUID
}

// PaymentOrderListPage holds a paginated result set plus total count.
type PaymentOrderListPage struct {
	Data    []PaymentOrder `json:"data"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
}

// PaymentOrderRepository is the persistence port for payment orders.
type PaymentOrderRepository interface {
	Create(ctx context.Context, params CreatePaymentOrderParams) (*PaymentOrder, error)
	ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*PaymentOrderListPage, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*PaymentOrder, error)
	UpdateStatus(ctx context.Context, id, tenantID uuid.UUID, from, to PaymentOrderStatus, approvedBy *uuid.UUID) (*PaymentOrder, error)
	MarkInvoicePaid(ctx context.Context, paymentOrderID, tenantID uuid.UUID) error
}

// Kafka event payloads for payment order transitions.
// All amounts are strings to preserve NUMERIC precision in transit.

type PaymentOrderCreatedEvent struct {
	PaymentOrderID string `json:"payment_order_id"`
	TenantID       string `json:"tenant_id"`
	InvoiceID      string `json:"invoice_id"`
	Amount         string `json:"amount"`
	Status         string `json:"status"`
}

type PaymentOrderApprovedEvent struct {
	PaymentOrderID string `json:"payment_order_id"`
	TenantID       string `json:"tenant_id"`
	InvoiceID      string `json:"invoice_id"`
	Amount         string `json:"amount"`
	ApprovedBy     string `json:"approved_by"`
}

type PaymentOrderPaidEvent struct {
	PaymentOrderID string `json:"payment_order_id"`
	TenantID       string `json:"tenant_id"`
	InvoiceID      string `json:"invoice_id"`
	Amount         string `json:"amount"`
	PaidAt         string `json:"paid_at"`
}

type PaymentOrderRejectedEvent struct {
	PaymentOrderID string `json:"payment_order_id"`
	TenantID       string `json:"tenant_id"`
	InvoiceID      string `json:"invoice_id"`
	Amount         string `json:"amount"`
}

// TelegramPaymentOrder represents a payment order initiated via Telegram bot.
type TelegramPaymentOrder struct {
	ID        uuid.UUID `json:"id"`
	OrderCode string    `json:"order_code"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserEmail string    `json:"user_email"`
	Plan      string    `json:"plan"`
	Amount    int       `json:"amount"`
	Period    string    `json:"period"`
	Status    string    `json:"status"`
}
