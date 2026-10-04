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

type grUsecase interface {
	Create(ctx context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.GoodsReceipt, error)
	ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.GoodsReceipt, error)
}

type GRHandler struct {
	uc grUsecase
}

func NewGRHandler(uc grUsecase) *GRHandler {
	return &GRHandler{uc: uc}
}

type grView struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	POID           string `json:"po_id"`
	VendorID       string `json:"vendor_id"`
	ReceivedQty    int    `json:"received_qty"`
	ReceivedAmount string `json:"received_amount"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	ReceivedAt     string `json:"received_at"`
	CreatedAt      string `json:"created_at"`
}

func toGRView(gr *domain.GoodsReceipt) grView {
	return grView{
		ID:             gr.ID.String(),
		TenantID:       gr.TenantID.String(),
		POID:           gr.POID.String(),
		VendorID:       gr.VendorID,
		ReceivedQty:    gr.ReceivedQty,
		ReceivedAmount: gr.ReceivedAmount.String(),
		Currency:       gr.Currency,
		Status:         string(gr.Status),
		ReceivedAt:     gr.ReceivedAt.Format(time.RFC3339),
		CreatedAt:      gr.CreatedAt.Format(time.RFC3339),
	}
}

type createGRRequest struct {
	POID           string `json:"po_id"`
	VendorID       string `json:"vendor_id"`
	ReceivedQty    int    `json:"received_qty"`
	ReceivedAmount string `json:"received_amount"`
	Currency       string `json:"currency"`
}

func (h *GRHandler) CreateGR(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64 KiB limit
	var req createGRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondError(w, r, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	poID, err := uuid.Parse(req.POID)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid po_id")
		return
	}

	receivedAmount, err := parseDecimalAmount(req.ReceivedAmount)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	params := domain.CreateGRParams{
		TenantID:       tenantID,
		POID:           poID,
		VendorID:       req.VendorID,
		ReceivedQty:    req.ReceivedQty,
		ReceivedAmount: receivedAmount,
		Currency:       req.Currency,
	}

	gr, err := h.uc.Create(r.Context(), params)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, r, http.StatusUnprocessableEntity, "invalid goods receipt data")
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "purchase order not found")
			return
		}
		log.Error().Err(err).Str("handler", "CreateGR").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusCreated, toGRView(gr))
}

func (h *GRHandler) ListGRs(w http.ResponseWriter, r *http.Request) {
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
		log.Error().Err(err).Str("handler", "ListGRs").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	type paginatedResponse struct {
		Data    []grView `json:"data"`
		Total   int      `json:"total"`
		Page    int      `json:"page"`
		PerPage int      `json:"per_page"`
	}

	views := make([]grView, 0, len(result.Data))
	for i := range result.Data {
		views = append(views, toGRView(&result.Data[i]))
	}
	respondJSON(w, http.StatusOK, paginatedResponse{
		Data:    views,
		Total:   result.Total,
		Page:    result.Page,
		PerPage: result.PerPage,
	})
}

func (h *GRHandler) GetGR(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid goods receipt id")
		return
	}

	gr, err := h.uc.GetByID(r.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "goods receipt not found")
			return
		}
		log.Error().Err(err).Str("handler", "GetGR").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toGRView(gr))
}
