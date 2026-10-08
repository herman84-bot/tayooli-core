package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
)

// ---------------------------------------------------------------------------
// Mock usecase
// ---------------------------------------------------------------------------

type mockDashboardUsecase struct {
	getSummaryFn func(ctx context.Context, tenantID uuid.UUID, days int) (*domain.DashboardSummary, error)
}

func (m *mockDashboardUsecase) GetSummary(ctx context.Context, tenantID uuid.UUID, days int) (*domain.DashboardSummary, error) {
	if m.getSummaryFn != nil {
		return m.getSummaryFn(ctx, tenantID, days)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func fakeDashboardSummary() *domain.DashboardSummary {
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
		POS: domain.POSDashStats{
			TodayRevenue:     decimal.NewFromInt(87_080),
			TodayOrdersCount: 2,
			TotalRevenue:     decimal.NewFromInt(87_080),
			TotalOrdersCount: 2,
			RecentOrders: []domain.POSRecentOrder{
				{OrderNumber: "POS-20261004-2569", CustomerName: "PT Ganda Dua", TotalAmount: decimal.NewFromInt(31_080), PaymentMethod: "CASH", Status: "COMPLETED"},
			},
		},
		WMS: domain.WMSDashStats{
			TotalSKUs:          3,
			TotalPhysicalUnits: decimal.NewFromInt(103),
			TotalWarehouses:    2,
			TotalLocations:     1,
			TodayMovements:     2,
			LowStockItems: []domain.LowStockItem{
				{SKU: "KOPI-TEST-001", Name: "Kopi Susu Test Enak", CurrentStock: decimal.NewFromInt(3), MinThreshold: decimal.NewFromInt(10)},
			},
		},
		Customers: domain.CustomerStats{Active: 4},
	}
}

// ---------------------------------------------------------------------------
// GetSummary handler tests
// ---------------------------------------------------------------------------

func TestGetSummary_OK(t *testing.T) {
	tenantID := uuid.New()
	summary := fakeDashboardSummary()

	uc := &mockDashboardUsecase{
		getSummaryFn: func(_ context.Context, tid uuid.UUID, _ int) (*domain.DashboardSummary, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID=%s, got %s", tenantID, tid)
			}
			return summary, nil
		},
	}
	h := handler.NewDashboardHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil), tenantID)
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify invoices section
	invoices, ok := body["invoices"].(map[string]any)
	if !ok {
		t.Fatalf("expected invoices object, got %T", body["invoices"])
	}
	if invoices["total"].(float64) != 150 {
		t.Errorf("expected total=150, got %v", invoices["total"])
	}
	if invoices["pending"].(float64) != 20 {
		t.Errorf("expected pending=20, got %v", invoices["pending"])
	}
	if invoices["total_amount"].(string) != "500000000" {
		t.Errorf("expected total_amount=500000000, got %v", invoices["total_amount"])
	}

	// Verify payments section
	payments, ok := body["payments"].(map[string]any)
	if !ok {
		t.Fatalf("expected payments object, got %T", body["payments"])
	}
	if payments["paid"].(float64) != 60 {
		t.Errorf("expected paid=60, got %v", payments["paid"])
	}

	// Verify monthly trend
	trend, ok := body["monthly_trend"].([]any)
	if !ok {
		t.Fatalf("expected monthly_trend array, got %T", body["monthly_trend"])
	}
	if len(trend) != 2 {
		t.Errorf("expected 2 monthly trends, got %d", len(trend))
	}

	// Verify top vendors
	topVendors, ok := body["top_vendors"].([]any)
	if !ok {
		t.Fatalf("expected top_vendors array, got %T", body["top_vendors"])
	}
	if len(topVendors) != 1 {
		t.Errorf("expected 1 top vendor, got %d", len(topVendors))
	}

	// Verify amounts are strings (decimal precision)
	vendor := topVendors[0].(map[string]any)
	amt, ok := vendor["total_amount"].(string)
	if !ok {
		t.Errorf("expected total_amount to be string, got %T", vendor["total_amount"])
	} else if amt != "120000000" {
		t.Errorf("expected top vendor amount=120000000, got %s", amt)
	}

	// Verify POS section
	pos, ok := body["pos"].(map[string]any)
	if !ok {
		t.Fatalf("expected pos object, got %T", body["pos"])
	}
	if pos["today_orders_count"].(float64) != 2 {
		t.Errorf("expected pos.today_orders_count=2, got %v", pos["today_orders_count"])
	}
	if pos["today_revenue"].(string) != "87080" {
		t.Errorf("expected pos.today_revenue=87080, got %v", pos["today_revenue"])
	}
	recentOrders, ok := pos["recent_orders"].([]any)
	if !ok || len(recentOrders) != 1 {
		t.Fatalf("expected 1 pos recent order, got %v", pos["recent_orders"])
	}

	// Verify WMS section
	wms, ok := body["wms"].(map[string]any)
	if !ok {
		t.Fatalf("expected wms object, got %T", body["wms"])
	}
	if wms["total_physical_units"].(string) != "103" {
		t.Errorf("expected wms.total_physical_units=103, got %v", wms["total_physical_units"])
	}
	lowStock, ok := wms["low_stock_items"].([]any)
	if !ok || len(lowStock) != 1 {
		t.Fatalf("expected 1 low stock item, got %v", wms["low_stock_items"])
	}
	item := lowStock[0].(map[string]any)
	if item["sku"].(string) != "KOPI-TEST-001" {
		t.Errorf("expected low stock sku=KOPI-TEST-001, got %v", item["sku"])
	}

	// Verify customers section
	customers, ok := body["customers"].(map[string]any)
	if !ok {
		t.Fatalf("expected customers object, got %T", body["customers"])
	}
	if customers["active"].(float64) != 4 {
		t.Errorf("expected customers.active=4, got %v", customers["active"])
	}
}

