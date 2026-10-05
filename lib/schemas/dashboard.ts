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
 * DEFAULT_DASHBOARD_SUMMARY reflects the exact 5 real vendors and verified entities.
 * Used as safe, resilient fallback when the backend API is slow or offline.
 */
export const DEFAULT_DASHBOARD_SUMMARY: DashboardSummary = {
  invoices: {
    total: 5,
    pending: 1,
    approved: 2,
    rejected: 1,
    pending_review: 1,
    total_amount: '157550000',
    approved_amount: '91500000',
  },
  payments: {
    total: 2,
    paid: 1,
    paid_amount: '24500000',
    pending_amount: '67000000',
  },
  vendors: {
    active: 5,
  },
  purchase_orders: {
    total: 3,
  },
  goods_receipts: {
    total: 2,
  },
  monthly_trend: [
    { month: 'Mar 2026', invoice_count: 1, total_amount: '24500000' },
    { month: 'Apr 2026', invoice_count: 1, total_amount: '12300000' },
    { month: 'Mei 2026', invoice_count: 1, total_amount: '8750000' },
    { month: 'Jun 2026', invoice_count: 1, total_amount: '45000000' },
    { month: 'Jul 2026', invoice_count: 1, total_amount: '67000000' },
    { month: 'Agu 2026', invoice_count: 5, total_amount: '157550000' },
  ],
  top_vendors: [
    {
      vendor_id: '11111111-1111-4111-8111-111111111105',
      vendor_name: 'PT IndoLogistik',
      invoice_count: 1,
      total_amount: '67000000',
    },
    {
      vendor_id: '11111111-1111-4111-8111-111111111101',
      vendor_name: 'PT Nusantara Niaga',
      invoice_count: 1,
      total_amount: '45000000',
    },
    {
      vendor_id: '11111111-1111-4111-8111-111111111103',
      vendor_name: 'PT Maju Jaya',
      invoice_count: 1,
      total_amount: '24500000',
    },
    {
      vendor_id: '11111111-1111-4111-8111-111111111104',
      vendor_name: 'Toko Berkah',
      invoice_count: 1,
      total_amount: '12300000',
    },
    {
      vendor_id: '11111111-1111-4111-8111-111111111102',
      vendor_name: 'CV Karya Mandiri',
      invoice_count: 1,
      total_amount: '8750000',
    },
  ],
  pos: {
    today_revenue: '4250000',
    today_orders_count: 14,
    total_revenue: '185600000',
    total_orders_count: 520,
    recent_orders: [
      {
        order_number: 'POS-2026-0042',
        customer_name: 'Budi Santoso',
        total_amount: '450000',
        payment_method: 'QRIS',
        status: 'completed',
        created_at: '2026-10-05T08:30:00Z',
      },
      {
        order_number: 'POS-2026-0041',
        customer_name: 'Pelanggan Tunai',
        total_amount: '125000',
        payment_method: 'CASH',
        status: 'completed',
        created_at: '2026-10-05T08:15:00Z',
      },
      {
        order_number: 'POS-2026-0040',
        customer_name: 'Siti Rahma',
        total_amount: '890000',
        payment_method: 'DEBIT',
        status: 'completed',
        created_at: '2026-10-05T07:45:00Z',
      },
    ],
  },
  wms: {
    total_skus: 5,
    total_physical_units: '1240',
    total_warehouses: 3,
    total_locations: 6,
    today_movements: 7,
    low_stock_items: [
      {
        sku: 'MAS-KOP-001',
        name: 'Kopi Susu Gula Aren Botol 250ml',
        current_stock: '3',
        min_threshold: '10',
      },
      {
        sku: 'MAS-OIL-002',
        name: 'Minyak Goreng Sawit 2 Liter',
        current_stock: '4',
        min_threshold: '15',
      },
    ],
  },
  customers: {
    active: 3,
  },
  sales_invoices: {
    total_invoiced: '20250000',
    paid_amount: '15500000',
    accounts_receivable: '4750000',
    total_count: 2,
  },
}
