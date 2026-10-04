package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/invoice"
)

// invoiceUsecase groups the operations the handler needs.
type invoiceUsecase interface {
	ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]domain.Invoice, error)
	ListInvoicesPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error)
	GetInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	CreateInvoice(ctx context.Context, params domain.CreateInvoiceParams) (*domain.Invoice, error)
	ApproveInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	RejectInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	AutoApproveInvoice(ctx context.Context, id, tenantID uuid.UUID) (*invoice.AutoApproveResult, error)
	RequeueInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
}

// InvoiceHandler handles HTTP requests for invoice resources.
type InvoiceHandler struct {
	uc invoiceUsecase
}

func NewInvoiceHandler(uc invoiceUsecase) *InvoiceHandler {
	return &InvoiceHandler{uc: uc}
}

// invoiceView is the JSON representation returned to clients.
// Amount is a string (e.g. "12345.6789") to preserve full NUMERIC(20,4) precision
// without float truncation — clients must parse it as a decimal, not a float.
type invoiceView struct {
	ID                string     `json:"id"`
	TenantID          string     `json:"tenant_id"`
	VendorID          string     `json:"vendor_id"`
	InvoiceNumber     string     `json:"invoice_number"`
	Amount            string     `json:"amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	AIConfidenceScore float64    `json:"ai_confidence_score"`
	POID              *string    `json:"po_id,omitempty"`
	MatchResult       *string    `json:"match_result,omitempty"`
	DueDate           *time.Time `json:"due_date,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func toView(inv *domain.Invoice) invoiceView {
	v := invoiceView{
		ID:                inv.ID.String(),
		TenantID:          inv.TenantID.String(),
		VendorID:          inv.VendorID,
		InvoiceNumber:     inv.InvoiceNumber,
		Amount:            inv.Amount.String(),
		Currency:          inv.Currency,
		Status:            string(inv.Status),
		AIConfidenceScore: inv.AIConfidenceScore,
		DueDate:           inv.DueDate,
		CreatedAt:         inv.CreatedAt,
		UpdatedAt:         inv.UpdatedAt,
	}
	if inv.POID != nil {
		s := inv.POID.String()
		v.POID = &s
	}
	v.MatchResult = inv.MatchResult
	return v
}

// ListInvoices godoc
// GET /api/v1/invoices?page=1&per_page=20
// Requires: Authorization: Bearer <JWT with tenant_id claim>
// When page/per_page query params are present (or always), returns a paginated
// envelope: { data: [...], total, page, per_page }.
func (h *InvoiceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination params with defaults.
	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}
	if perPage > 100 {
		perPage = 100
	}

	result, err := h.uc.ListInvoicesPaged(r.Context(), tenantID, page, perPage)
	if err != nil {
		log.Error().Err(err).Str("handler", "ListInvoices").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	type paginatedResponse struct {
		Data    []invoiceView `json:"data"`
		Total   int           `json:"total"`
		Page    int           `json:"page"`
		PerPage int           `json:"per_page"`
	}

	views := make([]invoiceView, 0, len(result.Data))
	for i := range result.Data {
		views = append(views, toView(&result.Data[i]))
	}
	respondJSON(w, http.StatusOK, paginatedResponse{
		Data:    views,
		Total:   result.Total,
		Page:    result.Page,
		PerPage: result.PerPage,
	})
}

// GetInvoice handles GET /api/v1/invoices/{id}
func (h *InvoiceHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid invoice id")
		return
	}
	inv, err := h.uc.GetInvoice(r.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "invoice not found")
			return
		}
		log.Error().Err(err).Str("handler", "GetInvoice").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, toView(inv))
}

// createInvoiceRequest is the JSON body for POST /api/v1/invoices.
// Amount is accepted as a string (e.g. "12345.6789") to prevent float truncation
// during JSON decode — the handler parses it with decimal.NewFromString.
type createInvoiceRequest struct {
	VendorID      string     `json:"vendor_id"`
	InvoiceNumber string     `json:"invoice_number"`
	Amount        string     `json:"amount"`
	Currency      string     `json:"currency"`
	DueDate       *time.Time `json:"due_date,omitempty"`
}

// CreateInvoice godoc
// POST /api/v1/invoices
func (h *InvoiceHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req createInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	amount, err := parseDecimalAmount(req.Amount)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	params := domain.CreateInvoiceParams{
		TenantID:      tenantID,
		VendorID:      req.VendorID,
		InvoiceNumber: req.InvoiceNumber,
		Amount:        amount,
		Currency:      req.Currency,
		DueDate:       req.DueDate,
	}
	if userID, ok := appMiddleware.GetUserID(r.Context()); ok {
		params.CreatedBy = &userID
	}

	inv, err := h.uc.CreateInvoice(r.Context(), params)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, r, http.StatusUnprocessableEntity, "invalid invoice data")
			return
		}
		log.Error().Err(err).Str("handler", "CreateInvoice").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, toView(inv))
}

// ApproveInvoice godoc
// POST /api/v1/invoices/{id}/approve
func (h *InvoiceHandler) ApproveInvoice(w http.ResponseWriter, r *http.Request) {
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

	inv, err := h.uc.ApproveInvoice(r.Context(), invoiceID, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "invoice not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "invoice is not in pending state")
			return
		}
		log.Error().Err(err).Str("handler", "ApproveInvoice").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toView(inv))
}

// RejectInvoice godoc
// POST /api/v1/invoices/{id}/reject
func (h *InvoiceHandler) RejectInvoice(w http.ResponseWriter, r *http.Request) {
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

	inv, err := h.uc.RejectInvoice(r.Context(), invoiceID, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "invoice not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "invoice is not in pending or pending_review state")
			return
		}
		log.Error().Err(err).Str("handler", "RejectInvoice").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toView(inv))
}

// RequeueInvoice godoc
// POST /api/v1/invoices/{id}/requeue
func (h *InvoiceHandler) RequeueInvoice(w http.ResponseWriter, r *http.Request) {
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

	inv, err := h.uc.RequeueInvoice(r.Context(), invoiceID, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "invoice not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "invoice is not in pending status")
			return
		}
		log.Error().Err(err).Str("handler", "RequeueInvoice").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toView(inv))
}

// AutoApproveInvoice godoc
// POST /api/v1/invoices/{id}/auto-approve
func (h *InvoiceHandler) AutoApproveInvoice(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.uc.AutoApproveInvoice(r.Context(), invoiceID, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "invoice not found")
			return
		}
		// AutoApproveUsecase may return ErrAutoApproveNotEligible as a conflict
		if errors.Is(err, invoice.ErrAutoApproveNotEligible) {
			respondError(w, r, http.StatusConflict, err.Error())
			return
		}
		log.Error().Err(err).Str("handler", "AutoApproveInvoice").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, result)
}
