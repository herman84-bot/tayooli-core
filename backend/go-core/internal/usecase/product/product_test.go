package product_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/product"
)

type mockProductRepo struct {
	products     map[uuid.UUID]*domain.Product
	hasMovements bool
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		products: make(map[uuid.UUID]*domain.Product),
	}
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
	if _, ok := m.products[p.ID]; !ok {
		return domain.ErrNotFound
	}
	m.products[p.ID] = p
	return nil
}

func (m *mockProductRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	p, ok := m.products[id]
	if !ok || p.TenantID != tenantID {
		return domain.ErrNotFound
	}
	delete(m.products, id)
	return nil
}

func (m *mockProductRepo) HasMovementsOrStock(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	return m.hasMovements, nil
}

type mockInventoryRepo struct{}

func (m *mockInventoryRepo) Create(ctx context.Context, i *domain.Inventory) error { return nil }
func (m *mockInventoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Inventory, error) {
	return nil, domain.ErrNotFound
}
func (m *mockInventoryRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) GetByProductID(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) UpdateQuantity(ctx context.Context, tenantID, id uuid.UUID, delta float64) error {
	return nil
}

func TestProductCRUD(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	pRepo := newMockProductRepo()
	iRepo := &mockInventoryRepo{}
	uc := product.New(pRepo, iRepo)

	// 1. Create Product
	created, err := uc.CreateProduct(ctx, tenantID, product.CreateProductRequest{
		Name:        "Kopi Susu Gula Aren",
		Description: "Minuman kopi",
		SKU:         "KOPI-001",
		Price:       18000,
	})
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}
	if created.SKU != "KOPI-001" {
		t.Errorf("expected SKU KOPI-001, got %s", created.SKU)
	}

	// 2. Update Product
	updated, err := uc.UpdateProduct(ctx, tenantID, created.ID, product.UpdateProductRequest{
		Name:        "Kopi Susu Gula Aren Spesial",
		Description: "Minuman kopi nikmat",
		SKU:         "KOPI-001-SP",
		Price:       20000,
	})
	if err != nil {
		t.Fatalf("UpdateProduct failed: %v", err)
	}
	if updated.Name != "Kopi Susu Gula Aren Spesial" || updated.Price != 20000 {
		t.Errorf("UpdateProduct fields mismatch: %+v", updated)
	}

	// 3. Test Conflict on Duplicate SKU
	p2, err := uc.CreateProduct(ctx, tenantID, product.CreateProductRequest{
		Name:  "Roti Bakar Coklat",
		SKU:   "ROTI-001",
		Price: 15000,
	})
	if err != nil {
		t.Fatalf("CreateProduct 2 failed: %v", err)
	}
	// Try updating p2 SKU to KOPI-001-SP -> should return ErrConflict
	_, err = uc.UpdateProduct(ctx, tenantID, p2.ID, product.UpdateProductRequest{
		Name:  "Roti Bakar Coklat",
		SKU:   "KOPI-001-SP",
		Price: 15000,
	})
	if err != domain.ErrConflict {
		t.Errorf("expected ErrConflict on duplicate SKU, got %v", err)
	}

	// 4. Test Delete blocked if has movements
	pRepo.hasMovements = true
	err = uc.DeleteProduct(ctx, tenantID, created.ID)
	if err != domain.ErrConflict {
		t.Errorf("expected ErrConflict when product has movements, got %v", err)
	}

	// 5. Test Delete succeeds if no movements
	pRepo.hasMovements = false
	err = uc.DeleteProduct(ctx, tenantID, created.ID)
	if err != nil {
		t.Fatalf("DeleteProduct failed: %v", err)
	}

	// 6. Verify deleted
	_, err = uc.GetProduct(ctx, tenantID, created.ID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after deletion, got %v", err)
	}
}
