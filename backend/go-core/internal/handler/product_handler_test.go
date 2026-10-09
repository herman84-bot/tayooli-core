package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/product"
)

type mockProductRepo struct {
	products     map[uuid.UUID]*domain.Product
	hasMovements bool
}

func (m *mockProductRepo) Create(ctx context.Context, p *domain.Product) error {
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	p, ok := m.products[id]
	if !ok || p.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (m *mockProductRepo) GetBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.Product, error) {
	for _, p := range m.products {
		if p.TenantID == tenantID && p.SKU == sku {
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockProductRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Product, error) {
	var list []domain.Product
	for _, p := range m.products {
		if p.TenantID == tenantID {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockProductRepo) ListByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]domain.Product, error) {
	var list []domain.Product
	for _, id := range ids {
		if p, ok := m.products[id]; ok && p.TenantID == tenantID {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockProductRepo) Update(ctx context.Context, p *domain.Product) error {
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.products, id)
	return nil
}

func (m *mockProductRepo) HasMovementsOrStock(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	return m.hasMovements, nil
}

func (m *mockProductRepo) ListInventoryFromWMS(ctx context.Context, tenantID uuid.UUID) ([]domain.InventoryItemWithProduct, error) {
	return nil, nil
}

type mockInventoryRepo struct{}

func (m *mockInventoryRepo) Create(ctx context.Context, i *domain.Inventory) error { return nil }
func (m *mockInventoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Inventory, error) {
	return nil, domain.ErrNotFound
}
func (m *mockInventoryRepo) GetByProductID(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) UpdateQuantity(ctx context.Context, tenantID, id uuid.UUID, delta float64) error {
	return nil
}

func TestProductHandler_ListAndMutations(t *testing.T) {
	tenantID := uuid.New()
	pRepo := &mockProductRepo{products: make(map[uuid.UUID]*domain.Product)}
	iRepo := &mockInventoryRepo{}
	u := uc.New(pRepo, iRepo)
	h := handler.NewProductHandler(u)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Get("/products", h.ListProducts)
	r.Get("/products/{id}", h.GetProduct)
	r.Post("/products", h.CreateProduct)
	r.Put("/products/{id}", h.UpdateProduct)
	r.Delete("/products/{id}", h.DeleteProduct)

	// 1. Create product
	body := `{"name":"Kopi Susu","sku":"KOP-01","price":15000}`
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var created domain.Product
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created failed: %v", err)
	}

	// 2. List products: must return {"data": [...]}
	req = httptest.NewRequest(http.MethodGet, "/products", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var listResp struct {
		Data []domain.Product `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode list response: %v, raw: %s", err, w.Body.String())
	}
	if len(listResp.Data) != 1 || listResp.Data[0].SKU != "KOP-01" {
		t.Fatalf("unexpected list data: %+v", listResp.Data)
	}

	// 3. Update product
	upBody := `{"name":"Kopi Susu Mantap","sku":"KOP-01-M","price":17000}`
	req = httptest.NewRequest(http.MethodPut, "/products/"+created.ID.String(), bytes.NewBufferString(upBody))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Delete product with conflict
	pRepo.hasMovements = true
	req = httptest.NewRequest(http.MethodDelete, "/products/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d", w.Code)
	}

	// 5. Delete product successfully
	pRepo.hasMovements = false
	req = httptest.NewRequest(http.MethodDelete, "/products/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 no content, got %d", w.Code)
	}
}
