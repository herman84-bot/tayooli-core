package invoice

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

const TopicInvoiceCreated = "invoice.created"

// EventPublisher is the port for publishing domain events.
// key must be a stable entity identifier (e.g. invoice UUID) to ensure
// related events land on the same Kafka partition for ordered consumption.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, key string, payload any) error
}

// CreateUsecase persists a new invoice and publishes an invoice.created event.
type CreateUsecase struct {
	repo      domain.InvoiceRepository
	publisher EventPublisher
}

func NewCreateUsecase(repo domain.InvoiceRepository, publisher EventPublisher) *CreateUsecase {
	return &CreateUsecase{repo: repo, publisher: publisher}
}

func (u *CreateUsecase) Execute(ctx context.Context, params domain.CreateInvoiceParams) (*domain.Invoice, error) {
	if params.VendorID == "" || params.InvoiceNumber == "" || params.Amount.LessThanOrEqual(decimal.Zero) || params.Currency == "" {
		return nil, domain.ErrInvalidInput
	}

	// MEDIUM-1: guard against Postgres NUMERIC(20,4) overflow.
	maxAmount, _ := decimal.NewFromString("9999999999999999.9999")
	if params.Amount.GreaterThan(maxAmount) {
		return nil, fmt.Errorf("amount exceeds maximum: %w", domain.ErrInvalidInput)
	}

	// MEDIUM-2: guard against silent rounding — NUMERIC(20,4) truncates sub-4-dp values,
	// which can produce a 0.0000 record that passes the > 0 check above.
	if params.Amount.Exponent() < -4 {
		return nil, fmt.Errorf("amount precision exceeds 4 decimal places: %w", domain.ErrInvalidInput)
	}

	inv, err := u.repo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("CreateUsecase.Execute: create invoice: %w", err)
	}
	// Publish a dedicated event struct so Python consumers receive stable
	// snake_case field names regardless of how the domain Invoice struct evolves.
	// Best-effort: failure to publish must not roll back the DB write.
	// Key is the invoice UUID so all events for the same invoice are ordered
	// within a single Kafka partition.
	// Amount is serialized as string to prevent float truncation in transit.
	event := domain.InvoiceCreatedEvent{
		InvoiceID: inv.ID.String(),
		TenantID:  inv.TenantID.String(),
		Amount:    inv.Amount.String(),
		Status:    string(inv.Status),
	}
	_ = u.publisher.Publish(ctx, TopicInvoiceCreated, inv.ID.String(), event)
	return inv, nil
}
