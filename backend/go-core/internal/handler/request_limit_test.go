package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// TestPOCreate64KiBLimit verifies that CreatePO rejects requests over 64 KiB
func TestPOCreate64KiBLimit(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockPOUsecase{}
	handler := NewPOHandler(uc)

	// Create a large JSON payload (over 64 KiB)
	largePayload := map[string]interface{}{
		"vendor_id": "vendor-1",
		"po_number": "PO-001",
		"amount":    "1000.00",
		"qty":       10,
		"currency":  "USD",
		"extra":     string(bytes.Repeat([]byte("x"), 65*1024)), // 65 KiB padding
	}
	body, _ := json.Marshal(largePayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set tenant in context
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreatePO(rr, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
}

// TestGRCreate64KiBLimit verifies that CreateGR rejects requests over 64 KiB
func TestGRCreate64KiBLimit(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockGRUsecase{}
	handler := NewGRHandler(uc)

	// Create a large JSON payload (over 64 KiB)
	largePayload := map[string]interface{}{
		"po_id":           uuid.New().String(),
		"vendor_id":       "vendor-1",
		"received_qty":    10,
		"received_amount": "1000.00",
		"currency":        "USD",
		"extra":           string(bytes.Repeat([]byte("x"), 65*1024)), // 65 KiB padding
	}
	body, _ := json.Marshal(largePayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set tenant in context
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.CreateGR(rr, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
}

// Mock PO usecase for testing
type mockPOUsecase struct{}

func (m *mockPOUsecase) Create(ctx context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error) {
	return &domain.PurchaseOrder{}, nil
}

func (m *mockPOUsecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.PurchaseOrder, error) {
	return nil, nil
}

func (m *mockPOUsecase) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error) {
	return &domain.PurchaseOrderListPage{Data: []domain.PurchaseOrder{}, Total: 0, Page: page, PerPage: perPage}, nil
}

func (m *mockPOUsecase) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error) {
	return nil, nil
}

// Mock GR usecase for testing
type mockGRUsecase struct{}

func (m *mockGRUsecase) Create(ctx context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error) {
	return &domain.GoodsReceipt{}, nil
}

func (m *mockGRUsecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.GoodsReceipt, error) {
	return nil, nil
}

func (m *mockGRUsecase) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error) {
	return &domain.GoodsReceiptListPage{Data: []domain.GoodsReceipt{}, Total: 0, Page: page, PerPage: perPage}, nil
}

func (m *mockGRUsecase) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.GoodsReceipt, error) {
	return nil, nil
}