package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type POStatus string

const (
	POStatusOpen              POStatus = "open"
	POStatusPartiallyReceived POStatus = "partially_received"
	POStatusReceived          POStatus = "received"
	POStatusClosed            POStatus = "closed"
	POStatusCancelled         POStatus = "cancelled"
)

type GRStatus string

const (
	GRStatusPending  GRStatus = "pending"
	GRStatusAccepted GRStatus = "accepted"
	GRStatusRejected GRStatus = "rejected"
)

type MatchResult string

const (
	MatchResultMatched        MatchResult = "matched"
	MatchResultAmountMismatch MatchResult = "amount_mismatch"
	MatchResultQtyMismatch    MatchResult = "qty_mismatch"
	MatchResultNoPO           MatchResult = "no_po"
	MatchResultNoGR           MatchResult = "no_gr"
	MatchResultPending        MatchResult = "pending"
)

type PurchaseOrder struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	VendorID  string          `json:"vendor_id"`
	PONumber  string          `json:"po_number"`
	Amount    decimal.Decimal `json:"amount"`
	Qty       int             `json:"qty"`
	Currency  string          `json:"currency"`
	Status    POStatus        `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type GoodsReceipt struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	POID           uuid.UUID       `json:"po_id"`
	VendorID       string          `json:"vendor_id"`
	ReceivedQty    int             `json:"received_qty"`
	ReceivedAmount decimal.Decimal `json:"received_amount"`
	Currency       string          `json:"currency"`
	Status         GRStatus        `json:"status"`
	ReceivedAt     time.Time       `json:"received_at"`
	CreatedAt      time.Time       `json:"created_at"`
}

type MatchRepository interface {
	GetPOByID(ctx context.Context, id, tenantID uuid.UUID) (*PurchaseOrder, error)
	GetGRByPOID(ctx context.Context, poID, tenantID uuid.UUID) (*GoodsReceipt, error)
	UpdateInvoiceMatchResult(ctx context.Context, invoiceID, tenantID uuid.UUID, poID *uuid.UUID, result MatchResult, status InvoiceStatus) error
}

type InvoiceApprovedEvent struct {
	InvoiceID   string `json:"invoice_id"`
	TenantID    string `json:"tenant_id"`
	Amount      string `json:"amount"`
	MatchResult string `json:"match_result"`
}

type InvoicePendingReviewEvent struct {
	InvoiceID   string `json:"invoice_id"`
	TenantID    string `json:"tenant_id"`
	Amount      string `json:"amount"`
	MatchResult string `json:"match_result"`
	Reason      string `json:"reason"`
}

// InvoiceRejectedEvent is the dedicated Kafka payload for the invoice.rejected topic.
// Using a dedicated struct (instead of re-using InvoiceApprovedEvent) gives
// consumers an unambiguous, stable contract for rejection events.
type InvoiceRejectedEvent struct {
	InvoiceID string `json:"invoice_id"`
	TenantID  string `json:"tenant_id"`
	Amount    string `json:"amount"`
}
