package handler

// Sprint 3 Outbound HTTP Handlers: Picking Tasks, Pack Station Scan,
// Damaged Reporting & Pack Finalization (Sentry-WMS §1.3, OCA §1.3).

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/shopspring/decimal"
)

func (h *WMSHandler) GetPickingTask(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	doID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid delivery order id")
		return
	}

	detail, err := h.uc.GetPickingTask(r.Context(), tenantID, userID, role, doID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": detail})
}

func (h *WMSHandler) StartPickingTask(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	doID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid delivery order id")
		return
	}

	detail, err := h.uc.StartPickingTask(r.Context(), tenantID, userID, role, doID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": detail})
}

func (h *WMSHandler) RecordPickingItem(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	doID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid delivery order id")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid task item id")
		return
	}

	var req struct {
		PickedQty decimal.Decimal `json:"picked_qty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	detail, err := h.uc.RecordPickingItem(r.Context(), tenantID, userID, role, doID, itemID, req.PickedQty)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": detail})
}

func (h *WMSHandler) ReportPickingDamaged(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	doID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid delivery order id")
		return
	}

	var req domain.PickingDamagedReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.uc.ReportPickingDamaged(r.Context(), tenantID, userID, role, doID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": res})
}

func (h *WMSHandler) ScanPackStationItem(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	doID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid delivery order id")
		return
	}

	var req domain.PackScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.uc.ScanPackStationItem(r.Context(), tenantID, userID, role, doID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": res})
}

func (h *WMSHandler) CompletePackStation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	doID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid delivery order id")
		return
	}

	var req domain.PackCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.uc.CompletePackStation(r.Context(), tenantID, userID, role, doID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": res})
}
