package sales_order_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/sales_order"
)

type mockSalesOrderRepo struct {
	created *domain.SalesOrder
}

func (m *mockSalesOrderRepo) Create(ctx context.Context, so *domain.SalesOrder) error {
	m.created = so
	return nil
}

func (m *mockSalesOrderRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	return nil, nil
}

func (m *mockSalesOrderRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	return nil, nil
}

type mockCustomerRepo struct {
	customers map[string]*domain.Customer
}

func (m *mockCustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	return nil
}

func (m *mockCustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	key := tenantID.String() + ":" + id.String()
	c, ok := m.customers[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (m *mockCustomerRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Customer, error) {
	return nil, nil
}

func TestSalesOrder_Create_CrossTenantCustomer_Rejected(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	customerOfTenantB := uuid.New()

	custRepo := &mockCustomerRepo{
		customers: map[string]*domain.Customer{
			tenantB.String() + ":" + customerOfTenantB.String(): {
				ID:       customerOfTenantB,
				TenantID: tenantB,
				Name:     "Tenant B Customer",
			},
		},
	}
	soRepo := &mockSalesOrderRepo{}

	usecase := uc.NewWithDeps(soRepo, custRepo)

	// Tenant A tries to create a sales order referencing Tenant B's customer
	_, err := usecase.Create(context.Background(), tenantA, uc.CreateRequest{
		CustomerID:  customerOfTenantB,
		OrderNumber: "SO-001",
		TotalAmount: decimal.NewFromInt(1000),
	})

	if err == nil {
		t.Fatal("expected error when referencing customer from another tenant, got nil")
	}
}

func TestSalesOrder_Create_OwnCustomer_Success(t *testing.T) {
	tenantA := uuid.New()
	customerOfTenantA := uuid.New()

	custRepo := &mockCustomerRepo{
		customers: map[string]*domain.Customer{
			tenantA.String() + ":" + customerOfTenantA.String(): {
				ID:       customerOfTenantA,
				TenantID: tenantA,
				Name:     "Tenant A Customer",
			},
		},
	}
	soRepo := &mockSalesOrderRepo{}

	usecase := uc.NewWithDeps(soRepo, custRepo)

	so, err := usecase.Create(context.Background(), tenantA, uc.CreateRequest{
		CustomerID:  customerOfTenantA,
		OrderNumber: "SO-001",
		TotalAmount: decimal.NewFromInt(1000),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if so.TenantID != tenantA {
		t.Errorf("expected tenantID %v, got %v", tenantA, so.TenantID)
	}
	if so.CustomerID != customerOfTenantA {
		t.Errorf("expected customerID %v, got %v", customerOfTenantA, so.CustomerID)
	}
}
