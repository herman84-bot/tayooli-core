package product_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/product"
)

type mockCategoryRepo struct {
	categories map[uuid.UUID]*domain.ProductCategory
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{categories: make(map[uuid.UUID]*domain.ProductCategory)}
}

func (m *mockCategoryRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.ProductCategory, error) {
	var list []domain.ProductCategory
	for _, c := range m.categories {
		if c.TenantID == tenantID {
			list = append(list, *c)
		}
	}
	return list, nil
}

func (m *mockCategoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ProductCategory, error) {
	c, ok := m.categories[id]
	if !ok || c.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (m *mockCategoryRepo) Create(ctx context.Context, c *domain.ProductCategory) error {
	for _, existing := range m.categories {
		if existing.TenantID == c.TenantID && strings.EqualFold(existing.Name, c.Name) {
			return domain.ErrConflict
		}
	}
	m.categories[c.ID] = c
	return nil
}

func (m *mockCategoryRepo) Rename(ctx context.Context, tenantID, id uuid.UUID, name string) error {
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

func (m *mockCategoryRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	c, ok := m.categories[id]
	if !ok || c.TenantID != tenantID {
		return domain.ErrNotFound
	}
	delete(m.categories, id)
	return nil
}

func TestProductCategories_CRUD(t *testing.T) {
	ctx := context.Background()
	tenantA := uuid.New()
	tenantB := uuid.New()

	pRepo := newMockProductRepo()
	cRepo := newMockCategoryRepo()
	uc := product.New(pRepo, &mockInventoryRepo{}, product.WithCategories(cRepo))

	// 1. Create category in tenant A
	cat, err := uc.CreateCategory(ctx, tenantA, "  Makanan Ringan  ")
	require.NoError(t, err)
	assert.Equal(t, "Makanan Ringan", cat.Name, "should normalize spaces")
	assert.Equal(t, tenantA, cat.TenantID)

	// 2. Reject duplicate in tenant A (case-insensitive)
	_, err = uc.CreateCategory(ctx, tenantA, "makanan ringan")
	assert.ErrorIs(t, err, domain.ErrConflict)

	// 3. Allow same name in tenant B (tenant isolation)
	catB, err := uc.CreateCategory(ctx, tenantB, "Makanan Ringan")
	require.NoError(t, err)
	assert.Equal(t, tenantB, catB.TenantID)

	// 4. Reject empty or oversized name
	_, err = uc.CreateCategory(ctx, tenantA, "   ")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	_, err = uc.CreateCategory(ctx, tenantA, strings.Repeat("A", 101))
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	// 5. Rename category
	renamed, err := uc.RenameCategory(ctx, tenantA, cat.ID, "Snack & Kudapan")
	require.NoError(t, err)
	assert.Equal(t, "Snack & Kudapan", renamed.Name)

	// 6. Delete category
	err = uc.DeleteCategory(ctx, tenantA, cat.ID)
	require.NoError(t, err)

	// 7. Verify deleted
	list, err := uc.ListCategories(ctx, tenantA)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestProduct_WithCategory(t *testing.T) {
	ctx := context.Background()
	tenantA := uuid.New()
	tenantB := uuid.New()

	pRepo := newMockProductRepo()
	cRepo := newMockCategoryRepo()
	uc := product.New(pRepo, &mockInventoryRepo{}, product.WithCategories(cRepo))

	// Seed category in tenant A and tenant B
	catA, err := uc.CreateCategory(ctx, tenantA, "Minuman")
	require.NoError(t, err)
	catB, err := uc.CreateCategory(ctx, tenantB, "Bahan Baku")
	require.NoError(t, err)

	// Create product in tenant A with valid category
	p, err := uc.CreateProduct(ctx, tenantA, product.CreateProductRequest{
		Name:       "Teh Botol",
		SKU:        "TEH-001",
		Price:      5000,
		CategoryID: &catA.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, &catA.ID, p.CategoryID)
	assert.Equal(t, "Minuman", p.CategoryName)

	// Reject cross-tenant category assignment
	_, err = uc.CreateProduct(ctx, tenantA, product.CreateProductRequest{
		Name:       "Gula Pasir",
		SKU:        "GUL-001",
		Price:      15000,
		CategoryID: &catB.ID, // belongs to tenant B!
	})
	assert.ErrorIs(t, err, domain.ErrInvalidInput, "foreign tenant category must be rejected as invalid input")

	// Update product: change category
	catA2, err := uc.CreateCategory(ctx, tenantA, "Kemasan")
	require.NoError(t, err)

	upd, err := uc.UpdateProduct(ctx, tenantA, p.ID, product.UpdateProductRequest{
		Name:       "Teh Botol 350ml",
		SKU:        "TEH-001",
		Price:      5500,
		CategoryID: product.OptionalUUID{Set: true, Value: &catA2.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, &catA2.ID, upd.CategoryID)
	assert.Equal(t, "Kemasan", upd.CategoryName)

	// Update product: clear category (set to null)
	updNull, err := uc.UpdateProduct(ctx, tenantA, p.ID, product.UpdateProductRequest{
		Name:       "Teh Botol 350ml",
		SKU:        "TEH-001",
		Price:      5500,
		CategoryID: product.OptionalUUID{Set: true, Value: nil},
	})
	require.NoError(t, err)
	assert.Nil(t, updNull.CategoryID)
	assert.Empty(t, updNull.CategoryName)
}

func (m *mockProductRepo) Reset() {
	m.products = make(map[uuid.UUID]*domain.Product)
}

// Suppress unused warning
var _ = time.Now
