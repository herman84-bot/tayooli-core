package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

type AIPermissionsHandler struct {
	repo *postgres.AIPermissionsRepo
}

func NewAIPermissionsHandler(repo *postgres.AIPermissionsRepo) *AIPermissionsHandler {
	return &AIPermissionsHandler{repo: repo}
}

// GetPermissions returns the current AI permissions and autonomy level for the tenant.
func (h *AIPermissionsHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}

	perms, err := h.repo.GetByTenantID(r.Context(), tenantID)
	if err != nil {
		RespondError(w, r, http.StatusInternalServerError, "failed to get ai permissions", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(perms)
}

// UpdatePermissions updates the tenant AI autonomy level, allowed scopes, or emergency kill switch.
// Only accessible to owner or admin.
func (h *AIPermissionsHandler) UpdatePermissions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var input domain.UpdateAIPermissionsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.AutonomyLevel != nil {
		al := *input.AutonomyLevel
		if al != "advisory" && al != "assisted" && al != "autopilot" {
			RespondError(w, r, http.StatusBadRequest, "invalid autonomy_level: must be advisory, assisted, or autopilot")
			return
		}
	}

	var updatedBy *uuid.UUID
	if uid, ok := middleware.GetUserID(r.Context()); ok {
		updatedBy = &uid
	}

	perms, err := h.repo.Update(r.Context(), tenantID, input, updatedBy)
	if err != nil {
		RespondError(w, r, http.StatusInternalServerError, "failed to update ai permissions", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(perms)
}
