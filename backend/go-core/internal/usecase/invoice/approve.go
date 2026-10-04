package invoice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

const TopicInvoiceApproved = "invoice.approved"

// ApproveUsecase transitions an invoice to the approved state and publishes
// an invoice.approved event. The tenant_id from the JWT claim is enforced
// so a tenant can only approve their own invoices.
type ApproveUsecase struct {
	repo      domain.InvoiceRepository
	publisher EventPublisher
}

func NewApproveUsecase(repo domain.InvoiceRepository, publisher EventPublisher) *ApproveUsecase {
	return &ApproveUsecase{repo: repo, publisher: publisher}
}

func (u *ApproveUsecase) Execute(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	inv, err := u.repo.Approve(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("ApproveUsecase.Execute: %w", err)
	}
	// Key is the invoice UUID so approval events for the same invoice are
	// ordered within the same Kafka partition as the invoice.created event.
	if err := u.publisher.Publish(ctx, TopicInvoiceApproved, inv.ID.String(), inv); err != nil {
		log.Error().Err(err).Str("invoice_id", inv.ID.String()).Msg("ApproveUsecase: failed to publish invoice.approved event")
	}
	return inv, nil
}
