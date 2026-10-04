package payment

import (
	"context"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// Facade is a convenience wrapper that satisfies the handler's
// paymentOrderUsecase interface by delegating to individual use-case structs.
type Facade struct {
	repo   domain.PaymentOrderRepository
	list   *ListUsecase
	create *CreateUsecase
	approve *ApproveUsecase
	pay    *PayUsecase
	reject *RejectUsecase
}

func NewFacade(poRepo domain.PaymentOrderRepository, invRepo domain.InvoiceRepository, publisher EventPublisher, auditRepo domain.AuditLogRepository) *Facade {
	return &Facade{
		repo:    poRepo,
		list:    NewListUsecase(poRepo),
		create:  NewCreateUsecase(poRepo, invRepo, publisher, auditRepo),
		approve: NewApproveUsecase(poRepo, publisher, auditRepo),
		pay:     NewPayUsecase(poRepo, publisher, auditRepo),
		reject:  NewRejectUsecase(poRepo, publisher, auditRepo),
	}
}

func (f *Facade) ListPaymentOrders(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error) {
	return f.list.ExecutePaged(ctx, tenantID, page, perPage)
}

func (f *Facade) GetPaymentOrder(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error) {
	return f.repo.GetByID(ctx, id, tenantID)
}

func (f *Facade) CreatePaymentOrder(ctx context.Context, params domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
	return f.create.Execute(ctx, params)
}

func (f *Facade) ApprovePaymentOrder(ctx context.Context, id, tenantID, approvedBy uuid.UUID, role string) (*domain.PaymentOrder, error) {
	return f.approve.Execute(ctx, id, tenantID, approvedBy, role)
}

func (f *Facade) PayPaymentOrder(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error) {
	return f.pay.Execute(ctx, id, tenantID)
}

func (f *Facade) RejectPaymentOrder(ctx context.Context, id, tenantID, rejectedBy uuid.UUID) (*domain.PaymentOrder, error) {
	return f.reject.Execute(ctx, id, tenantID, rejectedBy)
}
