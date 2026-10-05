package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWMSUsecase struct {
	createWarehouseFn     func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateWarehouseRequest) (*domain.Warehouse, error)
	getWarehouseFn        func(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID) (*domain.Warehouse, error)
	listWarehousesFn      func(ctx context.Context, tenantID, userID uuid.UUID, role string) ([]domain.Warehouse, error)
	assignUserWarehouseFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, targetUserID, warehouseID uuid.UUID) error

	createLocationFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateLocationRequest) (*domain.WarehouseLocation, error)
	listLocationsFn  func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.WarehouseLocation, error)

	resolveBarcodeFn   func(ctx context.Context, tenantID uuid.UUID, code string) (*domain.ResolvedProduct, error)
	createBarcodeFn    func(ctx context.Context, tenantID uuid.UUID, req uc.CreateBarcodeRequest) (*domain.ProductBarcode, error)
	createSKUMappingFn func(ctx context.Context, tenantID uuid.UUID, req uc.CreateSKUMappingRequest) (*domain.ProductSKUMapping, error)

	createTransferFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateTransferRequest) (*domain.StockTransfer, error)
	submitTransferFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)
	approveTransferFn  func(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)
	rejectTransferFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID, reason string) (*domain.StockTransfer, error)
	dispatchTransferFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)
	receiveTransferFn  func(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)
	getTransferFn      func(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, []domain.StockTransferItem, error)
	listTransfersFn    func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockTransfer, error)
	cancelTransferFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)

	createDeliveryOrderFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateDeliveryOrderRequest) (*domain.DeliveryOrder, error)
	dispatchDeliveryOrderFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, error)
	getDeliveryOrderFn      func(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, []domain.DeliveryOrderItem, error)
	listDeliveryOrdersFn    func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.DeliveryOrder, error)

	createStockOpnameFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateStockOpnameRequest) (*domain.StockOpname, error)
	getStockOpnameFn      func(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID) (*domain.StockOpname, []domain.StockOpnameItem, error)
	listStockOpnamesFn    func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockOpname, error)
	addOpnameItemFn       func(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID, req uc.AddOpnameItemRequest) (*domain.StockOpnameItem, error)
	completeStockOpnameFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID) (*domain.StockOpname, error)

	createStockScrapFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateStockScrapRequest) (*domain.StockScrap, error)
	listStockScrapsFn  func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockScrap, error)

	createStockReceiptFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error)
	updateStockReceiptFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, req uc.StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error)
	getStockReceiptFn    func(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, []domain.StockReceiptItem, error)
	listStockReceiptsFn  func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID, status *domain.StockReceiptStatus, receiptType *domain.StockReceiptType) ([]domain.StockReceipt, error)
	postStockReceiptFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, error)
	cancelStockReceiptFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, reason string) (*domain.StockReceipt, error)

	listStockMovementsFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, productID, locationID *uuid.UUID, limit int) ([]domain.StockMovement, error)
	listStockSummaryFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockSummary, error)

	importMarketplaceOrdersFn func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.ImportMarketplaceOrdersRequest) (*uc.ImportMarketplaceOrdersResponse, error)
	resolveSKUMappingFn       func(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateSKUMappingRequest) (*domain.ProductSKUMapping, error)
	listMarketplaceBatchesFn  func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.MarketplaceImportBatch, error)
	listMarketplaceOrdersFn   func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID, batchID *uuid.UUID, status *domain.MarketplaceOrderStatus) ([]domain.MarketplaceOrder, error)
	getMarketplaceOrderFn     func(ctx context.Context, tenantID, userID uuid.UUID, role string, orderID uuid.UUID) (*domain.MarketplaceOrder, error)
	listSKUMappingsFn         func(ctx context.Context, tenantID uuid.UUID, channelName string) ([]domain.ProductSKUMapping, error)
}

