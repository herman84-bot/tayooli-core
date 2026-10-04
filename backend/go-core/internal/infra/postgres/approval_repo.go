package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type approvalRepository struct {
	db *sql.DB
}

func NewApprovalRepository(db *sql.DB) domain.ApprovalRepository {
	return &approvalRepository{db: db}
}

func (r *approvalRepository) CreateWorkflow(ctx context.Context, w *domain.ApprovalWorkflow) error {
	tenantID, err := uuid.Parse(w.TenantID)
	if err != nil {
		return fmt.Errorf("approvalRepository.CreateWorkflow: parse tenant id: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("approvalRepository.CreateWorkflow: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("approvalRepository.CreateWorkflow: set tenant: %w", err)
	}

	query := `INSERT INTO approval_workflows (id, tenant_id, name, steps, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err := tx.ExecContext(ctx, query, w.ID, w.TenantID, w.Name, w.Steps, w.CreatedAt, w.UpdatedAt); err != nil {
		return fmt.Errorf("approvalRepository.CreateWorkflow: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("approvalRepository.CreateWorkflow: commit: %w", err)
	}
	return nil
}

func (r *approvalRepository) GetWorkflow(ctx context.Context, id string, tenantID uuid.UUID) (*domain.ApprovalWorkflow, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.GetWorkflow: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("approvalRepository.GetWorkflow: set tenant: %w", err)
	}

	query := `SELECT id, tenant_id, name, steps, created_at, updated_at FROM approval_workflows WHERE id = $1 AND tenant_id = $2`
	w := &domain.ApprovalWorkflow{}
	err = tx.QueryRowContext(ctx, query, id, tenantID).Scan(&w.ID, &w.TenantID, &w.Name, &w.Steps, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.GetWorkflow: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("approvalRepository.GetWorkflow: commit: %w", err)
	}
	return w, nil
}

func (r *approvalRepository) CreateRequest(ctx context.Context, req *domain.ApprovalRequest) error {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return fmt.Errorf("approvalRepository.CreateRequest: parse tenant id: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("approvalRepository.CreateRequest: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("approvalRepository.CreateRequest: set tenant: %w", err)
	}

	// Empty workflow_id is inserted as NULL: auto-created approval requests
	// (e.g. high-value invoice approvals) have no assigned workflow.
	var workflowID any
	if req.WorkflowID == "" {
		workflowID = nil
	} else {
		workflowID = req.WorkflowID
	}

	query := `INSERT INTO approval_requests (id, tenant_id, workflow_id, target_type, target_id, status, current_step_index, requested_by, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	if _, err := tx.ExecContext(ctx, query, req.ID, req.TenantID, workflowID, req.TargetType, req.TargetID, req.Status, req.CurrentStepIndex, req.RequestedBy, req.CreatedAt, req.UpdatedAt); err != nil {
		return fmt.Errorf("approvalRepository.CreateRequest: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("approvalRepository.CreateRequest: commit: %w", err)
	}
	return nil
}

func (r *approvalRepository) UpdateRequest(ctx context.Context, req *domain.ApprovalRequest) error {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequest: parse tenant id: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequest: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequest: set tenant: %w", err)
	}

	query := `UPDATE approval_requests SET status = $1, current_step_index = $2, updated_at = $3 WHERE id = $4 AND tenant_id = $5`
	if _, err := tx.ExecContext(ctx, query, req.Status, req.CurrentStepIndex, req.UpdatedAt, req.ID, req.TenantID); err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequest: exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequest: commit: %w", err)
	}
	return nil
}

func (r *approvalRepository) GetRequest(ctx context.Context, id string, tenantID uuid.UUID) (*domain.ApprovalRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.GetRequest: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("approvalRepository.GetRequest: set tenant: %w", err)
	}

	query := `SELECT id, tenant_id, workflow_id, target_type, target_id, status, current_step_index, requested_by, created_at, updated_at FROM approval_requests WHERE id = $1 AND tenant_id = $2`
	req := &domain.ApprovalRequest{}
	err = tx.QueryRowContext(ctx, query, id, tenantID).Scan(&req.ID, &req.TenantID, &req.WorkflowID, &req.TargetType, &req.TargetID, &req.Status, &req.CurrentStepIndex, &req.RequestedBy, &req.CreatedAt, &req.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.GetRequest: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("approvalRepository.GetRequest: commit: %w", err)
	}
	return req, nil
}

// ── ListRequests ────────────────────────────────────────────────────────────

const listApprovalRequestsByTenant = `
SELECT id, tenant_id, workflow_id, target_type, target_id, status,
       current_step_index, requested_by, approved_by, approved_at,
       rejected_by, rejected_at, rejection_reason, created_at, updated_at
FROM approval_requests
WHERE tenant_id = $1
ORDER BY created_at DESC`

func (r *approvalRepository) ListRequests(ctx context.Context, tenantID uuid.UUID) ([]domain.ApprovalRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequests: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequests: set tenant: %w", err)
	}

	rows, err := tx.QueryContext(ctx, listApprovalRequestsByTenant, tenantID)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequests: query: %w", err)
	}
	defer rows.Close()

	var results []domain.ApprovalRequest
	for rows.Next() {
		ar, err := scanApprovalRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("approvalRepository.ListRequests: scan: %w", err)
		}
		results = append(results, *ar)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequests: rows err: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequests: close rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequests: commit: %w", err)
	}
	return results, nil
}

const countApprovalRequestsByTenant = `SELECT COUNT(*) FROM approval_requests WHERE tenant_id = $1`

