package handler

// Sprint 2 QC Inbound & Karantina endpoints (ADR-014 Invariant 3).

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

type qcActor struct {
	tenantID, userID uuid.UUID
	role             string
}

func qcContext(w http.ResponseWriter, r *http.Request) (qcActor, bool) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return qcActor{}, false
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	return qcActor{tenantID, userID, appMiddleware.GetRole(r.Context())}, true
}

func parseQCUUID(w http.ResponseWriter, r *http.Request, raw, name string) (uuid.UUID, bool) {
	if raw == "" {
		RespondError(w, r, http.StatusBadRequest, name+" parameter is required")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid "+name+" uuid")
		return uuid.Nil, false
	}
	return id, true
}

// GET /api/v1/wms/receipts/{id}/qc -> {data: QCInspectionDetail|null}
func (h *WMSHandler) GetReceiptQC(w http.ResponseWriter, r *http.Request) {
	a, ok := qcContext(w, r)
	if !ok {
		return
	}
	id, ok := parseQCUUID(w, r, chi.URLParam(r, "id"), "receipt id")
	if !ok {
		return
	}
	d, err := h.uc.GetReceiptQC(r.Context(), a.tenantID, a.userID, a.role, id)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": d})
}

// POST /api/v1/wms/receipts/{id}/qc
func (h *WMSHandler) SubmitQCInspection(w http.ResponseWriter, r *http.Request) {
	a, ok := qcContext(w, r)
	if !ok {
		return
	}
	id, ok := parseQCUUID(w, r, chi.URLParam(r, "id"), "receipt id")
	if !ok {
		return
	}
	var in domain.QCInspectionInput
	if !decodeReceiptJSON(w, r, &in) {
		return
	}
	d, err := h.uc.SubmitQCInspection(r.Context(), a.tenantID, a.userID, a.role, id, in)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"data": d})
}

// GET /api/v1/wms/qc-inspections?warehouse_id=
func (h *WMSHandler) ListQCInspections(w http.ResponseWriter, r *http.Request) {
	a, ok := qcContext(w, r)
	if !ok {
		return
	}
	wh, ok := parseQCUUID(w, r, r.URL.Query().Get("warehouse_id"), "warehouse_id")
	if !ok {
		return
	}
	list, err := h.uc.ListQCInspections(r.Context(), a.tenantID, a.userID, a.role, wh)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if list == nil {
		list = []domain.QCInspection{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": list})
}

// GET /api/v1/wms/quarantine?warehouse_id=
func (h *WMSHandler) ListQuarantineStock(w http.ResponseWriter, r *http.Request) {
	a, ok := qcContext(w, r)
	if !ok {
		return
	}
	wh, ok := parseQCUUID(w, r, r.URL.Query().Get("warehouse_id"), "warehouse_id")
	if !ok {
		return
	}
	list, err := h.uc.ListQuarantineStock(r.Context(), a.tenantID, a.userID, a.role, wh)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if list == nil {
		list = []domain.BatchBalance{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": list})
}

// POST /api/v1/wms/quarantine/release
func (h *WMSHandler) ReleaseQuarantine(w http.ResponseWriter, r *http.Request) {
	h.quarantineAction(w, r, false)
}

// POST /api/v1/wms/quarantine/scrap
func (h *WMSHandler) ScrapQuarantine(w http.ResponseWriter, r *http.Request) {
	h.quarantineAction(w, r, true)
}

func (h *WMSHandler) quarantineAction(w http.ResponseWriter, r *http.Request, scrap bool) {
	a, ok := qcContext(w, r)
	if !ok {
		return
	}
	var in domain.QuarantineActionInput
	if !decodeReceiptJSON(w, r, &in) {
		return
	}
	var (
		mov *domain.StockMovement
		err error
	)
	if scrap {
		mov, err = h.uc.ScrapQuarantine(r.Context(), a.tenantID, a.userID, a.role, in)
	} else {
		mov, err = h.uc.ReleaseQuarantine(r.Context(), a.tenantID, a.userID, a.role, in)
	}
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": mov})
}
