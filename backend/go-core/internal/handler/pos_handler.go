package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pos"
)

type POSHandler struct {
	uc *uc.Usecase
}

func NewPOSHandler(u *uc.Usecase) *POSHandler {
	return &POSHandler{uc: u}
}

func (h *POSHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())

	var req uc.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	res, err := h.uc.Checkout(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *POSHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	orders, err := h.uc.ListOrders(r.Context(), tenantID, limit)
	if err != nil {
		http.Error(w, "failed to list pos orders", http.StatusInternalServerError)
		return
	}
	if orders == nil {
		orders = []domain.POSOrder{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": orders,
	})
}

func (h *POSHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}

	order, err := h.uc.GetOrderByID(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get order", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
