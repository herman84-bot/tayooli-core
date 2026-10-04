package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	salesInvoiceUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/sales_invoice"
	salesOrderUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/sales_order"
)

type mockSOHandlerRepo struct{}

func (m *mockSOHandlerRepo) Create(ctx context.Context, so *domain.SalesOrder) error {
	return nil
}
func (m *mockSOHandlerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	return nil, nil
}
func (m *mockSOHandlerRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	return nil, nil
}

type mockSIHandlerRepo struct{}

func (m *mockSIHandlerRepo) Create(ctx context.Context, si *domain.SalesInvoice) error {
	return nil
}
func (m *mockSIHandlerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesInvoice, error) {
	return nil, nil
}
func (m *mockSIHandlerRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesInvoice, error) {
	return nil, nil
}

type mockSIPublisher struct{}

func (m *mockSIPublisher) Publish(ctx context.Context, topic, key string, payload interface{}) error {
	return nil
}

func TestSalesOrderHandler_Create_UnauthorizedWithoutTenant(t *testing.T) {
	soUC := salesOrderUC.New(&mockSOHandlerRepo{})
	h := handler.NewSalesOrderHandler(soUC)

	body, _ := json.Marshal(map[string]any{
		"order_number": "SO-001",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales-orders", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestSalesInvoiceHandler_Create_UnauthorizedWithoutTenant(t *testing.T) {
	siUC := salesInvoiceUC.New(&mockSIHandlerRepo{}, &mockSIPublisher{})
	h := handler.NewSalesInvoiceHandler(siUC)

	body, _ := json.Marshal(map[string]any{
		"invoice_number": "INV-001",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sales-invoices", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
}
