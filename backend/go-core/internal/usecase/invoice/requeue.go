package invoice

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// RequeueUsecase re-queues a pending invoice by re-publishing the InvoiceCreatedEvent
// to the invoice.created Kafka topic. It does NOT change the invoice status —
// it remains "pending" so the OCR worker can pick it up again.
type RequeueUsecase struct {
	repo      domain.InvoiceRepository
	publisher EventPublisher
}

func NewRequeueUsecase(repo domain.InvoiceRepository, publisher EventPublisher) *RequeueUsecase {
	return &RequeueUsecase{repo: repo, publisher: publisher}
}

func (uc *RequeueUsecase) Execute(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	inv, err := uc.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err // domain.ErrNotFound propagates
	}
	if inv.Status != domain.StatusPending {
		return nil, domain.ErrConflict // "invoice is not in pending status"
	}

	event := domain.InvoiceCreatedEvent{
		InvoiceID:        inv.ID.String(),
		TenantID:         inv.TenantID.String(),
		Amount:           inv.Amount.String(),
		Status:           string(domain.StatusPending),
		RawDocumentBytes: "", // not available on requeue; worker will re-fetch if needed
		MimeType:         "",
	}

	if err := uc.publisher.Publish(ctx, TopicInvoiceCreated, inv.ID.String(), event); err != nil {
		// Non-fatal: DB is source of truth. Log and continue.
		log.Printf("requeue: kafka publish failed for invoice %s: %v", inv.ID, err)
	}
	return inv, nil
}