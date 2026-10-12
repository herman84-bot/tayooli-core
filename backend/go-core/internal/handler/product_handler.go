package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

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
	qv := r.URL.Query()
	// Legacy shape (no paging params): full list. Kept until every caller is
	// migrated to server-side search; see SearchProducts below.
	if !qv.Has("page") && !qv.Has("per_page") && !qv.Has("q") && !qv.Has("category") {
		res, err := h.uc.ListProducts(r.Context(), tenantID)
		if err != nil {
			writeError(w, err)
			return
		}
		if res == nil {
			res = []domain.Product{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": res})
		return
	}

	page, ok1 := positiveIntParam(qv.Get("page"), 1)
	perPage, ok2 := positiveIntParam(qv.Get("per_page"), 20)
	q := strings.TrimSpace(qv.Get("q"))
	if !ok1 || !ok2 || utf8.RuneCountInString(q) > maxProductQueryLen {
		http.Error(w, "invalid input: page/per_page harus angka >= 1, q maksimal 100 karakter", http.StatusBadRequest)
		return
	}
	category, ok3 := parseCategoryFilter(qv.Get("category"))
	if !ok3 {
		http.Error(w, "invalid input: category harus UUID kategori atau 'none'", http.StatusBadRequest)
		return
	}
	perPage = min(perPage, maxProductsPerPage)

	res, total, err := h.uc.SearchProducts(r.Context(), tenantID, domain.ProductFilter{
		Query: q, Category: category, Limit: perPage, Offset: (page - 1) * perPage,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if res == nil {
		res = []domain.Product{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": res, "total": total, "page": page, "per_page": perPage,
	})
}

const (
	maxProductsPerPage = 100 // Shopify-style hard cap on page size
	maxProductQueryLen = 100
)

// categoryNone selects products without a category in ?category=.
const categoryNone = "none"

// parseCategoryFilter parses ?category=: empty = any, "none" = uncategorized,
// otherwise a category UUID. Tenant scoping happens in the repo query, so a
// foreign UUID simply matches nothing.
func parseCategoryFilter(raw string) (domain.CategoryFilter, bool) {
	switch raw = strings.TrimSpace(raw); raw {
	case "":
		return domain.CategoryFilter{}, true
	case categoryNone:
		return domain.CategoryFilter{Uncategorized: true}, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return domain.CategoryFilter{}, false
	}
	return domain.CategoryFilter{ID: &id}, true
}

// positiveIntParam parses an optional positive integer; empty uses def.
func positiveIntParam(raw string, def int) (int, bool) {
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
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
