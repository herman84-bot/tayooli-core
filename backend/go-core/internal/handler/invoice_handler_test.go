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
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/invoice"
)

// ---------------------------------------------------------------------------
// Mock usecase — satisfies handler's private invoiceUsecase interface
// structurally (Go duck-typing; no need to name the private type).
// ---------------------------------------------------------------------------

type mockInvoiceUsecase struct {
	listFn           func(ctx context.Context, tenantID uuid.UUID) ([]domain.Invoice, error)
	listPagedFn      func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error)
	getInvoiceFn     func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	createFn         func(ctx context.Context, params domain.CreateInvoiceParams) (*domain.Invoice, error)
	approveFn        func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	rejectFn         func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	autoApproveFn    func(ctx context.Context, id, tenantID uuid.UUID) (*invoice.AutoApproveResult, error)
	requeueFn        func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
}

func (m *mockInvoiceUsecase) ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]domain.Invoice, error) {
	return m.listFn(ctx, tenantID)
}
func (m *mockInvoiceUsecase) ListInvoicesPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error) {
	if m.listPagedFn != nil {
		return m.listPagedFn(ctx, tenantID, page, perPage)
	}
	return nil, nil
}
func (m *mockInvoiceUsecase) GetInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return m.getInvoiceFn(ctx, id, tenantID)
}
func (m *mockInvoiceUsecase) CreateInvoice(ctx context.Context, params domain.CreateInvoiceParams) (*domain.Invoice, error) {
	return m.createFn(ctx, params)
}
func (m *mockInvoiceUsecase) ApproveInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return m.approveFn(ctx, id, tenantID)
}
func (m *mockInvoiceUsecase) RejectInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return m.rejectFn(ctx, id, tenantID)
}
func (m *mockInvoiceUsecase) AutoApproveInvoice(ctx context.Context, id, tenantID uuid.UUID) (*invoice.AutoApproveResult, error) {
	if m.autoApproveFn != nil {
		return m.autoApproveFn(ctx, id, tenantID)
	}
	return nil, nil
}
func (m *mockInvoiceUsecase) RequeueInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	if m.requeueFn != nil {
		return m.requeueFn(ctx, id, tenantID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// withTenant injects a tenant UUID into the request context, simulating what
// TenantMiddleware does for real requests.
func withTenant(r *http.Request, tenantID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), appMiddleware.TenantIDKey, tenantID)
	return r.WithContext(ctx)
}

