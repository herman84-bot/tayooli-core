package sales_order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type Usecase struct {
	repo         domain.SalesOrderRepository
	customerRepo domain.CustomerRepository
}

func New(repo domain.SalesOrderRepository, opts ...Option) *Usecase {
	return &Usecase{repo: repo}
}

// Option configures the Usecase.
type Option func(*Usecase)

// WithCustomerRepo sets the customer repository for cross-tenant validation.
func WithCustomerRepo(cr domain.CustomerRepository) Option {
	return func(u *Usecase) { u.customerRepo = cr }
}

// NewWithDeps creates a Usecase with explicit dependencies.
func NewWithDeps(repo domain.SalesOrderRepository, customerRepo domain.CustomerRepository) *Usecase {
	return &Usecase{repo: repo, customerRepo: customerRepo}
}

type CreateRequest struct {
	CustomerID  uuid.UUID       `json:"customer_id"`
	OrderNumber string          `json:"order_number"`
	TotalAmount decimal.Decimal `json:"total_amount"`
}

func (u *Usecase) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*domain.SalesOrder, error) {
	// Validate customer belongs to this tenant
	if u.customerRepo != nil {
		_, err := u.customerRepo.GetByID(ctx, tenantID, req.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("customer not found or does not belong to tenant: %w", err)
		}
	}

	so := &domain.SalesOrder{
		ID:          uuid.New(),
		TenantID:    tenantID,
		CustomerID:  req.CustomerID,
		OrderNumber: req.OrderNumber,
		TotalAmount: req.TotalAmount,
		Status:      "PENDING",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := u.repo.Create(ctx, so); err != nil {
		return nil, err
	}
	return so, nil
}

func (u *Usecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	return u.repo.GetByID(ctx, tenantID, id)
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	return u.repo.List(ctx, tenantID)
}
