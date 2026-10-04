package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// ---------------------------------------------------------------------------
// Mock usecase
// ---------------------------------------------------------------------------

type mockApprovalUsecase struct {
	listFn       func(ctx context.Context, tenantID uuid.UUID) ([]domain.ApprovalRequest, error)
	listPagedFn  func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.ApprovalRequestListPage, error)
	getFn        func(ctx context.Context, id, tenantID uuid.UUID) (*domain.ApprovalRequest, error)
	approveFn    func(ctx context.Context, id, tenantID, userID uuid.UUID) (*domain.ApprovalRequest, error)
	rejectFn     func(ctx context.Context, id, tenantID, userID uuid.UUID, reason string) (*domain.ApprovalRequest, error)
}

func (m *mockApprovalUsecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.ApprovalRequest, error) {
	return m.listFn(ctx, tenantID)
}
func (m *mockApprovalUsecase) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.ApprovalRequestListPage, error) {
	if m.listPagedFn != nil {
		return m.listPagedFn(ctx, tenantID, page, perPage)
	}
	return &domain.ApprovalRequestListPage{Data: []domain.ApprovalRequest{}, Total: 0, Page: page, PerPage: perPage}, nil
}
func (m *mockApprovalUsecase) Get(ctx context.Context, id, tenantID uuid.UUID) (*domain.ApprovalRequest, error) {
	return m.getFn(ctx, id, tenantID)
}
func (m *mockApprovalUsecase) Approve(ctx context.Context, id, tenantID, userID uuid.UUID) (*domain.ApprovalRequest, error) {
	return m.approveFn(ctx, id, tenantID, userID)
}
func (m *mockApprovalUsecase) Reject(ctx context.Context, id, tenantID, userID uuid.UUID, reason string) (*domain.ApprovalRequest, error) {
	return m.rejectFn(ctx, id, tenantID, userID, reason)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func withApprovalTenant(r *http.Request, tenantID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), appMiddleware.TenantIDKey, tenantID)
	return r.WithContext(ctx)
}