// withChiParam attaches a chi URL parameter, required when the handler calls
// chi.URLParam(r, "id").
func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func fakeInvoice(tenantID uuid.UUID) domain.Invoice {
	return domain.Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		VendorID:      "vendor-1",
		InvoiceNumber: "INV-001",
		Amount:        decimal.New(10_000, 0),
		Currency:      "IDR",
		Status:        domain.StatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// ---------------------------------------------------------------------------
// ListInvoices handler tests
// ---------------------------------------------------------------------------

func TestListInvoices_OK(t *testing.T) {
	tenantID := uuid.New()
	inv := fakeInvoice(tenantID)

	uc := &mockInvoiceUsecase{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error) {
			return &domain.InvoiceListPage{
				Data:  []domain.Invoice{inv},
				Total: 1,
				Page:  1,
				PerPage: 20,
			}, nil
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.ListInvoices(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", body["data"])
	}
	if len(data) != 1 {
		t.Errorf("expected 1 invoice, got %d", len(data))
	}
}

func TestListInvoices_Unauthorized_NoTenantInContext(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	// No tenant injected — simulates missing/invalid JWT
	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
	rr := httptest.NewRecorder()

	h.ListInvoices(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestListInvoices_InternalError(t *testing.T) {
	uc := &mockInvoiceUsecase{
		listPagedFn: func(_ context.Context, _ uuid.UUID, _, _ int) (*domain.InvoiceListPage, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil), uuid.New())
	rr := httptest.NewRecorder()

	h.ListInvoices(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// CreateInvoice handler tests
// ---------------------------------------------------------------------------

func TestCreateInvoice_OK(t *testing.T) {
	tenantID := uuid.New()
	inv := fakeInvoice(tenantID)

	uc := &mockInvoiceUsecase{
		createFn: func(_ context.Context, _ domain.CreateInvoiceParams) (*domain.Invoice, error) {
			return &inv, nil
		},
	}
	h := handler.NewInvoiceHandler(uc)

	// amount is a string, not a JSON number — matches the new createInvoiceRequest.Amount string field.
	body := `{"vendor_id":"vendor-1","invoice_number":"INV-001","amount":"10000","currency":"IDR"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.CreateInvoice(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateInvoice_BadRequestOnInvalidAmountString(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	cases := []struct {
		name string
		body string
	}{
		{"non-numeric amount", `{"vendor_id":"v1","invoice_number":"INV-001","amount":"abc","currency":"IDR"}`},
		{"zero amount", `{"vendor_id":"v1","invoice_number":"INV-001","amount":"0","currency":"IDR"}`},
		{"negative amount", `{"vendor_id":"v1","invoice_number":"INV-001","amount":"-50","currency":"IDR"}`},
		{"empty amount string", `{"vendor_id":"v1","invoice_number":"INV-001","amount":"","currency":"IDR"}`},
		// MEDIUM-1 regression: value exceeds NUMERIC(20,4) ceiling → must not reach DB.
		{"overflow amount", `{"vendor_id":"v1","invoice_number":"INV-001","amount":"10000000000000000.9999","currency":"IDR"}`},
		// MEDIUM-2 regression: > 4 dp silently rounds to 0.0000 in Postgres → must be rejected.
		{"too many decimal places", `{"vendor_id":"v1","invoice_number":"INV-001","amount":"0.000049999","currency":"IDR"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = withTenant(req, uuid.New())
			rr := httptest.NewRecorder()

			h.CreateInvoice(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestCreateInvoice_Unauthorized_NoTenant(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()

	h.CreateInvoice(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCreateInvoice_BadRequestOnMalformedJSON(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString("not-json{{{"))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreateInvoice(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateInvoice_UnprocessableOnInvalidInput(t *testing.T) {
	uc := &mockInvoiceUsecase{
		createFn: func(_ context.Context, _ domain.CreateInvoiceParams) (*domain.Invoice, error) {
			return nil, domain.ErrInvalidInput
		},
	}
	h := handler.NewInvoiceHandler(uc)

	// vendor_id is empty — usecase returns ErrInvalidInput → 422.
	// amount is a valid string so the handler-layer decimal parse passes
	// and the invalid-input error surfaces from the usecase mock.
	body := `{"vendor_id":"","invoice_number":"INV-001","amount":"100","currency":"IDR"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreateInvoice(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// ApproveInvoice handler tests
// ---------------------------------------------------------------------------

func TestApproveInvoice_OK(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	inv := fakeInvoice(tenantID)
	inv.ID = invoiceID
	inv.Status = domain.StatusApproved

	uc := &mockInvoiceUsecase{
		approveFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return &inv, nil
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/approve", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.ApproveInvoice(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "approved" {
		t.Errorf("expected status=approved, got %v", body["status"])
	}
}

func TestApproveInvoice_NotFound(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()

	uc := &mockInvoiceUsecase{
		approveFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/approve", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.ApproveInvoice(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestApproveInvoice_BadRequestOnInvalidUUID(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/not-a-uuid/approve", nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.ApproveInvoice(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestApproveInvoice_Conflict(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()

	uc := &mockInvoiceUsecase{
		approveFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrConflict
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/approve", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.ApproveInvoice(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestApproveInvoice_Unauthorized_NoTenant(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	invoiceID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/approve", nil)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.ApproveInvoice(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// RejectInvoice handler tests
// ---------------------------------------------------------------------------

func TestRejectInvoice_Success(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	inv := fakeInvoice(tenantID)
	inv.ID = invoiceID
	inv.Status = domain.StatusRejected

	uc := &mockInvoiceUsecase{
		rejectFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return &inv, nil
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/reject", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.RejectInvoice(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "rejected" {
		t.Errorf("expected status=rejected, got %v", body["status"])
	}
}

func TestRejectInvoice_AlreadyApproved(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()

	uc := &mockInvoiceUsecase{
		rejectFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrConflict
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/reject", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.RejectInvoice(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRejectInvoice_NotFound(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()

	uc := &mockInvoiceUsecase{
		rejectFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/reject", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.RejectInvoice(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestRejectInvoice_Unauthorized(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	invoiceID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoiceID.String()+"/reject", nil)
	req = withChiParam(req, "id", invoiceID.String())
	// No tenant injected — simulates missing/invalid JWT
	rr := httptest.NewRecorder()

	h.RejectInvoice(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRejectInvoice_BadRequestOnInvalidUUID(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/not-a-uuid/reject", nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.RejectInvoice(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Decimal precision round-trip test
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// GetInvoice handler tests
// ---------------------------------------------------------------------------

func TestGetInvoice_Success(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	inv := fakeInvoice(tenantID)
	inv.ID = invoiceID

	uc := &mockInvoiceUsecase{
		getInvoiceFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Invoice, error) {
			return &inv, nil
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invoiceID.String(), nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.GetInvoice(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["id"] != invoiceID.String() {
		t.Errorf("expected id=%s, got %v", invoiceID.String(), body["id"])
	}
}

func TestGetInvoice_NotFound(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()

	uc := &mockInvoiceUsecase{
		getInvoiceFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invoiceID.String(), nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", invoiceID.String())
	rr := httptest.NewRecorder()

	h.GetInvoice(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetInvoice_Unauthorized(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	invoiceID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invoiceID.String(), nil)
	req = withChiParam(req, "id", invoiceID.String())
	// No tenant in context — simulates missing/invalid JWT
	rr := httptest.NewRecorder()

	h.GetInvoice(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetInvoice_InvalidUUID(t *testing.T) {
	uc := &mockInvoiceUsecase{}
	h := handler.NewInvoiceHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/not-a-uuid", nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.GetInvoice(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// TestCreateInvoice_DecimalPrecisionPreserved verifies that a high-precision
// amount string (e.g. "12345.6789") survives the handler's JSON decode without
// being converted to float64. The handler must parse it with decimal.NewFromString
// and forward the exact decimal.Decimal value to the usecase layer.
//
// This is the regression guard for the float64 → decimal.Decimal migration:
// if the handler were to use a float64 field in createInvoiceRequest, the
// value would be silently truncated to 12345.67890625 (IEEE 754) before the
// usecase ever saw it.
func TestCreateInvoice_DecimalPrecisionPreserved(t *testing.T) {
	tenantID := uuid.New()

	const amountStr = "12345.6789"
	want, _ := decimal.NewFromString(amountStr)

	var capturedParams domain.CreateInvoiceParams

	uc := &mockInvoiceUsecase{
		createFn: func(_ context.Context, p domain.CreateInvoiceParams) (*domain.Invoice, error) {
			capturedParams = p
			// Echo the received amount back so the response body can be checked too.
			inv := fakeInvoice(tenantID)
			inv.Amount = p.Amount
			return &inv, nil
		},
	}
	h := handler.NewInvoiceHandler(uc)

	body := `{"vendor_id":"vendor-1","invoice_number":"INV-999","amount":"12345.6789","currency":"IDR"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.CreateInvoice(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// 1. Verify the exact decimal value reached the usecase (no float conversion).
	if !capturedParams.Amount.Equal(want) {
		t.Errorf("decimal precision lost at handler→usecase boundary: got %s, want %s",
			capturedParams.Amount.String(), amountStr)
	}

	// 2. Verify the response body carries amount as a JSON string, not a JSON number.
	var respBody map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	gotAmtStr, ok := respBody["amount"].(string)
	if !ok {
		t.Errorf("response amount field must be JSON string type, got %T — float serialization regression",
			respBody["amount"])
	} else if gotAmtStr != amountStr {
		t.Errorf("response amount precision mismatch: got %q, want %q", gotAmtStr, amountStr)
	}
}