func TestGetSummary_ExposesCrossModuleSections(t *testing.T) {
	s := fakeDashboardSummary()
	s.WMS.TotalStockValue = decimal.NewFromInt(9_278_000)
	s.POS.AverageBasketSize = decimal.NewFromInt(43_540)
	s.SalesOrders = domain.SalesOrderDashStats{Total: 3, Confirmed: 2, Pending: 1}
	s.Outbound = domain.OutboundDashStats{
		QtyToday: decimal.NewFromInt(150),
		QtyMonth: decimal.NewFromInt(4500),
		TopProducts: []domain.TopOutboundProduct{
			{ProductID: uuid.New(), ProductName: "Beras Premium", ProductSKU: "BRS-01", Quantity: decimal.NewFromInt(120)},
		},
		TopCustomers: []domain.TopCustomer{
			{CustomerName: "PT Mitra Jaya", OrderCount: 5, TotalRevenue: decimal.NewFromInt(25_000_000)},
		},
	}
	s.Financial = domain.FinancialOverview{
		TotalRevenue:   decimal.NewFromInt(3_087_080),
		NetCashBalance: decimal.NewFromInt(-1_412_920),
	}
	h := handler.NewDashboardHandler(&mockDashboardUsecase{
		getSummaryFn: func(context.Context, uuid.UUID, int) (*domain.DashboardSummary, error) { return s, nil },
	})
	rr := httptest.NewRecorder()
	h.GetSummary(rr, withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary?period=7d", nil), uuid.New()))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body struct {
		Financial           map[string]any   `json:"financial_overview"`
		Sales               map[string]any   `json:"sales_orders"`
		POS                 map[string]any   `json:"pos"`
		WMS                 map[string]any   `json:"wms"`
		Outbound            map[string]any   `json:"outbound"`
		OutboundQtyToday    string           `json:"outbound_qty_today"`
		OutboundQtyMonth    string           `json:"outbound_qty_month"`
		TopOutboundProducts []map[string]any `json:"top_outbound_products"`
		TopCustomers        []map[string]any `json:"top_customers"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Financial["total_revenue"] != "3087080" || body.Financial["net_cash_balance"] != "-1412920" {
		t.Errorf("financial_overview wrong: %v", body.Financial)
	}
	if body.Sales["pending"].(float64) != 1 || body.Sales["confirmed"].(float64) != 2 {
		t.Errorf("sales_orders wrong: %v", body.Sales)
	}
	if body.POS["average_basket_size"] != "43540" {
		t.Errorf("average_basket_size wrong: %v", body.POS["average_basket_size"])
	}
	if body.WMS["total_stock_value"] != "9278000" {
		t.Errorf("total_stock_value wrong: %v", body.WMS["total_stock_value"])
	}
	if body.OutboundQtyToday != "150" || body.OutboundQtyMonth != "4500" {
		t.Errorf("outbound qty aliases wrong: today=%s, month=%s", body.OutboundQtyToday, body.OutboundQtyMonth)
	}
	if len(body.TopOutboundProducts) != 1 || body.TopOutboundProducts[0]["product_sku"] != "BRS-01" {
		t.Errorf("top_outbound_products wrong: %v", body.TopOutboundProducts)
	}
	if len(body.TopCustomers) != 1 || body.TopCustomers[0]["customer_name"] != "PT Mitra Jaya" {
		t.Errorf("top_customers wrong: %v", body.TopCustomers)
	}
}

func TestGetSummary_Unauthorized(t *testing.T) {
	uc := &mockDashboardUsecase{}
	h := handler.NewDashboardHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetSummary_InternalError(t *testing.T) {
	uc := &mockDashboardUsecase{
		getSummaryFn: func(_ context.Context, _ uuid.UUID, _ int) (*domain.DashboardSummary, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := handler.NewDashboardHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil), uuid.New())
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestGetSummary_EmptyResult(t *testing.T) {
	uc := &mockDashboardUsecase{
		getSummaryFn: func(_ context.Context, _ uuid.UUID, _ int) (*domain.DashboardSummary, error) {
			return &domain.DashboardSummary{
				MonthlyTrend: []domain.MonthlyTrend{},
				TopVendors:   []domain.TopVendor{},
			}, nil
		},
	}
	h := handler.NewDashboardHandler(uc)

	req := withTenant(httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil), uuid.New())
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Empty arrays should be "[]" not null
	trend, ok := body["monthly_trend"].([]any)
	if !ok {
		t.Fatalf("expected monthly_trend array, got %T", body["monthly_trend"])
	}
	if len(trend) != 0 {
		t.Errorf("expected 0 monthly trends, got %d", len(trend))
	}
}
