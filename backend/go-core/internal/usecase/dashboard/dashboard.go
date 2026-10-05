package dashboard

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// Usecase implements the dashboard summary query logic.
type Usecase struct {
	repo domain.DashboardRepository
}

// New creates a dashboard Usecase.
func New(repo domain.DashboardRepository) *Usecase {
	return &Usecase{repo: repo}
}

// GetSummary returns aggregated dashboard metrics for a tenant.
func (u *Usecase) GetSummary(ctx context.Context, tenantID uuid.UUID) (*domain.DashboardSummary, error) {
	s, err := u.repo.GetSummary(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	deriveFinancials(s)
	return s, nil
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
