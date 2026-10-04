package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/domain"
)

type AccountUseCase struct {
	repo domain.AccountRepository
}

func NewAccountUseCase(repo domain.AccountRepository) *AccountUseCase {
	return &AccountUseCase{repo: repo}
}

func (u *AccountUseCase) CreateAccount(ctx context.Context, account *domain.Account) error {
	return u.repo.Create(ctx, account)
}

func (u *AccountUseCase) ListAccounts(ctx context.Context, tenantID uuid.UUID) ([]*domain.Account, error) {
	return u.repo.List(ctx, tenantID)
}
