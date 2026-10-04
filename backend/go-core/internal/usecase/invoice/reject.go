package invoice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

const TopicInvoiceRejected = "invoice.rejected"

// RejectUsecase transitions an invoice to the rejected state and publishes
// an invoice.rejected event. Allowed source statuses are 'pending' and
// 'pending_review'. The tenant_id from the JWT claim is enforced so a
// tenant can only reject their own invoices.
type RejectUsecase struct {
	repo      domain.InvoiceRepository
	publisher EventPublisher
}

func NewRejectUsecase(repo domain.InvoiceRepository, publisher EventPublisher) *RejectUsecase {
	return &RejectUsecase{repo: repo, publisher: publisher}
}

func (u *RejectUsecase) Execute(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	inv, err := u.repo.Reject(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("RejectUsecase.Execute: %w", err)
	}
	// Publish event (non-fatal): DB is source of truth.
	event := domain.InvoiceRejectedEvent{
		InvoiceID: inv.ID.String(),
		TenantID:  inv.TenantID.String(),
		Amount:    inv.Amount.String(),
	}
	if err := u.publisher.Publish(ctx, TopicInvoiceRejected, inv.ID.String(), event); err != nil {
		log.Error().Err(err).Str("invoice_id", inv.ID.String()).Msg("RejectUsecase: failed to publish invoice.rejected event")
	}
	return inv, nil
}
