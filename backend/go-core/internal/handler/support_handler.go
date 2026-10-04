package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// TelegramSender abstracts sending messages to Telegram.
type TelegramSender interface {
	ReportIssue(category, description, userEmail, tenantID string) error
}

// SupportHandler handles support report endpoints.
type SupportHandler struct {
	telegram TelegramSender
}

func NewSupportHandler(telegram TelegramSender) *SupportHandler {
	return &SupportHandler{telegram: telegram}
}

// RegisterSecuredRoutes registers auth-required support endpoints.
func (h *SupportHandler) RegisterSecuredRoutes(r chi.Router) {
	r.Post("/support/report", h.ReportIssue)
}

// ReportIssue handles POST /api/v1/support/report
func (h *SupportHandler) ReportIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Category    string `json:"category"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Category == "" || req.Description == "" {
		respondError(w, r, http.StatusBadRequest, "category and description are required")
		return
	}

	// Get user info from context
	userEmail := "unknown"
	tenantID := "unknown"
	if uid, ok := appMiddleware.GetUserID(r.Context()); ok {
		userEmail = uid.String()
	}
	if tid, ok := appMiddleware.GetTenantID(r.Context()); ok {
		tenantID = tid.String()
	}

	// Send to Telegram
	if err := h.telegram.ReportIssue(req.Category, req.Description, userEmail, tenantID); err != nil {
		respondError(w, r, http.StatusInternalServerError, "failed to send report")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "sent",
		"message": "Laporan sudah diteruskan. Tim kami akan segera merespon.",
	})
}
