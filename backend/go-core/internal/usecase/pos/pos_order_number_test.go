package pos_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pos"
)

// uniquePOSRepo enforces the same invariant the database enforces through
// `pos_orders_tenant_id_order_number_key` (see
// migrations/026_pos_and_wms_ledger.sql: UNIQUE (tenant_id, order_number)).
// A collision is reported the way PostgreSQL reports it, so a test failure
// carries the same signal a cashier would see in production.
type uniquePOSRepo struct {
	mu         sync.Mutex
	seen       map[string]struct{}
	orders     []domain.POSOrder
	duplicates []string
}

func newUniquePOSRepo() *uniquePOSRepo {
	return &uniquePOSRepo{seen: make(map[string]struct{})}
}

func (m *uniquePOSRepo) CreateOrder(ctx context.Context, order *domain.POSOrder, items []domain.POSOrderItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s|%s", order.TenantID, order.OrderNumber)
	if _, exists := m.seen[key]; exists {
		m.duplicates = append(m.duplicates, order.OrderNumber)
		return fmt.Errorf(`POSRepo.CreateOrder: exec order: pq: duplicate key value violates unique constraint "pos_orders_tenant_id_order_number_key" (23505)`)
	}
	m.seen[key] = struct{}{}
	m.orders = append(m.orders, *order)
	return nil
}

func (m *uniquePOSRepo) CreateSale(ctx context.Context, sale *domain.POSSale) error {
	return m.CreateOrder(ctx, sale.Order, sale.Items)
}

func (m *uniquePOSRepo) GetOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.POSOrder, error) {
	return nil, domain.ErrNotFound
}

func (m *uniquePOSRepo) ListOrders(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.POSOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.POSOrder(nil), m.orders...), nil
}

// TestPOSCheckout_OrderNumbersStayUniqueUnderConcurrentCashiers reproduces the
// production failure mode: several cashiers pressing "bayar" at the same time.
// Every checkout must succeed, because two different sales can never share an
// order number.
func TestPOSCheckout_OrderNumbersStayUniqueUnderConcurrentCashiers(t *testing.T) {
	const workers = 8
	const perWorker = 400 // 3.200 sales/day: above the 9.000-slot ceiling's safe range
	const total = workers * perWorker

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
	posRepo := newUniquePOSRepo()
	// inventory/wms/customer/sales repos are nil on purpose: this test isolates
	// order-number generation, the only thing the bug is about.
	uc := pos.New(posRepo, pRepo, nil, nil, nil, nil, nil)

	req := pos.CheckoutRequest{
		Items: []pos.CheckoutItemRequest{
			{
				ProductID: productID,
				Quantity:  decimal.NewFromInt(1),
				Price:     decimal.NewFromInt(145000),
				Discount:  decimal.Zero,
			},
		},
		Payments: []pos.CheckoutPaymentRequest{
			{Method: "CASH", Amount: decimal.NewFromInt(145000)},
		},
	}

	start := make(chan struct{})
	errCh := make(chan error, total)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < perWorker; i++ {
				if _, err := uc.Checkout(ctx, tenantID, userID, req); err != nil {
					errCh <- err
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Fatalf("%d/%d checkouts failed; first error: %v", len(errs), total, errs[0])
	}

	orders, err := posRepo.ListOrders(ctx, tenantID, total)
	if err != nil {
		t.Fatalf("ListOrders failed: %v", err)
	}
	if len(orders) != total {
		t.Fatalf("expected %d persisted orders, got %d", total, len(orders))
	}

	numbers := make(map[string]struct{}, len(orders))
	for _, o := range orders {
		numbers[o.OrderNumber] = struct{}{}
	}
	if len(numbers) != total {
		t.Fatalf("expected %d distinct order numbers, got %d (duplicates: %v)",
			total, len(numbers), posRepo.duplicates)
	}
	for _, n := range posRepo.duplicates {
		if strings.Contains(n, "duplicate") {
			t.Fatalf("unexpected duplicate marker in order number: %s", n)
		}
	}
}
