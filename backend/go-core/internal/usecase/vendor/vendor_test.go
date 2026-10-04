package vendor_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/vendor"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockVendorRepo struct {
	createFn       func(ctx context.Context, params domain.CreateVendorParams) (*domain.Vendor, error)
	listPagedFn    func(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error)
	getByIDFn      func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Vendor, error)
	updateFn       func(ctx context.Context, params domain.UpdateVendorParams) (*domain.Vendor, error)
	softDeleteFn   func(ctx context.Context, id, tenantID uuid.UUID) error
	addRatingFn    func(ctx context.Context, params domain.AddVendorRatingParams) (*domain.VendorRating, error)
	existsByNameFn func(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)
}

func (m *mockVendorRepo) Create(ctx context.Context, params domain.CreateVendorParams) (*domain.Vendor, error) {
	return m.createFn(ctx, params)
}
func (m *mockVendorRepo) ListPaged(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error) {
	return m.listPagedFn(ctx, tenantID, search, page, perPage)
}
func (m *mockVendorRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Vendor, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id, tenantID)
	}
	return &domain.Vendor{ID: id, TenantID: tenantID}, nil
}
func (m *mockVendorRepo) Update(ctx context.Context, params domain.UpdateVendorParams) (*domain.Vendor, error) {
	return m.updateFn(ctx, params)
}
func (m *mockVendorRepo) SoftDelete(ctx context.Context, id, tenantID uuid.UUID) error {
	return m.softDeleteFn(ctx, id, tenantID)
}
func (m *mockVendorRepo) AddRating(ctx context.Context, params domain.AddVendorRatingParams) (*domain.VendorRating, error) {
	return m.addRatingFn(ctx, params)
}
func (m *mockVendorRepo) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(ctx, tenantID, name)
	}
	return false, nil
}

type mockAuditLogRepo struct {
	entries []domain.AuditLogEntry
	err     error
}

