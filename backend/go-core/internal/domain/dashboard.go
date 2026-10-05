package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DashboardSummary holds all aggregated metrics for the dashboard endpoint.
type DashboardSummary struct {
	Invoices       InvoiceStats     `json:"invoices"`
	Payments       PaymentStats     `json:"payments"`
	Vendors        VendorStats      `json:"vendors"`
	PurchaseOrders POStats          `json:"purchase_orders"`
	GoodsReceipts  GRStats          `json:"goods_receipts"`
	MonthlyTrend   []MonthlyTrend   `json:"monthly_trend"`
	TopVendors     []TopVendor      `json:"top_vendors"`
	POS            POSDashStats     `json:"pos"`
	WMS            WMSDashStats     `json:"wms"`
	Customers      CustomerStats    `json:"customers"`
	SalesInvoices  SalesInvoiceDashStats `json:"sales_invoices"`
	SalesOrders    SalesOrderDashStats   `json:"sales_orders"`
	// Financial is derived by the usecase from the raw stats above.
	Financial FinancialOverview `json:"financial_overview"`
}

// SalesOrderDashStats holds Order-to-Cash sales order counts.
type SalesOrderDashStats struct {
	Total     int `json:"total"`
	Confirmed int `json:"confirmed"`
	Pending   int `json:"pending"`
}

// FinancialOverview combines sales (OTC + POS) and procurement (P2P) money flows.
// Revenue is sourced from sales_invoices only, because every POS checkout also
// writes a PAID "INV-POS" sales invoice; summing pos_orders too would double count.
type FinancialOverview struct {
	TotalRevenue       decimal.Decimal `json:"total_revenue"`
	CashInflow         decimal.Decimal `json:"cash_inflow"`
	AccountsReceivable decimal.Decimal `json:"accounts_receivable"`
	TotalExpense       decimal.Decimal `json:"total_expense"`
	CashOutflow        decimal.Decimal `json:"cash_outflow"`
	AccountsPayable    decimal.Decimal `json:"accounts_payable"`
	NetCashBalance     decimal.Decimal `json:"net_cash_balance"`
}

// POSDashStats holds aggregated POS (kasir) metrics for the dashboard.
type POSDashStats struct {
	TodayRevenue      decimal.Decimal    `json:"today_revenue"`
	TodayOrdersCount  int                `json:"today_orders_count"`
	TotalRevenue      decimal.Decimal    `json:"total_revenue"`
	TotalOrdersCount  int                `json:"total_orders_count"`
	AverageBasketSize decimal.Decimal    `json:"average_basket_size"`
	RecentOrders      []POSRecentOrder   `json:"recent_orders"`
}

// POSRecentOrder is a slim view of a POS order for the dashboard's recent list.
type POSRecentOrder struct {
	OrderNumber   string          `json:"order_number"`
	CustomerName  string          `json:"customer_name"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	PaymentMethod string          `json:"payment_method"`
	Status        string          `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
}

// WMSDashStats holds aggregated warehouse/inventory metrics for the dashboard.
type WMSDashStats struct {
	TotalSKUs          int             `json:"total_skus"`
	TotalPhysicalUnits decimal.Decimal `json:"total_physical_units"`
	TotalWarehouses    int             `json:"total_warehouses"`
	TotalLocations     int             `json:"total_locations"`
	TodayMovements     int             `json:"today_movements"`
	// TotalStockValue = Σ(on-hand qty × product selling price). Products have no
	// cost column, so this is a retail-price valuation, not a cost valuation.
	TotalStockValue decimal.Decimal `json:"total_stock_value"`
	LowStockItems   []LowStockItem  `json:"low_stock_items"`
}

// LowStockItem represents a product whose total stock is at or below the
// low-stock threshold used by the dashboard warning widget.
type LowStockItem struct {
	SKU           string          `json:"sku"`
	Name          string          `json:"name"`
	CurrentStock  decimal.Decimal `json:"current_stock"`
	MinThreshold  decimal.Decimal `json:"min_threshold"`
}

// CustomerStats holds aggregated customer metrics for the dashboard.
type CustomerStats struct {
	Active int `json:"active"`
}

// SalesInvoiceDashStats holds aggregated Order-to-Cash (AR) metrics.
type SalesInvoiceDashStats struct {
	TotalInvoiced       decimal.Decimal `json:"total_invoiced"`
	PaidAmount          decimal.Decimal `json:"paid_amount"`
	AccountsReceivable  decimal.Decimal `json:"accounts_receivable"`
	TotalCount          int             `json:"total_count"`
}

// InvoiceStats holds aggregated invoice metrics.
type InvoiceStats struct {
	Total          int             `json:"total"`
	Pending        int             `json:"pending"`
	Approved       int             `json:"approved"`
	Rejected       int             `json:"rejected"`
	PendingReview  int             `json:"pending_review"`
	TotalAmount    decimal.Decimal `json:"total_amount"`
	ApprovedAmount decimal.Decimal `json:"approved_amount"`
}

// PaymentStats holds aggregated payment order metrics.
type PaymentStats struct {
	Total           int             `json:"total"`
	Paid            int             `json:"paid"`
	PaidAmount      decimal.Decimal `json:"paid_amount"`
	PendingAmount   decimal.Decimal `json:"pending_amount"`
}

// VendorStats holds aggregated vendor metrics.
type VendorStats struct {
	Active int `json:"active"`
}

// POStats holds aggregated purchase order metrics.
type POStats struct {
	Total int `json:"total"`
}

// GRStats holds aggregated goods receipt metrics.
type GRStats struct {
	Total int `json:"total"`
}

// MonthlyTrend represents one month of invoice data for chart rendering.
type MonthlyTrend struct {
	Month        string          `json:"month"`
	InvoiceCount int             `json:"invoice_count"`
	TotalAmount  decimal.Decimal `json:"total_amount"`
}

// TopVendor represents a vendor ranked by total invoice amount.
type TopVendor struct {
	VendorID     string          `json:"vendor_id"`
	VendorName   string          `json:"vendor_name"`
	InvoiceCount int             `json:"invoice_count"`
	TotalAmount  decimal.Decimal `json:"total_amount"`
}

// DashboardRepository is the persistence port for dashboard aggregation queries.
type DashboardRepository interface {
	GetSummary(ctx context.Context, tenantID uuid.UUID) (*DashboardSummary, error)
}
