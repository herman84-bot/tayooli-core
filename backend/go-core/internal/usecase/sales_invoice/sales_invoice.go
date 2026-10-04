package sales_invoice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload interface{}) error
}

type Usecase struct {
	repo          domain.SalesInvoiceRepository
	salesOrderRepo domain.SalesOrderRepository
	publisher     EventPublisher
}

func New(repo domain.SalesInvoiceRepository, publisher EventPublisher, opts ...Option) *Usecase {
	return &Usecase{repo: repo, publisher: publisher}
}

// Option configures the Usecase.
type Option func(*Usecase)

// WithSalesOrderRepo sets the sales order repository for cross-tenant validation.
func WithSalesOrderRepo(sor domain.SalesOrderRepository) Option {
	return func(u *Usecase) { u.salesOrderRepo = sor }
}

// NewWithDeps creates a Usecase with explicit dependencies.
func NewWithDeps(repo domain.SalesInvoiceRepository, salesOrderRepo domain.SalesOrderRepository, publisher EventPublisher) *Usecase {
	return &Usecase{repo: repo, salesOrderRepo: salesOrderRepo, publisher: publisher}
}

type CreateRequest struct {
	SalesOrderID  uuid.UUID       `json:"sales_order_id"`
	InvoiceNumber string          `json:"invoice_number"`
	Amount        decimal.Decimal `json:"amount"`
	DueDate       time.Time       `json:"due_date"`
}

func (u *Usecase) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*domain.SalesInvoice, error) {
	// Validate sales order belongs to this tenant
	if u.salesOrderRepo != nil {
		_, err := u.salesOrderRepo.GetByID(ctx, tenantID, req.SalesOrderID)
		if err != nil {
			return nil, fmt.Errorf("sales order not found or does not belong to tenant: %w", err)
		}
	}

	si := &domain.SalesInvoice{
		ID:           uuid.New(),
		TenantID:     tenantID,
		SalesOrderID: req.SalesOrderID,
		InvoiceNumber: req.InvoiceNumber,
		Amount:       req.Amount,
		Status:       "UNPAID",
		DueDate:      req.DueDate,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := u.repo.Create(ctx, si); err != nil {
		return nil, err
	}

	event := domain.SalesInvoiceCreatedEvent{
		EventID:        uuid.New().String(),
		TenantID:       si.TenantID.String(),
		SalesInvoiceID: si.ID.String(),
		InvoiceNumber:  si.InvoiceNumber,
		Amount:         si.Amount.String(),
		Currency:       "IDR",
		Timestamp:      time.Now().Format(time.RFC3339),
	}
	_ = u.publisher.Publish(ctx, "sales_invoice.created", si.ID.String(), event)

	return si, nil
}

func (u *Usecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesInvoice, error) {
	return u.repo.GetByID(ctx, tenantID, id)
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesInvoice, error) {
	return u.repo.List(ctx, tenantID)
}
