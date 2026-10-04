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
)

// paymentOrderUsecase groups the operations the handler needs.
type paymentOrderUsecase interface {
	ListPaymentOrders(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error)
	GetPaymentOrder(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error)
	CreatePaymentOrder(ctx context.Context, params domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error)
	ApprovePaymentOrder(ctx context.Context, id, tenantID, approvedBy uuid.UUID, role string) (*domain.PaymentOrder, error)
	PayPaymentOrder(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error)
	RejectPaymentOrder(ctx context.Context, id, tenantID, rejectedBy uuid.UUID) (*domain.PaymentOrder, error)
}

// PaymentOrderHandler handles HTTP requests for payment order resources.
type PaymentOrderHandler struct {
	uc paymentOrderUsecase
}

func NewPaymentOrderHandler(uc paymentOrderUsecase) *PaymentOrderHandler {
	return &PaymentOrderHandler{uc: uc}
}

// paymentOrderView is the JSON representation returned to clients.
// Amount is a string (e.g. "12345.67") to preserve NUMERIC(15,2) precision.
type paymentOrderView struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	InvoiceID       string     `json:"invoice_id"`
	Amount          string     `json:"amount"`
	Currency        string     `json:"currency"`
	PaymentMethod   string     `json:"payment_method"`
	ReferenceNumber *string    `json:"reference_number,omitempty"`
	Status          string     `json:"status"`
	Notes           *string    `json:"notes,omitempty"`
	CreatedBy       *string    `json:"created_by,omitempty"`
	ApprovedBy      *string    `json:"approved_by,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func toPaymentOrderView(po *domain.PaymentOrder) paymentOrderView {
	v := paymentOrderView{
		ID:              po.ID.String(),
		TenantID:        po.TenantID.String(),
		InvoiceID:       po.InvoiceID.String(),
		Amount:          po.Amount.String(),
		Currency:        po.Currency,
		PaymentMethod:   po.PaymentMethod,
		ReferenceNumber: po.ReferenceNumber,
		Status:          string(po.Status),
		Notes:           po.Notes,
		PaidAt:          po.PaidAt,
		CreatedAt:       po.CreatedAt,
		UpdatedAt:       po.UpdatedAt,
	}
	if po.CreatedBy != nil {
		s := po.CreatedBy.String()
		v.CreatedBy = &s
	}
	if po.ApprovedBy != nil {
		s := po.ApprovedBy.String()
		v.ApprovedBy = &s
	}
	return v
}

// ── ListPaymentOrders ────────────────────────────────────────────────────────

// ListPaymentOrders godoc
// GET /api/v1/payment-orders?page=1&per_page=20
func (h *PaymentOrderHandler) ListPaymentOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

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

	result, err := h.uc.ListPaymentOrders(r.Context(), tenantID, page, perPage)
	if err != nil {
		log.Error().Err(err).Str("handler", "ListPaymentOrders").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	type paginatedResponse struct {
		Data    []paymentOrderView `json:"data"`
		Total   int                `json:"total"`
		Page    int                `json:"page"`
		PerPage int                `json:"per_page"`
	}

	views := make([]paymentOrderView, 0, len(result.Data))
	for i := range result.Data {
		views = append(views, toPaymentOrderView(&result.Data[i]))
	}
	respondJSON(w, http.StatusOK, paginatedResponse{
		Data:    views,
		Total:   result.Total,
		Page:    result.Page,
		PerPage: result.PerPage,
	})
}

// ── GetPaymentOrder ──────────────────────────────────────────────────────────

// GetPaymentOrder handles GET /api/v1/payment-orders/{id}
func (h *PaymentOrderHandler) GetPaymentOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid payment order id")
		return
	}
	po, err := h.uc.GetPaymentOrder(r.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "payment order not found")
			return
		}
		log.Error().Err(err).Str("handler", "GetPaymentOrder").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}
	respondJSON(w, http.StatusOK, toPaymentOrderView(po))
}

// ── CreatePaymentOrder ───────────────────────────────────────────────────────

// createPaymentOrderRequest is the JSON body for POST /api/v1/payment-orders.
type createPaymentOrderRequest struct {
	InvoiceID       string  `json:"invoice_id"`
	Amount          string  `json:"amount"`
	Currency        string  `json:"currency"`
	PaymentMethod   string  `json:"payment_method"`
	ReferenceNumber *string `json:"reference_number,omitempty"`
	Notes           *string `json:"notes,omitempty"`
}

// CreatePaymentOrder godoc
// POST /api/v1/payment-orders
func (h *PaymentOrderHandler) CreatePaymentOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req createPaymentOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	invoiceID, err := uuid.Parse(req.InvoiceID)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid invoice_id")
		return
	}

	amount, err := parseDecimalAmount(req.Amount)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if req.Currency == "" {
		req.Currency = "IDR"
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "bank_transfer"
	}

	// Extract user_id from context for created_by (best-effort).
	var createdBy *uuid.UUID
	if userID, uok := appMiddleware.GetUserID(r.Context()); uok {
		createdBy = &userID
	}

	params := domain.CreatePaymentOrderParams{
		TenantID:        tenantID,
		InvoiceID:       invoiceID,
		Amount:          amount,
		Currency:        req.Currency,
		PaymentMethod:   req.PaymentMethod,
		ReferenceNumber: req.ReferenceNumber,
		Notes:           req.Notes,
		CreatedBy:       createdBy,
	}

	po, err := h.uc.CreatePaymentOrder(r.Context(), params)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "invoice not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "invoice is not approved")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, r, http.StatusUnprocessableEntity, err.Error())
			return
		}
		log.Error().Err(err).Str("handler", "CreatePaymentOrder").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, toPaymentOrderView(po))
}

// ── ApprovePaymentOrder ──────────────────────────────────────────────────────

// ApprovePaymentOrder godoc
// POST /api/v1/payment-orders/{id}/approve
func (h *PaymentOrderHandler) ApprovePaymentOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	poID, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid payment order id")
		return
	}

	// The approver is the authenticated user.
	var approvedBy uuid.UUID
	if userID, uok := appMiddleware.GetUserID(r.Context()); uok {
		approvedBy = userID
	} else {
		approvedBy = uuid.Nil
	}

	role := appMiddleware.GetRole(r.Context())

	po, err := h.uc.ApprovePaymentOrder(r.Context(), poID, tenantID, approvedBy, role)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "payment order not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "payment order is not in draft status")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			respondError(w, r, http.StatusForbidden, "forbidden: only cfo or admin can approve payment orders")
			return
		}
		log.Error().Err(err).Str("handler", "ApprovePaymentOrder").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toPaymentOrderView(po))
}

// ── PayPaymentOrder ──────────────────────────────────────────────────────────

// PayPaymentOrder godoc
// POST /api/v1/payment-orders/{id}/pay
func (h *PaymentOrderHandler) PayPaymentOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	poID, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid payment order id")
		return
	}

	po, err := h.uc.PayPaymentOrder(r.Context(), poID, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "payment order not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "payment order is not in approved status")
			return
		}
		log.Error().Err(err).Str("handler", "PayPaymentOrder").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toPaymentOrderView(po))
}

// ── RejectPaymentOrder ───────────────────────────────────────────────────────

// RejectPaymentOrder godoc
// POST /api/v1/payment-orders/{id}/reject
func (h *PaymentOrderHandler) RejectPaymentOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	poID, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid payment order id")
		return
	}

	// The rejector is the authenticated user.
	var rejectedBy uuid.UUID
	if userID, uok := appMiddleware.GetUserID(r.Context()); uok {
		rejectedBy = userID
	} else {
		rejectedBy = uuid.Nil
	}

	po, err := h.uc.RejectPaymentOrder(r.Context(), poID, tenantID, rejectedBy)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "payment order not found")
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			respondError(w, r, http.StatusConflict, "payment order is not in draft status")
			return
		}
		log.Error().Err(err).Str("handler", "RejectPaymentOrder").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toPaymentOrderView(po))
}
