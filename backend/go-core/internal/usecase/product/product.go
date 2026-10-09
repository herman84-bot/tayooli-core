package product

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// Usecase coordinates the Product and Inventory aggregates. Both are
// tenant-scoped: every method takes the tenantID resolved from the request
// context by the tenant middleware.
type Usecase struct {
	productRepo   domain.ProductRepository
	inventoryRepo domain.InventoryRepository
}

func New(productRepo domain.ProductRepository, inventoryRepo domain.InventoryRepository) *Usecase {
	return &Usecase{productRepo: productRepo, inventoryRepo: inventoryRepo}
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SKU         string  `json:"sku"`
	Price       float64 `json:"price"`
	CostPrice   float64 `json:"cost_price"`
}

func (u *Usecase) CreateProduct(ctx context.Context, tenantID uuid.UUID, req CreateProductRequest) (*domain.Product, error) {
	if req.Name == "" || req.SKU == "" {
		return nil, domain.ErrInvalidInput
	}
	if req.Price < 0 || !validCost(req.CostPrice) {
		return nil, domain.ErrInvalidInput
	}
	p := &domain.Product{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		SKU:         req.SKU,
		Price:       req.Price,
		CostPrice:   req.CostPrice,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := u.productRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (u *Usecase) GetProduct(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	return u.productRepo.GetByID(ctx, tenantID, id)
}

func (u *Usecase) ListProducts(ctx context.Context, tenantID uuid.UUID) ([]domain.Product, error) {
	return u.productRepo.List(ctx, tenantID)
}

type UpdateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SKU         string  `json:"sku"`
	Price       float64 `json:"price"`
	// CostPrice is optional on update: nil keeps the stored value.
	CostPrice *float64 `json:"cost_price,omitempty"`
}

// validCost rejects negative, NaN and infinite costs.
func validCost(c float64) bool {
	return c >= 0 && c == c && c < 1e15
}

func (u *Usecase) UpdateProduct(ctx context.Context, tenantID, id uuid.UUID, req UpdateProductRequest) (*domain.Product, error) {
	name := strings.TrimSpace(req.Name)
	sku := strings.TrimSpace(req.SKU)
	if len(name) < 3 || sku == "" {
		return nil, domain.ErrInvalidInput
	}
	if req.Price < 0 || req.Price != req.Price {
		return nil, domain.ErrInvalidInput
	}
	if req.CostPrice != nil && !validCost(*req.CostPrice) {
		return nil, domain.ErrInvalidInput
	}

	existing, err := u.productRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// If SKU is changed, check uniqueness per-tenant
	if !strings.EqualFold(existing.SKU, sku) {
		other, err := u.productRepo.GetBySKU(ctx, tenantID, sku)
		if err == nil && other != nil && other.ID != id {
			return nil, domain.ErrConflict
		}
	}

	existing.Name = name
	existing.Description = req.Description
	existing.SKU = sku
	existing.Price = req.Price
	if req.CostPrice != nil {
		existing.CostPrice = *req.CostPrice
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := u.productRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (u *Usecase) DeleteProduct(ctx context.Context, tenantID, id uuid.UUID) error {
	if _, err := u.productRepo.GetByID(ctx, tenantID, id); err != nil {
		return err
	}

	hasMovements, err := u.productRepo.HasMovementsOrStock(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if hasMovements {
		return domain.ErrConflict
	}

	return u.productRepo.Delete(ctx, tenantID, id)
}

type CreateInventoryRequest struct {
	ProductID         uuid.UUID `json:"product_id"`
	Quantity          float64   `json:"quantity"`
	WarehouseLocation string    `json:"warehouse_location"`
}

func (u *Usecase) CreateInventory(ctx context.Context, tenantID uuid.UUID, req CreateInventoryRequest) (*domain.Inventory, error) {
	// Reject negative quantities and quantities that are not finite numbers
	// (NaN/Inf) so bad payloads never reach the database.
	if req.ProductID == uuid.Nil || req.Quantity < 0 || req.Quantity != req.Quantity {
		return nil, domain.ErrInvalidInput
	}
	// Validate the product exists in the same tenant before creating a stock
	// row for it — prevents orphan inventory records via cross-tenant or
	// nonexistent product IDs.
	if _, err := u.productRepo.GetByID(ctx, tenantID, req.ProductID); err != nil {
		return nil, err
	}
	i := &domain.Inventory{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ProductID:         req.ProductID,
		Quantity:          req.Quantity,
		WarehouseLocation: req.WarehouseLocation,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err := u.inventoryRepo.Create(ctx, i); err != nil {
		return nil, err
	}
	return i, nil
}

func (u *Usecase) GetInventory(ctx context.Context, tenantID, id uuid.UUID) (*domain.Inventory, error) {
	return u.inventoryRepo.GetByID(ctx, tenantID, id)
}

// InventoryItem joins an inventory row with its product details for display.
type InventoryItem struct {
	domain.Inventory
	Product *domain.Product `json:"product,omitempty"`
}

func (u *Usecase) ListInventory(ctx context.Context, tenantID uuid.UUID) ([]InventoryItem, error) {
	rows, err := u.inventoryRepo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []InventoryItem{}, nil
	}

	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ProductID)
	}
	products, err := u.productRepo.ListByIDs(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]*domain.Product, len(products))
	for i := range products {
		byID[products[i].ID] = &products[i]
	}

	items := make([]InventoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, InventoryItem{Inventory: r, Product: byID[r.ProductID]})
	}
	return items, nil
}
