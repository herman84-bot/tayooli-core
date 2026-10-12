package dashboard

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// Usecase implements the dashboard summary query logic.
type Usecase struct {
	repo  domain.DashboardRepository
	cache *summaryCache // nil = no caching
}

// New creates a dashboard Usecase without caching.
func New(repo domain.DashboardRepository) *Usecase {
	return &Usecase{repo: repo}
}

// NewWithCache creates a dashboard Usecase that caches each tenant's summary
// for ttl and coalesces concurrent identical requests into one DB fetch.
func NewWithCache(repo domain.DashboardRepository, ttl time.Duration) *Usecase {
	return &Usecase{repo: repo, cache: &summaryCache{ttl: ttl, entries: map[cacheKey]*cacheEntry{}}}
}

// GetSummary returns aggregated dashboard metrics for a tenant with a period filter.
func (u *Usecase) GetSummary(ctx context.Context, tenantID uuid.UUID, days int) (*domain.DashboardSummary, error) {
	fetch := func() (*domain.DashboardSummary, error) {
		// Detached from the caller's cancellation: the result is shared with
		// coalesced waiters, so one client disconnecting must not fail them all.
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		s, err := u.repo.GetSummary(fctx, tenantID, days)
		if err != nil {
			return nil, err
		}
		deriveFinancials(s)
		return s, nil
	}
	if u.cache == nil {
		return fetch()
	}
	return u.cache.get(ctx, cacheKey{tenant: tenantID, days: days}, fetch)
}

// deriveFinancials fills cross-module figures from the raw per-module stats.
func deriveFinancials(s *domain.DashboardSummary) {
	if s.POS.TotalOrdersCount > 0 {
		s.POS.AverageBasketSize = s.POS.TotalRevenue.
			Div(decimal.NewFromInt(int64(s.POS.TotalOrdersCount))).Round(2)
	}
	s.Financial = domain.FinancialOverview{
		// sales_invoices already contains POS sales (PAID INV-POS), so no POS add-on.
		TotalRevenue:       s.SalesInvoices.TotalInvoiced,
		CashInflow:         s.SalesInvoices.PaidAmount,
		AccountsReceivable: s.SalesInvoices.AccountsReceivable,
		TotalExpense:       s.Invoices.ApprovedAmount,
		CashOutflow:        s.Payments.PaidAmount,
		AccountsPayable:    s.Payments.PendingAmount,
		NetCashBalance:     s.SalesInvoices.PaidAmount.Sub(s.Payments.PaidAmount),
	}
}
