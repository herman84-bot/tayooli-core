package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/midtrans"
)

// MidtransStatusChecker re-verifies a transaction server-to-server.
// Built per tenant because credentials are BYO (each tenant owns its account).
type MidtransStatusChecker interface {
	VerifyTransactionStatus(ctx context.Context, orderID string) (*midtrans.TransactionStatus, error)
}

// MidtransClientFactory builds a checker from a tenant's credentials.
type MidtransClientFactory func(serverKey, clientKey string, isProduction bool) MidtransStatusChecker

// DefaultMidtransClientFactory uses the real SDK client.
func DefaultMidtransClientFactory(serverKey, clientKey string, isProduction bool) MidtransStatusChecker {
	return midtrans.NewClient(serverKey, clientKey, isProduction)
}

// PaymentWebhookHandler handles incoming payment gateway webhooks.
type PaymentWebhookHandler struct {
	repo       domain.PaymentGatewayRepository
	newChecker MidtransClientFactory
}

func NewPaymentWebhookHandler(repo domain.PaymentGatewayRepository, factory MidtransClientFactory) *PaymentWebhookHandler {
	if factory == nil {
		factory = DefaultMidtransClientFactory
	}
	return &PaymentWebhookHandler{repo: repo, newChecker: factory}
}

type midtransNotification struct {
	OrderID      string `json:"order_id"`
	StatusCode   string `json:"status_code"`
	GrossAmount  string `json:"gross_amount"`
	SignatureKey string `json:"signature_key"`
}

func ack(w http.ResponseWriter, note string) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok", "note": note})
}

// HandleMidtransWebhook handles POST /api/v1/webhooks/midtrans (public, no auth).
//
// Trust model — the body is never trusted for state changes:
//  1. tenant resolved from order_id (we generated it, see midtrans.BuildOrderID)
//  2. SHA512 signature verified with that tenant's server key
//  3. status + amount re-fetched from Midtrans API with the tenant's key
//  4. only then is the transaction marked completed (idempotent)
//
// Unknown/foreign orders are acknowledged with 200 so Midtrans stops retrying.
// Transient failures return 5xx so Midtrans retries.
func (h *PaymentWebhookHandler) HandleMidtransWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "failed to read body")
		return
	}
	var n midtransNotification
	if err := json.Unmarshal(body, &n); err != nil || n.OrderID == "" {
		RespondError(w, r, http.StatusBadRequest, "invalid notification payload")
		return
	}

	tenantID, err := midtrans.TenantFromOrderID(n.OrderID)
	if err != nil {
		ack(w, "ignored: foreign order id")
		return
	}

	cfg, err := h.repo.GetConfig(r.Context(), tenantID, "midtrans")
	if errors.Is(err, domain.ErrNotFound) {
		ack(w, "ignored: tenant has no midtrans config")
		return
	}
	if err != nil {
		RespondError(w, r, http.StatusInternalServerError, "failed to load config")
		return
	}
	if cfg.ServerKey == nil || *cfg.ServerKey == "" {
		ack(w, "ignored: tenant server key empty")
		return
	}

	if !midtrans.VerifySignature(n.OrderID, n.StatusCode, n.GrossAmount, *cfg.ServerKey, n.SignatureKey) {
		slog.Warn("midtrans webhook signature invalid", "tenant_id", tenantID, "order_id", n.OrderID)
		RespondError(w, r, http.StatusForbidden, "invalid signature")
		return
	}

	tx, err := h.repo.GetTransactionByOrderID(r.Context(), tenantID, n.OrderID)
	if errors.Is(err, domain.ErrNotFound) {
		ack(w, "ignored: unknown transaction")
		return
	}
	if err != nil {
		RespondError(w, r, http.StatusInternalServerError, "failed to load transaction")
		return
	}
	if tx.Status == "completed" {
		ack(w, "already completed")
		return
	}

	clientKey := ""
	if cfg.ClientKey != nil {
		clientKey = *cfg.ClientKey
	}
	st, err := h.newChecker(*cfg.ServerKey, clientKey, cfg.IsProduction).VerifyTransactionStatus(r.Context(), n.OrderID)
	if err != nil {
		slog.Error("midtrans status re-check failed", "tenant_id", tenantID, "order_id", n.OrderID, "error", err)
		RespondError(w, r, http.StatusBadGateway, "status verification failed")
		return
	}
	if st.Amount != tx.Amount.IntPart() {
		slog.Warn("midtrans amount mismatch", "tenant_id", tenantID, "order_id", n.OrderID, "gateway", st.Amount, "stored", tx.Amount.String())
		RespondError(w, r, http.StatusBadRequest, "amount mismatch")
		return
	}

	newStatus := ""
	switch {
	case midtrans.IsPaidStatus(st.Status):
		newStatus = "completed"
	case st.Status == "expire":
		newStatus = "expired"
	case st.Status == "deny" || st.Status == "cancel" || st.Status == "failure":
		newStatus = "failed"
	default:
		ack(w, "pending")
		return
	}

	var completedAt *time.Time
	if newStatus == "completed" {
		now := time.Now().UTC()
		completedAt = &now
	}
	if err := h.repo.UpdateTransactionStatus(r.Context(), tenantID, n.OrderID, newStatus, completedAt); err != nil {
		RespondError(w, r, http.StatusInternalServerError, "failed to update transaction")
		return
	}
	slog.Info("midtrans webhook processed", "tenant_id", tenantID, "order_id", n.OrderID, "status", newStatus)
	ack(w, newStatus)
}
