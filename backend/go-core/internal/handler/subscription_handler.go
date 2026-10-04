package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/pakasir"
)

// SubscriptionUsecase defines the business logic interface for subscriptions.
type SubscriptionUsecase interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSubscription, error)
	Create(ctx context.Context, tenantID uuid.UUID, plan domain.SubscriptionPlan, period domain.BillingPeriod) (*domain.TenantSubscription, error)
	Upgrade(ctx context.Context, tenantID uuid.UUID, plan domain.SubscriptionPlan, period domain.BillingPeriod) error
	Cancel(ctx context.Context, tenantID uuid.UUID) error
	GetPlanLimits(ctx context.Context, plan string) (*domain.PlanLimits, error)
	GetUsage(ctx context.Context, tenantID uuid.UUID) (map[string]int, error)
	ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionInvoice, error)
	GeneratePaymentLink(ctx context.Context, tenantID uuid.UUID, invoiceID string) (string, error)
	HandleWebhook(ctx context.Context, orderID, project, status string, amount int) error
}

// SubscriptionNotifier sends notifications for subscription events.
type SubscriptionNotifier interface {
	NotifyNewSubscription(tenantID, plan, period string) error
	NotifyUpgrade(tenantID, fromPlan, toPlan string) error
	NotifyCancel(tenantID, plan string) error
}

type SubscriptionHandler struct {
	uc         SubscriptionUsecase
	notifier   SubscriptionNotifier
	pakasirKey string
}

func NewSubscriptionHandler(uc SubscriptionUsecase, notifier SubscriptionNotifier, pakasirKey string) *SubscriptionHandler {
	return &SubscriptionHandler{uc: uc, notifier: notifier, pakasirKey: pakasirKey}
}

// RegisterSecuredRoutes registers auth-required subscription endpoints.
// rateLimit is applied only to POST /subscription/pay (payment-link
// generation); read/status endpoints stay unlimited.
func (h *SubscriptionHandler) RegisterSecuredRoutes(r chi.Router, rateLimit func(http.Handler) http.Handler) {
	r.Route("/subscription", func(r chi.Router) {
		r.Get("/", h.GetSubscription)
		r.Post("/", h.CreateSubscription)
		r.Patch("/", h.UpdateSubscription)
		r.Delete("/", h.CancelSubscription)

		r.Get("/invoices", h.ListInvoices)
		r.With(rateLimit).Post("/pay", h.GeneratePaymentLink)
	})

	r.Route("/plans", func(r chi.Router) {
		r.Get("/", h.ListPlans)
		r.Get("/{plan}/limits", h.GetPlanLimits)
	})

	r.Get("/usage", h.GetUsage)
}

// RegisterWebhookRoutes registers unauthenticated webhook endpoints.
func (h *SubscriptionHandler) RegisterWebhookRoutes(r chi.Router) {
	r.Post("/webhooks/pakasir", h.HandlePakasirWebhook)
}

// GetSubscription handles GET /api/v1/subscription
func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	if appMiddleware.IsBillingDisabled() {
		respondJSON(w, http.StatusOK, map[string]any{
			"id":                   "sub-demo-enterprise",
			"tenant_id":            tenantID,
			"plan":                 "enterprise",
			"status":               "active",
			"billing_period":       "annual",
			"current_period_start": time.Now().AddDate(-1, 0, 0),
			"current_period_end":   time.Now().AddDate(10, 0, 0),
			"created_at":           time.Now().AddDate(-1, 0, 0),
			"updated_at":           time.Now(),
		})
		return
	}

	sub, err := h.uc.GetByTenantID(r.Context(), tenantID)
	if err != nil {
		if err == domain.ErrNotFound {
			respondError(w, r, http.StatusNotFound, "subscription not found")
			return
		}
		respondError(w, r, http.StatusInternalServerError, "failed to get subscription")
		return
	}

	respondJSON(w, http.StatusOK, sub)
}

