package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/customer"
)

type CustomerHandler struct {
	uc *uc.Usecase
}

func NewCustomerHandler(u *uc.Usecase) *CustomerHandler {
	return &CustomerHandler{uc: u}
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req uc.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.uc.Create(r.Context(), tenantID, req)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := h.uc.List(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := h.uc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}
