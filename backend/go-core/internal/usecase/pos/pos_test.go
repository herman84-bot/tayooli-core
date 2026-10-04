package pos_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pos"
)

type mockPOSRepo struct {
	orders []domain.POSOrder
	items  map[uuid.UUID][]domain.POSOrderItem
}

func (m *mockPOSRepo) CreateOrder(ctx context.Context, order *domain.POSOrder, items []domain.POSOrderItem) error {
	m.orders = append(m.orders, *order)
	m.items[order.ID] = items
	return nil
}

func (m *mockPOSRepo) GetOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.POSOrder, error) {
	for _, o := range m.orders {
		if o.ID == id && o.TenantID == tenantID {
			cpy := o
			cpy.Items = m.items[id]
			return &cpy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockPOSRepo) ListOrders(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.POSOrder, error) {
	var list []domain.POSOrder
	for _, o := range m.orders {
		if o.TenantID == tenantID {
			list = append(list, o)
		}
	}
	return list, nil
}

type mockProductRepo struct {
	products map[uuid.UUID]*domain.Product
}

func (m *mockProductRepo) Create(ctx context.Context, p *domain.Product) error { return nil }
func (m *mockProductRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	p, ok := m.products[id]
	if !ok || p.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}
func (m *mockProductRepo) GetBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.Product, error) {
	return nil, domain.ErrNotFound
}
func (m *mockProductRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Product, error) {
	return nil, nil
}
func (m *mockProductRepo) ListByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]domain.Product, error) {
	return nil, nil
}
func (m *mockProductRepo) Update(ctx context.Context, p *domain.Product) error { return nil }
func (m *mockProductRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error { return nil }
func (m *mockProductRepo) HasMovementsOrStock(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	return false, nil
}

type mockInventoryRepo struct {
	stocks map[uuid.UUID]*domain.Inventory
}

func (m *mockInventoryRepo) Create(ctx context.Context, i *domain.Inventory) error { return nil }
func (m *mockInventoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Inventory, error) {
	return nil, domain.ErrNotFound
}
func (m *mockInventoryRepo) GetByProductID(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.Inventory, error) {
	var list []domain.Inventory
	for _, inv := range m.stocks {
		if inv.TenantID == tenantID && inv.ProductID == productID {
			list = append(list, *inv)
		}
	}
	return list, nil
}
func (m *mockInventoryRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Inventory, error) {
	return nil, nil
}
func (m *mockInventoryRepo) UpdateQuantity(ctx context.Context, tenantID, id uuid.UUID, delta float64) error {
	if inv, ok := m.stocks[id]; ok {
		inv.Quantity += delta
	}
	return nil
}

type mockSalesOrderRepo struct {
	orders []domain.SalesOrder
}

func (m *mockSalesOrderRepo) Create(ctx context.Context, so *domain.SalesOrder) error {
	m.orders = append(m.orders, *so)
	return nil
}
func (m *mockSalesOrderRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	return nil, domain.ErrNotFound
}
func (m *mockSalesOrderRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	return m.orders, nil
}

type mockSalesInvoiceRepo struct {
	invoices []domain.SalesInvoice
}

func (m *mockSalesInvoiceRepo) Create(ctx context.Context, si *domain.SalesInvoice) error {
	m.invoices = append(m.invoices, *si)
	return nil
}
func (m *mockSalesInvoiceRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesInvoice, error) {
	return nil, domain.ErrNotFound
}
func (m *mockSalesInvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesInvoice, error) {
	return m.invoices, nil
}

type mockCustomerRepo struct {
	customers []domain.Customer
}

func (m *mockCustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	m.customers = append(m.customers, *c)
	return nil
}
func (m *mockCustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	return nil, domain.ErrNotFound
}
func (m *mockCustomerRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Customer, error) {
	return m.customers, nil
}

func TestPOSCheckout_SuccessAndStockDeduction(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	productID := uuid.New()
	pRepo := &mockProductRepo{
		products: map[uuid.UUID]*domain.Product{
			productID: {
				ID:       productID,
				TenantID: tenantID,
				Name:     "Beras Rojolele 10kg",
				SKU:      "SKU-ROJO-10K",
				Price:    145000,
			},
		},
	}

	invID := uuid.New()
	iRepo := &mockInventoryRepo{
		stocks: map[uuid.UUID]*domain.Inventory{
			invID: {
				ID:        invID,
				TenantID:  tenantID,
				ProductID: productID,
				Quantity:  10,
			},
		},
	}

	posRepo := &mockPOSRepo{items: make(map[uuid.UUID][]domain.POSOrderItem)}
	soRepo := &mockSalesOrderRepo{}
	siRepo := &mockSalesInvoiceRepo{}
	custRepo := &mockCustomerRepo{}

	uc := pos.New(posRepo, pRepo, iRepo, nil, custRepo, soRepo, siRepo)

	// 1. Normal Checkout
	req := pos.CheckoutRequest{
		Items: []pos.CheckoutItemRequest{
			{
				ProductID: productID,
				Quantity:  decimal.NewFromInt(2),
				Price:     decimal.NewFromInt(145000),
				Discount:  decimal.Zero,
			},
		},
		Payments: []pos.CheckoutPaymentRequest{
			{
				Method: "CASH",
				Amount: decimal.NewFromInt(300000),
			},
		},
		Tax:      decimal.NewFromInt(31900),
		Discount: decimal.Zero,
	}

	res, err := uc.Checkout(ctx, tenantID, userID, req)
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	if res.OrderNumber == "" {
		t.Errorf("expected non-empty order number")
	}
	expectedTotal := decimal.NewFromInt(290000 + 31900)
	if !res.Total.Equal(expectedTotal) {
		t.Errorf("expected total %s, got %s", expectedTotal, res.Total)
	}

	// Verify stock was deducted from 10 to 8
	if iRepo.stocks[invID].Quantity != 8 {
		t.Errorf("expected stock 8, got %.2f", iRepo.stocks[invID].Quantity)
	}

	// Verify Sales Order & Invoice created
	if len(soRepo.orders) != 1 {
		t.Errorf("expected 1 sales order created")
	}
	if len(siRepo.invoices) != 1 {
		t.Errorf("expected 1 sales invoice created")
	}

	// 2. Reject Checkout when Qty > Available Stock
	overReq := pos.CheckoutRequest{
		Items: []pos.CheckoutItemRequest{
			{
				ProductID: productID,
				Quantity:  decimal.NewFromInt(15), // available is now 8!
				Price:     decimal.NewFromInt(145000),
			},
		},
	}
	_, err = uc.Checkout(ctx, tenantID, userID, overReq)
	if err == nil {
		t.Fatalf("expected error when qty > stock, got nil")
	}
	// Verify stock remained unchanged at 8
	if iRepo.stocks[invID].Quantity != 8 {
		t.Errorf("expected stock to remain 8, got %.2f", iRepo.stocks[invID].Quantity)
	}
}
