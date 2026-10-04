package ai

import (
	"context"

	aiclient "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/ai"
)

// AIClient abstracts the AI service client for testability.
type AIClient interface {
	Ingest(ctx context.Context, req aiclient.IngestRequest) (*aiclient.IngestResponse, error)
	GetStatus(ctx context.Context, invoiceID, tenantID string) (*aiclient.StatusResponse, error)
}

// Usecase orchestrates AI inference operations for invoices.
type Usecase struct {
	aiClient AIClient
}

// New creates a new AI usecase with the given AI client.
func New(aiClient AIClient) *Usecase {
	return &Usecase{aiClient: aiClient}
}

// TriggerInference submits an invoice for AI-powered anomaly detection
// and GL account suggestion. Used by the invoice handler after OCR extraction.
func (u *Usecase) TriggerInference(ctx context.Context, invoiceID, tenantID, amount, vendorID, extractedText string) (*aiclient.IngestResponse, error) {
	return u.aiClient.Ingest(ctx, aiclient.IngestRequest{
		InvoiceID:     invoiceID,
		TenantID:      tenantID,
		Amount:        amount,
		VendorID:      vendorID,
		ExtractedText: extractedText,
	})
}

// GetStatus retrieves the current AI inference status for an invoice.
func (u *Usecase) GetStatus(ctx context.Context, invoiceID, tenantID string) (*aiclient.StatusResponse, error) {
	return u.aiClient.GetStatus(ctx, invoiceID, tenantID)
}
