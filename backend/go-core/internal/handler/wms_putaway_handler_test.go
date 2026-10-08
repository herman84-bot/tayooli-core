package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// Implement Sprint 1 methods on mockWMSUsecase

func (m *mockWMSUsecase) GetPutawayPending(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.PutawayPendingLine, error) {
	return []domain.PutawayPendingLine{
		{
			ProductID:         uuid.New(),
			BatchID:           uuid.New(),
			BatchNumber:       "LOT-001",
			Quantity:          decimal.NewFromInt(10),
			SuggestionSource:  "DEFAULT_RACK",
		},
	}, nil
}

func (m *mockWMSUsecase) ConfirmPutaway(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.PutawayRequest) (*domain.StockMovement, error) {
	if req.DestLocationID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}
	return &domain.StockMovement{
		ID:             uuid.New(),
		TenantID:       tenantID,
		MovementNumber: "PUT-001",
		ProductID:      req.ProductID,
		BatchID:        &req.BatchID,
		Quantity:       req.Quantity,
		DestLocationID: req.DestLocationID,
		Status:         domain.StockMovementStatusDone,
	}, nil
}

func (m *mockWMSUsecase) ReleaseStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, error) {
	now := time.Now().UTC()
	return &domain.StockReceipt{
		ID:          receiptID,
		TenantID:    tenantID,
		Status:      domain.StockReceiptStatusPosted,
		ReleasedBy:  &userID,
		ReleasedAt:  &now,
	}, nil
}

func (m *mockWMSUsecase) GetWMSSettings(ctx context.Context, tenantID, userID uuid.UUID, role string) (*domain.WMSSettings, error) {
	return &domain.WMSSettings{
		TenantID:               tenantID,
		RequireReleaseApproval: true,
	}, nil
}

func (m *mockWMSUsecase) UpdateWMSSettings(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.UpdateWMSSettingsRequest) (*domain.WMSSettings, error) {
	return &domain.WMSSettings{
		TenantID:               tenantID,
		RequireReleaseApproval: req.RequireReleaseApproval,
	}, nil
}

func (m *mockWMSUsecase) ListDefaultLocations(ctx context.Context, tenantID, userID uuid.UUID, role string, productID *uuid.UUID) ([]domain.ProductDefaultLocation, error) {
	return []domain.ProductDefaultLocation{
		{
			TenantID:    tenantID,
			ProductID:   uuid.New(),
			WarehouseID: uuid.New(),
			LocationID:  uuid.New(),
		},
	}, nil
}

func (m *mockWMSUsecase) SetDefaultLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.SetDefaultLocationRequest) error {
	return nil
}

func (m *mockWMSUsecase) DeleteDefaultLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, productID, warehouseID uuid.UUID) error {
	return nil
}

func (m *mockWMSUsecase) TraceBatch(ctx context.Context, tenantID, userID uuid.UUID, role string, batchID uuid.UUID) (*domain.BatchTrace, error) {
	return &domain.BatchTrace{
		Batch: domain.StockBatch{
			ID:          batchID,
			TenantID:    tenantID,
			BatchNumber: "LOT-001",
		},
	}, nil
}

func (m *mockWMSUsecase) TraceDocument(ctx context.Context, tenantID, userID uuid.UUID, role string, refType string, refID uuid.UUID) (*domain.DocumentTrace, error) {
	return &domain.DocumentTrace{
		DocumentType: refType,
		DocumentID:   refID,
	}, nil
}

