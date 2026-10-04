// Package approval implements use-cases for the approval workflow.
// Status flow: pending -> approved | rejected.
// Every mutation writes an audit log entry.
package approval

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
)

// Repository defines the persistence port the use-case needs.
// It is a subset of domain.ApprovalRepository.
type Repository interface {
	CreateRequest(ctx context.Context, req *domain.ApprovalRequest) error
	ListRequests(ctx context.Context, tenantID uuid.UUID) ([]domain.ApprovalRequest, error)
	ListRequestsPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.ApprovalRequestListPage, error)
	GetRequestByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.ApprovalRequest, error)
	UpdateRequestStatus(ctx context.Context, req *domain.ApprovalRequest) error
}

// Usecase orchestrates approval business logic.
type Usecase struct {
	repo      Repository
	auditRepo domain.AuditLogRepository
}

// New creates an approval Usecase.
func New(repo Repository, auditRepo domain.AuditLogRepository) *Usecase {
	return &Usecase{repo: repo, auditRepo: auditRepo}
}

// List returns all approval requests for a tenant.
func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.ApprovalRequest, error) {
	return u.repo.ListRequests(ctx, tenantID)
}

// ListPaged returns a page of approval requests plus the total count for a tenant.
func (u *Usecase) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.ApprovalRequestListPage, error) {
	return u.repo.ListRequestsPaged(ctx, tenantID, page, perPage)
}

// Get returns a single approval request scoped to the tenant.
func (u *Usecase) Get(ctx context.Context, id, tenantID uuid.UUID) (*domain.ApprovalRequest, error) {
	return u.repo.GetRequestByID(ctx, id, tenantID)
}

// Approve transitions an approval request from "pending" (or "PENDING") to "approved".
// Returns ErrInvalidStatus if the request is not in a pending state.
func (u *Usecase) Approve(ctx context.Context, id, tenantID uuid.UUID, userID uuid.UUID) (*domain.ApprovalRequest, error) {
	ar, err := u.repo.GetRequestByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if ar.Status != "pending" && ar.Status != "PENDING" {
		return nil, domain.ErrInvalidStatus
	}

	approvedBy := userID.String()
	now := time.Now()
	ar.Status = "approved"
	ar.ApprovedBy = &approvedBy
	ar.ApprovedAt = &now

	if err := u.repo.UpdateRequestStatus(ctx, ar); err != nil {
		return nil, err
	}

	// Audit log -- best-effort.
	details, _ := json.Marshal(map[string]string{
		"approval_id": ar.ID,
		"old_status":  "pending",
		"new_status":  "approved",
		"target_type": ar.TargetType,
		"target_id":   ar.TargetID,
	})
	auditErr := u.auditRepo.Create(ctx, domain.AuditLogEntry{
		TenantID:   tenantID,
		UserID:     &userID,
		EntityType: "approval",
		EntityID:   id,
		Action:     "approved",
		Details:    details,
	})
	if auditErr != nil {
		log.Error().Err(auditErr).Str("approval_id", ar.ID).Msg("Approve: failed to write audit log")
	}

	return ar, nil
}

// Reject transitions an approval request from "pending" (or "PENDING") to "rejected".
// Returns ErrInvalidStatus if the request is not in a pending state.
func (u *Usecase) Reject(ctx context.Context, id, tenantID uuid.UUID, userID uuid.UUID, reason string) (*domain.ApprovalRequest, error) {
	ar, err := u.repo.GetRequestByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if ar.Status != "pending" && ar.Status != "PENDING" {
		return nil, domain.ErrInvalidStatus
	}

	rejectedBy := userID.String()
	now := time.Now()
	ar.Status = "rejected"
	ar.RejectedBy = &rejectedBy
	ar.RejectedAt = &now
	ar.RejectionReason = reason

	if err := u.repo.UpdateRequestStatus(ctx, ar); err != nil {
		return nil, err
	}

	// Audit log -- best-effort.
	details, _ := json.Marshal(map[string]string{
		"approval_id":      ar.ID,
		"old_status":       "pending",
		"new_status":       "rejected",
		"target_type":      ar.TargetType,
		"target_id":        ar.TargetID,
		"rejection_reason": reason,
	})
	auditErr := u.auditRepo.Create(ctx, domain.AuditLogEntry{
		TenantID:   tenantID,
		UserID:     &userID,
		EntityType: "approval",
		EntityID:   id,
		Action:     "rejected",
		Details:    details,
	})
	if auditErr != nil {
		log.Error().Err(auditErr).Str("approval_id", ar.ID).Msg("Reject: failed to write audit log")
	}

	return ar, nil
}

// CreateInvoiceApproval creates a pending approval request targeting an
// invoice. It is invoked by the invoice facade when a high-value invoice is
// created (amount >= the configured threshold).
func (u *Usecase) CreateInvoiceApproval(ctx context.Context, tenantID, invoiceID uuid.UUID, requestedBy uuid.UUID) (*domain.ApprovalRequest, error) {
	now := time.Now()
	approvalID := uuid.New()
	req := &domain.ApprovalRequest{
		ID:               approvalID.String(),
		TenantID:         tenantID.String(),
		TargetType:       "invoice",
		TargetID:         invoiceID.String(),
		Status:           "pending",
		CurrentStepIndex: 0,
		RequestedBy:      requestedBy.String(),
		CreatedAt:        now,
		UpdatedAt:        now,
		// WorkflowID is intentionally empty: auto-created requests have no
		// assigned workflow. The repository inserts NULL for empty values.
	}

	if err := u.repo.CreateRequest(ctx, req); err != nil {
		return nil, err
	}

	// Audit log — best-effort.
	details, _ := json.Marshal(map[string]string{
		"target_type": "invoice",
		"target_id":   invoiceID.String(),
		"requested_by": requestedBy.String(),
	})
	if auditErr := u.auditRepo.Create(ctx, domain.AuditLogEntry{
		TenantID:   tenantID,
		UserID:     &requestedBy,
		EntityType: "approval",
		EntityID:   approvalID,
		Action:     "created",
		Details:    details,
	}); auditErr != nil {
		log.Error().Err(auditErr).Str("approval_id", req.ID).Msg("CreateInvoiceApproval: failed to write audit log")
	}

	return req, nil
}
