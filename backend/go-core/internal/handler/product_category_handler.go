package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// Product categories share ProductHandler (same usecase, same role policy):
// reads for any tenant member, writes gated at the router like product writes.

// maxCategoryBodyBytes bounds the request body; a category is just a name.
const maxCategoryBodyBytes = 4 << 10

type categoryRequest struct {
	Name string `json:"name"`
}

// writeCategoryError maps domain errors with category-specific messages
// (writeError's conflict text is about SKUs).
func writeCategoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		http.Error(w, "not found: kategori tidak ditemukan", http.StatusNotFound)
	case errors.Is(err, domain.ErrInvalidInput):
		http.Error(w, "invalid input: nama kategori wajib diisi, maksimal 100 karakter", http.StatusBadRequest)
	case errors.Is(err, domain.ErrConflict):
		http.Error(w, "conflict: nama kategori sudah ada", http.StatusConflict)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func decodeCategoryRequest(w http.ResponseWriter, r *http.Request) (categoryRequest, bool) {
	var req categoryRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCategoryBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return req, false
	}
	return req, true
}

// ListCategories handles GET /product-categories.
func (h *ProductHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := h.uc.ListCategories(r.Context(), tenantID)
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": res})
}

// CreateCategory handles POST /product-categories.
func (h *ProductHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	req, ok := decodeCategoryRequest(w, r)
	if !ok {
		return
	}
	res, err := h.uc.CreateCategory(r.Context(), tenantID, req.Name)
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, res)
}

// RenameCategory handles PUT/PATCH /product-categories/{id}.
func (h *ProductHandler) RenameCategory(w http.ResponseWriter, r *http.Request) {
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
	req, ok := decodeCategoryRequest(w, r)
	if !ok {
		return
	}
	res, err := h.uc.RenameCategory(r.Context(), tenantID, id, req.Name)
	if err != nil {
		writeCategoryError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, res)
}

// DeleteCategory handles DELETE /product-categories/{id}. Products in the
// category become uncategorized.
func (h *ProductHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
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
	if err := h.uc.DeleteCategory(r.Context(), tenantID, id); err != nil {
		writeCategoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
