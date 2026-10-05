import { z } from 'zod'

// Coercion helpers for resilient data parsing
const amountString = z.union([z.string(), z.number()]).transform((v) => String(v))
const safeCount = z.union([z.string(), z.number()]).transform((v) => {
  const n = typeof v === 'number' ? v : parseInt(v, 10)
  return Number.isNaN(n) ? 0 : Math.max(0, Math.floor(n))
})

export const InvoiceStatsSchema = z.object({
  total: safeCount.default(0),
  pending: safeCount.default(0),
  approved: safeCount.default(0),
  rejected: safeCount.default(0),
  pending_review: safeCount.default(0),
  total_amount: amountString.default('0'),
  approved_amount: amountString.default('0'),
})
export type InvoiceStats = z.infer<typeof InvoiceStatsSchema>

export const PaymentStatsSchema = z.object({
  total: safeCount.default(0),
  paid: safeCount.default(0),
  paid_amount: amountString.default('0'),
  pending_amount: amountString.default('0'),
})
export type PaymentStats = z.infer<typeof PaymentStatsSchema>

export const VendorStatsSchema = z.object({
  active: safeCount.default(0),
})
export type VendorStats = z.infer<typeof VendorStatsSchema>

export const POStatsSchema = z.object({
  total: safeCount.default(0),
})

export const GRStatsSchema = z.object({
  total: safeCount.default(0),
})

export const MonthlyTrendSchema = z.object({
  month: z.string(),
  invoice_count: safeCount.default(0),
  total_amount: amountString.default('0'),
})
export type MonthlyTrend = z.infer<typeof MonthlyTrendSchema>

export const TopVendorSchema = z.object({
  vendor_id: z.string(),
  vendor_name: z.string(),
  invoice_count: safeCount.default(0),
  total_amount: amountString.default('0'),
})
export type TopVendor = z.infer<typeof TopVendorSchema>

export const POSRecentOrderSchema = z.object({
  order_number: z.string(),
  customer_name: z.string(),
  total_amount: amountString.default('0'),
  payment_method: z.string().default('CASH'),
  status: z.string().default('completed'),
  created_at: z.string().default(''),
})
export type POSRecentOrder = z.infer<typeof POSRecentOrderSchema>

export const POSStatsSchema = z.object({
  today_revenue: amountString.default('0'),
  today_orders_count: safeCount.default(0),
  total_revenue: amountString.default('0'),
  total_orders_count: safeCount.default(0),
  recent_orders: z.array(POSRecentOrderSchema).default([]),
})
export type POSStats = z.infer<typeof POSStatsSchema>

export const LowStockItemSchema = z.object({
  sku: z.string(),
  name: z.string(),
  current_stock: amountString.default('0'),
  min_threshold: amountString.default('0'),
})
export type LowStockItem = z.infer<typeof LowStockItemSchema>

export const WMSStatsSchema = z.object({
  total_skus: safeCount.default(0),
  total_physical_units: amountString.default('0'),
  total_warehouses: safeCount.default(0),
  total_locations: safeCount.default(0),
  today_movements: safeCount.default(0),
  low_stock_items: z.array(LowStockItemSchema).default([]),
})
export type WMSStats = z.infer<typeof WMSStatsSchema>

export const CustomerStatsSchema = z.object({
  active: safeCount.default(0),
})
export type CustomerStats = z.infer<typeof CustomerStatsSchema>

export const SalesInvoiceStatsSchema = z.object({
  total_invoiced: amountString.default('0'),
  paid_amount: amountString.default('0'),
  accounts_receivable: amountString.default('0'),
  total_count: safeCount.default(0),
})
export type SalesInvoiceStats = z.infer<typeof SalesInvoiceStatsSchema>

export const DashboardSummarySchema = z.object({
  invoices: InvoiceStatsSchema,
  payments: PaymentStatsSchema,
  vendors: VendorStatsSchema,
  purchase_orders: POStatsSchema,
  goods_receipts: GRStatsSchema,
  monthly_trend: z.array(MonthlyTrendSchema),
  top_vendors: z.array(TopVendorSchema),
  pos: POSStatsSchema,
  wms: WMSStatsSchema,
  customers: CustomerStatsSchema,
  sales_invoices: SalesInvoiceStatsSchema,
})
export type DashboardSummary = z.infer<typeof DashboardSummarySchema>

/**
 * EMPTY_DASHBOARD_SUMMARY: all-zero placeholder used only while loading or when
 * the backend fails. It intentionally contains NO fabricated numbers so the UI
 * never shows data that does not exist in the database.
 */
export const EMPTY_DASHBOARD_SUMMARY: DashboardSummary = {
  invoices: { total: 0, pending: 0, approved: 0, rejected: 0, pending_review: 0, total_amount: '0', approved_amount: '0' },
  payments: { total: 0, paid: 0, paid_amount: '0', pending_amount: '0' },
  vendors: { active: 0 },
  purchase_orders: { total: 0 },
  goods_receipts: { total: 0 },
  monthly_trend: [],
  top_vendors: [],
  pos: { today_revenue: '0', today_orders_count: 0, total_revenue: '0', total_orders_count: 0, recent_orders: [] },
  wms: { total_skus: 0, total_physical_units: '0', total_warehouses: 0, total_locations: 0, today_movements: 0, low_stock_items: [] },
  customers: { active: 0 },
  sales_invoices: { total_invoiced: '0', paid_amount: '0', accounts_receivable: '0', total_count: 0 },
}
