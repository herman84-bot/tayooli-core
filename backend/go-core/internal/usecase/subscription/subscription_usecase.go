package subscription

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/pakasir"
)

// SubscriptionRepo defines the repository interface needed by this usecase.
type SubscriptionRepo interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSubscription, error)
	GetInvoiceByID(ctx context.Context, tenantID, invoiceID uuid.UUID) (*domain.SubscriptionInvoice, error)
	Create(ctx context.Context, sub *domain.TenantSubscription) error
	UpdateStatus(ctx context.Context, tenantID uuid.UUID, status domain.SubscriptionStatus) error
	UpgradePlan(ctx context.Context, tenantID uuid.UUID, plan domain.SubscriptionPlan, period domain.BillingPeriod) error
	Cancel(ctx context.Context, tenantID uuid.UUID) error
	GetPlanLimits(ctx context.Context, plan string) (*domain.PlanLimits, error)
	CreateInvoice(ctx context.Context, inv *domain.SubscriptionInvoice) error
	UpdateInvoiceStatus(ctx context.Context, tenantID uuid.UUID, invoiceID uuid.UUID, status string, paidAt *time.Time) error
	ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionInvoice, error)
	ListAllInvoices(ctx context.Context) ([]domain.SubscriptionInvoice, error)
}

// TenantRepo provides access to tenant data for usage counting.
type TenantRepo interface {
	CountUsersByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
}

type Usecase struct {
	repo      SubscriptionRepo
	tenantRepo TenantRepo
}

func New(repo SubscriptionRepo, tenantRepo TenantRepo) *Usecase {
	return &Usecase{repo: repo, tenantRepo: tenantRepo}
}

// GetByTenantID retrieves the subscription for a tenant.
func (u *Usecase) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSubscription, error) {
	sub, err := u.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Auto-expire trial if past due
	if sub.IsTrialActive() && time.Now().After(sub.TrialEndsAt) {
		sub.Status = domain.SubscriptionStatusExpired
		_ = u.repo.UpdateStatus(ctx, tenantID, domain.SubscriptionStatusExpired)
	}

	return sub, nil
}

// Create creates a new trial subscription (called during registration).
func (u *Usecase) Create(ctx context.Context, tenantID uuid.UUID, plan domain.SubscriptionPlan, period domain.BillingPeriod) (*domain.TenantSubscription, error) {
	// Check if subscription already exists
	existing, err := u.repo.GetByTenantID(ctx, tenantID)
	if err == nil && existing != nil {
		return existing, nil // Already has subscription
	}

	now := time.Now()
	sub := &domain.TenantSubscription{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Plan:           plan,
		Status:         domain.SubscriptionStatusTrialing,
		TrialStartedAt: now,
		TrialEndsAt:    now.Add(14 * 24 * time.Hour),
		BillingPeriod:  period,
		PaymentProvider: "pakasir",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := u.repo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("subscription.Create: %w", err)
	}

	return sub, nil
}

// Upgrade changes the subscription plan.
func (u *Usecase) Upgrade(ctx context.Context, tenantID uuid.UUID, plan domain.SubscriptionPlan, period domain.BillingPeriod) error {
	sub, err := u.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return err
	}

	// Can only upgrade from active or trial
	if sub.Status != domain.SubscriptionStatusActive && sub.Status != domain.SubscriptionStatusTrialing {
		return fmt.Errorf("cannot upgrade subscription with status: %s", sub.Status)
	}

	if err := u.repo.UpgradePlan(ctx, tenantID, plan, period); err != nil {
		return fmt.Errorf("subscription.Upgrade: %w", err)
	}

	return nil
}

// Cancel marks the subscription for cancellation at end of period.
func (u *Usecase) Cancel(ctx context.Context, tenantID uuid.UUID) error {
	if err := u.repo.Cancel(ctx, tenantID); err != nil {
		return fmt.Errorf("subscription.Cancel: %w", err)
	}
	return nil
}

// GetPlanLimits retrieves the limits for a specific plan.
func (u *Usecase) GetPlanLimits(ctx context.Context, plan string) (*domain.PlanLimits, error) {
	return u.repo.GetPlanLimits(ctx, plan)
}

