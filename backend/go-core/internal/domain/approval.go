package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ApprovalWorkflow struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Steps     string    `json:"steps"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ApprovalRequestListPage holds a paginated result set plus total count.
type ApprovalRequestListPage struct {
	Data    []ApprovalRequest `json:"data"`
	Total   int               `json:"total"`
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
}

type ApprovalRequest struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	WorkflowID       string     `json:"workflow_id"`
	TargetType       string     `json:"target_type"`
	TargetID         string     `json:"target_id"`
	Status           string     `json:"status"`
	CurrentStepIndex int        `json:"current_step_index"`
	RequestedBy      string     `json:"requested_by"`
	ApprovedBy       *string    `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	RejectedBy       *string    `json:"rejected_by,omitempty"`
	RejectedAt       *time.Time `json:"rejected_at,omitempty"`
	RejectionReason  string     `json:"rejection_reason,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ApprovalRepository interface {
	CreateWorkflow(ctx context.Context, workflow *ApprovalWorkflow) error
	GetWorkflow(ctx context.Context, id string, tenantID uuid.UUID) (*ApprovalWorkflow, error)
	CreateRequest(ctx context.Context, req *ApprovalRequest) error
	UpdateRequest(ctx context.Context, req *ApprovalRequest) error
	GetRequest(ctx context.Context, id string, tenantID uuid.UUID) (*ApprovalRequest, error)
	// ListRequests returns all approval requests for a tenant.
	ListRequests(ctx context.Context, tenantID uuid.UUID) ([]ApprovalRequest, error)
	// ListRequestsPaged returns a page of approval requests plus the total count for the tenant.
	ListRequestsPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*ApprovalRequestListPage, error)
	// GetRequestByID returns a single approval request by id scoped to a tenant.
	GetRequestByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*ApprovalRequest, error)
	// UpdateRequestStatus updates an existing approval request (approve/reject).
	UpdateRequestStatus(ctx context.Context, req *ApprovalRequest) error
}
