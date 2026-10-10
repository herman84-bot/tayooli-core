package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/team"
)

// teamUsecase abstracts team management operations.
type teamUsecase interface {
	ListMembers(ctx context.Context, tenantID uuid.UUID) ([]domain.User, error)
	ChangeRole(ctx context.Context, tenantID, userID uuid.UUID, newRole string, warehouseIDs []uuid.UUID, requesterID uuid.UUID) error
	RemoveMember(ctx context.Context, tenantID, userID uuid.UUID, requesterID uuid.UUID) error
	Invite(ctx context.Context, tenantID uuid.UUID, email, role string, warehouseIDs []uuid.UUID, requesterID uuid.UUID) (*team.InviteResult, error)
	ResendInvitation(ctx context.Context, tenantID, userID uuid.UUID) (*team.InviteResult, error)
}

// TeamHandler handles HTTP requests for team management.
type TeamHandler struct {
	uc teamUsecase
}

func NewTeamHandler(uc teamUsecase) *TeamHandler {
	return &TeamHandler{uc: uc}
}

// teamMemberResponse is the JSON shape returned for each team member.
type teamMemberResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Role               string                     `json:"role"`
	AssignedWarehouses []domain.AssignedWarehouse `json:"assigned_warehouses"`
	CreatedAt          string                     `json:"created_at"`
}

func toTeamMemberResponse(m domain.User) teamMemberResponse {
	whs := m.AssignedWarehouses
	if whs == nil {
		whs = []domain.AssignedWarehouse{}
	}
	return teamMemberResponse{
		ID:                 m.ID.String(),
		Email:              m.Email,
		FullName:           m.FullName,
		Role:               m.Role,
		AssignedWarehouses: whs,
		CreatedAt:          m.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func parseWarehouseIDs(raw []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(raw))
	seen := map[uuid.UUID]bool{}
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ListMembers handles GET /api/v1/settings/team.
func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	members, err := h.uc.ListMembers(r.Context(), tenantID)
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "failed to list members")
		return
	}

	resp := make([]teamMemberResponse, len(members))
	for i, m := range members {
		resp[i] = toTeamMemberResponse(m)
	}

	respondJSON(w, http.StatusOK, map[string]any{"members": resp})
}

// changeRoleRequest is the JSON body for PATCH /api/v1/settings/team/{id}/role.
type changeRoleRequest struct {
	Role         string   `json:"role"`
	WarehouseIDs []string `json:"warehouse_ids"`
}

// ChangeRole handles PATCH /api/v1/settings/team/{id}/role.
func (h *TeamHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	requesterID, ok := appMiddleware.GetUserID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid user id")
		return
	}

	var req changeRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Role == "" {
		respondError(w, r, http.StatusBadRequest, "role is required")
		return
	}

	whIDs, err := parseWarehouseIDs(req.WarehouseIDs)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid warehouse id")
		return
	}

	if err := h.uc.ChangeRole(r.Context(), tenantID, userID, req.Role, whIDs, requesterID); err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "role updated"})
}

// inviteMemberRequest is the JSON body for POST /api/v1/settings/team.
type inviteMemberRequest struct {
	Email        string   `json:"email"`
	Role         string   `json:"role"`
	WarehouseIDs []string `json:"warehouse_ids"`
}

// InviteMember handles POST /api/v1/settings/team.
func (h *TeamHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	requesterID, ok := appMiddleware.GetUserID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		respondError(w, r, http.StatusBadRequest, "email is required")
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}

	whIDs, err := parseWarehouseIDs(req.WarehouseIDs)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid warehouse id")
		return
	}

	res, err := h.uc.Invite(r.Context(), tenantID, req.Email, req.Role, whIDs, requesterID)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, inviteResponse(res, "anggota ditambahkan"))
}

func inviteResponse(res *team.InviteResult, base string) map[string]any {
	msg := base + " dan email undangan terkirim"
	if !res.EmailSent {
		msg = base + ", tetapi email undangan TIDAK terkirim: " + res.EmailError
	}
	return map[string]any{
		"message":     msg,
		"email_sent":  res.EmailSent,
		"email_error": res.EmailError,
		"member":      toTeamMemberResponse(*res.User),
	}
}

// ResendInvitation handles POST /api/v1/settings/team/{id}/resend-invite.
func (h *TeamHandler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid user id")
		return
	}
	res, err := h.uc.ResendInvitation(r.Context(), tenantID, userID)
	if err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, inviteResponse(res, "undangan diproses"))
}

// RemoveMember handles DELETE /api/v1/settings/team/{id}.
func (h *TeamHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	requesterID, ok := appMiddleware.GetUserID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.uc.RemoveMember(r.Context(), tenantID, userID, requesterID); err != nil {
		respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "member removed"})
}
