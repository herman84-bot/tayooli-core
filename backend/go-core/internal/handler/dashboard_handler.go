package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// dashboardUsecase groups the operations the handler needs.
type dashboardUsecase interface {
	GetSummary(ctx context.Context, tenantID uuid.UUID) (*domain.DashboardSummary, error)
}

// DashboardHandler handles HTTP requests for dashboard analytics.
type DashboardHandler struct {
	uc dashboardUsecase
}

func NewDashboardHandler(uc dashboardUsecase) *DashboardHandler {
	return &DashboardHandler{uc: uc}
}

// ── JSON response views ──────────────────────────────────────────────────────
// Amount fields are strings to preserve NUMERIC(20,4) precision in JSON.

type invoiceStatsView struct {
	Total         int    `json:"total"`
	Pending       int    `json:"pending"`
	Approved      int    `json:"approved"`
	Rejected      int    `json:"rejected"`
	PendingReview int    `json:"pending_review"`
	TotalAmount   string `json:"total_amount"`
	ApprovedAmount string `json:"approved_amount"`
}

type paymentStatsView struct {
	Total        int    `json:"total"`
	Paid         int    `json:"paid"`
	PaidAmount   string `json:"paid_amount"`
	PendingAmount string `json:"pending_amount"`
}

type vendorStatsView struct {
	Active int `json:"active"`
}

type poStatsView struct {
	Total int `json:"total"`
}

type grStatsView struct {
	Total int `json:"total"`
}

type monthlyTrendView struct {
	Month        string `json:"month"`
	InvoiceCount int    `json:"invoice_count"`
	TotalAmount  string `json:"total_amount"`
}

type topVendorView struct {
	VendorID     string `json:"vendor_id"`
	VendorName   string `json:"vendor_name"`
	InvoiceCount int    `json:"invoice_count"`
	TotalAmount  string `json:"total_amount"`
}

