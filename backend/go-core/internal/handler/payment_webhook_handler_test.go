package handler_test

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/midtrans"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// mockStatusChecker verifies transaction status without making real API calls.
type mockStatusChecker struct {
	status string
	amount int64
	err    error
}

func (m *mockStatusChecker) VerifyTransactionStatus(ctx context.Context, orderID string) (*midtrans.TransactionStatus, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &midtrans.TransactionStatus{
		Status:  m.status,
		Amount:  m.amount,
		PaymentType: "qris",
	}, nil
}

func TestHandleMidtransWebhook_ValidSignature(t *testing.T) {
	tenantID := uuid.New()
	serverKey := "SK-test-123"
	orderID := midtrans.BuildOrderID(tenantID)
	statusCode := "200"
	grossAmount := "10000.00"

	sig := sha512.Sum512([]byte(orderID + statusCode + grossAmount + serverKey))
	signature := hex.EncodeToString(sig[:])

	repo := newMockPaymentGatewayRepo()
	repo.configs[tenantID.String()+":midtrans"] = &domain.TenantPaymentConfig{
		TenantID:     tenantID,
		Provider:     "midtrans",
		ServerKey:    &serverKey,
		IsProduction: false,
	}
	repo.transactions[orderID] = &domain.PaymentTransaction{
		TenantID:   tenantID,
		OrderID:    orderID,
		Amount:     decimal.NewFromInt(10000),
		Status:     "pending",
		Provider:   "midtrans",
	}

	factory := func(sk, ck string, prod bool) handler.MidtransStatusChecker {
		return &mockStatusChecker{status: "settlement", amount: 10000}
	}

	h := handler.NewPaymentWebhookHandler(repo, factory)

	payload := map[string]string{
		"order_id":       orderID,
		"status_code":    statusCode,
		"gross_amount":   grossAmount,
		"signature_key":  signature,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhooks/midtrans", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleMidtransWebhook(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "completed", repo.transactions[orderID].Status)
	require.NotNil(t, repo.transactions[orderID].CompletedAt)
}

func TestHandleMidtransWebhook_BadSignature(t *testing.T) {
	tenantID := uuid.New()
	serverKey := "SK-test-123"
	orderID := midtrans.BuildOrderID(tenantID)

	repo := newMockPaymentGatewayRepo()
	repo.configs[tenantID.String()+":midtrans"] = &domain.TenantPaymentConfig{
		TenantID:     tenantID,
		Provider:     "midtrans",
		ServerKey:    &serverKey,
		IsProduction: false,
	}
	repo.transactions[orderID] = &domain.PaymentTransaction{
		TenantID:   tenantID,
		OrderID:    orderID,
		Amount:     decimal.NewFromInt(10000),
		Status:     "pending",
		Provider:   "midtrans",
	}

	h := handler.NewPaymentWebhookHandler(repo, nil)

	payload := map[string]string{
		"order_id":       orderID,
		"status_code":    "200",
		"gross_amount":   "10000.00",
		"signature_key":  "invalid-sig-xxxxx",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhooks/midtrans", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleMidtransWebhook(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Equal(t, "pending", repo.transactions[orderID].Status)
}

func TestHandleMidtransWebhook_Idempotency(t *testing.T) {
	tenantID := uuid.New()
	serverKey := "SK-test-123"
	orderID := midtrans.BuildOrderID(tenantID)
	statusCode := "200"
	grossAmount := "10000.00"

	sig := sha512.Sum512([]byte(orderID + statusCode + grossAmount + serverKey))
	signature := hex.EncodeToString(sig[:])

	completedAt := time.Now().UTC()
	repo := newMockPaymentGatewayRepo()
	repo.configs[tenantID.String()+":midtrans"] = &domain.TenantPaymentConfig{
		TenantID:     tenantID,
		Provider:     "midtrans",
		ServerKey:    &serverKey,
		IsProduction: false,
	}
	repo.transactions[orderID] = &domain.PaymentTransaction{
		TenantID:    tenantID,
		OrderID:     orderID,
		Amount:      decimal.NewFromInt(10000),
		Status:      "completed",
		CompletedAt: &completedAt,
		Provider:    "midtrans",
	}

	h := handler.NewPaymentWebhookHandler(repo, nil)

	payload := map[string]string{
		"order_id":       orderID,
		"status_code":    statusCode,
		"gross_amount":   grossAmount,
		"signature_key":  signature,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhooks/midtrans", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleMidtransWebhook(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "completed", repo.transactions[orderID].Status)
	require.Equal(t, completedAt, *repo.transactions[orderID].CompletedAt)
}

func TestHandleMidtransWebhook_UnknownOrderID(t *testing.T) {
	h := handler.NewPaymentWebhookHandler(newMockPaymentGatewayRepo(), nil)

	payload := map[string]string{
		"order_id":       "INVALID-123",
		"status_code":    "200",
		"gross_amount":   "10000.00",
		"signature_key":  "xyz",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhooks/midtrans", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleMidtransWebhook(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandleMidtransWebhook_AmountMismatch(t *testing.T) {
	tenantID := uuid.New()
	serverKey := "SK-test-123"
	orderID := midtrans.BuildOrderID(tenantID)
	sig := sha512.Sum512([]byte(orderID + "200" + "10000.00" + serverKey))

	repo := newMockPaymentGatewayRepo()
	repo.configs[tenantID.String()+":midtrans"] = &domain.TenantPaymentConfig{TenantID: tenantID, Provider: "midtrans", ServerKey: &serverKey}
	repo.transactions[orderID] = &domain.PaymentTransaction{TenantID: tenantID, OrderID: orderID, Amount: decimal.NewFromInt(10000), Status: "pending"}

	factory := func(sk, ck string, prod bool) handler.MidtransStatusChecker {
		return &mockStatusChecker{status: "settlement", amount: 500}
	}
	h := handler.NewPaymentWebhookHandler(repo, factory)
	body, _ := json.Marshal(map[string]string{"order_id": orderID, "status_code": "200", "gross_amount": "10000.00", "signature_key": hex.EncodeToString(sig[:])})
	w := httptest.NewRecorder()
	h.HandleMidtransWebhook(w, httptest.NewRequest("POST", "/webhooks/midtrans", bytes.NewReader(body)))

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "pending", repo.transactions[orderID].Status)
}
