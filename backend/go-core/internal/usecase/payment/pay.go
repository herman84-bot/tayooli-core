package payment

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

const TopicPaymentOrderPaid = "payment_order.paid"

// PayUsecase transitions a payment order from approved to paid, marks the
// related invoice as paid, and publishes a payment_order.paid event.
type PayUsecase struct {
	repo      domain.PaymentOrderRepository
	publisher EventPublisher
	auditRepo domain.AuditLogRepository
}

func NewPayUsecase(repo domain.PaymentOrderRepository, publisher EventPublisher, auditRepo domain.AuditLogRepository) *PayUsecase {
	return &PayUsecase{repo: repo, publisher: publisher, auditRepo: auditRepo}
}

func (u *PayUsecase) Execute(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error) {
	// First get the payment order to return it and use its data for the event.
	po, err := u.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if po.Status != domain.PaymentOrderStatusApproved {
		return nil, domain.ErrConflict
	}

	// Mark both the payment order as paid and the invoice as paid in one tx.
	if err := u.repo.MarkInvoicePaid(ctx, id, tenantID); err != nil {
		return nil, err
	}

	// Re-fetch the updated payment order to return accurate state.
	updatedPO, err := u.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Publish event — best-effort.
	event := domain.PaymentOrderPaidEvent{
		PaymentOrderID: updatedPO.ID.String(),
		TenantID:       updatedPO.TenantID.String(),
		InvoiceID:      updatedPO.InvoiceID.String(),
		Amount:         updatedPO.Amount.String(),
	}
	if updatedPO.PaidAt != nil {
		event.PaidAt = updatedPO.PaidAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if err := u.publisher.Publish(ctx, TopicPaymentOrderPaid, updatedPO.ID.String(), event); err != nil {
		log.Error().Err(err).Str("payment_order_id", updatedPO.ID.String()).Msg("PayUsecase: failed to publish payment_order.paid event")
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"payment_order_id": updatedPO.ID.String(),
		"old_status":       string(domain.PaymentOrderStatusApproved),
		"new_status":       string(domain.PaymentOrderStatusPaid),
		"amount":           updatedPO.Amount.String(),
		"invoice_id":       updatedPO.InvoiceID.String(),
	})
	auditErr := u.auditRepo.Create(ctx, domain.AuditLogEntry{
		TenantID:   tenantID,
		EntityType: "payment_order",
		EntityID:   updatedPO.ID,
		Action:     "paid",
		Details:    details,
	})
	if auditErr != nil {
		log.Error().Err(auditErr).Str("payment_order_id", updatedPO.ID.String()).Msg("PayUsecase: failed to write audit log")
	}

	return updatedPO, nil
}
