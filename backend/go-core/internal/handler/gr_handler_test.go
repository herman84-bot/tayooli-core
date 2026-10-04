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
)

// ---------------------------------------------------------------------------
// Mock GR usecase — structurally satisfies handler's private grUsecase interface
// ---------------------------------------------------------------------------

type mockGRUsecase struct {
	createFn     func(ctx context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error)
	listFn       func(ctx context.Context, tenantID uuid.UUID) ([]domain.GoodsReceipt, error)
	listPagedFn  func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error)
	getByIDFn    func(ctx context.Context, id, tenantID uuid.UUID) (*domain.GoodsReceipt, error)
}

func (m *mockGRUsecase) Create(ctx context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error) {
	return m.createFn(ctx, params)
}
func (m *mockGRUsecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.GoodsReceipt, error) {
	return m.listFn(ctx, tenantID)
}
func (m *mockGRUsecase) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error) {
	if m.listPagedFn != nil {
		return m.listPagedFn(ctx, tenantID, page, perPage)
	}
	return &domain.GoodsReceiptListPage{Data: []domain.GoodsReceipt{}, Total: 0, Page: page, PerPage: perPage}, nil
}
func (m *mockGRUsecase) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.GoodsReceipt, error) {
	return m.getByIDFn(ctx, id, tenantID)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fakeGRForHandler(tenantID, poID uuid.UUID) *domain.GoodsReceipt {
	return &domain.GoodsReceipt{
		ID:             uuid.New(),
		TenantID:       tenantID,
		POID:           poID,
		VendorID:       "vendor-1",
		ReceivedQty:    5,
		ReceivedAmount: decimal.New(250_000, 0),
		Currency:       "IDR",
		Status:         domain.GRStatusPending,
		ReceivedAt:     time.Now(),
		CreatedAt:      time.Now(),
	}
}

// ---------------------------------------------------------------------------
// POST /api/v1/goods-receipts
// ---------------------------------------------------------------------------

func TestCreateGR_OK_Returns201WithAmountsAsString(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	gr := fakeGRForHandler(tenantID, poID)

	uc := &mockGRUsecase{
		createFn: func(_ context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error) {
			// Verify the tenantID from context was forwarded correctly
			if params.TenantID != tenantID {
				return nil, errors.New("wrong tenantID in params")
			}
			if params.POID != poID {
				return nil, errors.New("wrong poID in params")
			}
			return gr, nil
		},
	}
	h := handler.NewGRHandler(uc)

	body, _ := json.Marshal(map[string]any{
		"po_id":           poID.String(),
		"vendor_id":       "vendor-1",
		"received_qty":    5,
		"received_amount": "250000",
		"currency":        "IDR",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.CreateGR(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// AC#4: received_amount must be a JSON string, not a number
	if _, ok := resp["received_amount"].(string); !ok {
		t.Errorf("expected received_amount to be JSON string, got %T: %v", resp["received_amount"], resp["received_amount"])
	}
	if resp["tenant_id"] != tenantID.String() {
		t.Errorf("unexpected tenant_id in response: %v", resp["tenant_id"])
	}
	if resp["po_id"] != poID.String() {
		t.Errorf("expected po_id=%s, got %v", poID, resp["po_id"])
	}
}

func TestCreateGR_Unauthorized_NoTenant_Returns401(t *testing.T) {
	uc := &mockGRUsecase{}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()

	h.CreateGR(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCreateGR_BadRequestOnMalformedJSON(t *testing.T) {
	uc := &mockGRUsecase{}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewBufferString("not-json{{{"))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreateGR(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreateGR_BadRequestOnInvalidPOID(t *testing.T) {
	uc := &mockGRUsecase{}
	h := handler.NewGRHandler(uc)

	body := `{"po_id":"not-a-uuid","vendor_id":"vendor-1","received_qty":5,"received_amount":"100"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreateGR(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid po_id, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateGR_BadRequestOnInvalidAmount(t *testing.T) {
	uc := &mockGRUsecase{}
	h := handler.NewGRHandler(uc)

	validPOID := uuid.New().String()
	cases := []struct {
		name   string
		amount string
	}{
		{"non-numeric", "abc"},
		{"zero", "0"},
		{"negative", "-100"},
		{"empty", ""},
		{"too many dp", "0.000049"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{
				"po_id":           validPOID,
				"vendor_id":       "vendor-1",
				"received_qty":    5,
				"received_amount": tc.amount,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = withTenant(req, uuid.New())
			rr := httptest.NewRecorder()

			h.CreateGR(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("amount %q: expected 400, got %d: %s", tc.amount, rr.Code, rr.Body.String())
			}
		})
	}
}

// AC#7: GR creation with cross-tenant PO must return 404 (not 500, not 200)
func TestCreateGR_CrossTenantPO_Returns404(t *testing.T) {
	myTenantID := uuid.New()
	otherTenantPoID := uuid.New()

	// The GR usecase returns ErrNotFound when PO belongs to another tenant.
	// The handler must translate that into HTTP 404 — not 500 or any data leak.
	uc := &mockGRUsecase{
		createFn: func(_ context.Context, params domain.CreateGRParams) (*domain.GoodsReceipt, error) {
			// Simulate: PO exists for a different tenant — usecase returns ErrNotFound
			// to prevent cross-tenant existence probing (same response as truly absent PO).
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewGRHandler(uc)

	body, _ := json.Marshal(map[string]any{
		"po_id":           otherTenantPoID.String(),
		"vendor_id":       "vendor-1",
		"received_qty":    5,
		"received_amount": "100",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, myTenantID)
	rr := httptest.NewRecorder()

	h.CreateGR(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("cross-tenant PO must return 404, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify the response body does not contain any tenant B data or 500
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if _, hasError := resp["error"]; !hasError {
		t.Errorf("404 response body should contain 'error' field, got: %v", resp)
	}
}

func TestCreateGR_UnprocessableOnErrInvalidInput(t *testing.T) {
	uc := &mockGRUsecase{
		createFn: func(_ context.Context, _ domain.CreateGRParams) (*domain.GoodsReceipt, error) {
			return nil, domain.ErrInvalidInput
		},
	}
	h := handler.NewGRHandler(uc)

	body, _ := json.Marshal(map[string]any{
		"po_id":           uuid.New().String(),
		"vendor_id":       "",
		"received_qty":    0,
		"received_amount": "100",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreateGR(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateGR_InternalError_Returns500(t *testing.T) {
	uc := &mockGRUsecase{
		createFn: func(_ context.Context, _ domain.CreateGRParams) (*domain.GoodsReceipt, error) {
			return nil, errors.New("db connection refused")
		},
	}
	h := handler.NewGRHandler(uc)

	body, _ := json.Marshal(map[string]any{
		"po_id":           uuid.New().String(),
		"vendor_id":       "vendor-1",
		"received_qty":    5,
		"received_amount": "100",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/goods-receipts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreateGR(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/goods-receipts
// ---------------------------------------------------------------------------

func TestListGRs_OK_Returns200WithPaginatedList(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	grs := []domain.GoodsReceipt{
		*fakeGRForHandler(tenantID, poID),
		*fakeGRForHandler(tenantID, poID),
	}

	uc := &mockGRUsecase{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error) {
			if tid != tenantID {
				t.Errorf("unexpected tenantID passed to usecase: %s", tid)
			}
			return &domain.GoodsReceiptListPage{Data: grs, Total: len(grs), Page: page, PerPage: perPage}, nil
		},
	}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts", nil)
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.ListGRs(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body paginatedEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Data) != 2 {
		t.Errorf("expected 2 GRs, got %d", len(body.Data))
	}
	if body.Total != 2 {
		t.Errorf("expected total 2, got %d", body.Total)
	}
	// received_amount must be string in every item
	for i, item := range body.Data {
		if _, ok := item["received_amount"].(string); !ok {
			t.Errorf("item[%d] received_amount must be JSON string, got %T", i, item["received_amount"])
		}
	}
}

// AC#5: empty list for new tenant returns 200 with empty data array in envelope
func TestListGRs_EmptyForNewTenant_Returns200EmptyArray(t *testing.T) {
	uc := &mockGRUsecase{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error) {
			return &domain.GoodsReceiptListPage{Data: []domain.GoodsReceipt{}, Total: 0, Page: page, PerPage: perPage}, nil
		},
	}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts", nil)
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.ListGRs(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var body paginatedEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Data) != 0 || body.Data == nil {
		t.Errorf("expected empty non-nil data array, got %#v", body.Data)
	}
}

// Multi-tenancy: usecase must be called with the authenticated tenant's ID
func TestListGRs_TenantIsolation_OnlyOwnTenantIDPassedToUsecase(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	var capturedTenantID uuid.UUID
	uc := &mockGRUsecase{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.GoodsReceiptListPage, error) {
			capturedTenantID = tid
			return &domain.GoodsReceiptListPage{Data: []domain.GoodsReceipt{}, Total: 0, Page: page, PerPage: perPage}, nil
		},
	}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts", nil)
	req = withTenant(req, tenantB)
	rr := httptest.NewRecorder()

	h.ListGRs(rr, req)

	if capturedTenantID != tenantB {
		t.Errorf("handler passed wrong tenantID to usecase: got %s, want %s (tenantB)", capturedTenantID, tenantB)
	}
	if capturedTenantID == tenantA {
		t.Errorf("handler leaked tenant A's ID into tenant B's request")
	}
}

func TestListGRs_Unauthorized_NoTenant_Returns401(t *testing.T) {
	uc := &mockGRUsecase{}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts", nil)
	rr := httptest.NewRecorder()

	h.ListGRs(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestListGRs_InternalError_Returns500(t *testing.T) {
	uc := &mockGRUsecase{
		listPagedFn: func(_ context.Context, _ uuid.UUID, _ int, _ int) (*domain.GoodsReceiptListPage, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts", nil)
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.ListGRs(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/goods-receipts/{id}
// ---------------------------------------------------------------------------

func TestGetGR_OK_Returns200(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	gr := fakeGRForHandler(tenantID, poID)

	uc := &mockGRUsecase{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.GoodsReceipt, error) {
			if id == gr.ID && tid == tenantID {
				return gr, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts/"+gr.ID.String(), nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", gr.ID.String())
	rr := httptest.NewRecorder()

	h.GetGR(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["id"] != gr.ID.String() {
		t.Errorf("expected id=%s, got %v", gr.ID, body["id"])
	}
	if _, ok := body["received_amount"].(string); !ok {
		t.Errorf("expected received_amount to be JSON string, got %T", body["received_amount"])
	}
}

// AC#6: GET /goods-receipts/{id} returns 404 when not found
func TestGetGR_NotFound_Returns404(t *testing.T) {
	uc := &mockGRUsecase{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.GoodsReceipt, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewGRHandler(uc)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts/"+id.String(), nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	h.GetGR(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// Multi-tenancy: tenant B cannot retrieve tenant A's GR by ID
func TestGetGR_CrossTenantLookup_Returns404(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	grID := uuid.New()

	uc := &mockGRUsecase{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.GoodsReceipt, error) {
			if tid == tenantA && id == grID {
				return fakeGRForHandler(tenantA, uuid.New()), nil
			}
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewGRHandler(uc)

	// Tenant B requests tenant A's GR
	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts/"+grID.String(), nil)
	req = withTenant(req, tenantB)
	req = withChiParam(req, "id", grID.String())
	rr := httptest.NewRecorder()

	h.GetGR(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("tenant B must receive 404 for tenant A GR, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetGR_Unauthorized_NoTenant_Returns401(t *testing.T) {
	uc := &mockGRUsecase{}
	h := handler.NewGRHandler(uc)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts/"+id.String(), nil)
	req = withChiParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	h.GetGR(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetGR_BadRequestOnInvalidUUID(t *testing.T) {
	uc := &mockGRUsecase{}
	h := handler.NewGRHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts/not-a-uuid", nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.GetGR(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetGR_InternalError_Returns500(t *testing.T) {
	uc := &mockGRUsecase{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.GoodsReceipt, error) {
			return nil, errors.New("db timeout")
		},
	}
	h := handler.NewGRHandler(uc)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/goods-receipts/"+id.String(), nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	h.GetGR(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}
