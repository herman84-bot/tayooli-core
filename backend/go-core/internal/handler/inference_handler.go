package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	aiclient "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/ai"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// inferenceUsecase groups the operations the handler needs.
type inferenceUsecase interface {
	TriggerInference(ctx context.Context, invoiceID, tenantID, amount, vendorID, extractedText string) (*aiclient.IngestResponse, error)
	GetStatus(ctx context.Context, invoiceID, tenantID string) (*aiclient.StatusResponse, error)
}

// InferenceHandler handles HTTP requests for AI inference operations.
type InferenceHandler struct {
	uc inferenceUsecase
}

// NewInferenceHandler creates a new InferenceHandler.
func NewInferenceHandler(uc inferenceUsecase) *InferenceHandler {
	return &InferenceHandler{uc: uc}
}

// ── Request / response types ────────────────────────────────────────────────

type ingestRequest struct {
	InvoiceID     string `json:"invoice_id"`
	Amount        string `json:"amount"`
	VendorID      string `json:"vendor_id"`
	ExtractedText string `json:"extracted_text"`
}

type ingestResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type statusResponse struct {
	InvoiceID          string  `json:"invoice_id"`
	Status             string  `json:"status"`
	AnomalyScore       float64 `json:"anomaly_score,omitempty"`
	SuggestedGLAccount string  `json:"suggested_gl_account,omitempty"`
	Error              string  `json:"error,omitempty"`
}

// ── Ingest ──────────────────────────────────────────────────────────────────

// Ingest handles POST /api/v1/inference/ingest.
// Triggers AI-powered analysis (anomaly detection + GL suggestion) for an invoice.
// Returns 202 Accepted with a job_id for async tracking.
func (h *InferenceHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.InvoiceID == "" {
		respondError(w, r, http.StatusBadRequest, "invoice_id is required")
		return
	}
	if _, err := uuid.Parse(req.InvoiceID); err != nil {
		respondError(w, r, http.StatusBadRequest, "invoice_id must be a valid UUID")
		return
	}
	if req.VendorID == "" {
		respondError(w, r, http.StatusBadRequest, "vendor_id is required")
		return
	}
	if req.Amount == "" {
		respondError(w, r, http.StatusBadRequest, "amount is required")
		return
	}

	result, err := h.uc.TriggerInference(r.Context(), req.InvoiceID, tenantID.String(), req.Amount, req.VendorID, req.ExtractedText)
	if err != nil {
		log.Error().Err(err).Str("handler", "Ingest").Str("invoice_id", req.InvoiceID).Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "failed to submit inference request")
		return
	}

	respondJSON(w, http.StatusAccepted, ingestResponse{
		JobID:  result.JobID,
		Status: result.Status,
	})
}

// ── GetStatus ───────────────────────────────────────────────────────────────

// GetStatus handles GET /api/v1/inference/status/{id}.
// Returns the current AI inference status for an invoice, including
// anomaly score and suggested GL account when available.
func (h *InferenceHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	invoiceID, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid invoice id")
		return
	}

	result, err := h.uc.GetStatus(r.Context(), invoiceID.String(), tenantID.String())
	if err != nil {
		log.Error().Err(err).Str("handler", "GetStatus").Str("invoice_id", invoiceID.String()).Msg("usecase failed")
		respondError(w, r, http.StatusNotFound, "inference not found")
		return
	}

	respondJSON(w, http.StatusOK, statusResponse{
		InvoiceID:          result.InvoiceID,
		Status:             result.Status,
		AnomalyScore:       result.AnomalyScore,
		SuggestedGLAccount: result.SuggestedGLAccount,
		Error:              result.Error,
	})
}
