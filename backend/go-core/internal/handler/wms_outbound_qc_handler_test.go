package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestWMSOutboundQCHandlers(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	doID := uuid.New()

	h := handler.NewWMSHandler(&mockWMSUsecase{})
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), appMiddleware.TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, appMiddleware.UserIDKey, userID)
			ctx = context.WithValue(ctx, appMiddleware.RoleKey, "warehouse")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h.RegisterRoutes(r)

	t.Run("POST /wms/delivery-orders/{id}/qc/pass", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"scanned_awb": "AWB-12345",
			"notes":       "Kemasan rapi",
		})
		req := httptest.NewRequest(http.MethodPost, "/wms/delivery-orders/"+doID.String()+"/qc/pass", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		data := resp["data"].(map[string]any)
		assert.Equal(t, string(domain.DeliveryOrderStatusQCPassed), data["status"])
	})

	t.Run("POST /wms/delivery-orders/{id}/qc/reject", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"reason": "Kemasan sobek",
			"notes":  "Perlu ganti kardus",
		})
		req := httptest.NewRequest(http.MethodPost, "/wms/delivery-orders/"+doID.String()+"/qc/reject", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		data := resp["data"].(map[string]any)
		assert.Equal(t, string(domain.DeliveryOrderStatusQCFailed), data["status"])
	})

	t.Run("GET /wms/outbound/qc/inspections", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/wms/outbound/qc/inspections", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
