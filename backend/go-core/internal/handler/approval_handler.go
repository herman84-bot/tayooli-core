package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// approvalUsecase groups the operations the handler needs.
type approvalUsecase interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.ApprovalRequest, error)
	ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.ApprovalRequestListPage, error)
	Get(ctx context.Context, id, tenantID uuid.UUID) (*domain.ApprovalRequest, error)
	Approve(ctx context.Context, id, tenantID, userID uuid.UUID) (*domain.ApprovalRequest, error)
	Reject(ctx context.Context, id, tenantID, userID uuid.UUID, reason string) (*domain.ApprovalRequest, error)
}

// ApprovalHandler handles HTTP requests for approval resources.
type ApprovalHandler struct {
	uc approvalUsecase
}

// NewApprovalHandler creates an ApprovalHandler.
func NewApprovalHandler(uc approvalUsecase) *ApprovalHandler {
	return &ApprovalHandler{uc: uc}
}

// approvalView is the JSON representation returned to clients.
type approvalView struct {
	ID               string  `json:"id"`
	TenantID         string  `json:"tenant_id"`
	WorkflowID       string  `json:"workflow_id"`
	TargetType       string  `json:"target_type"`
	TargetID         string  `json:"target_id"`
	Status           string  `json:"status"`
	CurrentStepIndex int     `json:"current_step_index"`
	RequestedBy      string  `json:"requested_by"`
	ApprovedBy       *string `json:"approved_by,omitempty"`
	ApprovedAt       *string `json:"approved_at,omitempty"`
	RejectedBy       *string `json:"rejected_by,omitempty"`
	RejectedAt       *string `json:"rejected_at,omitempty"`
	RejectionReason  string  `json:"rejection_reason,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func toApprovalView(ar *domain.ApprovalRequest) approvalView {
	v := approvalView{
		ID:               ar.ID,
		TenantID:         ar.TenantID,
		WorkflowID:       ar.WorkflowID,
		TargetType:       ar.TargetType,
		TargetID:         ar.TargetID,
		Status:           ar.Status,
		CurrentStepIndex: ar.CurrentStepIndex,
		RequestedBy:      ar.RequestedBy,
		RejectionReason:  ar.RejectionReason,
		CreatedAt:        ar.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        ar.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if ar.ApprovedBy != nil {
		v.ApprovedBy = ar.ApprovedBy
	}
	if ar.ApprovedAt != nil {
		s := ar.ApprovedAt.Format("2006-01-02T15:04:05Z07:00")
		v.ApprovedAt = &s
	}
	if ar.RejectedBy != nil {
		v.RejectedBy = ar.RejectedBy
	}
	if ar.RejectedAt != nil {
		s := ar.RejectedAt.Format("2006-01-02T15:04:05Z07:00")
		v.RejectedAt = &s
	}
	return v
}

// ── ListApprovals ───────────────────────────────────────────────────────────

// ListApprovals godoc
// GET /api/v1/approvals
func (h *ApprovalHandler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination params with defaults (mirrors GET /invoices).
	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			perPage = n
		}
	}
	if perPage > 100 {
		perPage = 100
	}

	result, err := h.uc.ListPaged(r.Context(), tenantID, page, perPage)
	if err != nil {
		log.Error().Err(err).Str("handler", "ListApprovals").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	type paginatedResponse struct {
		Data    []approvalView `json:"data"`
		Total   int            `json:"total"`
		Page    int            `json:"page"`
		PerPage int            `json:"per_page"`
	}

	views := make([]approvalView, 0, len(result.Data))
	for i := range result.Data {
		views = append(views, toApprovalView(&result.Data[i]))
	}
	respondJSON(w, http.StatusOK, paginatedResponse{
		Data:    views,
		Total:   result.Total,
		Page:    result.Page,
		PerPage: result.PerPage,
	})
}

// ── GetApproval ─────────────────────────────────────────────────────────────

// GetApproval handles GET /api/v1/approvals/{id}
func (h *ApprovalHandler) GetApproval(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid approval id")
		return
	}

	ar, err := h.uc.Get(r.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "approval not found")
			return
		}
		log.Error().Err(err).Str("handler", "GetApproval").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toApprovalView(ar))
}

// ── ApproveApproval ─────────────────────────────────────────────────────────

// ApproveApproval godoc
// POST /api/v1/approvals/{id}/approve
func (h *ApprovalHandler) ApproveApproval(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid approval id")
		return
	}

	// The approver is the authenticated user.
	var userID uuid.UUID
	if uid, uok := appMiddleware.GetUserID(r.Context()); uok {
		userID = uid
	} else {
		userID = uuid.Nil
	}

	ar, err := h.uc.Approve(r.Context(), id, tenantID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "approval not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			respondError(w, r, http.StatusConflict, "approval is not in pending state")
			return
		}
		log.Error().Err(err).Str("handler", "ApproveApproval").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toApprovalView(ar))
}

// ── RejectApproval ──────────────────────────────────────────────────────────

// rejectApprovalRequest is the JSON body for POST /api/v1/approvals/{id}/reject.
type rejectApprovalRequest struct {
	Reason string `json:"reason"`
}

// RejectApproval godoc
// POST /api/v1/approvals/{id}/reject
func (h *ApprovalHandler) RejectApproval(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseUUIDParam(r, "id")
	if !ok {
		respondError(w, r, http.StatusBadRequest, "invalid approval id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req rejectApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	var userID uuid.UUID
	if uid, uok := appMiddleware.GetUserID(r.Context()); uok {
		userID = uid
	} else {
		userID = uuid.Nil
	}

	ar, err := h.uc.Reject(r.Context(), id, tenantID, userID, req.Reason)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, r, http.StatusNotFound, "approval not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			respondError(w, r, http.StatusConflict, "approval is not in pending state")
			return
		}
		log.Error().Err(err).Str("handler", "RejectApproval").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toApprovalView(ar))
}
