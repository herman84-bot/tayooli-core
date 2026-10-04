package match

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type InvoiceFetcher interface {
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, topic string, key string, payload any) error
}

type Usecase struct {
	invoiceRepo  InvoiceFetcher
	matchRepo    domain.MatchRepository
	publisher    EventPublisher
	tolerancePct decimal.Decimal
}

func NewUsecase(invoiceRepo InvoiceFetcher, matchRepo domain.MatchRepository, publisher EventPublisher, tolerancePct decimal.Decimal) *Usecase {
	return &Usecase{
		invoiceRepo:  invoiceRepo,
		matchRepo:    matchRepo,
		publisher:    publisher,
		tolerancePct: tolerancePct,
	}
}

func (u *Usecase) Execute(ctx context.Context, invoiceID, tenantID uuid.UUID) error {
	inv, err := u.invoiceRepo.GetByID(ctx, invoiceID, tenantID)
	if err != nil {
		return fmt.Errorf("MatchUsecase.Execute: get invoice: %w", err)
	}

	if inv.POID == nil {
		return u.escalate(ctx, inv, domain.MatchResultNoPO, "no purchase order linked to invoice")
	}

	po, err := u.matchRepo.GetPOByID(ctx, *inv.POID, tenantID)
	if err != nil {
		return u.escalate(ctx, inv, domain.MatchResultNoPO, "purchase order not found")
	}

	gr, err := u.matchRepo.GetGRByPOID(ctx, po.ID, tenantID)
	if err != nil {
		return u.escalate(ctx, inv, domain.MatchResultNoGR, "goods receipt not found")
	}

	if po.Amount.IsZero() {
		return u.escalate(ctx, inv, domain.MatchResultAmountMismatch, "purchase order has zero amount")
	}

	if gr.ReceivedQty != po.Qty {
		return u.escalate(ctx, inv, domain.MatchResultQtyMismatch,
			fmt.Sprintf("qty mismatch: po=%d received=%d", po.Qty, gr.ReceivedQty))
	}

	// |inv.amount - po.amount| / po.amount * 100 <= tolerancePct
	hundred := decimal.NewFromInt(100)
	diff := inv.Amount.Sub(po.Amount).Abs()
	pctDiff := diff.Div(po.Amount).Mul(hundred)
	if pctDiff.GreaterThan(u.tolerancePct) {
		return u.escalate(ctx, inv, domain.MatchResultAmountMismatch,
			fmt.Sprintf("amount mismatch: po=%s invoice=%s diff=%s%%",
				po.Amount.String(), inv.Amount.String(), pctDiff.StringFixed(4)))
	}

	return u.approve(ctx, inv)
}

func (u *Usecase) approve(ctx context.Context, inv *domain.Invoice) error {
	if err := u.matchRepo.UpdateInvoiceMatchResult(ctx, inv.ID, inv.TenantID, inv.POID,
		domain.MatchResultMatched, domain.StatusApproved); err != nil {
		return fmt.Errorf("MatchUsecase.approve: update: %w", err)
	}
	event := domain.InvoiceApprovedEvent{
		InvoiceID:   inv.ID.String(),
		TenantID:    inv.TenantID.String(),
		Amount:      inv.Amount.String(),
		MatchResult: string(domain.MatchResultMatched),
	}
	if err := u.publisher.Publish(ctx, "invoice.approved", inv.ID.String(), event); err != nil {
		return fmt.Errorf("MatchUsecase.approve: publish: %w", err)
	}
	return nil
}

func (u *Usecase) escalate(ctx context.Context, inv *domain.Invoice, result domain.MatchResult, reason string) error {
	if err := u.matchRepo.UpdateInvoiceMatchResult(ctx, inv.ID, inv.TenantID, inv.POID,
		result, domain.StatusPendingReview); err != nil {
		return fmt.Errorf("MatchUsecase.escalate: update: %w", err)
	}
	event := domain.InvoicePendingReviewEvent{
		InvoiceID:   inv.ID.String(),
		TenantID:    inv.TenantID.String(),
		Amount:      inv.Amount.String(),
		MatchResult: string(result),
		Reason:      reason,
	}
	if err := u.publisher.Publish(ctx, "invoice.pending_review", inv.ID.String(), event); err != nil {
		return fmt.Errorf("MatchUsecase.escalate: publish: %w", err)
	}
	return nil
}