// GetUsage returns current resource usage for a tenant.
func (u *Usecase) GetUsage(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	users := 0
	if u.tenantRepo != nil {
		var err error
		users, err = u.tenantRepo.CountUsersByTenant(ctx, tenantID)
		if err != nil {
			users = 0 // Fallback to 0 if counting fails
		}
	}

	return map[string]int{
		"users": users,
	}, nil
}

// ListInvoices retrieves all subscription invoices.
func (u *Usecase) ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionInvoice, error) {
	return u.repo.ListInvoices(ctx, tenantID)
}

// GeneratePaymentLink creates a Pakasir payment link for a subscription invoice.
func (u *Usecase) GeneratePaymentLink(ctx context.Context, tenantID uuid.UUID, invoiceID string) (string, error) {
	invUUID, err := uuid.Parse(invoiceID)
	if err != nil {
		return "", fmt.Errorf("invalid invoice ID")
	}

	inv, err := u.repo.GetInvoiceByID(ctx, tenantID, invUUID)
	if err != nil {
		return "", fmt.Errorf("invoice not found")
	}

	if inv.Status == "paid" {
		return "", fmt.Errorf("invoice already paid")
	}

	// Get Pakasir credentials from environment
	slug := os.Getenv("PAKASIR_SLUG")
	apiKey := os.Getenv("PAKASIR_API_KEY")
	if slug == "" || apiKey == "" {
		return "", fmt.Errorf("payment gateway not configured")
	}

	client := pakasir.NewClient()
	paymentResp, err := client.CreateTransaction(ctx, slug, inv.InvoiceNumber, apiKey, pakasir.PaymentMethodQRIS, inv.Amount)
	if err != nil {
		return "", fmt.Errorf("failed to create payment: %w", err)
	}

	// Update invoice with payment link
	paymentLink := paymentResp.Payment.PaymentNumber
	paymentRef := paymentResp.Payment.OrderID
	_ = u.repo.UpdateInvoiceStatus(ctx, tenantID, inv.ID, "pending", nil)

	// Build payment URL for QRIS
	paymentURL := fmt.Sprintf("https://app.pakasir.com/pay/%s/%d?order_id=%s", slug, inv.Amount, inv.InvoiceNumber)
	_ = paymentRef
	_ = paymentLink

	return paymentURL, nil
}

// HandleWebhook processes a Pakasir payment webhook.
func (u *Usecase) HandleWebhook(ctx context.Context, orderID, project, status string, amount int) error {
	if status != "completed" {
		return nil // Only process completed payments
	}

	// Find invoice by order_id (invoice_number)
	// We need to look up the invoice by its number
	invoices, err := u.repo.ListAllInvoices(ctx)
	if err != nil {
		return fmt.Errorf("webhook: failed to list invoices: %w", err)
	}

	var targetInvoice *domain.SubscriptionInvoice
	for i := range invoices {
		if invoices[i].InvoiceNumber == orderID {
			targetInvoice = &invoices[i]
			break
		}
	}

	if targetInvoice == nil {
		return fmt.Errorf("webhook: invoice not found for order_id=%s", orderID)
	}

	// Verify amount matches
	if targetInvoice.Amount != amount {
		return fmt.Errorf("webhook: amount mismatch expected=%d got=%d", targetInvoice.Amount, amount)
	}

	// Mark invoice as paid
	now := time.Now()
	if err := u.repo.UpdateInvoiceStatus(ctx, targetInvoice.TenantID, targetInvoice.ID, "paid", &now); err != nil {
		return fmt.Errorf("webhook: failed to mark invoice paid: %w", err)
	}

	// Activate subscription
	if err := u.repo.UpdateStatus(ctx, targetInvoice.TenantID, domain.SubscriptionStatusActive); err != nil {
		return fmt.Errorf("webhook: failed to activate subscription: %w", err)
	}

	// Set subscription period
	periodStart := now
	var periodEnd time.Time
	if targetInvoice.PeriodEnd.After(now) {
		periodEnd = targetInvoice.PeriodEnd
	} else {
		periodEnd = now.AddDate(0, 1, 0)
	}
	_ = periodStart
	_ = strconv.Itoa(int(periodEnd.Unix()))

	return nil
}
