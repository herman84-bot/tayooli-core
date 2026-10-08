package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
)

// -----------------------------------------------------------------------------
// 1. Putaway Handlers (sentry-wms §1.2 step 2)
// -----------------------------------------------------------------------------

// GetPutawayPending handles GET /api/v1/wms/putaway/pending?warehouse_id=
func (h *WMSHandler) GetPutawayPending(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	whIDStr := r.URL.Query().Get("warehouse_id")
	if whIDStr == "" {
		RespondError(w, r, http.StatusBadRequest, "warehouse_id parameter is required")
		return
	}
	whID, err := uuid.Parse(whIDStr)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
		return
	}

	lines, err := h.uc.GetPutawayPending(r.Context(), tenantID, userID, role, whID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if lines == nil {
		lines = []domain.PutawayPendingLine{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": lines})
}

// ConfirmPutaway handles POST /api/v1/wms/putaway/confirm
func (h *WMSHandler) ConfirmPutaway(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var req uc.PutawayRequest
	if !decodeReceiptJSON(w, r, &req) {
		return
	}

	mov, err := h.uc.ConfirmPutaway(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": mov})
}

// -----------------------------------------------------------------------------
// 2. Release Approval Handlers (PDF-06)
// -----------------------------------------------------------------------------

// ReleaseStockReceipt handles POST /api/v1/wms/receipts/{id}/release
func (h *WMSHandler) ReleaseStockReceipt(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	idStr := chi.URLParam(r, "id")
	rcID, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid receipt id uuid")
		return
	}

	rc, err := h.uc.ReleaseStockReceipt(r.Context(), tenantID, userID, role, rcID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": rc})
}

// -----------------------------------------------------------------------------
// 3. WMS Settings Handlers (PDF-06)
// -----------------------------------------------------------------------------

// GetWMSSettings handles GET /api/v1/wms/settings
func (h *WMSHandler) GetWMSSettings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	settings, err := h.uc.GetWMSSettings(r.Context(), tenantID, userID, role)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": settings})
}

// UpdateWMSSettings handles PUT /api/v1/wms/settings
func (h *WMSHandler) UpdateWMSSettings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var req uc.UpdateWMSSettingsRequest
	if !decodeReceiptJSON(w, r, &req) {
		return
	}

	settings, err := h.uc.UpdateWMSSettings(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": settings})
}

// -----------------------------------------------------------------------------
// 4. Product Default Locations Handlers (CR-03)
// -----------------------------------------------------------------------------

// ListDefaultLocations handles GET /api/v1/wms/default-locations?product_id=
func (h *WMSHandler) ListDefaultLocations(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var prodIDPtr *uuid.UUID
	if prodStr := r.URL.Query().Get("product_id"); prodStr != "" {
		pID, err := uuid.Parse(prodStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid product_id uuid")
			return
		}
		prodIDPtr = &pID
	}

	defs, err := h.uc.ListDefaultLocations(r.Context(), tenantID, userID, role, prodIDPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if defs == nil {
		defs = []domain.ProductDefaultLocation{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": defs})
}

// SetDefaultLocation handles POST /api/v1/wms/default-locations
func (h *WMSHandler) SetDefaultLocation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var req uc.SetDefaultLocationRequest
	if !decodeReceiptJSON(w, r, &req) {
		return
	}

	if err := h.uc.SetDefaultLocation(r.Context(), tenantID, userID, role, req); err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"message": "default location set successfully"})
}

// DeleteDefaultLocation handles DELETE /api/v1/wms/default-locations?product_id=&warehouse_id=
func (h *WMSHandler) DeleteDefaultLocation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	prodStr := r.URL.Query().Get("product_id")
	whStr := r.URL.Query().Get("warehouse_id")
	if prodStr == "" || whStr == "" {
		RespondError(w, r, http.StatusBadRequest, "product_id and warehouse_id query parameters are required")
		return
	}
	prodID, err1 := uuid.Parse(prodStr)
	whID, err2 := uuid.Parse(whStr)
	if err1 != nil || err2 != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid product_id or warehouse_id uuid")
		return
	}

	if err := h.uc.DeleteDefaultLocation(r.Context(), tenantID, userID, role, prodID, whID); err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"message": "default location removed successfully"})
}

// -----------------------------------------------------------------------------
// 5. Traceability Handlers (KO-1c) & Audit
// -----------------------------------------------------------------------------

// TraceBatch handles GET /api/v1/wms/trace/batch/{id}
func (h *WMSHandler) TraceBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	idStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid batch id uuid")
		return
	}

	trace, err := h.uc.TraceBatch(r.Context(), tenantID, userID, role, batchID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": trace})
}

// TraceDocument handles GET /api/v1/wms/trace/document?type=&id=
func (h *WMSHandler) TraceDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	refType := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("type")))
	idStr := r.URL.Query().Get("id")
	if refType == "" || idStr == "" {
		RespondError(w, r, http.StatusBadRequest, "type and id parameters are required")
		return
	}
	refID, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid id uuid")
		return
	}

	docTrace, err := h.uc.TraceDocument(r.Context(), tenantID, userID, role, refType, refID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": docTrace})
}

// ListAuditTrail handles GET /api/v1/wms/audit-trail?entity_type=&entity_id=
func (h *WMSHandler) ListAuditTrail(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	entityType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("entity_type")))
	idStr := r.URL.Query().Get("entity_id")
	if entityType == "" || idStr == "" {
		RespondError(w, r, http.StatusBadRequest, "entity_type and entity_id parameters are required")
		return
	}
	entityID, err := uuid.Parse(idStr)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid entity_id uuid")
		return
	}

	entries, err := h.uc.ListAuditTrail(r.Context(), tenantID, userID, role, entityType, entityID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if entries == nil {
		entries = []domain.AuditTrailEntry{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": entries})
}