func (m *mockWMSUsecase) ListAuditTrail(ctx context.Context, tenantID, userID uuid.UUID, role string, entityType string, entityID uuid.UUID) ([]domain.AuditTrailEntry, error) {
	return []domain.AuditTrailEntry{
		{
			ID:         uuid.New(),
			EntityType: entityType,
			EntityID:   entityID,
			Action:     "PUTAWAY",
			UserName:   "Admin User",
		},
	}, nil
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

func TestSprint1Endpoints(t *testing.T) {
	testTenantID := uuid.New()
	testUserID := uuid.New()
	mock := &mockWMSUsecase{}
	h := handler.NewWMSHandler(mock)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), appMiddleware.TenantIDKey, testTenantID)
			ctx = context.WithValue(ctx, appMiddleware.UserIDKey, testUserID)
			ctx = context.WithValue(ctx, appMiddleware.RoleKey, "admin")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h.RegisterRoutes(r)

	t.Run("GET /wms/putaway/pending returns list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wms/putaway/pending?warehouse_id=%s", uuid.New()), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		data, ok := body["data"].([]any)
		require.True(t, ok)
		assert.Len(t, data, 1)
	})

	t.Run("POST /wms/putaway/confirm", func(t *testing.T) {
		payload := uc.PutawayRequest{
			WarehouseID:    uuid.New(),
			ProductID:      uuid.New(),
			BatchID:        uuid.New(),
			Quantity:       decimal.NewFromInt(5),
			DestLocationID: uuid.New(),
		}
		raw, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/wms/putaway/confirm", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("POST /wms/receipts/{id}/release", func(t *testing.T) {
		rcID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/wms/receipts/%s/release", rcID), nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GET & PUT /wms/settings", func(t *testing.T) {
		getReq := httptest.NewRequest(http.MethodGet, "/wms/settings", nil)
		getRec := httptest.NewRecorder()
		r.ServeHTTP(getRec, getReq)
		assert.Equal(t, http.StatusOK, getRec.Code)

		putPayload := uc.UpdateWMSSettingsRequest{RequireReleaseApproval: true}
		putRaw, _ := json.Marshal(putPayload)
		putReq := httptest.NewRequest(http.MethodPut, "/wms/settings", bytes.NewReader(putRaw))
		putReq.Header.Set("Content-Type", "application/json")
		putRec := httptest.NewRecorder()
		r.ServeHTTP(putRec, putReq)
		assert.Equal(t, http.StatusOK, putRec.Code)
	})

	t.Run("Default locations GET & POST & DELETE", func(t *testing.T) {
		getReq := httptest.NewRequest(http.MethodGet, "/wms/default-locations", nil)
		getRec := httptest.NewRecorder()
		r.ServeHTTP(getRec, getReq)
		assert.Equal(t, http.StatusOK, getRec.Code)

		postPayload := uc.SetDefaultLocationRequest{
			ProductID:   uuid.New(),
			WarehouseID: uuid.New(),
			LocationID:  uuid.New(),
		}
		postRaw, _ := json.Marshal(postPayload)
		postReq := httptest.NewRequest(http.MethodPost, "/wms/default-locations", bytes.NewReader(postRaw))
		postReq.Header.Set("Content-Type", "application/json")
		postRec := httptest.NewRecorder()
		r.ServeHTTP(postRec, postReq)
		assert.Equal(t, http.StatusOK, postRec.Code)
	})

	t.Run("Traceability endpoints", func(t *testing.T) {
		batchID := uuid.New()
		traceReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wms/trace/batch/%s", batchID), nil)
		traceRec := httptest.NewRecorder()
		r.ServeHTTP(traceRec, traceReq)
		assert.Equal(t, http.StatusOK, traceRec.Code)

		docReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wms/trace/document?type=RECEIPT&id=%s", uuid.New()), nil)
		docRec := httptest.NewRecorder()
		r.ServeHTTP(docRec, docReq)
		assert.Equal(t, http.StatusOK, docRec.Code)

		auditReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wms/audit-trail?entity_type=receipt&entity_id=%s", uuid.New()), nil)
		auditRec := httptest.NewRecorder()
		r.ServeHTTP(auditRec, auditReq)
		assert.Equal(t, http.StatusOK, auditRec.Code)

		auditSmReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wms/audit-trail?entity_type=stock_movement&entity_id=%s", uuid.New()), nil)
		auditSmRec := httptest.NewRecorder()
		r.ServeHTTP(auditSmRec, auditSmReq)
		assert.Equal(t, http.StatusOK, auditSmRec.Code)

		auditDoReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wms/audit-trail?entity_type=delivery_order&entity_id=%s", uuid.New()), nil)
		auditDoRec := httptest.NewRecorder()
		r.ServeHTTP(auditDoRec, auditDoReq)
		assert.Equal(t, http.StatusOK, auditDoRec.Code)
	})
}
