package invoice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// AutoApproveInput represents the input for auto-approval check.
type AutoApproveInput struct {
	InvoiceID uuid.UUID `validate:"required,uuid"`
	TenantID  uuid.UUID `validate:"required,uuid"`
}

// AutoApproveResult represents the result of an auto-approval check.
type AutoApproveResult struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

// ErrAutoApproveNotEligible is returned when the invoice is not eligible for auto-approval.
var ErrAutoApproveNotEligible = errors.New("invoice is not eligible for auto-approval")

// AutoApproveUsecase checks if an invoice qualifies for auto-approval
// (ai_confidence_score >= 0.95) and approves it if eligible.
type AutoApproveUsecase struct {
	repo     domain.InvoiceRepository
	approve  *ApproveUsecase
	validate *validator.Validate
}

func NewAutoApproveUsecase(repo domain.InvoiceRepository, approve *ApproveUsecase) *AutoApproveUsecase {
	return &AutoApproveUsecase{
		repo:     repo,
		approve:  approve,
		validate: validator.New(),
	}
}

func (u *AutoApproveUsecase) Execute(ctx context.Context, input AutoApproveInput) (*AutoApproveResult, error) {
	// Validate input
	if err := u.validate.Struct(input); err != nil {
		return nil, fmt.Errorf("autoApproveUsecase.Execute: validation: %w", err)
	}

	// Apply 5s timeout for DB operations
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Load invoice with tenant isolation
	inv, err := u.repo.GetByID(ctx, input.InvoiceID, input.TenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return &AutoApproveResult{
				Approved: false,
				Reason:   "invoice not found",
			}, nil
		}
		return nil, fmt.Errorf("autoApproveUsecase.Execute: get invoice: %w", err)
	}

	// Check if invoice is already approved or rejected
	if inv.Status == domain.StatusApproved {
		return &AutoApproveResult{
			Approved: true,
			Reason:   "invoice already approved",
		}, nil
	}
	if inv.Status == domain.StatusRejected {
		return &AutoApproveResult{
			Approved: false,
			Reason:   "invoice already rejected",
		}, nil
	}

	// Check if invoice is in a state that can be auto-approved
	// Only pending or pending_review invoices with high confidence can be auto-approved
	if inv.Status != domain.StatusPending && inv.Status != domain.StatusPendingReview {
		return &AutoApproveResult{
			Approved: false,
			Reason:   fmt.Sprintf("invoice status %q is not eligible for auto-approval", inv.Status),
		}, nil
	}

	// Check AI confidence score
	const autoApproveThreshold = 0.95
	if inv.AIConfidenceScore < autoApproveThreshold {
		return &AutoApproveResult{
			Approved: false,
			Reason:   fmt.Sprintf("AI confidence score %.2f is below auto-approve threshold %.2f", inv.AIConfidenceScore, autoApproveThreshold),
		}, nil
	}

	// Check if invoice has OCR processed status (indicates AI has processed it)
	if inv.Status == domain.StatusPending && inv.AIConfidenceScore > 0 {
		// Invoice is pending but has AI score - check if it was processed
		// This is valid for auto-approval
	}

	// Approve the invoice using existing approval logic
	approvedInv, err := u.approve.Execute(ctx, input.InvoiceID, input.TenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return &AutoApproveResult{
				Approved: false,
				Reason:   "invoice not found",
			}, nil
		}
		if errors.Is(err, domain.ErrConflict) {
			return &AutoApproveResult{
				Approved: false,
				Reason:   "invoice is not in a state that can be approved",
			}, nil
		}
		return nil, fmt.Errorf("autoApproveUsecase.Execute: approve: %w", err)
	}

	return &AutoApproveResult{
		Approved: true,
		Reason:   fmt.Sprintf("Auto-approved: AI confidence score %.2f >= %.2f", approvedInv.AIConfidenceScore, autoApproveThreshold),
	}, nil
}