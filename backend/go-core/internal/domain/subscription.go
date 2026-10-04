package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the current state of a tenant's subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusExpired  SubscriptionStatus = "expired"
)

// SubscriptionPlan represents available pricing tiers.
type SubscriptionPlan string

const (
	PlanTrial      SubscriptionPlan = "trial"
	PlanStarter    SubscriptionPlan = "starter"
	PlanBisnis     SubscriptionPlan = "bisnis"
	PlanEnterprise SubscriptionPlan = "enterprise"
)

// BillingPeriod represents the billing cycle.
type BillingPeriod string

const (
	BillingPeriodMonthly BillingPeriod = "monthly"
	BillingPeriodAnnual  BillingPeriod = "annual"
)

// TenantSubscription represents a tenant's subscription to Tayooli.
type TenantSubscription struct {
	ID                uuid.UUID          `json:"id"`
	TenantID          uuid.UUID          `json:"tenant_id"`
	Plan              SubscriptionPlan   `json:"plan"`
	Status            SubscriptionStatus `json:"status"`
	TrialStartedAt    time.Time          `json:"trial_started_at"`
	TrialEndsAt       time.Time          `json:"trial_ends_at"`
	BillingPeriod     BillingPeriod      `json:"billing_period"`
	CurrentPeriodStart *time.Time        `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time        `json:"current_period_end,omitempty"`
	PaymentProvider    string             `json:"payment_provider,omitempty"`
	CancelAt          *time.Time         `json:"cancel_at,omitempty"`
	CancelledAt       *time.Time         `json:"cancelled_at,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// IsExpired returns true if the subscription has expired.
func (s *TenantSubscription) IsExpired() bool {
	return s.Status == SubscriptionStatusExpired || s.Status == SubscriptionStatusCancelled
}

// IsTrialActive returns true if the trial is still ongoing.
func (s *TenantSubscription) IsTrialActive() bool {
	return s.Status == SubscriptionStatusTrialing && time.Now().Before(s.TrialEndsAt)
}

// IsActive returns true if the subscription is active and paid.
func (s *TenantSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive
}

// CanWrite returns true if the tenant can perform write operations.
func (s *TenantSubscription) CanWrite() bool {
	if s.IsExpired() {
		return false
	}
	if s.IsTrialActive() {
		return true
	}
	if s.IsActive() {
		return true
	}
	return false
}

// PlanLimits defines the resource limits for a specific plan.
type PlanLimits struct {
	ID                uuid.UUID `json:"id"`
	Plan              string    `json:"plan"`
	MaxUsers          int       `json:"max_users"`
	MaxVendors        int       `json:"max_vendors"`
	MaxInvoicesPerMonth int     `json:"max_invoices_per_month"`
	MaxOCRPerMonth    int       `json:"max_ocr_per_month"`
	MultiEntity       bool      `json:"multi_entity"`
	APIAccess         bool      `json:"api_access"`
	CreatedAt         time.Time `json:"created_at"`
}

// IsUnlimited returns true if the limit is -1 (unlimited).
func (l *PlanLimits) IsUnlimited(limit int) bool {
	return limit == -1
}

// SubscriptionInvoice represents a billing invoice for subscription.
type SubscriptionInvoice struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	SubscriptionID   uuid.UUID  `json:"subscription_id"`
	InvoiceNumber    string     `json:"invoice_number"`
	Amount           int        `json:"amount"`
	Currency         string     `json:"currency"`
	Status           string     `json:"status"`
	PeriodStart      time.Time  `json:"period_start"`
	PeriodEnd        time.Time  `json:"period_end"`
	PaymentLink      *string    `json:"payment_link,omitempty"`
	PaymentProvider  string     `json:"payment_provider"`
	PaymentReference *string    `json:"payment_reference,omitempty"`
	DueDate          time.Time  `json:"due_date"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// SubscriptionStatus constants
const (
	InvoiceStatusDraft = "draft"
	InvoiceStatusOpen  = "open"
	InvoiceStatusPaid  = "paid"
	InvoiceStatusVoid  = "void"
)
