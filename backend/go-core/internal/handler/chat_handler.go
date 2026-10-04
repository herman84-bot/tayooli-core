package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/chat"
)

// ChatUsecase defines the business logic interface for chat.
type ChatUsecase interface {
	SendMessage(ctx context.Context, tenantID, userID, sessionID, message string) (string, string, error)
	GetHistory(tenantID, sessionID string) []chat.Message
}

// ChatHandler handles chat HTTP endpoints.
type ChatHandler struct {
	uc ChatUsecase
}

func NewChatHandler(uc ChatUsecase) *ChatHandler {
	return &ChatHandler{uc: uc}
}

// RegisterSecuredRoutes registers auth-required chat endpoints.
// rateLimit is applied only to POST /chat/message — the endpoint that triggers
// an LLM call (cost + latency). GET /history stays cheap and unlimited.
func (h *ChatHandler) RegisterSecuredRoutes(r chi.Router, rateLimit func(http.Handler) http.Handler) {
	r.Route("/chat", func(r chi.Router) {
		r.With(rateLimit).Post("/message", h.SendMessage)
		r.Get("/history", h.GetHistory)
	})
}

// SendMessage handles POST /api/v1/chat/message
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Message == "" {
		respondError(w, r, http.StatusBadRequest, "message is required")
		return
	}

	tid, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID := tid.String()

	userID := ""
	if uid, ok := appMiddleware.GetUserID(r.Context()); ok {
		userID = uid.String()
	}

	reply, sessionID, err := h.uc.SendMessage(r.Context(), tenantID, userID, req.SessionID, req.Message)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"reply":      "Layanan chat sedang tidak tersedia. Silakan coba lagi.",
			"session_id": req.SessionID,
			"error":      err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"reply":      reply,
		"session_id": sessionID,
	})
}

// GetHistory handles GET /api/v1/chat/history?session_id=xxx
func (h *ChatHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		respondError(w, r, http.StatusBadRequest, "session_id is required")
		return
	}

	tid, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID := tid.String()

	history := h.uc.GetHistory(tenantID, sessionID)
	if history == nil {
		respondJSON(w, http.StatusOK, []chat.Message{})
		return
	}

	respondJSON(w, http.StatusOK, history)
}


