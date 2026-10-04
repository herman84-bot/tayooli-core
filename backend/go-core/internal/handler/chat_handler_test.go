package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/chat"
)

type mockChatUsecase struct {
	lastTenantID string
	lastUserID   string
}

func (m *mockChatUsecase) SendMessage(ctx context.Context, tenantID, userID, sessionID, message string) (string, string, error) {
	m.lastTenantID = tenantID
	m.lastUserID = userID
	return "mock reply", sessionID, nil
}

func (m *mockChatUsecase) GetHistory(tenantID, sessionID string) []chat.Message {
	m.lastTenantID = tenantID
	return []chat.Message{}
}

func TestChatHandler_SendMessage_Unauthorized_WithoutTenantContext(t *testing.T) {
	uc := &mockChatUsecase{}
	h := handler.NewChatHandler(uc)

	body, _ := json.Marshal(map[string]string{
		"message": "hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/message", bytes.NewReader(body))
	// NO tenant context injected!
	rr := httptest.NewRecorder()

	h.SendMessage(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestChatHandler_SendMessage_Authorized_WithTenantContext(t *testing.T) {
	uc := &mockChatUsecase{}
	h := handler.NewChatHandler(uc)

	tenantID := uuid.New()
	userID := uuid.New()

	body, _ := json.Marshal(map[string]string{
		"message": "hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/message", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.SendMessage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if uc.lastTenantID != tenantID.String() {
		t.Errorf("expected tenantID %s, got %s", tenantID.String(), uc.lastTenantID)
	}
}

func TestChatHandler_GetHistory_Unauthorized_WithoutTenantContext(t *testing.T) {
	uc := &mockChatUsecase{}
	h := handler.NewChatHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/history?session_id=s123", nil)
	// NO tenant context injected!
	rr := httptest.NewRecorder()

	h.GetHistory(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestChatHandler_GetHistory_Authorized_WithTenantContext(t *testing.T) {
	uc := &mockChatUsecase{}
	h := handler.NewChatHandler(uc)

	tenantID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/history?session_id=s123", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	h.GetHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if uc.lastTenantID != tenantID.String() {
		t.Errorf("expected tenantID %s, got %s", tenantID.String(), uc.lastTenantID)
	}
}
