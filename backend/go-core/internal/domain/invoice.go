package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type InvoiceStatus string

const (
	StatusPending       InvoiceStatus = "pending"
	StatusApproved      InvoiceStatus = "approved"
	StatusRejected      InvoiceStatus = "rejected"
	StatusPendingReview InvoiceStatus = "pending_review"
)

type Invoice struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	VendorID          string          `json:"vendor_id"`
	InvoiceNumber     string          `json:"invoice_number"`
	Amount            decimal.Decimal `json:"amount"`
	Currency          string          `json:"currency"`
	Status            InvoiceStatus   `json:"status"`
	AIConfidenceScore float64         `json:"ai_confidence_score"`
	POID              *uuid.UUID      `json:"po_id,omitempty"`
	MatchResult       *string         `json:"match_result,omitempty"`
	DueDate           *time.Time      `json:"due_date,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// InvoiceCreatedEvent is the dedicated Kafka payload for the invoice.created topic.
// Using a dedicated struct (instead of marshalling Invoice directly) gives Python
// a stable, snake_case contract that won't break if the domain struct evolves.
// Amount is serialized as a JSON string (e.g. "12345.6789") to prevent float
// truncation in transit — Python must treat it as a string, never cast to float.
type InvoiceCreatedEvent struct {
	InvoiceID        string `json:"invoice_id"`
	TenantID         string `json:"tenant_id"`
	Amount           string `json:"amount"`
	Status           string `json:"status"`
	RawDocumentBytes string `json:"raw_document_bytes,omitempty"` // base64-encoded, optional
	MimeType         string `json:"mime_type,omitempty"`
}

type CreateInvoiceParams struct {
	TenantID      uuid.UUID
	VendorID      string
	InvoiceNumber string
	Amount        decimal.Decimal
	Currency      string
	POID          *uuid.UUID
	DueDate       *time.Time
	// CreatedBy is the authenticated user who created the invoice. It is set
	// server-side from the JWT context and used (when present) as the
	// requested_by of auto-created approval requests.
	CreatedBy *uuid.UUID
}

// InvoiceListPage holds a paginated result set plus total count.
type InvoiceListPage struct {
	Data    []Invoice `json:"data"`
	Total   int       `json:"total"`
	Page    int       `json:"page"`
	PerPage int       `json:"per_page"`
}

type InvoiceRepository interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]Invoice, error)
	ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*InvoiceListPage, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Invoice, error)
	Create(ctx context.Context, params CreateInvoiceParams) (*Invoice, error)
	Approve(ctx context.Context, id, tenantID uuid.UUID) (*Invoice, error)
	Reject(ctx context.Context, id, tenantID uuid.UUID) (*Invoice, error)
	// UpdateOCRResult persists the OCR pipeline output for an invoice.
	// tenantID is placed before invoiceID to match the tenant-first convention
	// used throughout the repository layer.
	UpdateOCRResult(ctx context.Context, tenantID, invoiceID uuid.UUID, score float64, extractedText, status string) error
}
