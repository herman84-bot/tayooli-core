package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// DashboardRepo implements domain.DashboardRepository using database/sql.
// All queries enforce tenant isolation via both application-level WHERE clauses
// and RLS through SET LOCAL app.current_tenant_id.
type DashboardRepo struct {
	db *sql.DB
}

func NewDashboardRepo(db *sql.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

// ── SQL Queries ──────────────────────────────────────────────────────────────

const dashboardInvoiceStats = `
SELECT
  COUNT(*)                                                    AS total,
  COUNT(*) FILTER (WHERE status = 'pending')                  AS pending,
  COUNT(*) FILTER (WHERE status = 'approved')                 AS approved,
  COUNT(*) FILTER (WHERE status = 'rejected')                 AS rejected,
  COUNT(*) FILTER (WHERE status = 'pending_review')           AS pending_review,
  COALESCE(SUM(amount), 0)                                    AS total_amount,
  COALESCE(SUM(amount) FILTER (WHERE status = 'approved'), 0) AS approved_amount
FROM invoices
WHERE tenant_id = $1`

const dashboardPaymentStats = `
SELECT
  COUNT(*)                                                    AS total,
  COUNT(*) FILTER (WHERE status = 'paid')                     AS paid,
  COALESCE(SUM(amount) FILTER (WHERE status = 'paid'), 0)     AS paid_amount,
  COALESCE(SUM(amount) FILTER (WHERE status = 'approved'), 0) AS pending_amount
FROM payment_orders
WHERE tenant_id = $1`

const dashboardVendorCount = `
SELECT COUNT(*) FROM vendors WHERE tenant_id = $1 AND status = 'active'`

const dashboardPOCount = `
SELECT COUNT(*) FROM purchase_orders WHERE tenant_id = $1`

const dashboardGRCount = `
SELECT COUNT(*) FROM goods_receipts WHERE tenant_id = $1`

const dashboardMonthlyTrend = `
SELECT
  TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month,
  COUNT(*)                                             AS invoice_count,
  COALESCE(SUM(amount), 0)                             AS total_amount
FROM invoices
WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '6 months'
GROUP BY DATE_TRUNC('month', created_at)
ORDER BY month`

const dashboardTopVendors = `
SELECT
  i.vendor_id,
  COALESCE(v.name, i.vendor_id) AS vendor_name,
  COUNT(*)                       AS invoice_count,
  COALESCE(SUM(i.amount), 0)     AS total_amount
FROM invoices i
LEFT JOIN vendors v ON v.id::text = i.vendor_id AND v.tenant_id = i.tenant_id
WHERE i.tenant_id = $1
GROUP BY i.vendor_id, v.name
ORDER BY total_amount DESC
LIMIT 5`

// ── POS (Point of Sale) metrics ─────────────────────────────────────────────
// pos_orders / pos_order_items come from migration 026_pos_and_wms_ledger.sql.

const dashboardPOSStats = `
SELECT
  COALESCE(SUM(total_amount) FILTER (WHERE created_at >= CURRENT_DATE), 0) AS today_revenue,
  COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE)                      AS today_count,
  COALESCE(SUM(total_amount), 0)                                          AS total_revenue,
  COUNT(*)                                                                AS total_count
FROM pos_orders
WHERE tenant_id = $1 AND status = 'COMPLETED'`

const dashboardPOSRecentOrders = `
SELECT
  o.order_number, COALESCE(c.name, 'Pelanggan Umum'), o.total_amount, o.payment_method, o.status, o.created_at
FROM pos_orders o
LEFT JOIN customers c ON c.id = o.customer_id AND c.tenant_id = o.tenant_id
WHERE o.tenant_id = $1
ORDER BY o.created_at DESC
LIMIT 5`

// ── WMS (Warehouse/Inventory) metrics ───────────────────────────────────────
// Uses the legacy `inventory` table (actual physical stock ledger consumed by
// POS checkout — see inventory_repo.go), not the WMS stock-movement ledger,
// since that is what POS deducts from and what the dashboard must reconcile.

const dashboardWMSCounts = `
SELECT
  (SELECT COUNT(*) FROM products WHERE tenant_id = $1)                                   AS total_skus,
  (SELECT COALESCE(SUM(quantity), 0) FROM inventory WHERE tenant_id = $1)                 AS total_units,
  (SELECT COUNT(*) FROM warehouses WHERE tenant_id = $1 AND is_active = true)             AS total_warehouses,
  (SELECT COUNT(*) FROM warehouse_locations WHERE tenant_id = $1)                         AS total_locations,
  (SELECT COUNT(*) FROM stock_movements WHERE tenant_id = $1 AND created_at >= CURRENT_DATE) AS today_movements,
  (SELECT COALESCE(SUM(i.quantity * p.price), 0)
     FROM inventory i JOIN products p ON p.id = i.product_id AND p.tenant_id = i.tenant_id
    WHERE i.tenant_id = $1)                                                               AS total_stock_value`

const dashboardSalesOrderStats = `
SELECT
  COUNT(*)                                       AS total,
  COUNT(*) FILTER (WHERE status = 'CONFIRMED')   AS confirmed,
  COUNT(*) FILTER (WHERE status = 'PENDING')     AS pending
FROM sales_orders
WHERE tenant_id = $1`

const dashboardLowStockItems = `
SELECT p.sku, p.name, COALESCE(s.quantity, 0) AS current_stock
FROM products p
LEFT JOIN (
  SELECT product_id, SUM(quantity) AS quantity
  FROM inventory
  WHERE tenant_id = $1
  GROUP BY product_id
) s ON p.id = s.product_id
WHERE p.tenant_id = $1 AND COALESCE(s.quantity, 0) <= 5
ORDER BY current_stock ASC
LIMIT 5`

const dashboardCustomerCount = `
SELECT COUNT(*) FROM customers WHERE tenant_id = $1`

const dashboardSalesInvoiceStats = `
SELECT
  COALESCE(SUM(amount), 0)                                      AS total_invoiced,
  COALESCE(SUM(amount) FILTER (WHERE status = 'PAID'), 0)       AS paid_amount,
  COALESCE(SUM(amount) FILTER (WHERE status = 'UNPAID'), 0)     AS accounts_receivable,
  COUNT(*)                                                      AS total_count
FROM sales_invoices
WHERE tenant_id = $1`

const dashboardOutboundCounts = `
SELECT
  COALESCE(SUM(quantity) FILTER (WHERE created_at >= CURRENT_DATE), 0) AS qty_today,
  COALESCE(SUM(quantity) FILTER (WHERE created_at >= DATE_TRUNC('month', CURRENT_DATE)), 0) AS qty_month
FROM stock_movements
WHERE tenant_id = $1 AND status = 'DONE'
  AND reference_type IN ('DELIVERY_ORDER', 'POS_SALE')`

const dashboardTopOutboundProducts = `
SELECT sm.product_id, COALESCE(p.name, ''), COALESCE(p.sku, ''), SUM(sm.quantity) AS total_qty
FROM stock_movements sm
JOIN products p ON p.id = sm.product_id AND p.tenant_id = sm.tenant_id
WHERE sm.tenant_id = $1 AND sm.status = 'DONE'
  AND sm.reference_type IN ('DELIVERY_ORDER', 'POS_SALE')
  AND sm.created_at >= NOW() - ($2 || ' days')::interval
GROUP BY sm.product_id, p.name, p.sku
ORDER BY total_qty DESC
LIMIT 10`

const dashboardTopCustomers = `
SELECT c.id, c.name, COUNT(*), COALESCE(SUM(so.total_amount), 0)
FROM sales_orders so
JOIN customers c ON c.id = so.customer_id AND c.tenant_id = so.tenant_id
WHERE so.tenant_id = $1 AND so.created_at >= NOW() - ($2 || ' days')::interval
GROUP BY c.id, c.name
ORDER BY SUM(so.total_amount) DESC
LIMIT 10`

// ── GetSummary ───────────────────────────────────────────────────────────────

func (r *DashboardRepo) GetSummary(ctx context.Context, tenantID uuid.UUID, days int) (*domain.DashboardSummary, error) {
	if days <= 0 {
		days = 30
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: set tenant: %w", err)
	}

	summary := &domain.DashboardSummary{}

	// 1. Invoice stats
	err = tx.QueryRowContext(ctx, dashboardInvoiceStats, tenantID).Scan(
		&summary.Invoices.Total,
		&summary.Invoices.Pending,
		&summary.Invoices.Approved,
		&summary.Invoices.Rejected,
		&summary.Invoices.PendingReview,
		&summary.Invoices.TotalAmount,
		&summary.Invoices.ApprovedAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: invoice stats: %w", err)
	}

	// 2. Payment stats
	err = tx.QueryRowContext(ctx, dashboardPaymentStats, tenantID).Scan(
		&summary.Payments.Total,
		&summary.Payments.Paid,
		&summary.Payments.PaidAmount,
		&summary.Payments.PendingAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: payment stats: %w", err)
	}

	// 3. Vendor count
	err = tx.QueryRowContext(ctx, dashboardVendorCount, tenantID).Scan(&summary.Vendors.Active)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: vendor count: %w", err)
	}

	// 4. PO count
	err = tx.QueryRowContext(ctx, dashboardPOCount, tenantID).Scan(&summary.PurchaseOrders.Total)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: PO count: %w", err)
	}

	// 5. GR count
	err = tx.QueryRowContext(ctx, dashboardGRCount, tenantID).Scan(&summary.GoodsReceipts.Total)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: GR count: %w", err)
	}

	// 6. Monthly trend (last 6 months)
	rows, err := tx.QueryContext(ctx, dashboardMonthlyTrend, tenantID)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: monthly trend: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mt domain.MonthlyTrend
		var amountStr string
		if err := rows.Scan(&mt.Month, &mt.InvoiceCount, &amountStr); err != nil {
			return nil, fmt.Errorf("DashboardRepo.GetSummary: scan monthly trend: %w", err)
		}
		amt, err := decimal.NewFromString(amountStr)
		if err != nil {
			return nil, fmt.Errorf("DashboardRepo.GetSummary: parse monthly amount %q: %w", amountStr, err)
		}
		mt.TotalAmount = amt
		summary.MonthlyTrend = append(summary.MonthlyTrend, mt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: monthly trend rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: close monthly rows: %w", err)
	}

	// 7. Top 5 vendors by amount
	rows2, err := tx.QueryContext(ctx, dashboardTopVendors, tenantID)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: top vendors: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var tv domain.TopVendor
		var amountStr string
		if err := rows2.Scan(&tv.VendorID, &tv.VendorName, &tv.InvoiceCount, &amountStr); err != nil {
			return nil, fmt.Errorf("DashboardRepo.GetSummary: scan top vendor: %w", err)
		}
		amt, err := decimal.NewFromString(amountStr)
		if err != nil {
			return nil, fmt.Errorf("DashboardRepo.GetSummary: parse vendor amount %q: %w", amountStr, err)
		}
		tv.TotalAmount = amt
		summary.TopVendors = append(summary.TopVendors, tv)
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: top vendors rows: %w", err)
	}
	if err := rows2.Close(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: close top vendor rows: %w", err)
	}

	// 8. POS stats (today + all-time revenue and order counts)
	err = tx.QueryRowContext(ctx, dashboardPOSStats, tenantID).Scan(
		&summary.POS.TodayRevenue,
		&summary.POS.TodayOrdersCount,
		&summary.POS.TotalRevenue,
		&summary.POS.TotalOrdersCount,
	)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: POS stats: %w", err)
	}

	// 9. POS recent orders (last 5)
	posRows, err := tx.QueryContext(ctx, dashboardPOSRecentOrders, tenantID)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: POS recent orders: %w", err)
	}
	defer posRows.Close()

	for posRows.Next() {
		var o domain.POSRecentOrder
		if err := posRows.Scan(&o.OrderNumber, &o.CustomerName, &o.TotalAmount, &o.PaymentMethod, &o.Status, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("DashboardRepo.GetSummary: scan POS order: %w", err)
		}
		summary.POS.RecentOrders = append(summary.POS.RecentOrders, o)
	}
	if err := posRows.Err(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: POS orders rows: %w", err)
	}
	if err := posRows.Close(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: close POS orders rows: %w", err)
	}

	// 10. WMS aggregate counts
	err = tx.QueryRowContext(ctx, dashboardWMSCounts, tenantID).Scan(
		&summary.WMS.TotalSKUs,
		&summary.WMS.TotalPhysicalUnits,
		&summary.WMS.TotalWarehouses,
		&summary.WMS.TotalLocations,
		&summary.WMS.TodayMovements,
		&summary.WMS.TotalStockValue,
	)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: WMS counts: %w", err)
	}

	// 11. Low stock items (<= 5 units on hand)
	lowStockRows, err := tx.QueryContext(ctx, dashboardLowStockItems, tenantID)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: low stock items: %w", err)
	}
	defer lowStockRows.Close()

	for lowStockRows.Next() {
		var item domain.LowStockItem
		if err := lowStockRows.Scan(&item.SKU, &item.Name, &item.CurrentStock); err != nil {
			return nil, fmt.Errorf("DashboardRepo.GetSummary: scan low stock item: %w", err)
		}
		item.MinThreshold = decimal.NewFromInt(10)
		summary.WMS.LowStockItems = append(summary.WMS.LowStockItems, item)
	}
	if err := lowStockRows.Err(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: low stock rows: %w", err)
	}
	if err := lowStockRows.Close(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: close low stock rows: %w", err)
	}

	// 12. Customer count
	err = tx.QueryRowContext(ctx, dashboardCustomerCount, tenantID).Scan(&summary.Customers.Active)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: customer count: %w", err)
	}

	// 13. Sales invoice (Order-to-Cash / AR) stats
	err = tx.QueryRowContext(ctx, dashboardSalesInvoiceStats, tenantID).Scan(
		&summary.SalesInvoices.TotalInvoiced,
		&summary.SalesInvoices.PaidAmount,
		&summary.SalesInvoices.AccountsReceivable,
		&summary.SalesInvoices.TotalCount,
	)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: sales invoice stats: %w", err)
	}

	// 14. Sales order (Order-to-Cash) counts
	err = tx.QueryRowContext(ctx, dashboardSalesOrderStats, tenantID).Scan(
		&summary.SalesOrders.Total,
		&summary.SalesOrders.Confirmed,
		&summary.SalesOrders.Pending,
	)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: sales order stats: %w", err)
	}

	// 15. Outbound metrics (CR-04a)
	err = tx.QueryRowContext(ctx, dashboardOutboundCounts, tenantID).Scan(
		&summary.Outbound.QtyToday,
		&summary.Outbound.QtyMonth,
	)
	if err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: outbound counts: %w", err)
	}

	topProdRows, err := tx.QueryContext(ctx, dashboardTopOutboundProducts, tenantID, days)
	if err == nil {
		defer topProdRows.Close()
		for topProdRows.Next() {
			var tp domain.TopOutboundProduct
			if err := topProdRows.Scan(&tp.ProductID, &tp.ProductName, &tp.ProductSKU, &tp.Quantity); err == nil {
				summary.Outbound.TopProducts = append(summary.Outbound.TopProducts, tp)
			}
		}
		_ = topProdRows.Close()
	}

	topCustRows, err := tx.QueryContext(ctx, dashboardTopCustomers, tenantID, days)
	if err == nil {
		defer topCustRows.Close()
		for topCustRows.Next() {
			var tc domain.TopCustomer
			var cID sql.NullString
			if err := topCustRows.Scan(&cID, &tc.CustomerName, &tc.OrderCount, &tc.TotalRevenue); err == nil {
				tc.CustomerID = nullUUIDToPtr(cID)
				summary.Outbound.TopCustomers = append(summary.Outbound.TopCustomers, tc)
			}
		}
		_ = topCustRows.Close()
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("DashboardRepo.GetSummary: commit: %w", err)
	}

	// Ensure slices are never nil (JSON "[]" instead of "null").
	if summary.MonthlyTrend == nil {
		summary.MonthlyTrend = []domain.MonthlyTrend{}
	}
	if summary.TopVendors == nil {
		summary.TopVendors = []domain.TopVendor{}
	}
	if summary.POS.RecentOrders == nil {
		summary.POS.RecentOrders = []domain.POSRecentOrder{}
	}
	if summary.WMS.LowStockItems == nil {
		summary.WMS.LowStockItems = []domain.LowStockItem{}
	}
	if summary.Outbound.TopProducts == nil {
		summary.Outbound.TopProducts = []domain.TopOutboundProduct{}
	}
	if summary.Outbound.TopCustomers == nil {
		summary.Outbound.TopCustomers = []domain.TopCustomer{}
	}

	return summary, nil
}
