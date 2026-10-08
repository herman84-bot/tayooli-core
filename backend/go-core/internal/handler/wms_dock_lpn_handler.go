package handler

// Sprint 5 WMS HTTP Handlers: Inbound Dock Scheduling, Appointments, and Pallet LPN Containerization
// ADR-014 Invariant 1 (Double-Entry Ledger per Batch), OCA/wms dock scheduling patterns.

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// -----------------------------------------------------------------------------
// Inbound Dock Bays
// -----------------------------------------------------------------------------

// ListDocks handles GET /api/v1/wms/docks
func (h *WMSHandler) ListDocks(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	whStr := r.URL.Query().Get("warehouse_id")
	if whStr == "" {
		RespondError(w, r, http.StatusBadRequest, "warehouse_id parameter is required")
		return
	}
	warehouseID, err := uuid.Parse(whStr)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
		return
	}

	var statusPtr *domain.DockStatus
	if stStr := r.URL.Query().Get("status"); stStr != "" {
		st := domain.DockStatus(stStr)
		statusPtr = &st
	}

	docks, err := h.uc.ListDocks(r.Context(), tenantID, userID, role, warehouseID, statusPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if docks == nil {
		docks = []domain.InboundDock{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": docks})
}

// CreateDock handles POST /api/v1/wms/docks
func (h *WMSHandler) CreateDock(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req domain.CreateDockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	dock, err := h.uc.CreateDock(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"data": dock})
}

// GetDock handles GET /api/v1/wms/docks/{id}
func (h *WMSHandler) GetDock(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	dockID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid dock id")
		return
	}

	dock, err := h.uc.GetDock(r.Context(), tenantID, userID, role, dockID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": dock})
}

// UpdateDockStatus handles PATCH /api/v1/wms/docks/{id}/status
func (h *WMSHandler) UpdateDockStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	dockID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid dock id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req domain.UpdateDockStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	dock, err := h.uc.UpdateDockStatus(r.Context(), tenantID, userID, role, dockID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": dock})
}

// -----------------------------------------------------------------------------
// Dock Appointments
// -----------------------------------------------------------------------------

// ListAppointments handles GET /api/v1/wms/dock-appointments
func (h *WMSHandler) ListAppointments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	whStr := r.URL.Query().Get("warehouse_id")
	if whStr == "" {
		RespondError(w, r, http.StatusBadRequest, "warehouse_id parameter is required")
		return
	}
	warehouseID, err := uuid.Parse(whStr)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
		return
	}

	var statusPtr *domain.AppointmentStatus
	if stStr := r.URL.Query().Get("status"); stStr != "" {
		st := domain.AppointmentStatus(stStr)
		statusPtr = &st
	}

	appointments, err := h.uc.ListAppointments(r.Context(), tenantID, userID, role, warehouseID, statusPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if appointments == nil {
		appointments = []domain.DockAppointment{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": appointments})
}

// CreateAppointment handles POST /api/v1/wms/dock-appointments
func (h *WMSHandler) CreateAppointment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req domain.CreateAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	appointment, err := h.uc.CreateAppointment(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"data": appointment})
}

// GetAppointment handles GET /api/v1/wms/dock-appointments/{id}
func (h *WMSHandler) GetAppointment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid appointment id")
		return
	}

	appointment, err := h.uc.GetAppointment(r.Context(), tenantID, userID, role, appID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": appointment})
}

// AssignDock handles POST /api/v1/wms/dock-appointments/{id}/assign
func (h *WMSHandler) AssignDock(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid appointment id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req domain.AssignDockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	appointment, err := h.uc.AssignDockToAppointment(r.Context(), tenantID, userID, role, appID, req.DockID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": appointment})
}

// UpdateAppointmentStatus handles PATCH /api/v1/wms/dock-appointments/{id}/status
func (h *WMSHandler) UpdateAppointmentStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid appointment id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req domain.UpdateAppointmentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	appointment, err := h.uc.UpdateAppointmentStatus(r.Context(), tenantID, userID, role, appID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": appointment})
}

// -----------------------------------------------------------------------------
// Stock LPNs (License Plate Numbers)
// -----------------------------------------------------------------------------

// ListLPNs handles GET /api/v1/wms/lpns
func (h *WMSHandler) ListLPNs(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	whStr := r.URL.Query().Get("warehouse_id")
	if whStr == "" {
		RespondError(w, r, http.StatusBadRequest, "warehouse_id parameter is required")
		return
	}
	warehouseID, err := uuid.Parse(whStr)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
		return
	}

	var statusPtr *domain.LPNStatus
	if stStr := r.URL.Query().Get("status"); stStr != "" {
		st := domain.LPNStatus(stStr)
		statusPtr = &st
	}

	lpns, err := h.uc.ListLPNs(r.Context(), tenantID, userID, role, warehouseID, statusPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if lpns == nil {
		lpns = []domain.StockLPN{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": lpns})
}

// CreateLPN handles POST /api/v1/wms/lpns
func (h *WMSHandler) CreateLPN(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req domain.CreateLPNRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	lpn, err := h.uc.CreateLPN(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"data": lpn})
}

// GetLPN handles GET /api/v1/wms/lpns/{id}
func (h *WMSHandler) GetLPN(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	lpnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid lpn id")
		return
	}

	detail, err := h.uc.GetLPN(r.Context(), tenantID, userID, role, lpnID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": detail})
}

// AddLPNItem handles POST /api/v1/wms/lpns/{id}/items
func (h *WMSHandler) AddLPNItem(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	lpnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid lpn id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req domain.AddLPNItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	detail, err := h.uc.AddLPNItem(r.Context(), tenantID, userID, role, lpnID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": detail})
}

// MoveLPN handles POST /api/v1/wms/lpns/{id}/move
func (h *WMSHandler) MoveLPN(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	lpnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid lpn id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req domain.MoveLPNRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	detail, err := h.uc.MoveLPN(r.Context(), tenantID, userID, role, lpnID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": detail})
}
