package pos_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pos"
)

func atomicFixture(t *testing.T) (*pos.Usecase, *mockPOSRepo, *mockInventoryRepo, *mockSalesOrderRepo, *mockSalesInvoiceRepo, uuid.UUID, uuid.UUID, [2]uuid.UUID, [2]uuid.UUID) {
	t.Helper()
	tenantID, userID := uuid.New(), uuid.New()
	prods := [2]uuid.UUID{uuid.New(), uuid.New()}
	invs := [2]uuid.UUID{uuid.New(), uuid.New()}
	pRepo := &mockProductRepo{products: map[uuid.UUID]*domain.Product{
		prods[0]: {ID: prods[0], TenantID: tenantID, Name: "Gula 1kg", SKU: "GULA-1", Price: 15000},
		prods[1]: {ID: prods[1], TenantID: tenantID, Name: "Teh 25s", SKU: "TEH-25", Price: 8000},
	}}
	iRepo := &mockInventoryRepo{stocks: map[uuid.UUID]*domain.Inventory{
		invs[0]: {ID: invs[0], TenantID: tenantID, ProductID: prods[0], Quantity: 10},
		invs[1]: {ID: invs[1], TenantID: tenantID, ProductID: prods[1], Quantity: 10},
	}}
	soRepo, siRepo := &mockSalesOrderRepo{}, &mockSalesInvoiceRepo{}
	posRepo := &mockPOSRepo{items: map[uuid.UUID][]domain.POSOrderItem{}, inv: iRepo, so: soRepo, si: siRepo}
	uc := pos.New(posRepo, pRepo, iRepo, nil, &mockCustomerRepo{}, soRepo, siRepo)
	return uc, posRepo, iRepo, soRepo, siRepo, tenantID, userID, prods, invs
}

func twoItemCart(prods [2]uuid.UUID) pos.CheckoutRequest {
	return pos.CheckoutRequest{Items: []pos.CheckoutItemRequest{
		{ProductID: prods[0], Quantity: decimal.NewFromInt(2)},
		{ProductID: prods[1], Quantity: decimal.NewFromInt(3)},
	}}
}

// Checkout must hand every write to ONE CreateSale call, with consistent links.
func TestCheckout_BuildsOneAtomicSale(t *testing.T) {
	uc, posRepo, _, _, _, tenantID, userID, prods, _ := atomicFixture(t)

	res, err := uc.Checkout(context.Background(), tenantID, userID, twoItemCart(prods))
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if len(posRepo.sales) != 1 {
		t.Fatalf("CreateSale calls = %d, want 1", len(posRepo.sales))
	}
	sale := posRepo.sales[0]
	if len(sale.Items) != 2 || len(sale.InventoryDeductions) != 2 {
		t.Fatalf("items=%d inventory deductions=%d, want 2/2", len(sale.Items), len(sale.InventoryDeductions))
	}
	if sale.SalesOrder == nil || sale.SalesInvoice == nil {
		t.Fatalf("sales order / invoice must be part of the atomic sale")
	}
	if *sale.Order.SalesOrderID != sale.SalesOrder.ID || *sale.Order.SalesInvoiceID != sale.SalesInvoice.ID {
		t.Fatalf("POS order must link the sales order and invoice written with it")
	}
	if sale.SalesInvoice.SalesOrderID != sale.SalesOrder.ID {
		t.Fatalf("invoice must reference the sales order")
	}
	if sale.Order.OrderNumber != res.OrderNumber {
		t.Fatalf("response order number %q != saved %q", res.OrderNumber, sale.Order.OrderNumber)
	}
	for _, it := range sale.Items {
		if it.POSOrderID != sale.Order.ID {
			t.Fatalf("item %s not linked to order", it.SKU)
		}
	}
}

// If saving the sale fails, nothing may change: no stock deducted, no sales
// order, no invoice, no POS order (the bug that lost 558 units in load test).
func TestCheckout_FailedSaveChangesNothing(t *testing.T) {
	uc, posRepo, iRepo, soRepo, siRepo, tenantID, userID, prods, invs := atomicFixture(t)
	posRepo.failSale = errors.New("pq: could not serialize access / connection lost")

	_, err := uc.Checkout(context.Background(), tenantID, userID, twoItemCart(prods))
	if err == nil {
		t.Fatal("expected checkout error")
	}
	if iRepo.stocks[invs[0]].Quantity != 10 || iRepo.stocks[invs[1]].Quantity != 10 {
		t.Fatalf("stock changed on failed checkout: %v / %v", iRepo.stocks[invs[0]].Quantity, iRepo.stocks[invs[1]].Quantity)
	}
	if len(soRepo.orders) != 0 || len(siRepo.invoices) != 0 || len(posRepo.orders) != 0 {
		t.Fatalf("partial writes: so=%d inv=%d pos=%d", len(soRepo.orders), len(siRepo.invoices), len(posRepo.orders))
	}
}

// Second line short of stock: the first line must not be deducted either.
func TestCheckout_ShortSecondLineDeductsNothing(t *testing.T) {
	uc, posRepo, iRepo, soRepo, _, tenantID, userID, prods, invs := atomicFixture(t)
	req := twoItemCart(prods)
	req.Items[1].Quantity = decimal.NewFromInt(50) // only 10 in stock

	_, err := uc.Checkout(context.Background(), tenantID, userID, req)
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("want ErrInsufficientStock, got %v", err)
	}
	if iRepo.stocks[invs[0]].Quantity != 10 {
		t.Fatalf("first line deducted although the sale failed: %v", iRepo.stocks[invs[0]].Quantity)
	}
	if len(posRepo.orders) != 0 || len(soRepo.orders) != 0 {
		t.Fatal("no order may be written for a refused sale")
	}
}

func TestPOSSaleLockOrderIsDeterministic(t *testing.T) {
	a, b := uuid.MustParse("00000000-0000-0000-0000-00000000000a"), uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	s1 := &domain.POSSale{StockDeductions: []domain.POSStockDeduction{{ProductID: b}, {ProductID: a}}, InventoryDeductions: []domain.POSInventoryDeduction{{ProductID: b}, {ProductID: a}}}
	s1.LockOrder()
	if s1.StockDeductions[0].ProductID != a || s1.InventoryDeductions[0].ProductID != a {
		t.Fatal("deductions must be sorted by product id so concurrent carts lock in the same order")
	}
}
