// Waits until a deployed backend exposes the dashboard summary fields added by
// a given commit, then exits 0. Useful right after pushing to confirm Zeabur
// actually rolled the new build out.
const { baseUrl, authHeaders, sleep } = require('./_live-helpers')

const fe = baseUrl()
const POLL_INTERVAL_MS = 20_000
const MAX_POLLS = 30 // ~10 minutes

;(async () => {
  const headers = await authHeaders(fe)

  for (let i = 0; i < MAX_POLLS; i++) {
    const res = await fetch(`${fe}/api/v1/dashboard/summary`, { headers })
    const body = res.ok ? await res.json() : null
    const hasFinancial = !!body?.financial_overview
    const hasSalesOrders = !!body?.sales_orders
    const hasBasket = body?.pos?.average_basket_size !== undefined
    const hasStockValue = body?.wms?.total_stock_value !== undefined

    if (hasFinancial && hasSalesOrders && hasBasket && hasStockValue) {
      console.log('DEPLOYED')
      console.log('financial_overview:', JSON.stringify(body.financial_overview))
      console.log('sales_orders:', JSON.stringify(body.sales_orders))
      console.log('average_basket_size:', body.pos.average_basket_size)
      console.log('total_stock_value:', body.wms.total_stock_value)
      process.exit(0)
    }

    console.log(
      `[${i}] waiting... financial=${hasFinancial} sales_orders=${hasSalesOrders} ` +
        `basket=${hasBasket} stock_value=${hasStockValue}` +
        (res.ok ? '' : ` (status ${res.status})`)
    )
    await sleep(POLL_INTERVAL_MS)
  }

  console.error(`TIMEOUT: new fields still absent after ${MAX_POLLS * (POLL_INTERVAL_MS / 60000)} minutes`)
  process.exit(1)
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