func (m *mockWMSUsecase) CreateWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateWarehouseRequest) (*domain.Warehouse, error) {
	if m.createWarehouseFn != nil {
		return m.createWarehouseFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) GetWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID) (*domain.Warehouse, error) {
	if m.getWarehouseFn != nil {
		return m.getWarehouseFn(ctx, tenantID, userID, role, id)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ListWarehouses(ctx context.Context, tenantID, userID uuid.UUID, role string) ([]domain.Warehouse, error) {
	if m.listWarehousesFn != nil {
		return m.listWarehousesFn(ctx, tenantID, userID, role)
	}
	return nil, nil
}

func (m *mockWMSUsecase) AssignUserWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, targetUserID, warehouseID uuid.UUID) error {
	if m.assignUserWarehouseFn != nil {
		return m.assignUserWarehouseFn(ctx, tenantID, userID, role, targetUserID, warehouseID)
	}
	return nil
}

func (m *mockWMSUsecase) CreateLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateLocationRequest) (*domain.WarehouseLocation, error) {
	if m.createLocationFn != nil {
		return m.createLocationFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ListLocations(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.WarehouseLocation, error) {
	if m.listLocationsFn != nil {
		return m.listLocationsFn(ctx, tenantID, userID, role, warehouseID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ResolveBarcode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.ResolvedProduct, error) {
	if m.resolveBarcodeFn != nil {
		return m.resolveBarcodeFn(ctx, tenantID, code)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CreateBarcode(ctx context.Context, tenantID uuid.UUID, req uc.CreateBarcodeRequest) (*domain.ProductBarcode, error) {
	if m.createBarcodeFn != nil {
		return m.createBarcodeFn(ctx, tenantID, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CreateSKUMapping(ctx context.Context, tenantID uuid.UUID, req uc.CreateSKUMappingRequest) (*domain.ProductSKUMapping, error) {
	if m.createSKUMappingFn != nil {
		return m.createSKUMappingFn(ctx, tenantID, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CreateTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateTransferRequest) (*domain.StockTransfer, error) {
	if m.createTransferFn != nil {
		return m.createTransferFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) SubmitTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	if m.submitTransferFn != nil {
		return m.submitTransferFn(ctx, tenantID, userID, role, transferID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ApproveTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	if m.approveTransferFn != nil {
		return m.approveTransferFn(ctx, tenantID, userID, role, transferID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) RejectTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID, reason string) (*domain.StockTransfer, error) {
	if m.rejectTransferFn != nil {
		return m.rejectTransferFn(ctx, tenantID, userID, role, transferID, reason)
	}
	return nil, nil
}

func (m *mockWMSUsecase) DispatchTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	if m.dispatchTransferFn != nil {
		return m.dispatchTransferFn(ctx, tenantID, userID, role, transferID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ReceiveTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	if m.receiveTransferFn != nil {
		return m.receiveTransferFn(ctx, tenantID, userID, role, transferID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) GetTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, []domain.StockTransferItem, error) {
	if m.getTransferFn != nil {
		return m.getTransferFn(ctx, tenantID, userID, role, transferID)
	}
	return nil, nil, nil
}

func (m *mockWMSUsecase) ListTransfers(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockTransfer, error) {
	if m.listTransfersFn != nil {
		return m.listTransfersFn(ctx, tenantID, userID, role, warehouseID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CancelTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error) {
	if m.cancelTransferFn != nil {
		return m.cancelTransferFn(ctx, tenantID, userID, role, transferID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CreateDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateDeliveryOrderRequest) (*domain.DeliveryOrder, error) {
	if m.createDeliveryOrderFn != nil {
		return m.createDeliveryOrderFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) DispatchDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, error) {
	if m.dispatchDeliveryOrderFn != nil {
		return m.dispatchDeliveryOrderFn(ctx, tenantID, userID, role, doID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) GetDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, []domain.DeliveryOrderItem, error) {
	if m.getDeliveryOrderFn != nil {
		return m.getDeliveryOrderFn(ctx, tenantID, userID, role, doID)
	}
	return nil, nil, nil
}

func (m *mockWMSUsecase) ListDeliveryOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.DeliveryOrder, error) {
	if m.listDeliveryOrdersFn != nil {
		return m.listDeliveryOrdersFn(ctx, tenantID, userID, role, warehouseID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CreateStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateStockOpnameRequest) (*domain.StockOpname, error) {
	if m.createStockOpnameFn != nil {
		return m.createStockOpnameFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) GetStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID) (*domain.StockOpname, []domain.StockOpnameItem, error) {
	if m.getStockOpnameFn != nil {
		return m.getStockOpnameFn(ctx, tenantID, userID, role, opnameID)
	}
	return nil, nil, nil
}

func (m *mockWMSUsecase) ListStockOpnames(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockOpname, error) {
	if m.listStockOpnamesFn != nil {
		return m.listStockOpnamesFn(ctx, tenantID, userID, role, warehouseID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) AddOpnameItem(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID, req uc.AddOpnameItemRequest) (*domain.StockOpnameItem, error) {
	if m.addOpnameItemFn != nil {
		return m.addOpnameItemFn(ctx, tenantID, userID, role, opnameID, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CompleteStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID) (*domain.StockOpname, error) {
	if m.completeStockOpnameFn != nil {
		return m.completeStockOpnameFn(ctx, tenantID, userID, role, opnameID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) CreateStockScrap(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateStockScrapRequest) (*domain.StockScrap, error) {
	if m.createStockScrapFn != nil {
		return m.createStockScrapFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ListStockScraps(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockScrap, error) {
	if m.listStockScrapsFn != nil {
		return m.listStockScrapsFn(ctx, tenantID, userID, role, warehouseID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ListStockMovements(ctx context.Context, tenantID, userID uuid.UUID, role string, productID, locationID *uuid.UUID, limit int) ([]domain.StockMovement, error) {
	if m.listStockMovementsFn != nil {
		return m.listStockMovementsFn(ctx, tenantID, userID, role, productID, locationID, limit)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ListStockSummary(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockSummary, error) {
	if m.listStockSummaryFn != nil {
		return m.listStockSummaryFn(ctx, tenantID, userID, role, warehouseID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ImportMarketplaceOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.ImportMarketplaceOrdersRequest) (*uc.ImportMarketplaceOrdersResponse, error) {
	if m.importMarketplaceOrdersFn != nil {
		return m.importMarketplaceOrdersFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ResolveSKUMapping(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateSKUMappingRequest) (*domain.ProductSKUMapping, error) {
	if m.resolveSKUMappingFn != nil {
		return m.resolveSKUMappingFn(ctx, tenantID, userID, role, req)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ListMarketplaceBatches(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.MarketplaceImportBatch, error) {
	if m.listMarketplaceBatchesFn != nil {
		return m.listMarketplaceBatchesFn(ctx, tenantID, userID, role, warehouseID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ListMarketplaceOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID, batchID *uuid.UUID, status *domain.MarketplaceOrderStatus) ([]domain.MarketplaceOrder, error) {
	if m.listMarketplaceOrdersFn != nil {
		return m.listMarketplaceOrdersFn(ctx, tenantID, userID, role, warehouseID, batchID, status)
	}
	return nil, nil
}

func (m *mockWMSUsecase) GetMarketplaceOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, orderID uuid.UUID) (*domain.MarketplaceOrder, error) {
	if m.getMarketplaceOrderFn != nil {
		return m.getMarketplaceOrderFn(ctx, tenantID, userID, role, orderID)
	}
	return nil, nil
}

func (m *mockWMSUsecase) ListSKUMappings(ctx context.Context, tenantID uuid.UUID, channelName string) ([]domain.ProductSKUMapping, error) {
	if m.listSKUMappingsFn != nil {
		return m.listSKUMappingsFn(ctx, tenantID, channelName)
	}
	return nil, nil
}

func withWMSAuth(r *http.Request, tenantID, userID uuid.UUID, role string) *http.Request {
	ctx := context.WithValue(r.Context(), appMiddleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, appMiddleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, appMiddleware.RoleKey, role)
	return r.WithContext(ctx)
}

func setupWMSTestRouter(mock *mockWMSUsecase) *chi.Mux {
	r := chi.NewRouter()
	h := handler.NewWMSHandler(mock)
	r.Route("/api/v1", func(r chi.Router) {
		h.RegisterRoutes(r)
	})
	return r
}

func TestWMSHandlerEndpoints(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	role := "admin"

	t.Run("GET /api/v1/wms/warehouses returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listWarehousesFn: func(ctx context.Context, tid, uid uuid.UUID, r string) ([]domain.Warehouse, error) {
				return []domain.Warehouse{
					{ID: uuid.New(), TenantID: tid, Code: "WH-1", Name: "Jakarta Hub", IsActive: true},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/warehouses", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "data")
	})

	t.Run("POST /api/v1/wms/warehouses creates warehouse with 201", func(t *testing.T) {
		whID := uuid.New()
		mock := &mockWMSUsecase{
			createWarehouseFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateWarehouseRequest) (*domain.Warehouse, error) {
				return &domain.Warehouse{
					ID:        whID,
					TenantID:  tid,
					Code:      req.Code,
					Name:      req.Name,
					IsActive:  true,
					CreatedAt: time.Now(),
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		body := uc.CreateWarehouseRequest{Code: "WH-NEW", Name: "Surabaya Center"}
		jsonBytes, _ := json.Marshal(body)

		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/warehouses", bytes.NewReader(jsonBytes)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var res domain.Warehouse
		err := json.NewDecoder(w.Body).Decode(&res)
		require.NoError(t, err)
		assert.Equal(t, whID, res.ID)
		assert.Equal(t, "WH-NEW", res.Code)
	})

	t.Run("GET /api/v1/wms/locations returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listLocationsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, whID *uuid.UUID) ([]domain.WarehouseLocation, error) {
				return []domain.WarehouseLocation{
					{ID: uuid.New(), TenantID: tid, Code: "RACK-01", Name: "Rack 01", Type: domain.LocationTypeInternal},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/locations", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /api/v1/wms/barcodes/resolve?code= resolves successfully", func(t *testing.T) {
		prodID := uuid.New()
		mock := &mockWMSUsecase{
			resolveBarcodeFn: func(ctx context.Context, tid uuid.UUID, code string) (*domain.ResolvedProduct, error) {
				return &domain.ResolvedProduct{
					ProductID:  prodID,
					SKU:        "SKU-100",
					Name:       "Test Product",
					Multiplier: decimal.NewFromInt(1),
					Source:     "SKU",
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/barcodes/resolve?code=SKU-100", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var res domain.ResolvedProduct
		err := json.NewDecoder(w.Body).Decode(&res)
		require.NoError(t, err)
		assert.Equal(t, prodID, res.ProductID)
		assert.Equal(t, "SKU", res.Source)
	})

	t.Run("GET /api/v1/wms/barcodes/resolve with unknown code returns 404", func(t *testing.T) {
		mock := &mockWMSUsecase{
			resolveBarcodeFn: func(ctx context.Context, tid uuid.UUID, code string) (*domain.ResolvedProduct, error) {
				return nil, domain.ErrBarcodeNotFound
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/barcodes/resolve?code=UNKNOWN", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("POST /api/v1/wms/transfers/{id}/dispatch updates to in-transit", func(t *testing.T) {
		trID := uuid.New()
		now := time.Now()
		mock := &mockWMSUsecase{
			dispatchTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockTransfer, error) {
				return &domain.StockTransfer{
					ID:           trID,
					TenantID:     tid,
					Status:       domain.TransferStatusInTransit,
					DispatchedAt: &now,
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/transfers/"+trID.String()+"/dispatch", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tr domain.StockTransfer
		err := json.NewDecoder(w.Body).Decode(&tr)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusInTransit, tr.Status)
	})

	t.Run("POST /api/v1/wms/transfers/{id}/dispatch with insufficient stock returns 422", func(t *testing.T) {
		trID := uuid.New()
		mock := &mockWMSUsecase{
			dispatchTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockTransfer, error) {
				return nil, domain.ErrInsufficientStock
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/transfers/"+trID.String()+"/dispatch", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("POST /api/v1/wms/transfers/{id}/receive confirms receipt", func(t *testing.T) {
		trID := uuid.New()
		now := time.Now()
		mock := &mockWMSUsecase{
			receiveTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockTransfer, error) {
				return &domain.StockTransfer{
					ID:         trID,
					TenantID:   tid,
					Status:     domain.TransferStatusReceived,
					ReceivedAt: &now,
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/transfers/"+trID.String()+"/receive", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var tr domain.StockTransfer
		err := json.NewDecoder(w.Body).Decode(&tr)
		require.NoError(t, err)
		assert.Equal(t, domain.TransferStatusReceived, tr.Status)
	})

	t.Run("GET /api/v1/wms/delivery-orders returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listDeliveryOrdersFn: func(ctx context.Context, tid, uid uuid.UUID, r string, whID *uuid.UUID) ([]domain.DeliveryOrder, error) {
				return []domain.DeliveryOrder{
					{ID: uuid.New(), TenantID: tid, DONumber: "DO-001", Status: domain.DeliveryOrderStatusDraft},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/delivery-orders", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /api/v1/wms/delivery-orders creates DO with 201", func(t *testing.T) {
		doID := uuid.New()
		mock := &mockWMSUsecase{
			createDeliveryOrderFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateDeliveryOrderRequest) (*domain.DeliveryOrder, error) {
				return &domain.DeliveryOrder{
					ID:       doID,
					TenantID: tid,
					DONumber: req.DONumber,
					Status:   domain.DeliveryOrderStatusDraft,
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		body := uc.CreateDeliveryOrderRequest{
			SalesOrderID: uuid.New(),
			WarehouseID:  uuid.New(),
			DONumber:     "DO-100",
			Items: []uc.CreateDeliveryOrderItemRequest{
				{ProductID: uuid.New(), Quantity: decimal.NewFromInt(5), LocationID: uuid.New()},
			},
		}
		jsonBytes, _ := json.Marshal(body)

		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/delivery-orders", bytes.NewReader(jsonBytes)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Multi-tenancy: Missing tenant context returns 401 Unauthorized", func(t *testing.T) {
		router := setupWMSTestRouter(&mockWMSUsecase{})

		endpoints := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/api/v1/wms/warehouses"},
			{http.MethodGet, "/api/v1/wms/locations"},
			{http.MethodGet, "/api/v1/wms/barcodes/resolve?code=123"},
			{http.MethodGet, "/api/v1/wms/transfers"},
			{http.MethodGet, "/api/v1/wms/delivery-orders"},
		}

		for _, ep := range endpoints {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, "endpoint %s %s should require tenant context", ep.method, ep.path)
		}
	})

	t.Run("Multi-tenancy: Cross-tenant lookup returns 404 Not Found", func(t *testing.T) {
		mock := &mockWMSUsecase{
			getWarehouseFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.Warehouse, error) {
				return nil, domain.ErrWarehouseNotFound
			},
			resolveBarcodeFn: func(ctx context.Context, tid uuid.UUID, code string) (*domain.ResolvedProduct, error) {
				return nil, domain.ErrBarcodeNotFound
			},
			getTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, transferID uuid.UUID) (*domain.StockTransfer, []domain.StockTransferItem, error) {
				return nil, nil, domain.ErrTransferNotFound
			},
			getDeliveryOrderFn: func(ctx context.Context, tid, uid uuid.UUID, r string, doID uuid.UUID) (*domain.DeliveryOrder, []domain.DeliveryOrderItem, error) {
				return nil, nil, domain.ErrDeliveryOrderNotFound
			},
		}
		router := setupWMSTestRouter(mock)

		// Cross-tenant warehouse lookup
		req1 := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/warehouses/"+uuid.New().String(), nil), tenantID, userID, role)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusNotFound, w1.Code)

		// Cross-tenant barcode lookup
		req2 := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/barcodes/resolve?code=OTHER-TENANT-BC", nil), tenantID, userID, role)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusNotFound, w2.Code)

		// Cross-tenant transfer lookup
		req3 := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/transfers/"+uuid.New().String(), nil), tenantID, userID, role)
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusNotFound, w3.Code)

		// Cross-tenant DO lookup
		req4 := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/delivery-orders/"+uuid.New().String(), nil), tenantID, userID, role)
		w4 := httptest.NewRecorder()
		router.ServeHTTP(w4, req4)
		assert.Equal(t, http.StatusNotFound, w4.Code)
	})

	t.Run("Security: Body size overflow returns 413 Request Entity Too Large", func(t *testing.T) {
		router := setupWMSTestRouter(&mockWMSUsecase{})

		// Create a valid JSON payload larger than 1MB (e.g. 1MB + 10KB)
		largeString := make([]byte, 1024*1024+10240)
		for i := range largeString {
			largeString[i] = 'A'
		}
		largePayload := []byte(`{"name":"` + string(largeString) + `"}`)

		endpoints := []string{
			"/api/v1/wms/warehouses",
			"/api/v1/wms/locations",
			"/api/v1/wms/transfers",
			"/api/v1/wms/delivery-orders",
		}

		for _, path := range endpoints {
			req := withWMSAuth(httptest.NewRequest(http.MethodPost, path, bytes.NewReader(largePayload)), tenantID, userID, role)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "endpoint %s should reject payload > 1MB with 413", path)
		}
	})

	t.Run("Security: Auditor write attempt is rejected with 403 Forbidden", func(t *testing.T) {
		mock := &mockWMSUsecase{
			createTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateTransferRequest) (*domain.StockTransfer, error) {
				if r == "auditor" {
					return nil, domain.ErrForbidden
				}
				return &domain.StockTransfer{ID: uuid.New()}, nil
			},
			dispatchTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, transferID uuid.UUID) (*domain.StockTransfer, error) {
				if r == "auditor" {
					return nil, domain.ErrForbidden
				}
				return &domain.StockTransfer{ID: transferID}, nil
			},
			createDeliveryOrderFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateDeliveryOrderRequest) (*domain.DeliveryOrder, error) {
				if r == "auditor" {
					return nil, domain.ErrForbidden
				}
				return &domain.DeliveryOrder{ID: uuid.New()}, nil
			},
			dispatchDeliveryOrderFn: func(ctx context.Context, tid, uid uuid.UUID, r string, doID uuid.UUID) (*domain.DeliveryOrder, error) {
				if r == "auditor" {
					return nil, domain.ErrForbidden
				}
				return &domain.DeliveryOrder{ID: doID}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		// Create transfer with auditor role
		trBody := uc.CreateTransferRequest{
			FromWarehouseID: uuid.New(),
			ToWarehouseID:   uuid.New(),
			Items: []uc.CreateTransferItemRequest{
				{ProductID: uuid.New(), RequestedQty: decimal.NewFromInt(1)},
			},
		}
		trBytes, _ := json.Marshal(trBody)
		reqTr := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/transfers", bytes.NewReader(trBytes)), tenantID, userID, "auditor")
		wTr := httptest.NewRecorder()
		router.ServeHTTP(wTr, reqTr)
		assert.Equal(t, http.StatusForbidden, wTr.Code)

		// Dispatch transfer with auditor role
		reqDispTr := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/transfers/"+uuid.New().String()+"/dispatch", nil), tenantID, userID, "auditor")
		wDispTr := httptest.NewRecorder()
		router.ServeHTTP(wDispTr, reqDispTr)
		assert.Equal(t, http.StatusForbidden, wDispTr.Code)

		// Create delivery order with auditor role
		doBody := uc.CreateDeliveryOrderRequest{
			WarehouseID: uuid.New(),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{ProductID: uuid.New(), Quantity: decimal.NewFromInt(1), LocationID: uuid.New()},
			},
		}
		doBytes, _ := json.Marshal(doBody)
		reqDO := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/delivery-orders", bytes.NewReader(doBytes)), tenantID, userID, "auditor")
		wDO := httptest.NewRecorder()
		router.ServeHTTP(wDO, reqDO)
		assert.Equal(t, http.StatusForbidden, wDO.Code)

		// Dispatch delivery order with auditor role
		reqDispDO := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/delivery-orders/"+uuid.New().String()+"/dispatch", nil), tenantID, userID, "auditor")
		wDispDO := httptest.NewRecorder()
		router.ServeHTTP(wDispDO, reqDispDO)
		assert.Equal(t, http.StatusForbidden, wDispDO.Code)
	})

	t.Run("Security: Location spoofing across warehouses is rejected with 403", func(t *testing.T) {
		mock := &mockWMSUsecase{
			createTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateTransferRequest) (*domain.StockTransfer, error) {
				return nil, domain.ErrUnauthorizedWarehouse
			},
			dispatchTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, transferID uuid.UUID) (*domain.StockTransfer, error) {
				return nil, domain.ErrUnauthorizedWarehouse
			},
			createDeliveryOrderFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateDeliveryOrderRequest) (*domain.DeliveryOrder, error) {
				return nil, domain.ErrUnauthorizedWarehouse
			},
			dispatchDeliveryOrderFn: func(ctx context.Context, tid, uid uuid.UUID, r string, doID uuid.UUID) (*domain.DeliveryOrder, error) {
				return nil, domain.ErrUnauthorizedWarehouse
			},
		}
		router := setupWMSTestRouter(mock)

		trBody := uc.CreateTransferRequest{
			FromWarehouseID: uuid.New(),
			ToWarehouseID:   uuid.New(),
			Items: []uc.CreateTransferItemRequest{
				{ProductID: uuid.New(), RequestedQty: decimal.NewFromInt(1)},
			},
		}
		trBytes, _ := json.Marshal(trBody)
		reqTr := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/transfers", bytes.NewReader(trBytes)), tenantID, userID, role)
		wTr := httptest.NewRecorder()
		router.ServeHTTP(wTr, reqTr)
		assert.Equal(t, http.StatusForbidden, wTr.Code)

		doBody := uc.CreateDeliveryOrderRequest{
			WarehouseID: uuid.New(),
			Items: []uc.CreateDeliveryOrderItemRequest{
				{ProductID: uuid.New(), Quantity: decimal.NewFromInt(1), LocationID: uuid.New()},
			},
		}
		doBytes, _ := json.Marshal(doBody)
		reqDO := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/delivery-orders", bytes.NewReader(doBytes)), tenantID, userID, role)
		wDO := httptest.NewRecorder()
		router.ServeHTTP(wDO, reqDO)
		assert.Equal(t, http.StatusForbidden, wDO.Code)
	})

	t.Run("Security: Dispatching PENDING_APPROVAL transfer is rejected with 400", func(t *testing.T) {
		mock := &mockWMSUsecase{
			dispatchTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, transferID uuid.UUID) (*domain.StockTransfer, error) {
				return nil, domain.ErrInvalidTransferStatus
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/transfers/"+uuid.New().String()+"/dispatch", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Security: Dispatching CANCELLED delivery order is rejected with 400", func(t *testing.T) {
		mock := &mockWMSUsecase{
			dispatchDeliveryOrderFn: func(ctx context.Context, tid, uid uuid.UUID, r string, doID uuid.UUID) (*domain.DeliveryOrder, error) {
				return nil, domain.ErrInvalidStatus
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/delivery-orders/"+uuid.New().String()+"/dispatch", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GET /api/v1/wms/opnames returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listStockOpnamesFn: func(ctx context.Context, tid, uid uuid.UUID, r string, warehouseID *uuid.UUID) ([]domain.StockOpname, error) {
				return []domain.StockOpname{
					{ID: uuid.New(), TenantID: tid, OpnameNumber: "OPN-001", Status: domain.StockOpnameStatusDraft},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/opnames", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string][]domain.StockOpname
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Len(t, resp["data"], 1)
	})

	t.Run("POST /api/v1/wms/opnames creates opname with 201", func(t *testing.T) {
		whID := uuid.New()
		mock := &mockWMSUsecase{
			createStockOpnameFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateStockOpnameRequest) (*domain.StockOpname, error) {
				return &domain.StockOpname{
					ID:           uuid.New(),
					TenantID:     tid,
					WarehouseID:  req.WarehouseID,
					OpnameNumber: "OPN-001",
					Status:       domain.StockOpnameStatusDraft,
					ConductedBy:  uid,
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		body := uc.CreateStockOpnameRequest{WarehouseID: whID}
		jsonBytes, _ := json.Marshal(body)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/opnames", bytes.NewReader(jsonBytes)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("GET /api/v1/wms/opnames/{id} returns 200 with opname and items", func(t *testing.T) {
		opID := uuid.New()
		mock := &mockWMSUsecase{
			getStockOpnameFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockOpname, []domain.StockOpnameItem, error) {
				return &domain.StockOpname{
						ID:           opID,
						TenantID:     tid,
						OpnameNumber: "OPN-001",
						Status:       domain.StockOpnameStatusDraft,
					}, []domain.StockOpnameItem{
						{ID: uuid.New(), OpnameID: opID, PhysicalQty: decimal.NewFromInt(10)},
					}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/opnames/"+opID.String(), nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["opname"])
		assert.NotNil(t, resp["items"])
	})

	t.Run("POST /api/v1/wms/opnames/{id}/items adds item with 201", func(t *testing.T) {
		opID := uuid.New()
		prodID := uuid.New()
		locID := uuid.New()
		mock := &mockWMSUsecase{
			addOpnameItemFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID, req uc.AddOpnameItemRequest) (*domain.StockOpnameItem, error) {
				return &domain.StockOpnameItem{
					ID:             uuid.New(),
					OpnameID:       id,
					TenantID:       tid,
					ProductID:      req.ProductID,
					LocationID:     req.LocationID,
					SystemQty:      decimal.NewFromInt(5),
					PhysicalQty:    req.PhysicalQty,
					DiscrepancyQty: req.PhysicalQty.Sub(decimal.NewFromInt(5)),
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		body := uc.AddOpnameItemRequest{
			ProductID:   prodID,
			LocationID:  locID,
			PhysicalQty: decimal.NewFromInt(8),
		}
		jsonBytes, _ := json.Marshal(body)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/opnames/"+opID.String()+"/items", bytes.NewReader(jsonBytes)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("POST /api/v1/wms/opnames/{id}/complete completes opname with 200", func(t *testing.T) {
		opID := uuid.New()
		mock := &mockWMSUsecase{
			completeStockOpnameFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockOpname, error) {
				return &domain.StockOpname{
					ID:           id,
					TenantID:     tid,
					OpnameNumber: "OPN-001",
					Status:       domain.StockOpnameStatusCompleted,
					ApprovedBy:   &uid,
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/opnames/"+opID.String()+"/complete", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var op domain.StockOpname
		err := json.NewDecoder(w.Body).Decode(&op)
		require.NoError(t, err)
		assert.Equal(t, domain.StockOpnameStatusCompleted, op.Status)
	})

	t.Run("GET /api/v1/wms/scraps returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listStockScrapsFn: func(ctx context.Context, tid, uid uuid.UUID, r string, warehouseID *uuid.UUID) ([]domain.StockScrap, error) {
				return []domain.StockScrap{
					{ID: uuid.New(), TenantID: tid, ScrapNumber: "SCRAP-001", Quantity: decimal.NewFromInt(3)},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/scraps", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string][]domain.StockScrap
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Len(t, resp["data"], 1)
	})

	t.Run("POST /api/v1/wms/scraps creates scrap with 201", func(t *testing.T) {
		mock := &mockWMSUsecase{
			createStockScrapFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateStockScrapRequest) (*domain.StockScrap, error) {
				return &domain.StockScrap{
					ID:               uuid.New(),
					TenantID:         tid,
					ScrapNumber:      "SCRAP-001",
					WarehouseID:      req.WarehouseID,
					ProductID:        req.ProductID,
					SourceLocationID: req.SourceLocationID,
					Quantity:         req.Quantity,
					Reason:           req.Reason,
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		body := uc.CreateStockScrapRequest{
			WarehouseID:      uuid.New(),
			ProductID:        uuid.New(),
			SourceLocationID: uuid.New(),
			Quantity:         decimal.NewFromInt(2),
			Reason:           "Damaged box",
		}
		jsonBytes, _ := json.Marshal(body)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/scraps", bytes.NewReader(jsonBytes)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Security: Stock Opname and Scrap errors mapped correctly", func(t *testing.T) {
		mock := &mockWMSUsecase{
			getStockOpnameFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockOpname, []domain.StockOpnameItem, error) {
				return nil, nil, domain.ErrOpnameNotFound
			},
			completeStockOpnameFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockOpname, error) {
				return nil, domain.ErrInvalidOpnameStatus
			},
			createStockScrapFn: func(ctx context.Context, tid, uid uuid.UUID, r string, req uc.CreateStockScrapRequest) (*domain.StockScrap, error) {
				return nil, domain.ErrInsufficientStock
			},
		}
		router := setupWMSTestRouter(mock)

		// Opname not found -> 404
		reqGet := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/opnames/"+uuid.New().String(), nil), tenantID, userID, role)
		wGet := httptest.NewRecorder()
		router.ServeHTTP(wGet, reqGet)
		assert.Equal(t, http.StatusNotFound, wGet.Code)

		// Invalid opname status transition -> 400
		reqComplete := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/opnames/"+uuid.New().String()+"/complete", nil), tenantID, userID, role)
		wComplete := httptest.NewRecorder()
		router.ServeHTTP(wComplete, reqComplete)
		assert.Equal(t, http.StatusBadRequest, wComplete.Code)

		// Insufficient stock on scrap -> 422
		scrapBody, _ := json.Marshal(uc.CreateStockScrapRequest{WarehouseID: uuid.New(), ProductID: uuid.New(), SourceLocationID: uuid.New(), Quantity: decimal.NewFromInt(99), Reason: "bad"})
		reqScrap := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/scraps", bytes.NewReader(scrapBody)), tenantID, userID, role)
		wScrap := httptest.NewRecorder()
		router.ServeHTTP(wScrap, reqScrap)
		assert.Equal(t, http.StatusUnprocessableEntity, wScrap.Code)
	})

	t.Run("Security: Payload exceeding 1MB on opnames and scraps is rejected with 413", func(t *testing.T) {
		router := setupWMSTestRouter(&mockWMSUsecase{})
		largeString := make([]byte, 1024*1024+10240)
		for i := range largeString {
			largeString[i] = 'A'
		}
		largePayload := []byte(`{"notes":"` + string(largeString) + `"}`)

		reqOpname := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/opnames", bytes.NewReader(largePayload)), tenantID, userID, role)
		wOpname := httptest.NewRecorder()
		router.ServeHTTP(wOpname, reqOpname)
		assert.Equal(t, http.StatusRequestEntityTooLarge, wOpname.Code)

		reqScrap := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/scraps", bytes.NewReader(largePayload)), tenantID, userID, role)
		wScrap := httptest.NewRecorder()
		router.ServeHTTP(wScrap, reqScrap)
		assert.Equal(t, http.StatusRequestEntityTooLarge, wScrap.Code)
	})

	// -------------------------------------------------------------------------
	// Marketplace Endpoints
	// -------------------------------------------------------------------------
	warehouseID := uuid.New()

	t.Run("POST /api/v1/wms/marketplace/import with JSON returns 201", func(t *testing.T) {
		mock := &mockWMSUsecase{
			importMarketplaceOrdersFn: func(ctx context.Context, tID, uID uuid.UUID, r string, req uc.ImportMarketplaceOrdersRequest) (*uc.ImportMarketplaceOrdersResponse, error) {
				return &uc.ImportMarketplaceOrdersResponse{
					Batch: &domain.MarketplaceImportBatch{
						ID:              uuid.New(),
						TenantID:        tID,
						BatchNumber:     "BATCH-MKT-SHOPEE-12345",
						Channel:         domain.MarketplaceChannelShopee,
						TotalOrders:     1,
						ProcessedOrders: 1,
						Status:          domain.MarketplaceBatchStatusCompleted,
					},
					Orders: []domain.MarketplaceOrder{
						{
							ID:              uuid.New(),
							TenantID:        tID,
							Channel:         domain.MarketplaceChannelShopee,
							ExternalOrderID: "240909SHP001",
							Status:          domain.MarketplaceOrderStatusCompleted,
						},
					},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		payload := []byte(fmt.Sprintf(`{"warehouse_id":"%s","channel":"SHOPEE","orders":[{"external_order_id":"240909SHP001","items":[{"external_sku":"SKU-1","quantity":2,"unit_price":10000}]}]}`, warehouseID))
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/marketplace/import", bytes.NewReader(payload)), tenantID, userID, role)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "BATCH-MKT-SHOPEE-12345")
	})

	t.Run("POST /api/v1/wms/marketplace/import with CSV returns 201", func(t *testing.T) {
		mock := &mockWMSUsecase{
			importMarketplaceOrdersFn: func(ctx context.Context, tID, uID uuid.UUID, r string, req uc.ImportMarketplaceOrdersRequest) (*uc.ImportMarketplaceOrdersResponse, error) {
				assert.Equal(t, domain.MarketplaceChannelShopee, req.Channel)
				assert.Len(t, req.Orders, 1)
				assert.Equal(t, "240909SHP002", req.Orders[0].ExternalOrderID)
				return &uc.ImportMarketplaceOrdersResponse{
					Batch: &domain.MarketplaceImportBatch{
						ID:              uuid.New(),
						TenantID:        tID,
						BatchNumber:     "BATCH-MKT-SHOPEE-CSV",
						TotalOrders:     1,
						ProcessedOrders: 1,
						Status:          domain.MarketplaceBatchStatusCompleted,
					},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		csvContent := "Nomor Pesanan,Nomor Referensi SKU,Nama Produk,Jumlah,Harga Satuan,Total Pembayaran\n240909SHP002,SKU-ABC,Kopi Susu,2,15000,30000\n"
		url := fmt.Sprintf("/api/v1/wms/marketplace/import?warehouse_id=%s&channel=SHOPEE", warehouseID)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, url, strings.NewReader(csvContent)), tenantID, userID, role)
		req.Header.Set("Content-Type", "text/csv")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "BATCH-MKT-SHOPEE-CSV")
	})

	t.Run("POST /api/v1/wms/marketplace/import duplicate error mapped to 409", func(t *testing.T) {
		mock := &mockWMSUsecase{
			importMarketplaceOrdersFn: func(ctx context.Context, tID, uID uuid.UUID, r string, req uc.ImportMarketplaceOrdersRequest) (*uc.ImportMarketplaceOrdersResponse, error) {
				return nil, domain.ErrDuplicateMarketplaceOrder
			},
		}
		router := setupWMSTestRouter(mock)
		payload := []byte(fmt.Sprintf(`{"warehouse_id":"%s","channel":"SHOPEE","orders":[{"external_order_id":"DUP-001"}]}`, warehouseID))
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/marketplace/import", bytes.NewReader(payload)), tenantID, userID, role)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "duplicate marketplace order")
	})

	t.Run("POST /api/v1/wms/marketplace/import forbidden warehouse access mapped to 403", func(t *testing.T) {
		mock := &mockWMSUsecase{
			importMarketplaceOrdersFn: func(ctx context.Context, tID, uID uuid.UUID, r string, req uc.ImportMarketplaceOrdersRequest) (*uc.ImportMarketplaceOrdersResponse, error) {
				return nil, domain.ErrUnauthorizedWarehouse
			},
		}
		router := setupWMSTestRouter(mock)
		payload := []byte(fmt.Sprintf(`{"warehouse_id":"%s","channel":"SHOPEE","orders":[{"external_order_id":"FORBIDDEN-001"}]}`, warehouseID))
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/marketplace/import", bytes.NewReader(payload)), tenantID, userID, role)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "unauthorized warehouse access")
	})

	t.Run("POST /api/v1/wms/marketplace/sku-mappings forbidden auditor mapped to 403", func(t *testing.T) {
		mock := &mockWMSUsecase{
			resolveSKUMappingFn: func(ctx context.Context, tID, uID uuid.UUID, r string, req uc.CreateSKUMappingRequest) (*domain.ProductSKUMapping, error) {
				return nil, domain.ErrForbidden
			},
		}
		router := setupWMSTestRouter(mock)
		payload := []byte(fmt.Sprintf(`{"product_id":"%s","channel_name":"TOKOPEDIA","external_sku":"KOPISUSU-DUS","multiplier":24}`, uuid.New()))
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/marketplace/sku-mappings", bytes.NewReader(payload)), tenantID, userID, "auditor")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "forbidden")
	})

	t.Run("Security: Payload exceeding 5MB on marketplace import is rejected with 413", func(t *testing.T) {
		router := setupWMSTestRouter(&mockWMSUsecase{})
		largeString := make([]byte, 5*1024*1024+10240)
		for i := range largeString {
			largeString[i] = 'X'
		}
		payload := []byte(`{"warehouse_id":"` + warehouseID.String() + `","csv_data":"` + string(largeString) + `"}`)
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/marketplace/import", bytes.NewReader(payload)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("GET /api/v1/wms/marketplace/batches returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listMarketplaceBatchesFn: func(ctx context.Context, tID, uID uuid.UUID, r string, whID *uuid.UUID) ([]domain.MarketplaceImportBatch, error) {
				return []domain.MarketplaceImportBatch{
					{
						ID:          uuid.New(),
						TenantID:    tID,
						BatchNumber: "BATCH-1",
						Channel:     domain.MarketplaceChannelShopee,
					},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/marketplace/batches", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "BATCH-1")
	})

	t.Run("GET /api/v1/wms/marketplace/orders returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listMarketplaceOrdersFn: func(ctx context.Context, tID, uID uuid.UUID, r string, whID, bID *uuid.UUID, st *domain.MarketplaceOrderStatus) ([]domain.MarketplaceOrder, error) {
				return []domain.MarketplaceOrder{
					{
						ID:              uuid.New(),
						TenantID:        tID,
						ExternalOrderID: "ORD-999",
						Channel:         domain.MarketplaceChannelTokopedia,
						Status:          domain.MarketplaceOrderStatusCompleted,
					},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/marketplace/orders", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-999")
	})

	t.Run("GET /api/v1/wms/marketplace/orders/{id} returns 200", func(t *testing.T) {
		orderID := uuid.New()
		mock := &mockWMSUsecase{
			getMarketplaceOrderFn: func(ctx context.Context, tID, uID uuid.UUID, r string, id uuid.UUID) (*domain.MarketplaceOrder, error) {
				return &domain.MarketplaceOrder{
					ID:              id,
					TenantID:        tID,
					ExternalOrderID: "ORD-SINGLE",
					Channel:         domain.MarketplaceChannelTikTok,
					Status:          domain.MarketplaceOrderStatusCompleted,
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/wms/marketplace/orders/%s", orderID), nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ORD-SINGLE")
	})

	t.Run("POST /api/v1/wms/marketplace/sku-mappings returns 201", func(t *testing.T) {
		prodID := uuid.New()
		mock := &mockWMSUsecase{
			resolveSKUMappingFn: func(ctx context.Context, tID, uID uuid.UUID, r string, req uc.CreateSKUMappingRequest) (*domain.ProductSKUMapping, error) {
				return &domain.ProductSKUMapping{
					ID:          uuid.New(),
					TenantID:    tID,
					ProductID:   req.ProductID,
					ChannelName: req.ChannelName,
					ExternalSKU: req.ExternalSKU,
					Multiplier:  decimal.NewFromInt(24),
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		payload := []byte(fmt.Sprintf(`{"product_id":"%s","channel_name":"TOKOPEDIA","external_sku":"KOPISUSU-DUS","multiplier":24}`, prodID))
		req := withWMSAuth(httptest.NewRequest(http.MethodPost, "/api/v1/wms/marketplace/sku-mappings", bytes.NewReader(payload)), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "KOPISUSU-DUS")
	})

	t.Run("GET /api/v1/wms/marketplace/sku-mappings returns 200", func(t *testing.T) {
		mock := &mockWMSUsecase{
			listSKUMappingsFn: func(ctx context.Context, tID uuid.UUID, ch string) ([]domain.ProductSKUMapping, error) {
				return []domain.ProductSKUMapping{
					{
						ID:          uuid.New(),
						TenantID:    tID,
						ChannelName: "SHOPEE",
						ExternalSKU: "SHP-SKU-1",
						Multiplier:  decimal.NewFromInt(1),
					},
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)
		req := withWMSAuth(httptest.NewRequest(http.MethodGet, "/api/v1/wms/marketplace/sku-mappings?channel_name=SHOPEE", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "SHP-SKU-1")
	})

	t.Run("Security: Missing tenant context returns 401", func(t *testing.T) {
		router := setupWMSTestRouter(&mockWMSUsecase{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/wms/marketplace/batches", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("DELETE /api/v1/wms/transfers/{id} cancels a draft with 200", func(t *testing.T) {
		transferID := uuid.New()
		mock := &mockWMSUsecase{
			cancelTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockTransfer, error) {
				assert.Equal(t, transferID, id)
				return &domain.StockTransfer{
					ID:             id,
					TenantID:       tid,
					TransferNumber: "TR-CANCEL-001",
					Status:         domain.TransferStatusCancelled,
				}, nil
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodDelete, "/api/v1/wms/transfers/"+transferID.String(), nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var res domain.StockTransfer
		require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
		assert.Equal(t, domain.TransferStatusCancelled, res.Status)
	})

	t.Run("DELETE /api/v1/wms/transfers/{id} returns 409 when not a draft", func(t *testing.T) {
		transferID := uuid.New()
		mock := &mockWMSUsecase{
			cancelTransferFn: func(ctx context.Context, tid, uid uuid.UUID, r string, id uuid.UUID) (*domain.StockTransfer, error) {
				return nil, domain.ErrTransferNotDraft
			},
		}
		router := setupWMSTestRouter(mock)

		req := withWMSAuth(httptest.NewRequest(http.MethodDelete, "/api/v1/wms/transfers/"+transferID.String(), nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "DRAFT")
	})

	t.Run("DELETE /api/v1/wms/transfers/{id} returns 400 on malformed id", func(t *testing.T) {
		router := setupWMSTestRouter(&mockWMSUsecase{})
		req := withWMSAuth(httptest.NewRequest(http.MethodDelete, "/api/v1/wms/transfers/not-a-uuid", nil), tenantID, userID, role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
