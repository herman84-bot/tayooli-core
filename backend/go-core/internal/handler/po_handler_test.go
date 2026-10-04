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
// Mock PO usecase — structurally satisfies handler's private poUsecase interface
// ---------------------------------------------------------------------------

type mockPOUsecase struct {
	createFn     func(ctx context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error)
	listFn       func(ctx context.Context, tenantID uuid.UUID) ([]domain.PurchaseOrder, error)
	listPagedFn  func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error)
	getByIDFn    func(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error)
}

func (m *mockPOUsecase) Create(ctx context.Context, params domain.CreatePOParams) (*domain.PurchaseOrder, error) {
	return m.createFn(ctx, params)
}
func (m *mockPOUsecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.PurchaseOrder, error) {
	return m.listFn(ctx, tenantID)
}
func (m *mockPOUsecase) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error) {
	if m.listPagedFn != nil {
		return m.listPagedFn(ctx, tenantID, page, perPage)
	}
	return &domain.PurchaseOrderListPage{Data: []domain.PurchaseOrder{}, Total: 0, Page: page, PerPage: perPage}, nil
}
func (m *mockPOUsecase) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PurchaseOrder, error) {
	return m.getByIDFn(ctx, id, tenantID)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fakePOForHandler(tenantID uuid.UUID) *domain.PurchaseOrder {
	return &domain.PurchaseOrder{
		ID:        uuid.New(),
		TenantID:  tenantID,
		VendorID:  "vendor-1",
		PONumber:  "PO-001",
		Amount:    decimal.New(500_000, 0),
		Qty:       10,
		Currency:  "IDR",
		Status:    domain.POStatusOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// POST /api/v1/purchase-orders
// ---------------------------------------------------------------------------

func TestCreatePO_OK_Returns201WithAmountAsString(t *testing.T) {
	tenantID := uuid.New()
	po := fakePOForHandler(tenantID)

	uc := &mockPOUsecase{
		createFn: func(_ context.Context, _ domain.CreatePOParams) (*domain.PurchaseOrder, error) {
			return po, nil
		},
	}
	h := handler.NewPOHandler(uc)

	body := `{"vendor_id":"vendor-1","po_number":"PO-001","amount":"500000","qty":10,"currency":"IDR"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.CreatePO(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// AC#1: amount must be a JSON string, not a number
	if _, ok := resp["amount"].(string); !ok {
		t.Errorf("expected amount to be JSON string, got %T: %v", resp["amount"], resp["amount"])
	}
	if resp["tenant_id"] != tenantID.String() {
		t.Errorf("unexpected tenant_id in response: %v", resp["tenant_id"])
	}
	if resp["status"] != "open" {
		t.Errorf("expected status=open, got %v", resp["status"])
	}
}

func TestCreatePO_Unauthorized_NoTenant_Returns401(t *testing.T) {
	uc := &mockPOUsecase{}
	h := handler.NewPOHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-orders", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()

	h.CreatePO(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCreatePO_BadRequestOnMalformedJSON(t *testing.T) {
	uc := &mockPOUsecase{}
	h := handler.NewPOHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-orders", bytes.NewBufferString("not-json{{{"))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreatePO(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCreatePO_BadRequestOnInvalidAmount(t *testing.T) {
	uc := &mockPOUsecase{}
	h := handler.NewPOHandler(uc)

	cases := []struct {
		name string
		body string
	}{
		{"non-numeric", `{"vendor_id":"v1","po_number":"PO-001","amount":"abc","qty":1}`},
		{"zero", `{"vendor_id":"v1","po_number":"PO-001","amount":"0","qty":1}`},
		{"negative", `{"vendor_id":"v1","po_number":"PO-001","amount":"-100","qty":1}`},
		{"empty string", `{"vendor_id":"v1","po_number":"PO-001","amount":"","qty":1}`},
		{"too many decimal places", `{"vendor_id":"v1","po_number":"PO-001","amount":"0.000049","qty":1}`},
		{"overflow", `{"vendor_id":"v1","po_number":"PO-001","amount":"10000000000000000.9999","qty":1}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-orders", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = withTenant(req, uuid.New())
			rr := httptest.NewRecorder()

			h.CreatePO(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestCreatePO_UnprocessableOnErrInvalidInput(t *testing.T) {
	uc := &mockPOUsecase{
		createFn: func(_ context.Context, _ domain.CreatePOParams) (*domain.PurchaseOrder, error) {
			return nil, domain.ErrInvalidInput
		},
	}
	h := handler.NewPOHandler(uc)

	// amount passes handler validation, invalid input surfaces from usecase
	body := `{"vendor_id":"","po_number":"","amount":"100","qty":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreatePO(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreatePO_InternalError_Returns500(t *testing.T) {
	uc := &mockPOUsecase{
		createFn: func(_ context.Context, _ domain.CreatePOParams) (*domain.PurchaseOrder, error) {
			return nil, errors.New("db connection refused")
		},
	}
	h := handler.NewPOHandler(uc)

	body := `{"vendor_id":"vendor-1","po_number":"PO-001","amount":"100","qty":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/purchase-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreatePO(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/purchase-orders
// ---------------------------------------------------------------------------

// paginatedEnvelope is the JSON shape returned by list endpoints.
type paginatedEnvelope struct {
	Data    []map[string]any `json:"data"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

func TestListPOs_OK_Returns200WithPaginatedList(t *testing.T) {
	tenantID := uuid.New()
	pos := []domain.PurchaseOrder{*fakePOForHandler(tenantID), *fakePOForHandler(tenantID)}

	uc := &mockPOUsecase{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error) {
			if tid != tenantID {
				t.Errorf("unexpected tenantID passed to usecase: %s", tid)
			}
			return &domain.PurchaseOrderListPage{Data: pos, Total: len(pos), Page: page, PerPage: perPage}, nil
		},
	}
	h := handler.NewPOHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders", nil)
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.ListPOs(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body paginatedEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Data) != 2 {
		t.Errorf("expected 2 POs, got %d", len(body.Data))
	}
	if body.Total != 2 {
		t.Errorf("expected total 2, got %d", body.Total)
	}
	if body.Page != 1 || body.PerPage != 20 {
		t.Errorf("expected defaults page=1 per_page=20, got page=%d per_page=%d", body.Page, body.PerPage)
	}
}

// AC#2: empty list for a new tenant must return empty data array in envelope
func TestListPOs_EmptyForNewTenant_Returns200EmptyArray(t *testing.T) {
	uc := &mockPOUsecase{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error) {
			return &domain.PurchaseOrderListPage{Data: []domain.PurchaseOrder{}, Total: 0, Page: page, PerPage: perPage}, nil
		},
	}
	h := handler.NewPOHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders", nil)
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.ListPOs(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body paginatedEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode empty-list response: %v", err)
	}
	if len(body.Data) != 0 || body.Data == nil {
		t.Errorf("expected empty non-nil data array, got %#v", body.Data)
	}
}

// Multi-tenancy: tenant B must not see tenant A's POs
func TestListPOs_TenantIsolation_OnlyOwnTenantIDPassedToUsecase(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	var capturedTenantID uuid.UUID
	uc := &mockPOUsecase{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.PurchaseOrderListPage, error) {
			capturedTenantID = tid
			// Simulate: only return POs for tenantA
			if tid == tenantA {
				return &domain.PurchaseOrderListPage{Data: []domain.PurchaseOrder{*fakePOForHandler(tenantA)}, Total: 1, Page: page, PerPage: perPage}, nil
			}
			return &domain.PurchaseOrderListPage{Data: []domain.PurchaseOrder{}, Total: 0, Page: page, PerPage: perPage}, nil
		},
	}
	h := handler.NewPOHandler(uc)

	// Request as tenant B — usecase must be called with tenant B's ID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders", nil)
	req = withTenant(req, tenantB)
	rr := httptest.NewRecorder()

	h.ListPOs(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if capturedTenantID != tenantB {
		t.Errorf("handler passed wrong tenantID to usecase: got %s, want %s", capturedTenantID, tenantB)
	}

	var body paginatedEnvelope
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	// Tenant B sees 0 results since usecase returned empty for B
	if len(body.Data) != 0 {
		t.Errorf("tenant B must not receive tenant A data: got %d items", len(body.Data))
	}
}

func TestListPOs_Unauthorized_NoTenant_Returns401(t *testing.T) {
	uc := &mockPOUsecase{}
	h := handler.NewPOHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders", nil)
	rr := httptest.NewRecorder()

	h.ListPOs(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestListPOs_InternalError_Returns500(t *testing.T) {
	uc := &mockPOUsecase{
		listPagedFn: func(_ context.Context, _ uuid.UUID, _ int, _ int) (*domain.PurchaseOrderListPage, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := handler.NewPOHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders", nil)
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.ListPOs(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/purchase-orders/{id}
// ---------------------------------------------------------------------------

func TestGetPO_OK_Returns200(t *testing.T) {
	tenantID := uuid.New()
	po := fakePOForHandler(tenantID)

	uc := &mockPOUsecase{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.PurchaseOrder, error) {
			if id == po.ID && tid == tenantID {
				return po, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewPOHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders/"+po.ID.String(), nil)
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", po.ID.String())
	rr := httptest.NewRecorder()

	h.GetPO(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["id"] != po.ID.String() {
		t.Errorf("expected id=%s, got %v", po.ID, body["id"])
	}
	// amount must be a string
	if _, ok := body["amount"].(string); !ok {
		t.Errorf("expected amount to be JSON string, got %T", body["amount"])
	}
}

// AC#3: GET /purchase-orders/{id} returns 404 when not found or cross-tenant
func TestGetPO_NotFound_Returns404(t *testing.T) {
	uc := &mockPOUsecase{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PurchaseOrder, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewPOHandler(uc)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders/"+id.String(), nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	h.GetPO(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// Multi-tenancy: tenant B requesting tenant A's PO ID must get 404 (not the PO data)
func TestGetPO_CrossTenantLookup_Returns404(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	poID := uuid.New()

	uc := &mockPOUsecase{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.PurchaseOrder, error) {
			// Only returns for tenant A
			if tid == tenantA && id == poID {
				return fakePOForHandler(tenantA), nil
			}
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewPOHandler(uc)

	// Request as tenant B
	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders/"+poID.String(), nil)
	req = withTenant(req, tenantB)
	req = withChiParam(req, "id", poID.String())
	rr := httptest.NewRecorder()

	h.GetPO(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("tenant B must receive 404 for tenant A PO, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetPO_Unauthorized_NoTenant_Returns401(t *testing.T) {
	uc := &mockPOUsecase{}
	h := handler.NewPOHandler(uc)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders/"+id.String(), nil)
	req = withChiParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	h.GetPO(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetPO_BadRequestOnInvalidUUID(t *testing.T) {
	uc := &mockPOUsecase{}
	h := handler.NewPOHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders/not-a-uuid", nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.GetPO(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetPO_InternalError_Returns500(t *testing.T) {
	uc := &mockPOUsecase{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PurchaseOrder, error) {
			return nil, errors.New("db timeout")
		},
	}
	h := handler.NewPOHandler(uc)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/purchase-orders/"+id.String(), nil)
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	h.GetPO(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}
