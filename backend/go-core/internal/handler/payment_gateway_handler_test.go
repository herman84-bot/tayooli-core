package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPaymentGatewayRepo struct {
	configs      map[string]*domain.TenantPaymentConfig
	transactions map[string]*domain.PaymentTransaction
}

func newMockPaymentGatewayRepo() *mockPaymentGatewayRepo {
	return &mockPaymentGatewayRepo{
		configs:      make(map[string]*domain.TenantPaymentConfig),
		transactions: make(map[string]*domain.PaymentTransaction),
	}
}

func (m *mockPaymentGatewayRepo) GetConfig(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.TenantPaymentConfig, error) {
	key := tenantID.String() + ":" + provider
	cfg, ok := m.configs[key]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cfg, nil
}

func (m *mockPaymentGatewayRepo) UpsertConfig(ctx context.Context, cfg *domain.TenantPaymentConfig) error {
	key := cfg.TenantID.String() + ":" + cfg.Provider
	m.configs[key] = cfg
	return nil
}

func (m *mockPaymentGatewayRepo) CreateTransaction(ctx context.Context, tx *domain.PaymentTransaction) error {
	m.transactions[tx.OrderID] = tx
	return nil
}

func (m *mockPaymentGatewayRepo) GetTransactionByOrderID(ctx context.Context, tenantID uuid.UUID, orderID string) (*domain.PaymentTransaction, error) {
	tx, ok := m.transactions[orderID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return tx, nil
}

func (m *mockPaymentGatewayRepo) UpdateTransactionStatus(ctx context.Context, tenantID uuid.UUID, orderID, status string, completedAt *time.Time) error {
	tx, ok := m.transactions[orderID]
	if !ok {
		return domain.ErrNotFound
	}
	tx.Status = status
	tx.CompletedAt = completedAt
	return nil
}

func TestPaymentGatewayHandler_GetConfig_NotFound(t *testing.T) {
	repo := newMockPaymentGatewayRepo()
	h := handler.NewPaymentGatewayHandler(repo)

	tenantID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/configs?provider=pakasir", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.GetConfig(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["hasCredentials"])
}

func TestPaymentGatewayHandler_UpsertAndGetConfig(t *testing.T) {
	repo := newMockPaymentGatewayRepo()
	h := handler.NewPaymentGatewayHandler(repo)

	tenantID := uuid.New()
	slug := "my-project"
	apiKey := "secret-api-key"

	upsertReq := handler.UpsertPaymentConfigRequest{
		Provider:          "pakasir",
		Slug:              &slug,
		APIKey:            &apiKey,
		IsActive:          true,
		GatewayFeePercent: decimal.NewFromFloat(0.7),
	}
	b, _ := json.Marshal(upsertReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/configs", bytes.NewReader(b))
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.UpsertConfig(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify GET returns masked credentials
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/payments/configs?provider=pakasir", nil)
	getReq = getReq.WithContext(ctx)
	getRec := httptest.NewRecorder()
	h.GetConfig(getRec, getReq)

	assert.Equal(t, http.StatusOK, getRec.Code)
	var getBody map[string]any
	err := json.Unmarshal(getRec.Body.Bytes(), &getBody)
	require.NoError(t, err)
	assert.Equal(t, true, getBody["hasCredentials"])
	assert.Equal(t, "my-project", getBody["slug"])
	assert.Nil(t, getBody["api_key"]) // api_key must be masked
}
