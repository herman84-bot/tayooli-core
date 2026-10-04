package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TenantPaymentConfig stores the gateway credentials per tenant.
type TenantPaymentConfig struct {
	ID                    uuid.UUID       `json:"id"`
	TenantID              uuid.UUID       `json:"tenant_id"`
	Provider              string          `json:"provider"`
	Slug                  *string         `json:"slug,omitempty"`
	APIKey                *string         `json:"api_key,omitempty"`
	ServerKey             *string         `json:"server_key,omitempty"`
	ClientKey             *string         `json:"client_key,omitempty"`
	IsProduction          bool            `json:"is_production"`
	IsActive              bool            `json:"is_active"`
	SettlementBankName    *string         `json:"settlement_bank_name,omitempty"`
	SettlementBankAccount *string         `json:"settlement_bank_account,omitempty"`
	SettlementHolderName  *string         `json:"settlement_holder_name,omitempty"`
	GatewayFeePercent     decimal.Decimal `json:"gateway_fee_percent"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// PaymentTransaction represents an individual payment gateway checkout/transaction.
type PaymentTransaction struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Provider    string          `json:"provider"`
	InvoiceID   string          `json:"invoice_id"`
	OrderID     string          `json:"order_id"`
	Amount      decimal.Decimal `json:"amount"`
	Method      string          `json:"method"`
	PaymentLink *string         `json:"payment_link,omitempty"`
	Status      string          `json:"status"` // pending, completed, failed, expired
	IsDemo      bool            `json:"is_demo"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// PaymentGatewayRepository defines the persistence port for gateway configs & transactions.
type PaymentGatewayRepository interface {
	GetConfig(ctx context.Context, tenantID uuid.UUID, provider string) (*TenantPaymentConfig, error)
	UpsertConfig(ctx context.Context, cfg *TenantPaymentConfig) error
	CreateTransaction(ctx context.Context, tx *PaymentTransaction) error
	GetTransactionByOrderID(ctx context.Context, tenantID uuid.UUID, orderID string) (*PaymentTransaction, error)
	UpdateTransactionStatus(ctx context.Context, tenantID uuid.UUID, orderID, status string, completedAt *time.Time) error
}
