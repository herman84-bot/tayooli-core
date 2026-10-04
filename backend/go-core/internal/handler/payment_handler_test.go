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

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// ---------------------------------------------------------------------------
// Mock usecase
// ---------------------------------------------------------------------------

type mockPaymentOrderUsecase struct {
	listFn    func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error)
	getFn     func(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error)
	createFn  func(ctx context.Context, params domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error)
	approveFn func(ctx context.Context, id, tenantID, approvedBy uuid.UUID, role string) (*domain.PaymentOrder, error)
	payFn     func(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error)
	rejectFn  func(ctx context.Context, id, tenantID, rejectedBy uuid.UUID) (*domain.PaymentOrder, error)
}

func (m *mockPaymentOrderUsecase) ListPaymentOrders(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error) {
	return m.listFn(ctx, tenantID, page, perPage)
}
func (m *mockPaymentOrderUsecase) GetPaymentOrder(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error) {
	return m.getFn(ctx, id, tenantID)
}
func (m *mockPaymentOrderUsecase) CreatePaymentOrder(ctx context.Context, params domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
	return m.createFn(ctx, params)
}
func (m *mockPaymentOrderUsecase) ApprovePaymentOrder(ctx context.Context, id, tenantID, approvedBy uuid.UUID, role string) (*domain.PaymentOrder, error) {
	return m.approveFn(ctx, id, tenantID, approvedBy, role)
}
func (m *mockPaymentOrderUsecase) PayPaymentOrder(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error) {
	return m.payFn(ctx, id, tenantID)
}
func (m *mockPaymentOrderUsecase) RejectPaymentOrder(ctx context.Context, id, tenantID, rejectedBy uuid.UUID) (*domain.PaymentOrder, error) {
	return m.rejectFn(ctx, id, tenantID, rejectedBy)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func withTenantAndUser(r *http.Request, tenantID, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), appMiddleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, appMiddleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func fakePaymentOrder(tenantID uuid.UUID) domain.PaymentOrder {
	return domain.PaymentOrder{
		ID:            uuid.New(),
		TenantID:      tenantID,
		InvoiceID:     uuid.New(),
		Amount:        decimal.New(5_000_000, 0),
		Currency:      "IDR",
		PaymentMethod: "bank_transfer",
		Status:        domain.PaymentOrderStatusDraft,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// ---------------------------------------------------------------------------
// ListPaymentOrders handler tests
// ---------------------------------------------------------------------------

func TestListPaymentOrders_OK(t *testing.T) {
	tenantID := uuid.New()
	po := fakePaymentOrder(tenantID)

	uc := &mockPaymentOrderUsecase{
		listFn: func(_ context.Context, _ uuid.UUID, _, _ int) (*domain.PaymentOrderListPage, error) {
			return &domain.PaymentOrderListPage{
				Data:    []domain.PaymentOrder{po},
				Total:   1,
				Page:    1,
				PerPage: 20,
			}, nil
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/payment-orders", nil), tenantID)
	rr := httptest.NewRecorder()

	h.ListPaymentOrders(rr, req)

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
		t.Errorf("expected 1 payment order, got %d", len(data))
	}
}

func TestListPaymentOrders_Unauthorized(t *testing.T) {
	uc := &mockPaymentOrderUsecase{}
	h := handler.NewPaymentOrderHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payment-orders", nil)
	rr := httptest.NewRecorder()

	h.ListPaymentOrders(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestListPaymentOrders_InternalError(t *testing.T) {
	uc := &mockPaymentOrderUsecase{
		listFn: func(_ context.Context, _ uuid.UUID, _, _ int) (*domain.PaymentOrderListPage, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/payment-orders", nil), uuid.New())
	rr := httptest.NewRecorder()

	h.ListPaymentOrders(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetPaymentOrder handler tests
// ---------------------------------------------------------------------------

func TestGetPaymentOrder_OK(t *testing.T) {
	tenantID := uuid.New()
	po := fakePaymentOrder(tenantID)

	uc := &mockPaymentOrderUsecase{
		getFn: func(_ context.Context, id, tid uuid.UUID) (*domain.PaymentOrder, error) {
			return &po, nil
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/payment-orders/"+po.ID.String(), nil), tenantID)
	req = withChiParam(req, "id", po.ID.String())
	rr := httptest.NewRecorder()

	h.GetPaymentOrder(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["id"] != po.ID.String() {
		t.Errorf("expected id=%s, got %v", po.ID.String(), body["id"])
	}
}

func TestGetPaymentOrder_NotFound(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockPaymentOrderUsecase{
		getFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/payment-orders/"+poID.String(), nil), tenantID)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.GetPaymentOrder(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetPaymentOrder_InvalidUUID(t *testing.T) {
	uc := &mockPaymentOrderUsecase{}
	h := handler.NewPaymentOrderHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/payment-orders/not-a-uuid", nil), uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.GetPaymentOrder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// CreatePaymentOrder handler tests
// ---------------------------------------------------------------------------

func TestCreatePaymentOrder_OK(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	invoiceID := uuid.New()
	po := fakePaymentOrder(tenantID)
	po.InvoiceID = invoiceID
	po.Status = domain.PaymentOrderStatusDraft

	uc := &mockPaymentOrderUsecase{
		createFn: func(_ context.Context, p domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
			return &po, nil
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	body := `{"invoice_id":"` + invoiceID.String() + `","amount":"5000000","currency":"IDR","payment_method":"bank_transfer"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantAndUser(req, tenantID, userID)
	rr := httptest.NewRecorder()

	h.CreatePaymentOrder(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreatePaymentOrder_BadRequest_MalformedJSON(t *testing.T) {
	uc := &mockPaymentOrderUsecase{}
	h := handler.NewPaymentOrderHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders", bytes.NewBufferString("not-json")), uuid.New())
	rr := httptest.NewRecorder()

	h.CreatePaymentOrder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreatePaymentOrder_Conflict_InvoiceNotApproved(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()

	uc := &mockPaymentOrderUsecase{
		createFn: func(_ context.Context, _ domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
			return nil, domain.ErrConflict
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	body := `{"invoice_id":"` + invoiceID.String() + `","amount":"5000000","currency":"IDR"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.CreatePaymentOrder(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreatePaymentOrder_Unauthorized(t *testing.T) {
	uc := &mockPaymentOrderUsecase{}
	h := handler.NewPaymentOrderHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()

	h.CreatePaymentOrder(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCreatePaymentOrder_InvalidAmount(t *testing.T) {
	uc := &mockPaymentOrderUsecase{}
	h := handler.NewPaymentOrderHandler(uc)

	invoiceID := uuid.New()
	cases := []struct {
		name string
		body string
	}{
		{"non-numeric amount", `{"invoice_id":"` + invoiceID.String() + `","amount":"abc","currency":"IDR"}`},
		{"zero amount", `{"invoice_id":"` + invoiceID.String() + `","amount":"0","currency":"IDR"}`},
		{"negative amount", `{"invoice_id":"` + invoiceID.String() + `","amount":"-100","currency":"IDR"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = withTenant(req, uuid.New())
			rr := httptest.NewRecorder()

			h.CreatePaymentOrder(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ApprovePaymentOrder handler tests
// ---------------------------------------------------------------------------

func TestApprovePaymentOrder_OK(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	po := fakePaymentOrder(tenantID)
	po.Status = domain.PaymentOrderStatusApproved

	uc := &mockPaymentOrderUsecase{
		approveFn: func(_ context.Context, id, tid, approvedBy uuid.UUID, role string) (*domain.PaymentOrder, error) {
			return &po, nil
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+po.ID.String()+"/approve", nil)
	req = withTenantAndUser(req, tenantID, userID)
	req = withChiParam(req, "id", po.ID.String())
	rr := httptest.NewRecorder()

	h.ApprovePaymentOrder(rr, req)

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

func TestApprovePaymentOrder_NotFound(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockPaymentOrderUsecase{
		approveFn: func(_ context.Context, _, _, _ uuid.UUID, _ string) (*domain.PaymentOrder, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/approve", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.ApprovePaymentOrder(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestApprovePaymentOrder_Conflict(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockPaymentOrderUsecase{
		approveFn: func(_ context.Context, _, _, _ uuid.UUID, _ string) (*domain.PaymentOrder, error) {
			return nil, domain.ErrConflict
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/approve", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.ApprovePaymentOrder(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestApprovePaymentOrder_Forbidden(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	uc := &mockPaymentOrderUsecase{
		approveFn: func(_ context.Context, _, _, _ uuid.UUID, _ string) (*domain.PaymentOrder, error) {
			return nil, domain.ErrForbidden
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/approve", nil)
	req = withTenantAndUser(req, tenantID, userID)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.ApprovePaymentOrder(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestApprovePaymentOrder_Unauthorized(t *testing.T) {
	uc := &mockPaymentOrderUsecase{}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/approve", nil)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.ApprovePaymentOrder(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// PayPaymentOrder handler tests
// ---------------------------------------------------------------------------

func TestPayPaymentOrder_OK(t *testing.T) {
	tenantID := uuid.New()
	po := fakePaymentOrder(tenantID)
	po.Status = domain.PaymentOrderStatusPaid
	now := time.Now()
	po.PaidAt = &now

	uc := &mockPaymentOrderUsecase{
		payFn: func(_ context.Context, id, tid uuid.UUID) (*domain.PaymentOrder, error) {
			return &po, nil
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+po.ID.String()+"/pay", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", po.ID.String())
	rr := httptest.NewRecorder()

	h.PayPaymentOrder(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["status"] != "paid" {
		t.Errorf("expected status=paid, got %v", body["status"])
	}
}

func TestPayPaymentOrder_Conflict_NotApproved(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockPaymentOrderUsecase{
		payFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrConflict
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/pay", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.PayPaymentOrder(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPayPaymentOrder_NotFound(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockPaymentOrderUsecase{
		payFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/pay", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.PayPaymentOrder(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// RejectPaymentOrder handler tests
// ---------------------------------------------------------------------------

func TestRejectPaymentOrder_OK(t *testing.T) {
	tenantID := uuid.New()
	po := fakePaymentOrder(tenantID)
	po.Status = domain.PaymentOrderStatusRejected

	uc := &mockPaymentOrderUsecase{
		rejectFn: func(_ context.Context, _, _, _ uuid.UUID) (*domain.PaymentOrder, error) {
			return &po, nil
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+po.ID.String()+"/reject", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", po.ID.String())
	rr := httptest.NewRecorder()

	h.RejectPaymentOrder(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["status"] != "rejected" {
		t.Errorf("expected status=rejected, got %v", body["status"])
	}
}

func TestRejectPaymentOrder_WithUserID(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	po := fakePaymentOrder(tenantID)
	po.Status = domain.PaymentOrderStatusRejected

	var capturedRejectedBy uuid.UUID
	uc := &mockPaymentOrderUsecase{
		rejectFn: func(_ context.Context, _, _, rejectedBy uuid.UUID) (*domain.PaymentOrder, error) {
			capturedRejectedBy = rejectedBy
			return &po, nil
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+po.ID.String()+"/reject", nil)
	req = withTenantAndUser(req, tenantID, userID)
	req = withChiParam(req, "id", po.ID.String())
	rr := httptest.NewRecorder()

	h.RejectPaymentOrder(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if capturedRejectedBy != userID {
		t.Errorf("expected rejected_by=%s, got %s", userID, capturedRejectedBy)
	}
}

func TestRejectPaymentOrder_Conflict(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockPaymentOrderUsecase{
		rejectFn: func(_ context.Context, _, _, _ uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrConflict
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/reject", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.RejectPaymentOrder(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRejectPaymentOrder_NotFound(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockPaymentOrderUsecase{
		rejectFn: func(_ context.Context, _, _, _ uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/reject", nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.RejectPaymentOrder(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestRejectPaymentOrder_Unauthorized(t *testing.T) {
	uc := &mockPaymentOrderUsecase{}
	h := handler.NewPaymentOrderHandler(uc)

	poID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/"+poID.String()+"/reject", nil)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.RejectPaymentOrder(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRejectPaymentOrder_InvalidUUID(t *testing.T) {
	uc := &mockPaymentOrderUsecase{}
	h := handler.NewPaymentOrderHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders/not-a-uuid/reject", nil), uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.RejectPaymentOrder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Decimal precision round-trip test
// ---------------------------------------------------------------------------

func TestCreatePaymentOrder_DecimalPrecisionPreserved(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()

	const amountStr = "5000000.50"
	// decimal.Decimal.String() normalizes trailing zeros, so "5000000.50" becomes "5000000.5".
	want, _ := decimal.NewFromString(amountStr)

	var capturedParams domain.CreatePaymentOrderParams

	uc := &mockPaymentOrderUsecase{
		createFn: func(_ context.Context, p domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
			capturedParams = p
			po := fakePaymentOrder(tenantID)
			po.Amount = p.Amount
			po.InvoiceID = p.InvoiceID
			return &po, nil
		},
	}
	h := handler.NewPaymentOrderHandler(uc)

	body := `{"invoice_id":"` + invoiceID.String() + `","amount":"5000000.50","currency":"IDR"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.CreatePaymentOrder(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if !capturedParams.Amount.Equal(want) {
		t.Errorf("decimal precision lost: got %s, want %s", capturedParams.Amount.String(), amountStr)
	}

	var respBody map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	gotAmtStr, ok := respBody["amount"].(string)
	if !ok {
		t.Errorf("response amount must be JSON string, got %T", respBody["amount"])
	} else if gotAmtStr != want.String() {
		t.Errorf("response amount: got %q, want %q", gotAmtStr, want.String())
	}
}
