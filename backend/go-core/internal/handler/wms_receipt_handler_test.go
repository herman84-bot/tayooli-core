package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mockWMSUsecase: Stock Receipts ---

func (m *mockWMSUsecase) CreateStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
	if m.createStockReceiptFn != nil {
		return m.createStockReceiptFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil, nil
}

func (m *mockWMSUsecase) UpdateStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, req uc.StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
	if m.updateStockReceiptFn != nil {
		return m.updateStockReceiptFn(ctx, tenantID, userID, role, receiptID, req)
	}
	return nil, nil, nil
}

func (m *mockWMSUsecase) GetStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
	if m.getStockReceiptFn != nil {
		return m.getStockReceiptFn(ctx, tenantID, userID, role, receiptID)
	}
	return nil, nil, nil
}

func (m *mockWMSUsecase) ListStockReceipts(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID, status *domain.StockReceiptStatus, receiptType *domain.StockReceiptType) ([]domain.StockReceipt, error) {
	if m.listStockReceiptsFn != nil {
		return m.listStockReceiptsFn(ctx, tenantID, userID, role, warehouseID, status, receiptType)
	}
	return nil, nil
}

func (m *mockWMSUsecase) PostStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, error) {
	if m.postStockReceiptFn != nil {
		return m.postStockReceiptFn(ctx, tenantID, userID, role, receiptID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CancelStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, reason string) (*domain.StockReceipt, error) {
	if m.cancelStockReceiptFn != nil {
		return m.cancelStockReceiptFn(ctx, tenantID, userID, role, receiptID, reason)
	}
	return nil, nil
}

func TestWMSStockReceiptHandlers(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	role := "admin"

	t.Run("POST /wms/receipts creates DRAFT with 201", func(t *testing.T) {
		rcID := uuid.New()
		var got uc.StockReceiptRequest
		mock := &mockWMSUsecase{
			createStockReceiptFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
				got = req
				return &domain.StockReceipt{ID: rcID, TenantID: tid, ReceiptNumber: "GR-20250101-ABCD1234", Status: domain.StockReceiptStatusDraft, SupplierName: req.SupplierName, ItemCount: 1},
					[]domain.StockReceiptItem{{ID: uuid.New(), ReceiptID: rcID, ProductID: req.Items[0].ProductID, AcceptedQty: req.Items[0].AcceptedQty, ProductName: "Kopi", ProductSKU: "KP-1"}}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		prodID := uuid.New()
		body := fmt.Sprintf(`{"warehouse_id":"%s","dest_location_id":"%s","supplier_name":"PT Sumber","source_ref":"PO-2026-001","items":[{"product_id":"%s","accepted_qty":"10","rejected_qty":"0"}]}`, uuid.New(), uuid.New(), prodID)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/receipts", bytes.NewBufferString(body)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		assert.Equal(t, "PT Sumber", got.SupplierName)
		require.Len(t, got.Items, 1)
		assert.True(t, got.Items[0].AcceptedQty.Equal(decimal.NewFromInt(10)))
		var resp map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp, "receipt")
		assert.Contains(t, resp, "items")
		var rc domain.StockReceipt
		require.NoError(t, json.Unmarshal(resp["receipt"], &rc))
		assert.Equal(t, rcID, rc.ID)
		assert.Equal(t, domain.StockReceiptStatusDraft, rc.Status)
	})

	t.Run("POST /wms/receipts validation error returns 400 with Indonesian message", func(t *testing.T) {
		mock := &mockWMSUsecase{
			createStockReceiptFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
				return nil, nil, &domain.StockReceiptValidationError{Msg: "Nama pemasok wajib diisi"}
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/receipts", bytes.NewBufferString(`{"items":[]}`)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Nama pemasok wajib diisi")
	})

	t.Run("POST /wms/receipts invalid json returns 400", func(t *testing.T) {
		router := setupWMSTestRouter(&mockWMSUsecase{})
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/receipts", bytes.NewBufferString(`{bad`)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST /wms/receipts/{id}/post returns 200 POSTED", func(t *testing.T) {
		rcID := uuid.New()
		mock := &mockWMSUsecase{
			postStockReceiptFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockReceipt, error) {
				assert.Equal(t, rcID, id)
				return &domain.StockReceipt{ID: id, Status: domain.StockReceiptStatusPosted, PostedBy: &uid}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/receipts/"+rcID.String()+"/post", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var rc domain.StockReceipt
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rc))
		assert.Equal(t, domain.StockReceiptStatusPosted, rc.Status)
	})

	t.Run("POST /wms/receipts/{id}/post twice returns 409", func(t *testing.T) {
		mock := &mockWMSUsecase{
			postStockReceiptFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockReceipt, error) {
				return nil, domain.ErrStockReceiptNotDraft
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/receipts/"+uuid.New().String()+"/post", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "tidak bisa diubah")
	})

	t.Run("POST /wms/receipts/{bad}/post returns 400", func(t *testing.T) {
		router := setupWMSTestRouter(&mockWMSUsecase{})
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/receipts/not-a-uuid/post", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST /wms/receipts/{id}/cancel with consumed stock returns 422", func(t *testing.T) {
		var gotReason string
		mock := &mockWMSUsecase{
			cancelStockReceiptFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID, reason string) (*domain.StockReceipt, error) {
				gotReason = reason
				return nil, fmt.Errorf("wrap: %w", domain.ErrStockReceiptStockConsumed)
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/receipts/"+uuid.New().String()+"/cancel", bytes.NewBufferString(`{"reason":"salah input"}`)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Equal(t, "salah input", gotReason)
	})

	t.Run("GET /wms/receipts returns data envelope", func(t *testing.T) {
		var gotStatus *domain.StockReceiptStatus
		mock := &mockWMSUsecase{
			listStockReceiptsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, wh *uuid.UUID, st *domain.StockReceiptStatus, tp *domain.StockReceiptType) ([]domain.StockReceipt, error) {
				gotStatus = st
				return nil, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/receipts?status=draft", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, gotStatus)
		assert.Equal(t, domain.StockReceiptStatusDraft, *gotStatus)
		assert.JSONEq(t, `{"data":[]}`, w.Body.String())
	})
}
