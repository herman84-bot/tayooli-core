package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// IsBillingDisabled checks if billing and trial enforcement is disabled for demo mode.
// Enabled by default for client demo unless DISABLE_BILLING="false".
func IsBillingDisabled() bool {
	v := os.Getenv("DISABLE_BILLING")
	if v == "false" || v == "0" {
		return false
	}
	return true
}

// SubscriptionStatus represents subscription status for middleware.
type SubscriptionStatus string

const (
	StatusTrialing  SubscriptionStatus = "trialing"
	StatusActive    SubscriptionStatus = "active"
	StatusPastDue   SubscriptionStatus = "past_due"
	StatusCancelled SubscriptionStatus = "cancelled"
	StatusExpired   SubscriptionStatus = "expired"
)

// Subscription represents subscription data for middleware.
type Subscription struct {
	Status     SubscriptionStatus
	Plan       string
	TrialEnds  time.Time
}

// PlanLimits represents plan limits for middleware.
type PlanLimits struct {
	MaxUsers          int
	MaxVendors        int
	MaxInvoicesPerMonth int
	MaxOCRPerMonth    int
	MultiEntity       bool
	APIAccess         bool
}

// SubscriptionProvider abstracts subscription data access for middleware.
type SubscriptionProvider interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*Subscription, error)
	GetPlanLimits(ctx context.Context, plan string) (*PlanLimits, error)
}

// UsageProvider abstracts usage counting for middleware.
type UsageProvider interface {
	CountUsers(ctx context.Context, tenantID uuid.UUID) (int, error)
}

// planGuardKey is the context key for subscription data.
type planGuardKey string

const PlanGuardDataKey planGuardKey = "plan_guard_data"

// PlanGuardData contains subscription check results for handlers.
type PlanGuardData struct {
	Subscription *Subscription
	Limits       *PlanLimits
	Usage        map[string]int
}

// GetPlanGuardData retrieves plan guard data from context.
func GetPlanGuardData(ctx context.Context) (*PlanGuardData, bool) {
	data, ok := ctx.Value(PlanGuardDataKey).(*PlanGuardData)
	return data, ok
}

// PlanGuard creates middleware that checks subscription status and plan limits.
// Only write operations (POST, PUT, PATCH, DELETE) are enforced.
// GET/HEAD are always allowed (read-only access).
func PlanGuard(subProvider SubscriptionProvider, usageProvider UsageProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip enforcement for read-only operations
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// In demo mode with billing disabled, allow all write operations without paywalls.
			if IsBillingDisabled() {
				next.ServeHTTP(w, r)
				return
			}

			tenantID, ok := GetTenantID(r.Context())
			if !ok {
				// No tenant context — skip (endpoint doesn't require auth)
				next.ServeHTTP(w, r)
				return
			}

			// Get subscription
			sub, err := subProvider.GetByTenantID(r.Context(), tenantID)
			if err != nil || sub == nil {
				// No subscription found — allow (might be registration or demo)
				next.ServeHTTP(w, r)
				return
			}

			// Check trial expiry
			if sub.Status == StatusTrialing && time.Now().After(sub.TrialEnds) {
				sub.Status = StatusExpired
			}

			// Check if subscription allows write operations
			if sub.Status == StatusExpired || sub.Status == StatusCancelled {
				respondPlanError(w, r, http.StatusForbidden, "subscription_expired", "Your subscription has expired. Please upgrade to continue.")
				return
			}

			// Check plan limits
			limits, err := subProvider.GetPlanLimits(r.Context(), sub.Plan)
			if err == nil && limits != nil && usageProvider != nil {
				usage := make(map[string]int)

				// Check user limit
				if limits.MaxUsers > 0 {
					userCount, err := usageProvider.CountUsers(r.Context(), tenantID)
					if err == nil {
						usage["users"] = userCount
						if userCount >= limits.MaxUsers {
							respondPlanError(w, r, http.StatusForbidden, "user_limit_reached",
								fmt.Sprintf("You've reached the maximum of %d users for your %s plan. Please upgrade.", limits.MaxUsers, sub.Plan))
							return
						}
					}
				}

				// Store plan guard data in context for handlers
				planData := &PlanGuardData{
					Subscription: sub,
					Limits:       limits,
					Usage:        usage,
				}
				ctx := context.WithValue(r.Context(), PlanGuardDataKey, planData)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No limits found — allow
			next.ServeHTTP(w, r)
		})
	}
}

func respondPlanError(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}


