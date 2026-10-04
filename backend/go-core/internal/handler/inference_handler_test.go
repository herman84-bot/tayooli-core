package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	aiclient "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/ai"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// ---------------------------------------------------------------------------
// Mock usecase
// ---------------------------------------------------------------------------

type mockInferenceUsecase struct {
	triggerFn func(ctx context.Context, invoiceID, tenantID, amount, vendorID, extractedText string) (*aiclient.IngestResponse, error)
	statusFn  func(ctx context.Context, invoiceID, tenantID string) (*aiclient.StatusResponse, error)
}

func (m *mockInferenceUsecase) TriggerInference(ctx context.Context, invoiceID, tenantID, amount, vendorID, extractedText string) (*aiclient.IngestResponse, error) {
	return m.triggerFn(ctx, invoiceID, tenantID, amount, vendorID, extractedText)
}

func (m *mockInferenceUsecase) GetStatus(ctx context.Context, invoiceID, tenantID string) (*aiclient.StatusResponse, error) {
	return m.statusFn(ctx, invoiceID, tenantID)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func withTenantInference(r *http.Request, tenantID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), appMiddleware.TenantIDKey, tenantID)
	return r.WithContext(ctx)
}

func newInferenceRouter(h *handler.InferenceHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/api/v1/inference/ingest", h.Ingest)
	r.Get("/api/v1/inference/status/{id}", h.GetStatus)
	return r
}

// ---------------------------------------------------------------------------
// Ingest handler tests
// ---------------------------------------------------------------------------

func TestIngest_Success(t *testing.T) {
	invoiceID := uuid.New()
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{
		triggerFn: func(_ context.Context, invID, tID, amount, vendorID, extracted string) (*aiclient.IngestResponse, error) {
			return &aiclient.IngestResponse{JobID: "job-123", Status: "queued"}, nil
		},
	}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	body := map[string]string{
		"invoice_id":     invoiceID.String(),
		"vendor_id":      "V-001",
		"amount":         "1500.00",
		"extracted_text": "Invoice #123 from Vendor",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inference/ingest", bytes.NewReader(bodyBytes))
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["job_id"] != "job-123" {
		t.Errorf("expected job_id 'job-123', got %q", resp["job_id"])
	}
	if resp["status"] != "queued" {
		t.Errorf("expected status 'queued', got %q", resp["status"])
	}
}

func TestIngest_Unauthorized(t *testing.T) {
	mock := &mockInferenceUsecase{}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	body := map[string]string{"invoice_id": uuid.New().String(), "vendor_id": "V-001", "amount": "100"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inference/ingest", bytes.NewReader(bodyBytes))
	// No tenant in context
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestIngest_MissingInvoiceID(t *testing.T) {
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	body := map[string]string{"vendor_id": "V-001", "amount": "100"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inference/ingest", bytes.NewReader(bodyBytes))
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIngest_InvalidInvoiceID(t *testing.T) {
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	body := map[string]string{"invoice_id": "not-a-uuid", "vendor_id": "V-001", "amount": "100"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inference/ingest", bytes.NewReader(bodyBytes))
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIngest_MissingVendorID(t *testing.T) {
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	body := map[string]string{"invoice_id": uuid.New().String(), "amount": "100"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inference/ingest", bytes.NewReader(bodyBytes))
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIngest_MissingAmount(t *testing.T) {
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	body := map[string]string{"invoice_id": uuid.New().String(), "vendor_id": "V-001"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inference/ingest", bytes.NewReader(bodyBytes))
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIngest_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inference/ingest", bytes.NewReader([]byte("{invalid")))
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestIngest_UsecaseError(t *testing.T) {
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{
		triggerFn: func(_ context.Context, _, _, _, _, _ string) (*aiclient.IngestResponse, error) {
			return nil, errors.New("kafka unavailable")
		},
	}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	body := map[string]string{"invoice_id": uuid.New().String(), "vendor_id": "V-001", "amount": "100"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inference/ingest", bytes.NewReader(bodyBytes))
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetStatus handler tests
// ---------------------------------------------------------------------------

func TestGetStatus_Success(t *testing.T) {
	invoiceID := uuid.New()
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{
		statusFn: func(_ context.Context, invID, tID string) (*aiclient.StatusResponse, error) {
			return &aiclient.StatusResponse{
				InvoiceID:          invID,
				Status:             "completed",
				AnomalyScore:       0.15,
				SuggestedGLAccount: "5100",
			}, nil
		},
	}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inference/status/"+invoiceID.String(), nil)
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["invoice_id"] != invoiceID.String() {
		t.Errorf("expected invoice_id %s, got %v", invoiceID, resp["invoice_id"])
	}
	if resp["status"] != "completed" {
		t.Errorf("expected status 'completed', got %v", resp["status"])
	}
}

func TestGetStatus_Unauthorized(t *testing.T) {
	mock := &mockInferenceUsecase{}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inference/status/"+uuid.New().String(), nil)
	// No tenant in context
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetStatus_InvalidID(t *testing.T) {
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inference/status/not-a-uuid", nil)
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetStatus_NotFound(t *testing.T) {
	invoiceID := uuid.New()
	tenantID := uuid.New()
	mock := &mockInferenceUsecase{
		statusFn: func(_ context.Context, _, _ string) (*aiclient.StatusResponse, error) {
			return nil, errors.New("not found")
		},
	}
	h := handler.NewInferenceHandler(mock)
	r := newInferenceRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inference/status/"+invoiceID.String(), nil)
	req = withTenantInference(req, tenantID)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
