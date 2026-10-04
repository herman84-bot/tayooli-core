package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

const TopicPaymentOrderCreated = "payment_order.created"

// EventPublisher is the port for publishing domain events.
// key must be a stable entity identifier (e.g. payment order UUID) to ensure
// related events land on the same Kafka partition for ordered consumption.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, key string, payload any) error
}

// CreateUsecase persists a new payment order and publishes a
// payment_order.created event.
type CreateUsecase struct {
	repo      domain.PaymentOrderRepository
	invRepo   domain.InvoiceRepository
	publisher EventPublisher
	auditRepo domain.AuditLogRepository
}

func NewCreateUsecase(repo domain.PaymentOrderRepository, invRepo domain.InvoiceRepository, publisher EventPublisher, auditRepo domain.AuditLogRepository) *CreateUsecase {
	return &CreateUsecase{repo: repo, invRepo: invRepo, publisher: publisher, auditRepo: auditRepo}
}

func (u *CreateUsecase) Execute(ctx context.Context, params domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
	// Validate required fields.
	if params.InvoiceID == uuid.Nil {
		return nil, fmt.Errorf("invoice_id is required: %w", domain.ErrInvalidInput)
	}
	if params.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive: %w", domain.ErrInvalidInput)
	}
	if params.Currency == "" {
		return nil, fmt.Errorf("currency is required: %w", domain.ErrInvalidInput)
	}
	if params.PaymentMethod == "" {
		params.PaymentMethod = "bank_transfer"
	}

	// Verify the invoice exists and is in 'approved' status.
	inv, err := u.invRepo.GetByID(ctx, params.InvoiceID, params.TenantID)
	if err != nil {
		return nil, fmt.Errorf("CreateUsecase.Execute: get invoice: %w", err)
	}
	if inv.Status != domain.StatusApproved {
		return nil, fmt.Errorf("invoice is not approved (current: %s): %w", inv.Status, domain.ErrConflict)
	}

	po, err := u.repo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("CreateUsecase.Execute: create payment order: %w", err)
	}

	// Publish event — best-effort, DB write is not rolled back on failure.
	event := domain.PaymentOrderCreatedEvent{
		PaymentOrderID: po.ID.String(),
		TenantID:       po.TenantID.String(),
		InvoiceID:      po.InvoiceID.String(),
		Amount:         po.Amount.String(),
		Status:         string(po.Status),
	}
	if err := u.publisher.Publish(ctx, TopicPaymentOrderCreated, po.ID.String(), event); err != nil {
		log.Error().Err(err).Str("payment_order_id", po.ID.String()).Msg("CreateUsecase: failed to publish payment_order.created event")
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"payment_order_id": po.ID.String(),
		"new_status":       string(po.Status),
		"amount":           po.Amount.String(),
		"invoice_id":       po.InvoiceID.String(),
	})
	auditEntry := domain.AuditLogEntry{
		TenantID:   po.TenantID,
		EntityType: "payment_order",
		EntityID:   po.ID,
		Action:     "created",
		Details:    details,
	}
	if params.CreatedBy != nil {
		auditEntry.UserID = params.CreatedBy
	}
	auditErr := u.auditRepo.Create(ctx, auditEntry)
	if auditErr != nil {
		log.Error().Err(auditErr).Str("payment_order_id", po.ID.String()).Msg("CreateUsecase: failed to write audit log")
	}

	return po, nil
}
