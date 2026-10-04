package customer

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type Usecase struct {
	repo domain.CustomerRepository
}

func New(repo domain.CustomerRepository) *Usecase {
	return &Usecase{repo: repo}
}

type CreateRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

func (u *Usecase) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*domain.Customer, error) {
	c := &domain.Customer{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Address:   req.Address,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := u.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *Usecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	return u.repo.GetByID(ctx, tenantID, id)
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Customer, error) {
	return u.repo.List(ctx, tenantID)
}