func withApprovalTenantAndUser(r *http.Request, tenantID, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), appMiddleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, appMiddleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func withApprovalChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func fakeApprovalRequest(tenantID string) domain.ApprovalRequest {
	return domain.ApprovalRequest{
		ID:               uuid.New().String(),
		TenantID:         tenantID,
		WorkflowID:       uuid.New().String(),
		TargetType:       "invoice",
		TargetID:         uuid.New().String(),
		Status:           "pending",
		CurrentStepIndex: 0,
		RequestedBy:      uuid.New().String(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// ---------------------------------------------------------------------------
// ListApprovals handler tests
// ---------------------------------------------------------------------------

func TestListApprovals_OK(t *testing.T) {
	tenantID := uuid.New()
	ar := fakeApprovalRequest(tenantID.String())

	uc := &mockApprovalUsecase{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.ApprovalRequestListPage, error) {
			return &domain.ApprovalRequestListPage{Data: []domain.ApprovalRequest{ar}, Total: 1, Page: page, PerPage: perPage}, nil
		},
	}
	h := handler.NewApprovalHandler(uc)

	req := withApprovalTenant(httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil), tenantID)
	rr := httptest.NewRecorder()

	h.ListApprovals(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body paginatedEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Errorf("expected 1 approval, got %d", len(body.Data))
	}
	if body.Total != 1 {
		t.Errorf("expected total 1, got %d", body.Total)
	}
}

func TestListApprovals_Unauthorized(t *testing.T) {
	uc := &mockApprovalUsecase{}
	h := handler.NewApprovalHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil)
	rr := httptest.NewRecorder()

	h.ListApprovals(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestListApprovals_InternalError(t *testing.T) {
	uc := &mockApprovalUsecase{
		listPagedFn: func(_ context.Context, _ uuid.UUID, _ int, _ int) (*domain.ApprovalRequestListPage, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := handler.NewApprovalHandler(uc)

	req := withApprovalTenant(httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil), uuid.New())
	rr := httptest.NewRecorder()

	h.ListApprovals(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetApproval handler tests
// ---------------------------------------------------------------------------

func TestGetApproval_OK(t *testing.T) {
	tenantID := uuid.New()
	ar := fakeApprovalRequest(tenantID.String())

	uc := &mockApprovalUsecase{
		getFn: func(_ context.Context, id, tid uuid.UUID) (*domain.ApprovalRequest, error) {
			return &ar, nil
		},
	}
	h := handler.NewApprovalHandler(uc)

	req := withApprovalTenant(httptest.NewRequest(http.MethodGet, "/api/v1/approvals/"+ar.ID, nil), tenantID)
	req = withApprovalChiParam(req, "id", ar.ID)
	rr := httptest.NewRecorder()

	h.GetApproval(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["id"] != ar.ID {
		t.Errorf("expected id=%s, got %v", ar.ID, body["id"])
	}
}

func TestGetApproval_NotFound(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockApprovalUsecase{
		getFn: func(_ context.Context, _, _ uuid.UUID) (*domain.ApprovalRequest, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewApprovalHandler(uc)

	approvalID := uuid.New()
	req := withApprovalTenant(httptest.NewRequest(http.MethodGet, "/api/v1/approvals/"+approvalID.String(), nil), tenantID)
	req = withApprovalChiParam(req, "id", approvalID.String())
	rr := httptest.NewRecorder()

	h.GetApproval(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetApproval_InvalidUUID(t *testing.T) {
	uc := &mockApprovalUsecase{}
	h := handler.NewApprovalHandler(uc)

	req := withApprovalTenant(httptest.NewRequest(http.MethodGet, "/api/v1/approvals/not-a-uuid", nil), uuid.New())
	req = withApprovalChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.GetApproval(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// ApproveApproval handler tests
// ---------------------------------------------------------------------------

func TestApproveApproval_OK(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	ar := fakeApprovalRequest(tenantID.String())
	ar.Status = "approved"

	uc := &mockApprovalUsecase{
		approveFn: func(_ context.Context, id, tid, uid uuid.UUID) (*domain.ApprovalRequest, error) {
			return &ar, nil
		},
	}
	h := handler.NewApprovalHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+ar.ID+"/approve", nil)
	req = withApprovalTenantAndUser(req, tenantID, userID)
	req = withApprovalChiParam(req, "id", ar.ID)
	rr := httptest.NewRecorder()

	h.ApproveApproval(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["status"] != "approved" {
		t.Errorf("expected status=approved, got %v", body["status"])
	}
}

func TestApproveApproval_NotFound(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockApprovalUsecase{
		approveFn: func(_ context.Context, _, _, _ uuid.UUID) (*domain.ApprovalRequest, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewApprovalHandler(uc)

	approvalID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalID.String()+"/approve", nil)
	req = withApprovalTenant(req, tenantID)
	req = withApprovalChiParam(req, "id", approvalID.String())
	rr := httptest.NewRecorder()

	h.ApproveApproval(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestApproveApproval_Conflict_InvalidStatus(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockApprovalUsecase{
		approveFn: func(_ context.Context, _, _, _ uuid.UUID) (*domain.ApprovalRequest, error) {
			return nil, domain.ErrInvalidStatus
		},
	}
	h := handler.NewApprovalHandler(uc)

	approvalID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalID.String()+"/approve", nil)
	req = withApprovalTenant(req, tenantID)
	req = withApprovalChiParam(req, "id", approvalID.String())
	rr := httptest.NewRecorder()

	h.ApproveApproval(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestApproveApproval_Unauthorized(t *testing.T) {
	uc := &mockApprovalUsecase{}
	h := handler.NewApprovalHandler(uc)

	approvalID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalID.String()+"/approve", nil)
	req = withApprovalChiParam(req, "id", approvalID.String())
	rr := httptest.NewRecorder()

	h.ApproveApproval(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestApproveApproval_WithUserID(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	ar := fakeApprovalRequest(tenantID.String())
	ar.Status = "approved"

	var capturedUserID uuid.UUID
	uc := &mockApprovalUsecase{
		approveFn: func(_ context.Context, _, _, uid uuid.UUID) (*domain.ApprovalRequest, error) {
			capturedUserID = uid
			return &ar, nil
		},
	}
	h := handler.NewApprovalHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+ar.ID+"/approve", nil)
	req = withApprovalTenantAndUser(req, tenantID, userID)
	req = withApprovalChiParam(req, "id", ar.ID)
	rr := httptest.NewRecorder()

	h.ApproveApproval(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if capturedUserID != userID {
		t.Errorf("expected user_id=%s, got %s", userID, capturedUserID)
	}
}

// ---------------------------------------------------------------------------
// RejectApproval handler tests
// ---------------------------------------------------------------------------

func TestRejectApproval_OK(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	ar := fakeApprovalRequest(tenantID.String())
	ar.Status = "rejected"

	uc := &mockApprovalUsecase{
		rejectFn: func(_ context.Context, id, tid, uid uuid.UUID, reason string) (*domain.ApprovalRequest, error) {
			return &ar, nil
		},
	}
	h := handler.NewApprovalHandler(uc)

	body := `{"reason":"not compliant"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+ar.ID+"/reject", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withApprovalTenantAndUser(req, tenantID, userID)
	req = withApprovalChiParam(req, "id", ar.ID)
	rr := httptest.NewRecorder()

	h.RejectApproval(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["status"] != "rejected" {
		t.Errorf("expected status=rejected, got %v", resp["status"])
	}
}

func TestRejectApproval_NotFound(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockApprovalUsecase{
		rejectFn: func(_ context.Context, _, _, _ uuid.UUID, _ string) (*domain.ApprovalRequest, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewApprovalHandler(uc)

	approvalID := uuid.New()
	body := `{"reason":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalID.String()+"/reject", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withApprovalTenant(req, tenantID)
	req = withApprovalChiParam(req, "id", approvalID.String())
	rr := httptest.NewRecorder()

	h.RejectApproval(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestRejectApproval_Conflict_InvalidStatus(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockApprovalUsecase{
		rejectFn: func(_ context.Context, _, _, _ uuid.UUID, _ string) (*domain.ApprovalRequest, error) {
			return nil, domain.ErrInvalidStatus
		},
	}
	h := handler.NewApprovalHandler(uc)

	approvalID := uuid.New()
	body := `{"reason":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalID.String()+"/reject", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withApprovalTenant(req, tenantID)
	req = withApprovalChiParam(req, "id", approvalID.String())
	rr := httptest.NewRecorder()

	h.RejectApproval(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestRejectApproval_Unauthorized(t *testing.T) {
	uc := &mockApprovalUsecase{}
	h := handler.NewApprovalHandler(uc)

	approvalID := uuid.New()
	body := `{"reason":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalID.String()+"/reject", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withApprovalChiParam(req, "id", approvalID.String())
	rr := httptest.NewRecorder()

	h.RejectApproval(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRejectApproval_MalformedJSON(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockApprovalUsecase{}
	h := handler.NewApprovalHandler(uc)

	approvalID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalID.String()+"/reject", bytes.NewBufferString("not-json"))
	req = withApprovalTenant(req, tenantID)
	req = withApprovalChiParam(req, "id", approvalID.String())
	rr := httptest.NewRecorder()

	h.RejectApproval(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestRejectApproval_InvalidUUID(t *testing.T) {
	uc := &mockApprovalUsecase{}
	h := handler.NewApprovalHandler(uc)

	body := `{"reason":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/not-a-uuid/reject", bytes.NewBufferString(body))
	req = withApprovalTenant(req, uuid.New())
	req = withApprovalChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.RejectApproval(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
