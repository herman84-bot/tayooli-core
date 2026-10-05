// Live smoke test for the deployed Tayooli backend.
//
// Verifies the dashboard summary contract (derived figures must equal the raw
// per-module figures) and the WMS transfer guardrails, using a real session.
//
// Credentials come from the environment so no secret is stored in the repo:
//   TAYOOLI_EMAIL / TAYOOLI_PASSWORD
// Optional: TAYOOLI_BASE_URL to target another deployment.
const FE = process.env.TAYOOLI_BASE_URL || 'https://tayooli.my.id'
const EMAIL = process.env.TAYOOLI_EMAIL
const PASSWORD = process.env.TAYOOLI_PASSWORD

if (!EMAIL || !PASSWORD) {
  console.error('Set TAYOOLI_EMAIL and TAYOOLI_PASSWORD before running this script.')
  process.exit(2)
}

const j = (b) => ({ method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify(b) })

;(async () => {
  const l = await fetch(`${FE}/api/v1/auth/login`, j({ email: EMAIL, password: PASSWORD }))
  if (!l.ok) throw new Error(`login failed: ${l.status} ${await l.text()}`)
  const cookie = (l.headers.getSetCookie() || []).map((c) => c.split(';')[0]).join('; ')
  const h = { cookie, 'content-type': 'application/json' }

  const s = await (await fetch(`${FE}/api/v1/dashboard/summary`, { headers: h })).json()
  console.log('financial_overview:', JSON.stringify(s.financial_overview))
  console.log('raw sales_invoices:', JSON.stringify(s.sales_invoices))
  console.log('wms:', JSON.stringify({ ...s.wms, low_stock_items: s.wms?.low_stock_items?.length }))
  console.log('pos.average_basket_size:', s.pos?.average_basket_size)

  // Derived financials must agree with the raw module totals, otherwise the
  // dashboard would double-count POS revenue that also produces a sales invoice.
  const consistent =
    s.financial_overview?.total_revenue === s.sales_invoices?.total_invoiced &&
    s.financial_overview?.cash_inflow === s.sales_invoices?.paid_amount &&
    s.financial_overview?.cash_outflow === s.payments?.paid_amount
  console.log('derived == raw:', consistent ? 'YES' : 'NO')

  const wh = await (await fetch(`${FE}/api/v1/wms/warehouses`, { headers: h })).json()
  const list = Array.isArray(wh) ? wh : wh.data || []
  const pr = await (await fetch(`${FE}/api/v1/products`, { headers: h })).json()
  const product = (Array.isArray(pr) ? pr : pr.data || [])[0]

  let allPass = consistent && s.financial_overview !== undefined

  // A transfer item without a source rack can never be dispatched, so the API
  // must refuse it instead of persisting a broken draft.
  if (list.length >= 2 && product) {
    const r = await fetch(
      `${FE}/api/v1/wms/transfers`,
      {
        ...j({
          from_warehouse_id: list[0].id,
          to_warehouse_id: list[1].id,
          transfer_number: `TR-PROBE-${Date.now()}`,
          items: [{ product_id: product.id, requested_qty: 1 }],
        }),
        headers: h,
      }
    )
    const body = await r.text()
    console.log('create transfer w/o source_location_id ->', r.status, body)
    allPass = allPass && r.status === 422
  }

  console.log(`\nRESULT: ${allPass ? 'PASS' : 'FAIL'}`)
  process.exit(allPass ? 0 : 1)
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
