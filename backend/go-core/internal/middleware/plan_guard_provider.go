package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// SubscriptionGetter abstracts fetching subscription for PlanGuard.
type SubscriptionGetter interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantSubscription, error)
}

// PlanLimitsGetter abstracts fetching plan limits for PlanGuard.
type PlanLimitsGetter interface {
	GetPlanLimits(ctx context.Context, plan string) (*domain.PlanLimits, error)
}

// UserCounter abstracts counting users for PlanGuard.
type UserCounter interface {
	CountUsersByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
}

// planGuardAdapter bridges domain types to middleware types.
type planGuardAdapter struct {
	subGetter  SubscriptionGetter
	limitsGetter PlanLimitsGetter
	userCounter  UserCounter
}

// NewPlanGuardAdapter creates a PlanGuard-compatible provider from domain interfaces.
func NewPlanGuardAdapter(sub SubscriptionGetter, limits PlanLimitsGetter, users UserCounter) *planGuardAdapter {
	return &planGuardAdapter{
		subGetter:    sub,
		limitsGetter: limits,
		userCounter:  users,
	}
}

func (a *planGuardAdapter) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	sub, err := a.subGetter.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &Subscription{
		Status:    SubscriptionStatus(sub.Status),
		Plan:      string(sub.Plan),
		TrialEnds: sub.TrialEndsAt,
	}, nil
}

func (a *planGuardAdapter) GetPlanLimits(ctx context.Context, plan string) (*PlanLimits, error) {
	limits, err := a.limitsGetter.GetPlanLimits(ctx, plan)
	if err != nil {
		return nil, err
	}
	return &PlanLimits{
		MaxUsers:          limits.MaxUsers,
		MaxVendors:        limits.MaxVendors,
		MaxInvoicesPerMonth: limits.MaxInvoicesPerMonth,
		MaxOCRPerMonth:    limits.MaxOCRPerMonth,
		MultiEntity:       limits.MultiEntity,
		APIAccess:         limits.APIAccess,
	}, nil
}

func (a *planGuardAdapter) CountUsers(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return a.userCounter.CountUsersByTenant(ctx, tenantID)
}
