package sales_invoice_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/sales_invoice"
)

type mockSalesInvoiceRepo struct {
	created *domain.SalesInvoice
}

func (m *mockSalesInvoiceRepo) Create(ctx context.Context, si *domain.SalesInvoice) error {
	m.created = si
	return nil
}

func (m *mockSalesInvoiceRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesInvoice, error) {
	return nil, nil
}

func (m *mockSalesInvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesInvoice, error) {
	return nil, nil
}

type mockSalesOrderRepo struct {
	orders map[string]*domain.SalesOrder
}

func (m *mockSalesOrderRepo) Create(ctx context.Context, so *domain.SalesOrder) error {
	return nil
}

func (m *mockSalesOrderRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	key := tenantID.String() + ":" + id.String()
	so, ok := m.orders[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return so, nil
}

func (m *mockSalesOrderRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	return nil, nil
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(ctx context.Context, topic, key string, payload interface{}) error {
	return nil
}

func TestSalesInvoice_Create_CrossTenantSalesOrder_Rejected(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	soOfTenantB := uuid.New()

	soRepo := &mockSalesOrderRepo{
		orders: map[string]*domain.SalesOrder{
			tenantB.String() + ":" + soOfTenantB.String(): {
				ID:       soOfTenantB,
				TenantID: tenantB,
			},
		},
	}
	siRepo := &mockSalesInvoiceRepo{}
	pub := &mockPublisher{}

	usecase := uc.NewWithDeps(siRepo, soRepo, pub)

	// Tenant A tries to create invoice referencing Tenant B's sales order
	_, err := usecase.Create(context.Background(), tenantA, uc.CreateRequest{
		SalesOrderID:  soOfTenantB,
		InvoiceNumber: "INV-001",
		Amount:        decimal.NewFromInt(1000),
		DueDate:       time.Now().Add(24 * time.Hour),
	})

	if err == nil {
		t.Fatal("expected error when referencing sales order from another tenant, got nil")
	}
}

func TestSalesInvoice_Create_OwnSalesOrder_Success(t *testing.T) {
	tenantA := uuid.New()
	soOfTenantA := uuid.New()

	soRepo := &mockSalesOrderRepo{
		orders: map[string]*domain.SalesOrder{
			tenantA.String() + ":" + soOfTenantA.String(): {
				ID:       soOfTenantA,
				TenantID: tenantA,
			},
		},
	}
	siRepo := &mockSalesInvoiceRepo{}
	pub := &mockPublisher{}

	usecase := uc.NewWithDeps(siRepo, soRepo, pub)

	si, err := usecase.Create(context.Background(), tenantA, uc.CreateRequest{
		SalesOrderID:  soOfTenantA,
		InvoiceNumber: "INV-001",
		Amount:        decimal.NewFromInt(1000),
		DueDate:       time.Now().Add(24 * time.Hour),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if si.TenantID != tenantA {
		t.Errorf("expected tenantID %v, got %v", tenantA, si.TenantID)
	}
	if si.SalesOrderID != soOfTenantA {
		t.Errorf("expected salesOrderID %v, got %v", soOfTenantA, si.SalesOrderID)
	}
}
