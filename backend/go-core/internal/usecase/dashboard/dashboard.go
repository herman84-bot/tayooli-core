package dashboard

import (
	"context"

	"github.com/google/uuid"
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
	return u.repo.GetSummary(ctx, tenantID)
}
