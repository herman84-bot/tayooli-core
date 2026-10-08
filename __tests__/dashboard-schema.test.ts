import { DashboardSummarySchema, EMPTY_DASHBOARD_SUMMARY } from '@/lib/schemas/dashboard'

// Payload shape from a backend deployed BEFORE financial_overview/sales_orders existed.
const legacyPayload = {
  invoices: { total: 1, pending: 0, approved: 1, rejected: 0, pending_review: 0, total_amount: '1500000', approved_amount: '1500000' },
  payments: { total: 1, paid: 1, paid_amount: '1500000', pending_amount: '0' },
  vendors: { active: 18 },
  purchase_orders: { total: 2 },
  goods_receipts: { total: 2 },
  monthly_trend: [],
  top_vendors: [],
  pos: { today_revenue: '87080', today_orders_count: 2, total_revenue: '87080', total_orders_count: 2, recent_orders: [] },
  wms: { total_skus: 3, total_physical_units: '103', total_warehouses: 2, total_locations: 1, today_movements: 2, low_stock_items: [] },
  customers: { active: 4 },
  sales_invoices: { total_invoiced: '3087080', paid_amount: '87080', accounts_receivable: '3000000', total_count: 3 },
}

describe('DashboardSummarySchema', () => {
  it('parses the new cross-module sections', () => {
    const parsed = DashboardSummarySchema.parse({
      ...legacyPayload,
      pos: { ...legacyPayload.pos, average_basket_size: '43540' },
      wms: { ...legacyPayload.wms, total_stock_value: '9278000' },
      sales_orders: { total: 3, confirmed: 2, pending: 1 },
      financial_overview: {
        total_revenue: '3087080', cash_inflow: '87080', accounts_receivable: '3000000',
        total_expense: '1500000', cash_outflow: '1500000', accounts_payable: '0', net_cash_balance: '-1412920',
      },
    })
    expect(parsed.financial_overview.net_cash_balance).toBe('-1412920')
    expect(parsed.sales_orders.pending).toBe(1)
    expect(parsed.pos.average_basket_size).toBe('43540')
    expect(parsed.wms.total_stock_value).toBe('9278000')
  })

  it('parses CR-04a outbound metrics and root aliases', () => {
    const parsed = DashboardSummarySchema.parse({
      ...legacyPayload,
      outbound: {
        qty_today: '150',
        qty_month: '4500',
        top_products: [
          { product_id: 'p-1', product_name: 'Beras Premium', product_sku: 'BRS-01', quantity: '120' },
        ],
        top_customers: [
          { customer_id: 'c-1', customer_name: 'PT Mitra Jaya', order_count: 5, total_revenue: '25000000' },
        ],
      },
      outbound_qty_today: '150',
      outbound_qty_month: '4500',
      top_outbound_products: [
        { product_id: 'p-1', product_name: 'Beras Premium', product_sku: 'BRS-01', quantity: '120' },
      ],
      top_customers: [
        { customer_id: 'c-1', customer_name: 'PT Mitra Jaya', order_count: 5, total_revenue: '25000000' },
      ],
    })
    expect(parsed.outbound.qty_today).toBe('150')
    expect(parsed.outbound.qty_month).toBe('4500')
    expect(parsed.outbound.top_products).toHaveLength(1)
    expect(parsed.outbound.top_customers).toHaveLength(1)
    expect(parsed.outbound_qty_today).toBe('150')
    expect(parsed.outbound_qty_month).toBe('4500')
    expect(parsed.top_outbound_products[0].product_sku).toBe('BRS-01')
    expect(parsed.top_customers[0].customer_name).toBe('PT Mitra Jaya')
  })

  it('stays backward compatible with an older backend payload (no crash, zero defaults)', () => {
    const parsed = DashboardSummarySchema.parse(legacyPayload)
    expect(parsed.financial_overview.total_revenue).toBe('0')
    expect(parsed.sales_orders).toEqual({ total: 0, confirmed: 0, pending: 0 })
    expect(parsed.wms.total_stock_value).toBe('0')
  })

  it('empty placeholder satisfies the schema', () => {
    expect(() => DashboardSummarySchema.parse(EMPTY_DASHBOARD_SUMMARY)).not.toThrow()
  })
})
