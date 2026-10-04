package payment

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

const TopicPaymentOrderRejected = "payment_order.rejected"

// RejectUsecase transitions a payment order to the rejected state from draft
// and publishes a payment_order.rejected event.
type RejectUsecase struct {
	repo      domain.PaymentOrderRepository
	publisher EventPublisher
	auditRepo domain.AuditLogRepository
}

func NewRejectUsecase(repo domain.PaymentOrderRepository, publisher EventPublisher, auditRepo domain.AuditLogRepository) *RejectUsecase {
	return &RejectUsecase{repo: repo, publisher: publisher, auditRepo: auditRepo}
}

// Execute transitions a payment order from draft to rejected.
// rejectedBy records who performed the rejection for audit trail.
func (u *RejectUsecase) Execute(ctx context.Context, id, tenantID, rejectedBy uuid.UUID) (*domain.PaymentOrder, error) {
	po, err := u.repo.UpdateStatus(ctx, id, tenantID, domain.PaymentOrderStatusDraft, domain.PaymentOrderStatusRejected, nil)
	if err != nil {
		return nil, err
	}

	// Publish event — best-effort.
	event := domain.PaymentOrderRejectedEvent{
		PaymentOrderID: po.ID.String(),
		TenantID:       po.TenantID.String(),
		InvoiceID:      po.InvoiceID.String(),
		Amount:         po.Amount.String(),
	}
	if err := u.publisher.Publish(ctx, TopicPaymentOrderRejected, po.ID.String(), event); err != nil {
		log.Error().Err(err).Str("payment_order_id", po.ID.String()).Msg("RejectUsecase: failed to publish payment_order.rejected event")
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"payment_order_id": po.ID.String(),
		"old_status":       string(domain.PaymentOrderStatusDraft),
		"new_status":       string(domain.PaymentOrderStatusRejected),
		"amount":           po.Amount.String(),
		"invoice_id":       po.InvoiceID.String(),
	})
	auditErr := u.auditRepo.Create(ctx, domain.AuditLogEntry{
		TenantID:   tenantID,
		UserID:     &rejectedBy,
		EntityType: "payment_order",
		EntityID:   po.ID,
		Action:     "rejected",
		Details:    details,
	})
	if auditErr != nil {
		log.Error().Err(auditErr).Str("payment_order_id", po.ID.String()).Msg("RejectUsecase: failed to write audit log")
	}

	return po, nil
}
