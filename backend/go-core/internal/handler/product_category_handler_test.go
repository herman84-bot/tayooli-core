package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/product"
)

type mockCategoryRepoForHandler struct {
	categories map[uuid.UUID]*domain.ProductCategory
}

func newMockCategoryRepoForHandler() *mockCategoryRepoForHandler {
	return &mockCategoryRepoForHandler{categories: make(map[uuid.UUID]*domain.ProductCategory)}
}

func (m *mockCategoryRepoForHandler) List(ctx context.Context, tenantID uuid.UUID) ([]domain.ProductCategory, error) {
	var list []domain.ProductCategory
	for _, c := range m.categories {
		if c.TenantID == tenantID {
			list = append(list, *c)
		}
	}
	return list, nil
}

func (m *mockCategoryRepoForHandler) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ProductCategory, error) {
	c, ok := m.categories[id]
	if !ok || c.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (m *mockCategoryRepoForHandler) Create(ctx context.Context, c *domain.ProductCategory) error {
	for _, existing := range m.categories {
		if existing.TenantID == c.TenantID && strings.EqualFold(existing.Name, c.Name) {
			return domain.ErrConflict
		}
	}
	m.categories[c.ID] = c
	return nil
}

func (m *mockCategoryRepoForHandler) Rename(ctx context.Context, tenantID, id uuid.UUID, name string) error {
	c, ok := m.categories[id]
	if !ok || c.TenantID != tenantID {
		return domain.ErrNotFound
	}
	for _, existing := range m.categories {
		if existing.TenantID == tenantID && existing.ID != id && strings.EqualFold(existing.Name, name) {
			return domain.ErrConflict
		}
	}
	c.Name = name
	return nil
}

func (m *mockCategoryRepoForHandler) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	c, ok := m.categories[id]
	if !ok || c.TenantID != tenantID {
		return domain.ErrNotFound
	}
	delete(m.categories, id)
	return nil
}

func setupCategoryTestRouter(pRepo domain.ProductRepository, cRepo domain.ProductCategoryRepository) (*chi.Mux, uuid.UUID) {
	tenantID := uuid.New()
	u := uc.New(pRepo, &mockInventoryRepo{}, uc.WithCategories(cRepo))
	h := handler.NewProductHandler(u)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/products", h.ListProducts)
	r.Get("/product-categories", h.ListCategories)
	r.Post("/product-categories", h.CreateCategory)
	r.Put("/product-categories/{id}", h.RenameCategory)
	r.Delete("/product-categories/{id}", h.DeleteCategory)
	return r, tenantID
}

func TestProductCategoryHandler_Endpoints(t *testing.T) {
	pRepo := &mockProductRepo{products: make(map[uuid.UUID]*domain.Product)}
	cRepo := newMockCategoryRepoForHandler()
	r, _ := setupCategoryTestRouter(pRepo, cRepo)

	// 1. Create category
	body := `{"name":"Minuman Segar"}`
	req := httptest.NewRequest(http.MethodPost, "/product-categories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created domain.ProductCategory
	err := json.NewDecoder(w.Body).Decode(&created)
	require.NoError(t, err)
	assert.Equal(t, "Minuman Segar", created.Name)

	// 2. Reject duplicate
	req = httptest.NewRequest(http.MethodPost, "/product-categories", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)

	// 3. List categories
	req = httptest.NewRequest(http.MethodGet, "/product-categories", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var listResp struct {
		Data []domain.ProductCategory `json:"data"`
	}
	err = json.NewDecoder(w.Body).Decode(&listResp)
	require.NoError(t, err)
	assert.Len(t, listResp.Data, 1)

	// 4. Rename category
	renameBody := `{"name":"Minuman & Jus"}`
	req = httptest.NewRequest(http.MethodPut, "/product-categories/"+created.ID.String(), bytes.NewBufferString(renameBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. Delete category
	req = httptest.NewRequest(http.MethodDelete, "/product-categories/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestProductHandler_CategoryFilterParam(t *testing.T) {
	pRepo := &mockProductRepo{products: make(map[uuid.UUID]*domain.Product)}
	cRepo := newMockCategoryRepoForHandler()
	r, _ := setupCategoryTestRouter(pRepo, cRepo)

	catID := uuid.New()
	// Filter by category UUID
	req := httptest.NewRequest(http.MethodGet, "/products?page=1&per_page=10&category="+catID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, &catID, pRepo.lastFilter.Category.ID)
	assert.False(t, pRepo.lastFilter.Category.Uncategorized)

	// Filter by none
	req = httptest.NewRequest(http.MethodGet, "/products?page=1&per_page=10&category=none", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, pRepo.lastFilter.Category.Uncategorized)
	assert.Nil(t, pRepo.lastFilter.Category.ID)

	// Reject invalid category parameter
	req = httptest.NewRequest(http.MethodGet, "/products?page=1&per_page=10&category=not-a-uuid", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