type posRecentOrderView struct {
	OrderNumber   string `json:"order_number"`
	CustomerName  string `json:"customer_name"`
	TotalAmount   string `json:"total_amount"`
	PaymentMethod string `json:"payment_method"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

type posStatsView struct {
	TodayRevenue     string               `json:"today_revenue"`
	TodayOrdersCount int                  `json:"today_orders_count"`
	TotalRevenue     string               `json:"total_revenue"`
	TotalOrdersCount int                  `json:"total_orders_count"`
	AverageBasket    string               `json:"average_basket_size"`
	RecentOrders     []posRecentOrderView `json:"recent_orders"`
}

type lowStockItemView struct {
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	CurrentStock string `json:"current_stock"`
	MinThreshold string `json:"min_threshold"`
}

type wmsStatsView struct {
	TotalSKUs          int                `json:"total_skus"`
	TotalPhysicalUnits string             `json:"total_physical_units"`
	TotalWarehouses    int                `json:"total_warehouses"`
	TotalLocations     int                `json:"total_locations"`
	TodayMovements     int                `json:"today_movements"`
	TotalStockValue    string             `json:"total_stock_value"`
	LowStockItems      []lowStockItemView `json:"low_stock_items"`
}

type salesOrderStatsView struct {
	Total     int `json:"total"`
	Confirmed int `json:"confirmed"`
	Pending   int `json:"pending"`
}

type financialOverviewView struct {
	TotalRevenue       string `json:"total_revenue"`
	CashInflow         string `json:"cash_inflow"`
	AccountsReceivable string `json:"accounts_receivable"`
	TotalExpense       string `json:"total_expense"`
	CashOutflow        string `json:"cash_outflow"`
	AccountsPayable    string `json:"accounts_payable"`
	NetCashBalance     string `json:"net_cash_balance"`
}

type customerStatsView struct {
	Active int `json:"active"`
}

type salesInvoiceStatsView struct {
	TotalInvoiced      string `json:"total_invoiced"`
	PaidAmount         string `json:"paid_amount"`
	AccountsReceivable string `json:"accounts_receivable"`
	TotalCount         int    `json:"total_count"`
}

type dashboardSummaryView struct {
	Invoices       invoiceStatsView      `json:"invoices"`
	Payments       paymentStatsView      `json:"payments"`
	Vendors        vendorStatsView       `json:"vendors"`
	PurchaseOrders poStatsView           `json:"purchase_orders"`
	GoodsReceipts  grStatsView           `json:"goods_receipts"`
	MonthlyTrend   []monthlyTrendView    `json:"monthly_trend"`
	TopVendors     []topVendorView       `json:"top_vendors"`
	POS            posStatsView          `json:"pos"`
	WMS            wmsStatsView          `json:"wms"`
	Customers      customerStatsView     `json:"customers"`
	SalesInvoices  salesInvoiceStatsView `json:"sales_invoices"`
	SalesOrders    salesOrderStatsView   `json:"sales_orders"`
	Financial      financialOverviewView `json:"financial_overview"`
}

func toDashboardSummaryView(s *domain.DashboardSummary) dashboardSummaryView {
	v := dashboardSummaryView{
		Invoices: invoiceStatsView{
			Total:          s.Invoices.Total,
			Pending:        s.Invoices.Pending,
			Approved:       s.Invoices.Approved,
			Rejected:       s.Invoices.Rejected,
			PendingReview:  s.Invoices.PendingReview,
			TotalAmount:    s.Invoices.TotalAmount.String(),
			ApprovedAmount: s.Invoices.ApprovedAmount.String(),
		},
		Payments: paymentStatsView{
			Total:         s.Payments.Total,
			Paid:          s.Payments.Paid,
			PaidAmount:    s.Payments.PaidAmount.String(),
			PendingAmount: s.Payments.PendingAmount.String(),
		},
		Vendors: vendorStatsView{
			Active: s.Vendors.Active,
		},
		PurchaseOrders: poStatsView{
			Total: s.PurchaseOrders.Total,
		},
		GoodsReceipts: grStatsView{
			Total: s.GoodsReceipts.Total,
		},
		MonthlyTrend: make([]monthlyTrendView, len(s.MonthlyTrend)),
		TopVendors:   make([]topVendorView, len(s.TopVendors)),
	}

	for i, mt := range s.MonthlyTrend {
		v.MonthlyTrend[i] = monthlyTrendView{
			Month:        mt.Month,
			InvoiceCount: mt.InvoiceCount,
			TotalAmount:  mt.TotalAmount.String(),
		}
	}

	for i, tv := range s.TopVendors {
		v.TopVendors[i] = topVendorView{
			VendorID:     tv.VendorID,
			VendorName:   tv.VendorName,
			InvoiceCount: tv.InvoiceCount,
			TotalAmount:  tv.TotalAmount.String(),
		}
	}

	v.POS = posStatsView{
		TodayRevenue:     s.POS.TodayRevenue.String(),
		TodayOrdersCount: s.POS.TodayOrdersCount,
		TotalRevenue:     s.POS.TotalRevenue.String(),
		TotalOrdersCount: s.POS.TotalOrdersCount,
		AverageBasket:    s.POS.AverageBasketSize.String(),
		RecentOrders:     make([]posRecentOrderView, len(s.POS.RecentOrders)),
	}
	for i, o := range s.POS.RecentOrders {
		v.POS.RecentOrders[i] = posRecentOrderView{
			OrderNumber:   o.OrderNumber,
			CustomerName:  o.CustomerName,
			TotalAmount:   o.TotalAmount.String(),
			PaymentMethod: o.PaymentMethod,
			Status:        o.Status,
			CreatedAt:     o.CreatedAt.Format(time.RFC3339),
		}
	}

	v.WMS = wmsStatsView{
		TotalSKUs:          s.WMS.TotalSKUs,
		TotalPhysicalUnits: s.WMS.TotalPhysicalUnits.String(),
		TotalWarehouses:    s.WMS.TotalWarehouses,
		TotalLocations:     s.WMS.TotalLocations,
		TodayMovements:     s.WMS.TodayMovements,
		TotalStockValue:    s.WMS.TotalStockValue.String(),
		LowStockItems:      make([]lowStockItemView, len(s.WMS.LowStockItems)),
	}
	for i, item := range s.WMS.LowStockItems {
		v.WMS.LowStockItems[i] = lowStockItemView{
			SKU:          item.SKU,
			Name:         item.Name,
			CurrentStock: item.CurrentStock.String(),
			MinThreshold: item.MinThreshold.String(),
		}
	}

	v.Customers = customerStatsView{Active: s.Customers.Active}

	v.SalesInvoices = salesInvoiceStatsView{
		TotalInvoiced:      s.SalesInvoices.TotalInvoiced.String(),
		PaidAmount:         s.SalesInvoices.PaidAmount.String(),
		AccountsReceivable: s.SalesInvoices.AccountsReceivable.String(),
		TotalCount:         s.SalesInvoices.TotalCount,
	}

	v.SalesOrders = salesOrderStatsView{
		Total:     s.SalesOrders.Total,
		Confirmed: s.SalesOrders.Confirmed,
		Pending:   s.SalesOrders.Pending,
	}

	f := s.Financial
	v.Financial = financialOverviewView{
		TotalRevenue:       f.TotalRevenue.String(),
		CashInflow:         f.CashInflow.String(),
		AccountsReceivable: f.AccountsReceivable.String(),
		TotalExpense:       f.TotalExpense.String(),
		CashOutflow:        f.CashOutflow.String(),
		AccountsPayable:    f.AccountsPayable.String(),
		NetCashBalance:     f.NetCashBalance.String(),
	}

	return v
}

// ── GetSummary ───────────────────────────────────────────────────────────────

// GetSummary handles GET /api/v1/dashboard/summary
func (h *DashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		respondError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	summary, err := h.uc.GetSummary(r.Context(), tenantID)
	if err != nil {
		log.Error().Err(err).Str("handler", "GetSummary").Msg("usecase failed")
		respondError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, toDashboardSummaryView(summary))
}

