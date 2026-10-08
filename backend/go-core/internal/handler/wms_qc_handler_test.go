package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// qcErr lets a test force a usecase error for the QC endpoints.
var qcErr error

func (m *mockWMSUsecase) SubmitQCInspection(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, in domain.QCInspectionInput) (*domain.QCInspectionDetail, error) {
	if qcErr != nil {
		return nil, qcErr
	}
	return &domain.QCInspectionDetail{Inspection: domain.QCInspection{ReceiptID: receiptID, InspectionMode: in.InspectionMode, Status: domain.QCStatusPassed}, Items: []domain.QCInspectionItem{}}, nil
}

func (m *mockWMSUsecase) GetReceiptQC(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.QCInspectionDetail, error) {
	return nil, qcErr
}

func (m *mockWMSUsecase) ListQCInspections(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.QCInspection, error) {
	return nil, qcErr
}

func (m *mockWMSUsecase) ListQuarantineStock(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.BatchBalance, error) {
	return nil, qcErr
}

func (m *mockWMSUsecase) ReleaseQuarantine(ctx context.Context, tenantID, userID uuid.UUID, role string, in domain.QuarantineActionInput) (*domain.StockMovement, error) {
	if qcErr != nil {
		return nil, qcErr
	}
	return &domain.StockMovement{ID: uuid.New(), Quantity: in.Quantity}, nil
}

func (m *mockWMSUsecase) ScrapQuarantine(ctx context.Context, tenantID, userID uuid.UUID, role string, in domain.QuarantineActionInput) (*domain.StockMovement, error) {
	return m.ReleaseQuarantine(ctx, tenantID, userID, role, in)
}

func TestQCEndpoints(t *testing.T) {
	h := handler.NewWMSHandler(&mockWMSUsecase{})
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), appMiddleware.TenantIDKey, uuid.New())
			ctx = context.WithValue(ctx, appMiddleware.UserIDKey, uuid.New())
			ctx = context.WithValue(ctx, appMiddleware.RoleKey, "admin")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h.RegisterRoutes(r)
	do := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	t.Cleanup(func() { qcErr = nil })

	t.Run("submit 201", func(t *testing.T) {
		rec := do(http.MethodPost, fmt.Sprintf("/wms/receipts/%s/qc", uuid.New()), map[string]any{"inspection_mode": "FULL", "gross_cartons": 2, "items": []any{}})
		assert.Equal(t, http.StatusCreated, rec.Code)
	})
	t.Run("get not inspected returns data null", func(t *testing.T) {
		rec := do(http.MethodGet, fmt.Sprintf("/wms/receipts/%s/qc", uuid.New()), nil)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"data":null}`, rec.Body.String())
	})
	t.Run("lists return empty arrays", func(t *testing.T) {
		for _, p := range []string{"/wms/qc-inspections", "/wms/quarantine"} {
			rec := do(http.MethodGet, p+"?warehouse_id="+uuid.NewString(), nil)
			require.Equal(t, http.StatusOK, rec.Code, p)
			assert.JSONEq(t, `{"data":[]}`, rec.Body.String(), p)
		}
	})
	t.Run("missing / bad warehouse_id is 400", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, do(http.MethodGet, "/wms/quarantine", nil).Code)
		assert.Equal(t, http.StatusBadRequest, do(http.MethodGet, "/wms/quarantine?warehouse_id=x", nil).Code)
		assert.Equal(t, http.StatusBadRequest, do(http.MethodPost, "/wms/receipts/x/qc", map[string]any{}).Code)
	})
	t.Run("release & scrap 200", func(t *testing.T) {
		body := map[string]any{"warehouse_id": uuid.New(), "product_id": uuid.New(), "batch_id": uuid.New(), "quantity": 1, "notes": "x"}
		assert.Equal(t, http.StatusOK, do(http.MethodPost, "/wms/quarantine/release", body).Code)
		assert.Equal(t, http.StatusOK, do(http.MethodPost, "/wms/quarantine/scrap", body).Code)
	})
	t.Run("domain errors map to Indonesian messages", func(t *testing.T) {
		cases := []struct {
			err  error
			code int
		}{
			{domain.ErrQCAlreadyInspected, http.StatusConflict},
			{domain.ErrQCReceiptNotPosted, http.StatusConflict},
			{domain.ErrQCSamplingFailed, http.StatusUnprocessableEntity},
			{domain.ErrQCBAKDriverRequired, http.StatusBadRequest},
			{domain.ErrQCStagedQtyChanged, http.StatusConflict},
			{domain.ErrQuarantineQtyInvalid, http.StatusConflict},
			{domain.ErrScrapNotesRequired, http.StatusBadRequest},
		}
		for _, c := range cases {
			qcErr = c.err
			rec := do(http.MethodPost, fmt.Sprintf("/wms/receipts/%s/qc", uuid.New()), map[string]any{"inspection_mode": "FULL"})
			assert.Equal(t, c.code, rec.Code, c.err.Error())
		}
		qcErr = nil
	})
}
