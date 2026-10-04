package dashboard_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/dashboard"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockDashboardRepo struct {
	getSummaryFn func(ctx context.Context, tenantID uuid.UUID) (*domain.DashboardSummary, error)
}

func (m *mockDashboardRepo) GetSummary(ctx context.Context, tenantID uuid.UUID) (*domain.DashboardSummary, error) {
	return m.getSummaryFn(ctx, tenantID)
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func fakeSummary() *domain.DashboardSummary {
	return &domain.DashboardSummary{
		Invoices: domain.InvoiceStats{
			Total:          150,
			Pending:        20,
			Approved:       100,
			Rejected:       10,
			PendingReview:  20,
			TotalAmount:    decimal.NewFromInt(500_000_000),
			ApprovedAmount: decimal.NewFromInt(400_000_000),
		},
		Payments: domain.PaymentStats{
			Total:         80,
			Paid:          60,
			PaidAmount:    decimal.NewFromInt(300_000_000),
			PendingAmount: decimal.NewFromInt(100_000_000),
		},
		Vendors: domain.VendorStats{
			Active: 45,
		},
		PurchaseOrders: domain.POStats{
			Total: 120,
		},
		GoodsReceipts: domain.GRStats{
			Total: 95,
		},
		MonthlyTrend: []domain.MonthlyTrend{
			{Month: "2026-01", InvoiceCount: 25, TotalAmount: decimal.NewFromInt(80_000_000)},
			{Month: "2026-02", InvoiceCount: 30, TotalAmount: decimal.NewFromInt(95_000_000)},
		},
		TopVendors: []domain.TopVendor{
			{VendorID: "v1", VendorName: "PT Maju Mundur", InvoiceCount: 15, TotalAmount: decimal.NewFromInt(120_000_000)},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestGetSummary_OK(t *testing.T) {
	tenantID := uuid.New()
	expected := fakeSummary()

	repo := &mockDashboardRepo{
		getSummaryFn: func(_ context.Context, tid uuid.UUID) (*domain.DashboardSummary, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID=%s, got %s", tenantID, tid)
			}
			return expected, nil
		},
	}

	uc := dashboard.New(repo)
	result, err := uc.GetSummary(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Invoices.Total != 150 {
		t.Errorf("expected total invoices=150, got %d", result.Invoices.Total)
	}
	if len(result.MonthlyTrend) != 2 {
		t.Errorf("expected 2 monthly trends, got %d", len(result.MonthlyTrend))
	}
	if len(result.TopVendors) != 1 {
		t.Errorf("expected 1 top vendor, got %d", len(result.TopVendors))
	}
}

func TestGetSummary_RepoError(t *testing.T) {
	repo := &mockDashboardRepo{
		getSummaryFn: func(_ context.Context, _ uuid.UUID) (*domain.DashboardSummary, error) {
			return nil, errors.New("db connection lost")
		},
	}

	uc := dashboard.New(repo)
	_, err := uc.GetSummary(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSummary_EmptyResult(t *testing.T) {
	repo := &mockDashboardRepo{
		getSummaryFn: func(_ context.Context, _ uuid.UUID) (*domain.DashboardSummary, error) {
			return &domain.DashboardSummary{
				MonthlyTrend: []domain.MonthlyTrend{},
				TopVendors:   []domain.TopVendor{},
			}, nil
		},
	}

	uc := dashboard.New(repo)
	result, err := uc.GetSummary(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Invoices.Total != 0 {
		t.Errorf("expected total invoices=0, got %d", result.Invoices.Total)
	}
}