// CreateSubscription handles POST /api/v1/subscription
func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Plan   string `json:"plan"`
		Period string `json:"period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	plan := domain.SubscriptionPlan(req.Plan)
	period := domain.BillingPeriod(req.Period)

	if plan == "" || (period != domain.BillingPeriodMonthly && period != domain.BillingPeriodAnnual) {
		respondError(w, r, http.StatusBadRequest, "invalid plan or period")
		return
	}

	sub, err := h.uc.Create(r.Context(), tenantID, plan, period)
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "failed to create subscription")
		return
	}

	// Notify via Telegram (async, non-blocking)
	if h.notifier != nil {
		go h.notifier.NotifyNewSubscription(tenantID.String(), string(plan), string(period))
	}

	respondJSON(w, http.StatusCreated, sub)
}

// UpdateSubscription handles PATCH /api/v1/subscription
func (h *SubscriptionHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Plan   string `json:"plan"`
		Period string `json:"period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	plan := domain.SubscriptionPlan(req.Plan)
	period := domain.BillingPeriod(req.Period)

	if plan == "" || (period != domain.BillingPeriodMonthly && period != domain.BillingPeriodAnnual) {
		respondError(w, r, http.StatusBadRequest, "invalid plan or period")
		return
	}

	if err := h.uc.Upgrade(r.Context(), tenantID, plan, period); err != nil {
		respondError(w, r, http.StatusInternalServerError, "failed to update subscription")
		return
	}

	// Notify via Telegram (async, non-blocking)
	if h.notifier != nil {
		go h.notifier.NotifyUpgrade(tenantID.String(), "", string(plan))
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// CancelSubscription handles DELETE /api/v1/subscription
func (h *SubscriptionHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.uc.Cancel(r.Context(), tenantID); err != nil {
		respondError(w, r, http.StatusInternalServerError, "failed to cancel subscription")
		return
	}

	// Notify via Telegram (async, non-blocking)
	if h.notifier != nil {
		go h.notifier.NotifyCancel(tenantID.String(), "")
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// ListPlans handles GET /api/v1/plans
func (h *SubscriptionHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans := []map[string]interface{}{
		{"name": "starter", "monthly": 199000, "annual": 159200},
		{"name": "bisnis", "monthly": 449000, "annual": 359200, "featured": true},
		{"name": "enterprise", "monthly": nil, "annual": nil},
	}
	respondJSON(w, http.StatusOK, plans)
}

// GetPlanLimits handles GET /api/v1/plans/{plan}/limits
func (h *SubscriptionHandler) GetPlanLimits(w http.ResponseWriter, r *http.Request) {
	plan := chi.URLParam(r, "plan")
	if plan == "" {
		respondError(w, r, http.StatusBadRequest, "plan is required")
		return
	}

	limits, err := h.uc.GetPlanLimits(r.Context(), plan)
	if err != nil {
		if err == domain.ErrNotFound {
			respondError(w, r, http.StatusNotFound, "plan not found")
			return
		}
		respondError(w, r, http.StatusInternalServerError, "failed to get plan limits")
		return
	}

	respondJSON(w, http.StatusOK, limits)
}

// GetUsage handles GET /api/v1/usage
func (h *SubscriptionHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	usage, err := h.uc.GetUsage(r.Context(), tenantID)
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "failed to get usage")
		return
	}

	respondJSON(w, http.StatusOK, usage)
}

// ListInvoices handles GET /api/v1/subscription/invoices
func (h *SubscriptionHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	invoices, err := h.uc.ListInvoices(r.Context(), tenantID)
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "failed to list invoices")
		return
	}

	respondJSON(w, http.StatusOK, invoices)
}

// GeneratePaymentLink handles POST /api/v1/subscription/pay
// Creates a Pakasir QRIS transaction and returns payment data for the frontend.
func (h *SubscriptionHandler) GeneratePaymentLink(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Plan   string `json:"plan"`
		Period string `json:"period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Plan == "" {
		respondError(w, r, http.StatusBadRequest, "plan is required")
		return
	}

	// Map plan to amount
	var amount int
	switch req.Plan {
	case "starter":
		if req.Period == "annual" {
			amount = 1592000
		} else {
			amount = 199000
		}
	case "bisnis":
		if req.Period == "annual" {
			amount = 3592000
		} else {
			amount = 449000
		}
	default:
		respondError(w, r, http.StatusBadRequest, "invalid plan")
		return
	}

	// Generate order code
	orderCode := fmt.Sprintf("TAY-%s-%s", tenantID.String()[:8], time.Now().Format("20060102150405"))

	// Create Pakasir QRIS transaction
	if h.pakasirKey != "" {
		pakasirClient := pakasir.NewClient()
		x, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		result, err := pakasirClient.CreateTransaction(
			x, "tayooli-erp", orderCode, h.pakasirKey, "qris", amount,
		)
		if err != nil {
			// Log error but still return payment URL as fallback
			fmt.Fprintf(os.Stderr, "pakasir create transaction error: %v\n", err)
		} else {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"order_code":       orderCode,
				"plan":             req.Plan,
				"amount":           amount,
				"period":           req.Period,
				"payment_url":      result.Payment.PaymentNumber,
				"payment_method":   result.Payment.PaymentMethod,
				"total_payment":    result.Payment.TotalPayment,
				"fee":              result.Payment.Fee,
				"expired_at":       result.Payment.ExpiredAt,
				"payment_number":   result.Payment.PaymentNumber,
				"status":           "pending",
			})
			return
		}
	}

	// Fallback: return payment URL without Pakasir integration
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"order_code":  orderCode,
		"plan":        req.Plan,
		"amount":      amount,
		"period":      req.Period,
		"payment_url": fmt.Sprintf("https://app.pakasir.com/pay/tayooli-erp/%s", orderCode),
		"status":      "pending",
	})
}

// HandlePakasirWebhook handles POST /api/v1/webhooks/pakasir
// Called by Pakasir when a payment is completed.
func (h *SubscriptionHandler) HandlePakasirWebhook(w http.ResponseWriter, r *http.Request) {
	// Webhook signature / token verification if configured
	webhookKey := os.Getenv("PAKASIR_WEBHOOK_KEY")
	if webhookKey != "" {
		token := r.Header.Get("X-Pakasir-Key")
		if token == "" {
			token = r.URL.Query().Get("key")
		}
		if token != webhookKey {
			respondError(w, r, http.StatusUnauthorized, "unauthorized webhook")
			return
		}
	}

	var payload struct {
		Amount        int    `json:"amount"`
		OrderID       string `json:"order_id"`
		Project       string `json:"project"`
		Status        string `json:"status"`
		PaymentMethod string `json:"payment_method"`
		CompletedAt   string `json:"completed_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	if err := h.uc.HandleWebhook(r.Context(), payload.OrderID, payload.Project, payload.Status, payload.Amount); err != nil {
		respondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Helper to generate invoice number
func generateInvoiceNumber() string {
	return "SUB-" + time.Now().Format("200601") + "-" + uuid.New().String()[:8]
}
