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

type poUsecase interface {
	Create(ctx context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.PurchaseOrder, error)
	ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error)
}

type POHandler struct {
	uc poUsecase
}

func NewPOHandler(uc poUsecase) *POHandler {
	return &POHandler{uc: uc}
}

type poView struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	VendorID  string `json:"vendor_id"`
	PONumber  string `json:"po_number"`
	Amount    string `json:"amount"`
	Qty       int    `json:"qty"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toPOView(po *domain.PurchaseOrder) poView {
	return poView{
		ID:        po.ID.String(),
		TenantID:  po.TenantID.String(),
		VendorID:  po.VendorID,
		PONumber:  po.PONumber,
		Amount:    po.Amount.String(),
		Qty:       po.Qty,
		Currency:  po.Currency,
		Status:    string(po.Status),
		CreatedAt: po.CreatedAt.Format(time.RFC3339),
		UpdatedAt: po.UpdatedAt.Format(time.RFC3339),
	}
}

type createPORequest struct {
	VendorID string `json:"vendor_id"`
	PONumber string `json:"po_number"`
	Amount   string `json:"amount"`
	Qty      int    `json:"qty"`
	Currency string `json:"currency"`
}

func (h *POHandler) CreatePO(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64 KiB limit
	var req createPORequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondError(w, r, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	amount, err := parseDecimalAmount(req.Amount)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	params := domain.CreatePOParams{
		TenantID: tenantID,
		VendorID: req.VendorID,
		PONumber: req.PONumber,
		Amount:   amount,
		Qty:      req.Qty,
		Currency: req.Currency,
	}

	po, err := h.uc.Create(r.Context(), params)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, r, http.StatusUnprocessableEntity, "invalid purchase order data")
			return
		}
		log.Error().Err(err).Str("handler", "CreatePO").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, toPOView(po))
}

func (h *POHandler) ListPOs(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination params with defaults (mirrors GET /invoices).
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

	result, err := h.uc.ListPaged(r.Context(), tenantID, page, perPage)
	if err != nil {
		log.Error().Err(err).Str("handler", "ListPOs").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	type paginatedResponse struct {
		Data    []poView `json:"data"`
		Total   int      `json:"total"`
		Page    int      `json:"page"`
		PerPage int      `json:"per_page"`
	}

	views := make([]poView, 0, len(result.Data))
	for i := range result.Data {
		views = append(views, toPOView(&result.Data[i]))
	}
	respondJSON(w, http.StatusOK, paginatedResponse{
		Data:    views,
		Total:   result.Total,
		Page:    result.Page,
		PerPage: result.PerPage,
	})
}

func (h *POHandler) GetPO(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid purchase order id")
		return
	}

	po, err := h.uc.GetByID(r.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "purchase order not found")
			return
		}
		log.Error().Err(err).Str("handler", "GetPO").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toPOView(po))
}
