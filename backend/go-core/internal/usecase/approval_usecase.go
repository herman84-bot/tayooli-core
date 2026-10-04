package usecase

import (
	"context"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/google/uuid"
	"time"
)

type ApprovalUsecase struct {
	repo domain.ApprovalRepository
}

func NewApprovalUsecase(repo domain.ApprovalRepository) *ApprovalUsecase {
	return &ApprovalUsecase{repo: repo}
}

func (u *ApprovalUsecase) CreateWorkflow(ctx context.Context, tenantID, name, steps string) (*domain.ApprovalWorkflow, error) {
	w := &domain.ApprovalWorkflow{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		Name:      name,
		Steps:     steps,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := u.repo.CreateWorkflow(ctx, w)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (u *ApprovalUsecase) CreateRequest(ctx context.Context, tenantID, workflowID, targetType, targetID, requestedBy string) (*domain.ApprovalRequest, error) {
	req := &domain.ApprovalRequest{
		ID:               uuid.NewString(),
		TenantID:         tenantID,
		WorkflowID:       workflowID,
		TargetType:       targetType,
		TargetID:         targetID,
		Status:           "PENDING",
		CurrentStepIndex: 0,
		RequestedBy:      requestedBy,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err := u.repo.CreateRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	return req, nil
}
