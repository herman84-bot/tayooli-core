package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

const TopicPaymentOrderApproved = "payment_order.approved"

// ApproveUsecase transitions a payment order from draft to approved and
// publishes a payment_order.approved event.
type ApproveUsecase struct {
	repo      domain.PaymentOrderRepository
	publisher EventPublisher
	auditRepo domain.AuditLogRepository
}

func NewApproveUsecase(repo domain.PaymentOrderRepository, publisher EventPublisher, auditRepo domain.AuditLogRepository) *ApproveUsecase {
	return &ApproveUsecase{repo: repo, publisher: publisher, auditRepo: auditRepo}
}

// Execute transitions a payment order from draft to approved.
// Only users with role "cfo" or "admin" are authorized.
func (u *ApproveUsecase) Execute(ctx context.Context, id, tenantID, approvedBy uuid.UUID, role string) (*domain.PaymentOrder, error) {
	// Enforce role at usecase layer.
	if role != "cfo" && role != "admin" {
		return nil, fmt.Errorf("role %q is not authorized to approve payment orders: %w", role, domain.ErrForbidden)
	}

	po, err := u.repo.UpdateStatus(ctx, id, tenantID, domain.PaymentOrderStatusDraft, domain.PaymentOrderStatusApproved, &approvedBy)
	if err != nil {
		return nil, fmt.Errorf("ApproveUsecase.Execute: %w", err)
	}

	// Publish event — best-effort.
	event := domain.PaymentOrderApprovedEvent{
		PaymentOrderID: po.ID.String(),
		TenantID:       po.TenantID.String(),
		InvoiceID:      po.InvoiceID.String(),
		Amount:         po.Amount.String(),
		ApprovedBy:     approvedBy.String(),
	}
	if err := u.publisher.Publish(ctx, TopicPaymentOrderApproved, po.ID.String(), event); err != nil {
		log.Error().Err(err).Str("payment_order_id", po.ID.String()).Msg("ApproveUsecase: failed to publish payment_order.approved event")
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"payment_order_id": po.ID.String(),
		"old_status":       string(domain.PaymentOrderStatusDraft),
		"new_status":       string(domain.PaymentOrderStatusApproved),
		"amount":           po.Amount.String(),
		"invoice_id":       po.InvoiceID.String(),
	})
	auditErr := u.auditRepo.Create(ctx, domain.AuditLogEntry{
		TenantID:   tenantID,
		UserID:     &approvedBy,
		EntityType: "payment_order",
		EntityID:   po.ID,
		Action:     "approved",
		Details:    details,
	})
	if auditErr != nil {
		log.Error().Err(auditErr).Str("payment_order_id", po.ID.String()).Msg("ApproveUsecase: failed to write audit log")
	}

	return po, nil
}
