package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
)

// -----------------------------------------------------------------------------
// Stock Receipt Handlers (Barang Masuk)
// -----------------------------------------------------------------------------

// CancelStockReceiptRequest is the body for POST /wms/receipts/{id}/cancel.
type CancelStockReceiptRequest struct {
	Reason string `json:"reason"`
}

func decodeReceiptJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return false
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return false
	}
	return true
}

// ListStockReceipts handles GET /api/v1/wms/receipts?warehouse_id=&status=
func (h *WMSHandler) ListStockReceipts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whIDPtr *uuid.UUID
	if whIDStr := r.URL.Query().Get("warehouse_id"); whIDStr != "" {
		parsed, err := uuid.Parse(whIDStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whIDPtr = &parsed
	}
	var statusPtr *domain.StockReceiptStatus
	if s := strings.TrimSpace(r.URL.Query().Get("status")); s != "" {
		st := domain.StockReceiptStatus(strings.ToUpper(s))
		statusPtr = &st
	}

	receipts, err := h.uc.ListStockReceipts(r.Context(), tenantID, userID, role, whIDPtr, statusPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if receipts == nil {
		receipts = []domain.StockReceipt{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": receipts})
}

// GetStockReceipt handles GET /api/v1/wms/receipts/{id}
func (h *WMSHandler) GetStockReceipt(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	receiptID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid receipt id")
		return
	}

	receipt, items, err := h.uc.GetStockReceipt(r.Context(), tenantID, userID, role, receiptID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if items == nil {
		items = []domain.StockReceiptItem{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"receipt": receipt, "items": items})
}

// CreateStockReceipt handles POST /api/v1/wms/receipts
func (h *WMSHandler) CreateStockReceipt(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var req uc.StockReceiptRequest
	if !decodeReceiptJSON(w, r, &req) {
		return
	}

	receipt, items, err := h.uc.CreateStockReceipt(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if items == nil {
		items = []domain.StockReceiptItem{}
	}
	respondJSON(w, http.StatusCreated, map[string]any{"receipt": receipt, "items": items})
}

// UpdateStockReceipt handles PUT /api/v1/wms/receipts/{id} (DRAFT only; replaces items)
func (h *WMSHandler) UpdateStockReceipt(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	receiptID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid receipt id")
		return
	}
	var req uc.StockReceiptRequest
	if !decodeReceiptJSON(w, r, &req) {
		return
	}

	receipt, items, err := h.uc.UpdateStockReceipt(r.Context(), tenantID, userID, role, receiptID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if items == nil {
		items = []domain.StockReceiptItem{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"receipt": receipt, "items": items})
}

// PostStockReceipt handles POST /api/v1/wms/receipts/{id}/post
func (h *WMSHandler) PostStockReceipt(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	receiptID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid receipt id")
		return
	}

	receipt, err := h.uc.PostStockReceipt(r.Context(), tenantID, userID, role, receiptID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, receipt)
}

// CancelStockReceipt handles POST /api/v1/wms/receipts/{id}/cancel  body {reason}
func (h *WMSHandler) CancelStockReceipt(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	receiptID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid receipt id")
		return
	}
	var req CancelStockReceiptRequest
	if !decodeReceiptJSON(w, r, &req) {
		return
	}

	receipt, err := h.uc.CancelStockReceipt(r.Context(), tenantID, userID, role, receiptID, req.Reason)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, receipt)
}
