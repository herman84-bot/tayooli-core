package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type mockVendorUsecase struct {
	createFn func(ctx context.Context, params domain.CreateVendorParams) (*domain.Vendor, error)
	listFn   func(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error)
	getFn    func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Vendor, error)
	updateFn func(ctx context.Context, params domain.UpdateVendorParams) (*domain.Vendor, error)
	deleteFn func(ctx context.Context, id, tenantID uuid.UUID, deletedBy *uuid.UUID) error
	rateFn   func(ctx context.Context, params domain.AddVendorRatingParams) (*domain.VendorRating, error)
}

func (m *mockVendorUsecase) CreateVendor(ctx context.Context, params domain.CreateVendorParams) (*domain.Vendor, error) {
	return m.createFn(ctx, params)
}
func (m *mockVendorUsecase) ListVendors(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error) {
	return m.listFn(ctx, tenantID, search, page, perPage)
}
func (m *mockVendorUsecase) GetVendor(ctx context.Context, id, tenantID uuid.UUID) (*domain.Vendor, error) {
	return m.getFn(ctx, id, tenantID)
}
func (m *mockVendorUsecase) UpdateVendor(ctx context.Context, params domain.UpdateVendorParams) (*domain.Vendor, error) {
	return m.updateFn(ctx, params)
}
func (m *mockVendorUsecase) DeleteVendor(ctx context.Context, id, tenantID uuid.UUID, deletedBy *uuid.UUID) error {
	return m.deleteFn(ctx, id, tenantID, deletedBy)
}
func (m *mockVendorUsecase) RateVendor(ctx context.Context, params domain.AddVendorRatingParams) (*domain.VendorRating, error) {
	return m.rateFn(ctx, params)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func withTenantAndUserVendor(r *http.Request, tenantID, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), appMiddleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, appMiddleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func fakeVendorDomain(tenantID uuid.UUID) domain.Vendor {
	return domain.Vendor{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        "PT Vendor Sejahtera",
		Status:      domain.VendorStatusActive,
		AvgRating:   decimal.NewFromFloat(4.50),
		RatingCount: 10,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ---------------------------------------------------------------------------
// ListVendors handler tests
// ---------------------------------------------------------------------------

func TestListVendors_OK(t *testing.T) {
	tenantID := uuid.New()
	v := fakeVendorDomain(tenantID)

	uc := &mockVendorUsecase{
		listFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) (*domain.VendorListPage, error) {
			return &domain.VendorListPage{
				Data:    []domain.Vendor{v},
				Total:   1,
				Page:    1,
				PerPage: 20,
			}, nil
		},
	}
	h := handler.NewVendorHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/vendors", nil), tenantID)
	rr := httptest.NewRecorder()

	h.ListVendors(rr, req)

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
		t.Errorf("expected 1 vendor, got %d", len(data))
	}
}

func TestListVendors_WithSearch(t *testing.T) {
	tenantID := uuid.New()
	var capturedSearch string

	uc := &mockVendorUsecase{
		listFn: func(_ context.Context, _ uuid.UUID, search string, _, _ int) (*domain.VendorListPage, error) {
			capturedSearch = search
			return &domain.VendorListPage{Data: []domain.Vendor{}, Total: 0, Page: 1, PerPage: 20}, nil
		},
	}
	h := handler.NewVendorHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/vendors?q=PT+Maju", nil), tenantID)
	rr := httptest.NewRecorder()

	h.ListVendors(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if capturedSearch != "PT Maju" {
		t.Errorf("expected search 'PT Maju', got %q", capturedSearch)
	}
}

func TestListVendors_Unauthorized(t *testing.T) {
	uc := &mockVendorUsecase{}
	h := handler.NewVendorHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendors", nil)
	rr := httptest.NewRecorder()

	h.ListVendors(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestListVendors_InternalError(t *testing.T) {
	uc := &mockVendorUsecase{
		listFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) (*domain.VendorListPage, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := handler.NewVendorHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/vendors", nil), uuid.New())
	rr := httptest.NewRecorder()

	h.ListVendors(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetVendor handler tests
// ---------------------------------------------------------------------------

func TestGetVendor_OK(t *testing.T) {
	tenantID := uuid.New()
	v := fakeVendorDomain(tenantID)

	uc := &mockVendorUsecase{
		getFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Vendor, error) {
			return &v, nil
		},
	}
	h := handler.NewVendorHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/vendors/"+v.ID.String(), nil), tenantID)
	req = withChiParam(req, "id", v.ID.String())
	rr := httptest.NewRecorder()

	h.GetVendor(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["id"] != v.ID.String() {
		t.Errorf("expected id=%s, got %v", v.ID.String(), body["id"])
	}
}

func TestGetVendor_NotFound(t *testing.T) {
	tenantID := uuid.New()
	uc := &mockVendorUsecase{
		getFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Vendor, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/vendors/"+vendorID.String(), nil), tenantID)
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.GetVendor(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetVendor_InvalidUUID(t *testing.T) {
	uc := &mockVendorUsecase{}
	h := handler.NewVendorHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/vendors/not-a-uuid", nil), uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.GetVendor(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// CreateVendor handler tests
// ---------------------------------------------------------------------------

func TestCreateVendor_OK(t *testing.T) {
	tenantID := uuid.New()
	v := fakeVendorDomain(tenantID)

	uc := &mockVendorUsecase{
		createFn: func(_ context.Context, p domain.CreateVendorParams) (*domain.Vendor, error) {
			v := fakeVendorDomain(p.TenantID)
			v.Name = p.Name
			return &v, nil
		},
	}
	h := handler.NewVendorHandler(uc)

	body := `{"name":"PT Vendor Sejahtera","email":"vendor@example.com","phone":"+62812345678"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.CreateVendor(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	_ = v // silence unused
}

func TestCreateVendor_BadRequest_MalformedJSON(t *testing.T) {
	uc := &mockVendorUsecase{}
	h := handler.NewVendorHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreateVendor(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateVendor_Unauthorized(t *testing.T) {
	uc := &mockVendorUsecase{}
	h := handler.NewVendorHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewBufferString(`{"name":"test"}`))
	rr := httptest.NewRecorder()

	h.CreateVendor(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCreateVendor_EmptyName(t *testing.T) {
	uc := &mockVendorUsecase{
		createFn: func(_ context.Context, _ domain.CreateVendorParams) (*domain.Vendor, error) {
			return nil, fmt.Errorf("name is required: %w", domain.ErrInvalidInput)
		},
	}
	h := handler.NewVendorHandler(uc)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	rr := httptest.NewRecorder()

	h.CreateVendor(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// UpdateVendor handler tests
// ---------------------------------------------------------------------------

func TestUpdateVendor_OK(t *testing.T) {
	tenantID := uuid.New()
	vendorID := uuid.New()
	v := fakeVendorDomain(tenantID)
	v.ID = vendorID
	v.Name = "Updated Name"

	uc := &mockVendorUsecase{
		updateFn: func(_ context.Context, p domain.UpdateVendorParams) (*domain.Vendor, error) {
			v := fakeVendorDomain(p.TenantID)
			v.ID = p.ID
			v.Name = p.Name
			return &v, nil
		},
	}
	h := handler.NewVendorHandler(uc)

	body := `{"name":"Updated Name","email":"updated@example.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/vendors/"+vendorID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.UpdateVendor(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var respBody map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if respBody["name"] != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %v", respBody["name"])
	}
}

func TestUpdateVendor_NotFound(t *testing.T) {
	uc := &mockVendorUsecase{
		updateFn: func(_ context.Context, _ domain.UpdateVendorParams) (*domain.Vendor, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	body := `{"name":"Test"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/vendors/"+vendorID.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.UpdateVendor(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateVendor_InvalidUUID(t *testing.T) {
	uc := &mockVendorUsecase{}
	h := handler.NewVendorHandler(uc)

	body := `{"name":"Test"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/vendors/not-a-uuid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.UpdateVendor(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// DeleteVendor handler tests
// ---------------------------------------------------------------------------

func TestDeleteVendor_OK(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	vendorID := uuid.New()

	var capturedDeletedBy *uuid.UUID
	uc := &mockVendorUsecase{
		deleteFn: func(_ context.Context, _, _ uuid.UUID, deletedBy *uuid.UUID) error {
			capturedDeletedBy = deletedBy
			return nil
		},
	}
	h := handler.NewVendorHandler(uc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/vendors/"+vendorID.String(), nil)
	req = withTenantAndUserVendor(req, tenantID, userID)
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.DeleteVendor(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if capturedDeletedBy == nil || *capturedDeletedBy != userID {
		t.Errorf("expected deleted_by=%s, got %v", userID, capturedDeletedBy)
	}
}

func TestDeleteVendor_NotFound(t *testing.T) {
	uc := &mockVendorUsecase{
		deleteFn: func(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID) error {
			return domain.ErrNotFound
		},
	}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	req := withTenant(httptest.NewRequest(http.MethodDelete, "/api/v1/vendors/"+vendorID.String(), nil), uuid.New())
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.DeleteVendor(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDeleteVendor_Conflict_AlreadyInactive(t *testing.T) {
	uc := &mockVendorUsecase{
		deleteFn: func(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID) error {
			return domain.ErrConflict
		},
	}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	req := withTenant(httptest.NewRequest(http.MethodDelete, "/api/v1/vendors/"+vendorID.String(), nil), uuid.New())
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.DeleteVendor(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestDeleteVendor_Unauthorized(t *testing.T) {
	uc := &mockVendorUsecase{}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/vendors/"+vendorID.String(), nil)
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.DeleteVendor(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// RateVendor handler tests
// ---------------------------------------------------------------------------

func TestRateVendor_OK(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	vendorID := uuid.New()

	var capturedRatedBy *uuid.UUID
	uc := &mockVendorUsecase{
		rateFn: func(_ context.Context, p domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			capturedRatedBy = p.RatedBy
			return &domain.VendorRating{
				ID:       uuid.New(),
				TenantID: p.TenantID,
				VendorID: p.VendorID,
				RatedBy:  p.RatedBy,
				Rating:   p.Rating,
				Comment:  p.Comment,
				CreatedAt: time.Now(),
			}, nil
		},
	}
	h := handler.NewVendorHandler(uc)

	body := `{"rating":5,"comment":"Excellent vendor"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+vendorID.String()+"/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantAndUserVendor(req, tenantID, userID)
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.RateVendor(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if capturedRatedBy == nil || *capturedRatedBy != userID {
		t.Errorf("expected rated_by=%s, got %v", userID, capturedRatedBy)
	}
}

func TestRateVendor_InvalidRating_TooLow(t *testing.T) {
	uc := &mockVendorUsecase{
		rateFn: func(_ context.Context, _ domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			return nil, fmt.Errorf("rating must be between 1 and 5: %w", domain.ErrInvalidInput)
		},
	}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	body := `{"rating":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+vendorID.String()+"/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.RateVendor(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRateVendor_InvalidRating_TooHigh(t *testing.T) {
	uc := &mockVendorUsecase{
		rateFn: func(_ context.Context, _ domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			return nil, fmt.Errorf("rating must be between 1 and 5: %w", domain.ErrInvalidInput)
		},
	}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	body := `{"rating":6}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+vendorID.String()+"/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.RateVendor(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRateVendor_Conflict_AlreadyRated(t *testing.T) {
	uc := &mockVendorUsecase{
		rateFn: func(_ context.Context, _ domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			return nil, domain.ErrConflict
		},
	}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	body := `{"rating":4}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+vendorID.String()+"/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, uuid.New())
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.RateVendor(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestRateVendor_Unauthorized(t *testing.T) {
	uc := &mockVendorUsecase{}
	h := handler.NewVendorHandler(uc)

	vendorID := uuid.New()
	body := `{"rating":3}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+vendorID.String()+"/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withChiParam(req, "id", vendorID.String())
	rr := httptest.NewRecorder()

	h.RateVendor(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRateVendor_InvalidUUID(t *testing.T) {
	uc := &mockVendorUsecase{}
	h := handler.NewVendorHandler(uc)

	body := `{"rating":3}`
	req := withTenant(httptest.NewRequest(http.MethodPost, "/api/v1/vendors/not-a-uuid/rate", bytes.NewBufferString(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	req = withChiParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.RateVendor(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Response shape tests
// ---------------------------------------------------------------------------

func TestCreateVendor_ResponseShape(t *testing.T) {
	tenantID := uuid.New()

	uc := &mockVendorUsecase{
		createFn: func(_ context.Context, p domain.CreateVendorParams) (*domain.Vendor, error) {
			v := fakeVendorDomain(p.TenantID)
			v.Name = p.Name
			email := "test@example.com"
			v.Email = &email
			return &v, nil
		},
	}
	h := handler.NewVendorHandler(uc)

	body := `{"name":"PT Test","email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenant(req, tenantID)
	rr := httptest.NewRecorder()

	h.CreateVendor(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var respBody map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"id", "tenant_id", "name", "status", "avg_rating", "rating_count", "created_at", "updated_at"}
	for _, field := range requiredFields {
		if _, ok := respBody[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}

	// avg_rating must be string (decimal representation)
	if _, ok := respBody["avg_rating"].(string); !ok {
		t.Errorf("avg_rating must be JSON string, got %T", respBody["avg_rating"])
	}
}

func TestListVendors_PaginationResponseShape(t *testing.T) {
	tenantID := uuid.New()

	uc := &mockVendorUsecase{
		listFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) (*domain.VendorListPage, error) {
			return &domain.VendorListPage{
				Data:    []domain.Vendor{fakeVendorDomain(tenantID)},
				Total:   1,
				Page:    1,
				PerPage: 20,
			}, nil
		},
	}
	h := handler.NewVendorHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/vendors", nil), tenantID)
	rr := httptest.NewRecorder()

	h.ListVendors(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	for _, field := range []string{"data", "total", "page", "per_page"} {
		if _, ok := body[field]; !ok {
			t.Errorf("pagination response missing field %q", field)
		}
	}
}
