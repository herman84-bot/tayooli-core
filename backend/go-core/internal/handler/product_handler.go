package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/product"
)

// ProductHandler serves the Products and Inventory resources. Reads are open
// to any authenticated tenant member; writes are role-gated at the router.
type ProductHandler struct {
	uc *uc.Usecase
}

func NewProductHandler(u *uc.Usecase) *ProductHandler {
	return &ProductHandler{uc: u}
}

// writeError maps domain errors to HTTP status codes. Unknown errors are
// returned as a generic 500 (the chi logger already records the request).
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, domain.ErrInvalidInput):
		http.Error(w, "invalid input", http.StatusBadRequest)
	case errors.Is(err, domain.ErrConflict):
		http.Error(w, "conflict: SKU sudah digunakan atau produk memiliki riwayat mutasi/stok", http.StatusConflict)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req uc.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.uc.CreateProduct(r.Context(), tenantID, req)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := h.uc.ListProducts(r.Context(), tenantID)
	if err != nil {
		writeError(w, err)
		return
	}
	if res == nil {
		res = []domain.Product{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
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
	res, err := h.uc.GetProduct(r.Context(), tenantID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
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
	var req uc.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.uc.UpdateProduct(r.Context(), tenantID, id, req)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
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
	if err := h.uc.DeleteProduct(r.Context(), tenantID, id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) CreateInventory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req uc.CreateInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	res, err := h.uc.CreateInventory(r.Context(), tenantID, req)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *ProductHandler) ListInventory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := h.uc.ListInventory(r.Context(), tenantID)
	if err != nil {
		writeError(w, err)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *ProductHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
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
	res, err := h.uc.GetInventory(r.Context(), tenantID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	json.NewEncoder(w).Encode(res)
}