func (m *mockAuditLogRepo) Create(_ context.Context, entry domain.AuditLogEntry) error {
	if m.err != nil {
		return m.err
	}
	m.entries = append(m.entries, entry)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fakeVendor(tenantID uuid.UUID) *domain.Vendor {
	name := "PT Vendor Sejahtera"
	return &domain.Vendor{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Status:      domain.VendorStatusActive,
		AvgRating:   decimal.Zero,
		RatingCount: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ---------------------------------------------------------------------------
// CreateUsecase tests
// ---------------------------------------------------------------------------

func TestCreateUsecase_Execute_OK(t *testing.T) {
	tenantID := uuid.New()
	audit := &mockAuditLogRepo{}
	repo := &mockVendorRepo{
		createFn: func(_ context.Context, p domain.CreateVendorParams) (*domain.Vendor, error) {
			v := fakeVendor(p.TenantID)
			v.Name = p.Name
			return v, nil
		},
	}

	uc := vendor.NewCreateUsecase(repo, audit)
	v, err := uc.Execute(context.Background(), domain.CreateVendorParams{
		TenantID: tenantID,
		Name:     "PT Vendor Sejahtera",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v == nil {
		t.Fatal("expected vendor, got nil")
	}
	if v.Name != "PT Vendor Sejahtera" {
		t.Errorf("expected name 'PT Vendor Sejahtera', got %q", v.Name)
	}
	// Verify audit log.
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "created" {
		t.Errorf("expected audit action 'created', got %q", audit.entries[0].Action)
	}
	if audit.entries[0].EntityType != "vendor" {
		t.Errorf("expected entity_type 'vendor', got %q", audit.entries[0].EntityType)
	}
}

func TestCreateUsecase_Execute_EmptyName(t *testing.T) {
	audit := &mockAuditLogRepo{}
	repo := &mockVendorRepo{}

	uc := vendor.NewCreateUsecase(repo, audit)
	_, err := uc.Execute(context.Background(), domain.CreateVendorParams{
		TenantID: uuid.New(),
		Name:     "",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateUsecase_Execute_JunkName(t *testing.T) {
	audit := &mockAuditLogRepo{}
	repo := &mockVendorRepo{}
	uc := vendor.NewCreateUsecase(repo, audit)

	for _, junk := range []string{"baru", "vendor", "perusahaan", "pemasok", "(isi nama vendor)", "a"} {
		_, err := uc.Execute(context.Background(), domain.CreateVendorParams{
			TenantID: uuid.New(),
			Name:     junk,
		})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for junk name %q, got %v", junk, err)
		}
	}
}

func TestCreateUsecase_Execute_DuplicateName(t *testing.T) {
	audit := &mockAuditLogRepo{}
	tenantID := uuid.New()
	repo := &mockVendorRepo{
		existsByNameFn: func(_ context.Context, _ uuid.UUID, name string) (bool, error) {
			return strings.EqualFold(name, "pt sudah ada"), nil
		},
	}
	uc := vendor.NewCreateUsecase(repo, audit)

	_, err := uc.Execute(context.Background(), domain.CreateVendorParams{
		TenantID: tenantID,
		Name:     "PT Sudah Ada",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict for duplicate name, got %v", err)
	}
}

func TestCreateUsecase_Execute_ContinuesIfAuditFails(t *testing.T) {
	tenantID := uuid.New()
	audit := &mockAuditLogRepo{err: errors.New("audit db down")}
	repo := &mockVendorRepo{
		createFn: func(_ context.Context, p domain.CreateVendorParams) (*domain.Vendor, error) {
			return fakeVendor(p.TenantID), nil
		},
	}

	uc := vendor.NewCreateUsecase(repo, audit)
	v, err := uc.Execute(context.Background(), domain.CreateVendorParams{
		TenantID: tenantID,
		Name:     "PT Vendor Sejahtera",
	})
	if err != nil {
		t.Errorf("audit failure should be swallowed, got %v", err)
	}
	if v == nil {
		t.Fatal("expected vendor, got nil")
	}
}

// ---------------------------------------------------------------------------
// ListUsecase tests
// ---------------------------------------------------------------------------

func TestListUsecase_ExecutePaged_OK(t *testing.T) {
	tenantID := uuid.New()
	v := fakeVendor(tenantID)

	repo := &mockVendorRepo{
		listPagedFn: func(_ context.Context, tid uuid.UUID, search string, page, perPage int) (*domain.VendorListPage, error) {
			return &domain.VendorListPage{
				Data:    []domain.Vendor{*v},
				Total:   1,
				Page:    page,
				PerPage: perPage,
			}, nil
		},
	}

	uc := vendor.NewListUsecase(repo)
	result, err := uc.ExecutePaged(context.Background(), tenantID, "", 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Data))
	}
}

func TestListUsecase_ExecutePaged_WithSearch(t *testing.T) {
	tenantID := uuid.New()

	var capturedSearch string
	repo := &mockVendorRepo{
		listPagedFn: func(_ context.Context, _ uuid.UUID, search string, _, _ int) (*domain.VendorListPage, error) {
			capturedSearch = search
			return &domain.VendorListPage{Data: []domain.Vendor{}, Total: 0, Page: 1, PerPage: 20}, nil
		},
	}

	uc := vendor.NewListUsecase(repo)
	_, err := uc.ExecutePaged(context.Background(), tenantID, "PT Maju", 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSearch != "PT Maju" {
		t.Errorf("expected search 'PT Maju', got %q", capturedSearch)
	}
}

func TestListUsecase_ExecutePaged_PropagatesError(t *testing.T) {
	repo := &mockVendorRepo{
		listPagedFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) (*domain.VendorListPage, error) {
			return nil, errors.New("db error")
		},
	}

	uc := vendor.NewListUsecase(repo)
	_, err := uc.ExecutePaged(context.Background(), uuid.New(), "", 1, 20)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// UpdateUsecase tests
// ---------------------------------------------------------------------------

func TestUpdateUsecase_Execute_OK(t *testing.T) {
	tenantID := uuid.New()
	vendorID := uuid.New()
	audit := &mockAuditLogRepo{}

	repo := &mockVendorRepo{
		updateFn: func(_ context.Context, p domain.UpdateVendorParams) (*domain.Vendor, error) {
			v := fakeVendor(p.TenantID)
			v.ID = p.ID
			v.Name = p.Name
			return v, nil
		},
	}

	uc := vendor.NewUpdateUsecase(repo, audit)
	v, err := uc.Execute(context.Background(), domain.UpdateVendorParams{
		ID:       vendorID,
		TenantID: tenantID,
		Name:     "Updated Name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %q", v.Name)
	}
	// Verify audit log.
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "updated" {
		t.Errorf("expected audit action 'updated', got %q", audit.entries[0].Action)
	}
}

func TestUpdateUsecase_Execute_EmptyName(t *testing.T) {
	repo := &mockVendorRepo{}
	audit := &mockAuditLogRepo{}

	uc := vendor.NewUpdateUsecase(repo, audit)
	_, err := uc.Execute(context.Background(), domain.UpdateVendorParams{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateUsecase_Execute_NotFound(t *testing.T) {
	repo := &mockVendorRepo{
		updateFn: func(_ context.Context, _ domain.UpdateVendorParams) (*domain.Vendor, error) {
			return nil, domain.ErrNotFound
		},
	}
	audit := &mockAuditLogRepo{}

	uc := vendor.NewUpdateUsecase(repo, audit)
	_, err := uc.Execute(context.Background(), domain.UpdateVendorParams{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Name:     "Test",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteUsecase tests
// ---------------------------------------------------------------------------

func TestDeleteUsecase_Execute_OK(t *testing.T) {
	tenantID := uuid.New()
	vendorID := uuid.New()
	deletedBy := uuid.New()
	audit := &mockAuditLogRepo{}

	repo := &mockVendorRepo{
		softDeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return nil
		},
	}

	uc := vendor.NewDeleteUsecase(repo, audit)
	err := uc.Execute(context.Background(), vendorID, tenantID, &deletedBy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify audit log.
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "deleted" {
		t.Errorf("expected audit action 'deleted', got %q", audit.entries[0].Action)
	}
	if audit.entries[0].UserID == nil || *audit.entries[0].UserID != deletedBy {
		t.Errorf("expected audit user_id %s, got %v", deletedBy, audit.entries[0].UserID)
	}
}

func TestDeleteUsecase_Execute_NotFound(t *testing.T) {
	repo := &mockVendorRepo{
		softDeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return domain.ErrNotFound
		},
	}
	audit := &mockAuditLogRepo{}

	uc := vendor.NewDeleteUsecase(repo, audit)
	err := uc.Execute(context.Background(), uuid.New(), uuid.New(), nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteUsecase_Execute_Conflict_AlreadyInactive(t *testing.T) {
	repo := &mockVendorRepo{
		softDeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return domain.ErrConflict
		},
	}
	audit := &mockAuditLogRepo{}

	uc := vendor.NewDeleteUsecase(repo, audit)
	err := uc.Execute(context.Background(), uuid.New(), uuid.New(), nil)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestDeleteUsecase_Execute_ContinuesIfAuditFails(t *testing.T) {
	tenantID := uuid.New()
	vendorID := uuid.New()
	audit := &mockAuditLogRepo{err: errors.New("audit db down")}

	repo := &mockVendorRepo{
		softDeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			return nil
		},
	}

	uc := vendor.NewDeleteUsecase(repo, audit)
	err := uc.Execute(context.Background(), vendorID, tenantID, nil)
	if err != nil {
		t.Errorf("audit failure should be swallowed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// RateUsecase tests
// ---------------------------------------------------------------------------

func TestRateUsecase_Execute_OK(t *testing.T) {
	tenantID := uuid.New()
	vendorID := uuid.New()
	ratedBy := uuid.New()
	audit := &mockAuditLogRepo{}

	repo := &mockVendorRepo{
		addRatingFn: func(_ context.Context, p domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			return &domain.VendorRating{
				ID:        uuid.New(),
				TenantID:  p.TenantID,
				VendorID:  p.VendorID,
				RatedBy:   p.RatedBy,
				Rating:    p.Rating,
				Comment:   p.Comment,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	comment := "Great vendor"
	uc := vendor.NewRateUsecase(repo, audit)
	rating, err := uc.Execute(context.Background(), domain.AddVendorRatingParams{
		TenantID: tenantID,
		VendorID: vendorID,
		RatedBy:  &ratedBy,
		Rating:   5,
		Comment:  &comment,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rating.Rating != 5 {
		t.Errorf("expected rating 5, got %d", rating.Rating)
	}
	if rating.Comment == nil || *rating.Comment != "Great vendor" {
		t.Errorf("expected comment 'Great vendor', got %v", rating.Comment)
	}
	// Verify audit log.
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "rated" {
		t.Errorf("expected audit action 'rated', got %q", audit.entries[0].Action)
	}
	if audit.entries[0].UserID == nil || *audit.entries[0].UserID != ratedBy {
		t.Errorf("expected audit user_id %s, got %v", ratedBy, audit.entries[0].UserID)
	}
}

func TestRateUsecase_Execute_VendorNotFound(t *testing.T) {
	tenantID := uuid.New()
	vendorID := uuid.New()
	audit := &mockAuditLogRepo{}

	repo := &mockVendorRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Vendor, error) {
			return nil, domain.ErrNotFound
		},
	}

	uc := vendor.NewRateUsecase(repo, audit)
	_, err := uc.Execute(context.Background(), domain.AddVendorRatingParams{
		TenantID: tenantID,
		VendorID: vendorID,
		Rating:   5,
	})
	if err == nil {
		t.Fatal("expected error for vendor belonging to another tenant, got nil")
	}
}

func TestRateUsecase_Execute_InvalidRating_TooLow(t *testing.T) {
	repo := &mockVendorRepo{}
	audit := &mockAuditLogRepo{}

	uc := vendor.NewRateUsecase(repo, audit)
	_, err := uc.Execute(context.Background(), domain.AddVendorRatingParams{
		TenantID: uuid.New(),
		VendorID: uuid.New(),
		Rating:   0,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for rating 0, got %v", err)
	}
}

func TestRateUsecase_Execute_InvalidRating_TooHigh(t *testing.T) {
	repo := &mockVendorRepo{}
	audit := &mockAuditLogRepo{}

	uc := vendor.NewRateUsecase(repo, audit)
	_, err := uc.Execute(context.Background(), domain.AddVendorRatingParams{
		TenantID: uuid.New(),
		VendorID: uuid.New(),
		Rating:   6,
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for rating 6, got %v", err)
	}
}

func TestRateUsecase_Execute_ContinuesIfAuditFails(t *testing.T) {
	tenantID := uuid.New()
	vendorID := uuid.New()
	audit := &mockAuditLogRepo{err: errors.New("audit db down")}

	repo := &mockVendorRepo{
		addRatingFn: func(_ context.Context, p domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			return &domain.VendorRating{
				ID:       uuid.New(),
				TenantID: p.TenantID,
				VendorID: p.VendorID,
				RatedBy:  p.RatedBy,
				Rating:   p.Rating,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	uc := vendor.NewRateUsecase(repo, audit)
	rating, err := uc.Execute(context.Background(), domain.AddVendorRatingParams{
		TenantID: tenantID,
		VendorID: vendorID,
		Rating:   3,
	})
	if err != nil {
		t.Errorf("audit failure should be swallowed, got %v", err)
	}
	if rating == nil {
		t.Fatal("expected rating, got nil")
	}
}

// ---------------------------------------------------------------------------
// Boundary tests: rating 1 and 5 are valid
// ---------------------------------------------------------------------------

func TestRateUsecase_Execute_BoundaryRating1(t *testing.T) {
	repo := &mockVendorRepo{
		addRatingFn: func(_ context.Context, p domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			return &domain.VendorRating{
				ID:       uuid.New(),
				TenantID: p.TenantID,
				VendorID: p.VendorID,
				Rating:   p.Rating,
				CreatedAt: time.Now(),
			}, nil
		},
	}
	audit := &mockAuditLogRepo{}

	uc := vendor.NewRateUsecase(repo, audit)
	rating, err := uc.Execute(context.Background(), domain.AddVendorRatingParams{
		TenantID: uuid.New(),
		VendorID: uuid.New(),
		Rating:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error for rating 1: %v", err)
	}
	if rating.Rating != 1 {
		t.Errorf("expected rating 1, got %d", rating.Rating)
	}
}

func TestRateUsecase_Execute_BoundaryRating5(t *testing.T) {
	repo := &mockVendorRepo{
		addRatingFn: func(_ context.Context, p domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			return &domain.VendorRating{
				ID:       uuid.New(),
				TenantID: p.TenantID,
				VendorID: p.VendorID,
				Rating:   p.Rating,
				CreatedAt: time.Now(),
			}, nil
		},
	}
	audit := &mockAuditLogRepo{}

	uc := vendor.NewRateUsecase(repo, audit)
	rating, err := uc.Execute(context.Background(), domain.AddVendorRatingParams{
		TenantID: uuid.New(),
		VendorID: uuid.New(),
		Rating:   5,
	})
	if err != nil {
		t.Fatalf("unexpected error for rating 5: %v", err)
	}
	if rating.Rating != 5 {
		t.Errorf("expected rating 5, got %d", rating.Rating)
	}
}

// ---------------------------------------------------------------------------
// Facade tests
// ---------------------------------------------------------------------------

func TestFacade_GetVendor(t *testing.T) {
	tenantID := uuid.New()
	v := fakeVendor(tenantID)

	repo := &mockVendorRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Vendor, error) {
			if id == v.ID && tid == tenantID {
				return v, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	audit := &mockAuditLogRepo{}

	f := vendor.New(repo, audit)
	got, err := f.GetVendor(context.Background(), v.ID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != v.ID {
		t.Errorf("expected ID %s, got %s", v.ID, got.ID)
	}
}

func TestFacade_GetVendor_NotFound(t *testing.T) {
	repo := &mockVendorRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Vendor, error) {
			return nil, domain.ErrNotFound
		},
	}
	audit := &mockAuditLogRepo{}

	f := vendor.New(repo, audit)
	_, err := f.GetVendor(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Audit log JSON keys (vendor entity type + action consistency)
// ---------------------------------------------------------------------------

func TestAuditLog_VendorCreateDetails(t *testing.T) {
	tenantID := uuid.New()
	audit := &mockAuditLogRepo{}
	repo := &mockVendorRepo{
		createFn: func(_ context.Context, p domain.CreateVendorParams) (*domain.Vendor, error) {
			return fakeVendor(p.TenantID), nil
		},
	}

	uc := vendor.NewCreateUsecase(repo, audit)
	_, err := uc.Execute(context.Background(), domain.CreateVendorParams{
		TenantID: tenantID,
		Name:     "PT Test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(audit.entries) == 0 {
		t.Fatal("expected audit entry")
	}

	var details map[string]string
	if err := json.Unmarshal(audit.entries[0].Details, &details); err != nil {
		t.Fatalf("failed to unmarshal audit details: %v", err)
	}

	// Verify snake_case keys
	requiredKeys := []string{"vendor_id", "name", "status"}
	for _, key := range requiredKeys {
		if _, ok := details[key]; !ok {
			t.Errorf("audit details missing key %q", key)
		}
	}
}

func TestAuditLog_VendorRateDetails(t *testing.T) {
	tenantID := uuid.New()
	vendorID := uuid.New()
	audit := &mockAuditLogRepo{}
	repo := &mockVendorRepo{
		addRatingFn: func(_ context.Context, p domain.AddVendorRatingParams) (*domain.VendorRating, error) {
			return &domain.VendorRating{
				ID:       uuid.New(),
				TenantID: p.TenantID,
				VendorID: p.VendorID,
				Rating:   p.Rating,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	uc := vendor.NewRateUsecase(repo, audit)
	_, err := uc.Execute(context.Background(), domain.AddVendorRatingParams{
		TenantID: tenantID,
		VendorID: vendorID,
		Rating:   4,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(audit.entries) == 0 {
		t.Fatal("expected audit entry")
	}

	var details map[string]string
	if err := json.Unmarshal(audit.entries[0].Details, &details); err != nil {
		t.Fatalf("failed to unmarshal audit details: %v", err)
	}

	if details["vendor_id"] != vendorID.String() {
		t.Errorf("expected vendor_id %s, got %s", vendorID.String(), details["vendor_id"])
	}
	if details["rating"] != "4" {
		t.Errorf("expected rating '4', got %q", details["rating"])
	}
}
