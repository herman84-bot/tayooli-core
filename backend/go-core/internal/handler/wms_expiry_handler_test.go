package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func expiryRouter(mock *mockWMSUsecase) *chi.Mux {
	h := handler.NewWMSHandler(mock)
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
	return r
}

func TestGetExpiryReportEndpoint(t *testing.T) {
	var gotDays int
	var gotWH *uuid.UUID
	mock := &mockWMSUsecase{
		getExpiryReportFn: func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID, days int) (*domain.ExpiryReport, error) {
			gotDays, gotWH = days, warehouseID
			return &domain.ExpiryReport{Days: 60, AsOf: "2026-11-01", Lines: []domain.ExpiringBatchLine{{BatchNumber: "LOT-1", Severity: domain.ExpirySeverityExpired}}}, nil
		},
	}
	r := expiryRouter(mock)
	whID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/wms/expiry-report?days=60&warehouse_id=%s", whID), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, 60, gotDays)
	require.NotNil(t, gotWH)
	assert.Equal(t, whID, *gotWH)

	var body struct {
		Data domain.ExpiryReport `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data.Lines, 1)
	assert.Equal(t, "LOT-1", body.Data.Lines[0].BatchNumber)

	for _, q := range []string{"days=abc", "days=-5", "warehouse_id=not-a-uuid"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wms/expiry-report?"+q, nil))
		assert.Equal(t, http.StatusBadRequest, rec.Code, q)
	}
}

func TestExpiredStockBlockedMapsTo422(t *testing.T) {
	mock := &mockWMSUsecase{
		getExpiryReportFn: func(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID, days int) (*domain.ExpiryReport, error) {
			return nil, domain.ErrExpiredStockBlocked
		},
	}
	rec := httptest.NewRecorder()
	expiryRouter(mock).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/wms/expiry-report", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.True(t, strings.Contains(rec.Body.String(), "kedaluwarsa"), rec.Body.String())
}
