package handler

// Sprint 4 Outbound HTTP Handlers: Shipping Manifests, Truck Loading Scan Verification,
// Driver Handover Dispatch & 8 SOP Outbound KPIs (ADR-014 Invariant 1, Master PRD §3.2, §4.1).

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// ListShippingManifests handles GET /api/v1/wms/manifests
func (h *WMSHandler) ListShippingManifests(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whIDPtr *uuid.UUID
	if whStr := r.URL.Query().Get("warehouse_id"); whStr != "" {
		parsed, err := uuid.Parse(whStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whIDPtr = &parsed
	}

	var statusPtr *domain.ShippingManifestStatus
	if stStr := r.URL.Query().Get("status"); stStr != "" {
		st := domain.ShippingManifestStatus(stStr)
		statusPtr = &st
	}

	var expNamePtr *string
	if exp := r.URL.Query().Get("expedition_name"); exp != "" {
		expNamePtr = &exp
	}

	manifests, err := h.uc.ListShippingManifests(r.Context(), tenantID, userID, role, whIDPtr, statusPtr, expNamePtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if manifests == nil {
		manifests = []domain.ShippingManifest{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": manifests})
}

// CreateShippingManifest handles POST /api/v1/wms/manifests
func (h *WMSHandler) CreateShippingManifest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var req domain.CreateShippingManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	manifest, err := h.uc.CreateShippingManifest(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"data": manifest})
}

// GetShippingManifest handles GET /api/v1/wms/manifests/{id}
func (h *WMSHandler) GetShippingManifest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	manifestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid manifest id")
		return
	}

	detail, err := h.uc.GetShippingManifest(r.Context(), tenantID, userID, role, manifestID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": detail})
}

// ScanDOLoading handles POST /api/v1/wms/manifests/{id}/loading-scan
func (h *WMSHandler) ScanDOLoading(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	manifestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid manifest id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	var req domain.LoadingScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	detail, err := h.uc.ScanDOLoading(r.Context(), tenantID, userID, role, manifestID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": detail})
}

// DispatchShippingManifest handles POST /api/v1/wms/manifests/{id}/dispatch
func (h *WMSHandler) DispatchShippingManifest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	manifestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid manifest id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2MB limit for signature SVG
	var req domain.DispatchShippingManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	manifest, err := h.uc.DispatchShippingManifest(r.Context(), tenantID, userID, role, manifestID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": manifest})
}

// GetWMSOutboundKPIs handles GET /api/v1/wms/kpi
func (h *WMSHandler) GetWMSOutboundKPIs(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whIDPtr *uuid.UUID
	if whStr := r.URL.Query().Get("warehouse_id"); whStr != "" {
		parsed, err := uuid.Parse(whStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whIDPtr = &parsed
	}

	kpis, err := h.uc.GetWMSOutboundKPIs(r.Context(), tenantID, userID, role, whIDPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": kpis})
}