const listApprovalRequestsByTenantPaged = `
SELECT id, tenant_id, workflow_id, target_type, target_id, status,
       current_step_index, requested_by, approved_by, approved_at,
       rejected_by, rejected_at, rejection_reason, created_at, updated_at
FROM approval_requests
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

// ListRequestsPaged returns a page of approval requests plus the total count for the tenant.
func (r *approvalRepository) ListRequestsPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.ApprovalRequestListPage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequestsPaged: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequestsPaged: set tenant: %w", err)
	}

	// Count total rows for this tenant.
	var total int
	if err := tx.QueryRowContext(ctx, countApprovalRequestsByTenant, tenantID).Scan(&total); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequestsPaged: count: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := tx.QueryContext(ctx, listApprovalRequestsByTenantPaged, tenantID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequestsPaged: query: %w", err)
	}
	defer rows.Close()

	var results []domain.ApprovalRequest
	for rows.Next() {
		ar, err := scanApprovalRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("approvalRepository.ListRequestsPaged: scan row: %w", err)
		}
		results = append(results, *ar)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequestsPaged: rows err: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequestsPaged: close rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("approvalRepository.ListRequestsPaged: commit: %w", err)
	}

	return &domain.ApprovalRequestListPage{
		Data:    results,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

// ── GetRequestByID ──────────────────────────────────────────────────────────

const getApprovalRequestByID = `
SELECT id, tenant_id, workflow_id, target_type, target_id, status,
       current_step_index, requested_by, approved_by, approved_at,
       rejected_by, rejected_at, rejection_reason, created_at, updated_at
FROM approval_requests
WHERE id = $1 AND tenant_id = $2`

func (r *approvalRepository) GetRequestByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.ApprovalRequest, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.GetRequestByID: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("approvalRepository.GetRequestByID: set tenant: %w", err)
	}

	row := tx.QueryRowContext(ctx, getApprovalRequestByID, id, tenantID)
	ar, err := scanApprovalRequestRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("approvalRepository.GetRequestByID: scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("approvalRepository.GetRequestByID: commit: %w", err)
	}
	return ar, nil
}

// ── UpdateRequestStatus ─────────────────────────────────────────────────────

const updateApprovalRequestStatus = `
UPDATE approval_requests
SET status = $1, approved_by = $2, approved_at = $3,
    rejected_by = $4, rejected_at = $5, rejection_reason = $6, updated_at = NOW()
WHERE id = $7 AND tenant_id = $8`

func (r *approvalRepository) UpdateRequestStatus(ctx context.Context, req *domain.ApprovalRequest) error {
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequestStatus: parse tenant id: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequestStatus: begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequestStatus: set tenant: %w", err)
	}

	res, err := tx.ExecContext(ctx, updateApprovalRequestStatus,
		req.Status,
		req.ApprovedBy,
		req.ApprovedAt,
		req.RejectedBy,
		req.RejectedAt,
		req.RejectionReason,
		req.ID,
		req.TenantID,
	)
	if err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequestStatus: exec: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequestStatus: rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("approvalRepository.UpdateRequestStatus: commit: %w", err)
	}
	return nil
}

// ── Scan helpers ────────────────────────────────────────────────────────────

func scanApprovalRequest(s *sql.Rows) (*domain.ApprovalRequest, error) {
	var ar domain.ApprovalRequest
	var workflowID, targetID sql.NullString
	var approvedBy, rejectedBy sql.NullString
	var approvedAt, rejectedAt sql.NullTime
	var rejectionReason sql.NullString

	err := s.Scan(
		&ar.ID, &ar.TenantID, &workflowID, &ar.TargetType, &targetID,
		&ar.Status, &ar.CurrentStepIndex, &ar.RequestedBy,
		&approvedBy, &approvedAt, &rejectedBy, &rejectedAt, &rejectionReason,
		&ar.CreatedAt, &ar.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if workflowID.Valid {
		ar.WorkflowID = workflowID.String
	}
	if targetID.Valid {
		ar.TargetID = targetID.String
	}
	if approvedBy.Valid {
		ar.ApprovedBy = &approvedBy.String
	}
	if approvedAt.Valid {
		t := approvedAt.Time
		ar.ApprovedAt = &t
	}
	if rejectedBy.Valid {
		ar.RejectedBy = &rejectedBy.String
	}
	if rejectedAt.Valid {
		t := rejectedAt.Time
		ar.RejectedAt = &t
	}
	if rejectionReason.Valid {
		ar.RejectionReason = rejectionReason.String
	}

	return &ar, nil
}

func scanApprovalRequestRow(row *sql.Row) (*domain.ApprovalRequest, error) {
	var ar domain.ApprovalRequest
	var workflowID, targetID sql.NullString
	var approvedBy, rejectedBy sql.NullString
	var approvedAt, rejectedAt sql.NullTime
	var rejectionReason sql.NullString

	err := row.Scan(
		&ar.ID, &ar.TenantID, &workflowID, &ar.TargetType, &targetID,
		&ar.Status, &ar.CurrentStepIndex, &ar.RequestedBy,
		&approvedBy, &approvedAt, &rejectedBy, &rejectedAt, &rejectionReason,
		&ar.CreatedAt, &ar.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if workflowID.Valid {
		ar.WorkflowID = workflowID.String
	}
	if targetID.Valid {
		ar.TargetID = targetID.String
	}
	if approvedBy.Valid {
		ar.ApprovedBy = &approvedBy.String
	}
	if approvedAt.Valid {
		t := approvedAt.Time
		ar.ApprovedAt = &t
	}
	if rejectedBy.Valid {
		ar.RejectedBy = &rejectedBy.String
	}
	if rejectedAt.Valid {
		t := rejectedAt.Time
		ar.RejectedAt = &t
	}
	if rejectionReason.Valid {
		ar.RejectionReason = rejectionReason.String
	}

	return &ar, nil
}
