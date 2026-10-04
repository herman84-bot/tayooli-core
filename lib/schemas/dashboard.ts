import { z } from 'zod'

export const InvoiceStatsSchema = z.object({
  total: z.number().int().nonnegative(),
  pending: z.number().int().nonnegative(),
  approved: z.number().int().nonnegative(),
  rejected: z.number().int().nonnegative(),
  pending_review: z.number().int().nonnegative(),
  total_amount: z.string(),
  approved_amount: z.string(),
})
export type InvoiceStats = z.infer<typeof InvoiceStatsSchema>

export const PaymentStatsSchema = z.object({
  total: z.number().int().nonnegative(),
  paid: z.number().int().nonnegative(),
  paid_amount: z.string(),
  pending_amount: z.string(),
})
export type PaymentStats = z.infer<typeof PaymentStatsSchema>

export const VendorStatsSchema = z.object({
  active: z.number().int().nonnegative(),
})
export type VendorStats = z.infer<typeof VendorStatsSchema>

export const POStatsSchema = z.object({
  total: z.number().int().nonnegative(),
})

export const GRStatsSchema = z.object({
  total: z.number().int().nonnegative(),
})

export const MonthlyTrendSchema = z.object({
  month: z.string(),
  invoice_count: z.number().int().nonnegative(),
  total_amount: z.string(),
})
export type MonthlyTrend = z.infer<typeof MonthlyTrendSchema>

export const TopVendorSchema = z.object({
  vendor_id: z.string(),
  vendor_name: z.string(),
  invoice_count: z.number().int().nonnegative(),
  total_amount: z.string(),
})
export type TopVendor = z.infer<typeof TopVendorSchema>

export const POSRecentOrderSchema = z.object({
  order_number: z.string(),
  customer_name: z.string(),
  total_amount: z.string(),
  payment_method: z.string(),
  status: z.string(),
  created_at: z.string(),
})
export type POSRecentOrder = z.infer<typeof POSRecentOrderSchema>

export const POSStatsSchema = z.object({
  today_revenue: z.string(),
  today_orders_count: z.number().int().nonnegative(),
  total_revenue: z.string(),
  total_orders_count: z.number().int().nonnegative(),
  recent_orders: z.array(POSRecentOrderSchema),
})
export type POSStats = z.infer<typeof POSStatsSchema>

export const LowStockItemSchema = z.object({
  sku: z.string(),
  name: z.string(),
  current_stock: z.string(),
  min_threshold: z.string(),
})
export type LowStockItem = z.infer<typeof LowStockItemSchema>

export const WMSStatsSchema = z.object({
  total_skus: z.number().int().nonnegative(),
  total_physical_units: z.string(),
  total_warehouses: z.number().int().nonnegative(),
  total_locations: z.number().int().nonnegative(),
  today_movements: z.number().int().nonnegative(),
  low_stock_items: z.array(LowStockItemSchema),
})
export type WMSStats = z.infer<typeof WMSStatsSchema>

export const CustomerStatsSchema = z.object({
  active: z.number().int().nonnegative(),
})
export type CustomerStats = z.infer<typeof CustomerStatsSchema>

export const SalesInvoiceStatsSchema = z.object({
  total_invoiced: z.string(),
  paid_amount: z.string(),
  accounts_receivable: z.string(),
  total_count: z.number().int().nonnegative(),
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
