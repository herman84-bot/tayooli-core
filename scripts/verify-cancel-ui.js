// Verifies the deployed transfers page shows the CANCELLED state correctly.
const { chromium } = require('@playwright/test')
const { baseUrl, cdpUrl, requireCredentials, sleep } = require('./_live-helpers')

const BASE = baseUrl()

;(async () => {
  const { email, password } = requireCredentials()
  const browser = await chromium.connectOverCDP(cdpUrl())
  const ctx = browser.contexts()[0]
  const page = await ctx.newPage()

  await ctx.clearCookies().catch(() => {})
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded' })
  await page.fill('input[type="email"]', email)
  await page.fill('input[type="password"]', password)
  await page.click('button[type="submit"]')
  await page.waitForURL(/dashboard|onboarding/, { timeout: 20000 })

  // Wait for the frontend deploy to serve the new UI.
  let ok = false
  for (let i = 0; i < 30; i++) {
    await page.goto(`${BASE}/wms/transfers`, { waitUntil: 'domcontentloaded' })
    await sleep(3500)
    const text = await page.evaluate(() => document.body.innerText)
    if (text.includes('Dibatalkan')) {
      ok = true
      console.log(`[${i}] UI updated`)
      break
    }
    console.log(`[${i}] waiting for frontend (no "Dibatalkan" yet)`)
    await sleep(16000)
  }

  const text = await page.evaluate(() => document.body.innerText)
  const checks = {
    cancelledBanner: text.includes('Dibatalkan'),
    noStockChangeNote: /Stok gudang tidak berubah/i.test(text),
    filterHasCancelled: text.includes('Dibatalkan'),
    noObjectObject: !text.includes('[object Object]'),
    // A cancelled transfer must not offer approve/reject/dispatch actions.
    noApproveButton: !/Setujui Transfer/.test(text),
  }
  console.log(JSON.stringify(checks, null, 2))

  await page.screenshot({ path: 'screenshots/live-transfers-cancelled.png', fullPage: true })
  await page.close()
  await browser.close()

  const pass = ok && Object.values(checks).every(Boolean)
  console.log(`\nRESULT: ${pass ? 'PASS' : 'FAIL'}`)
  process.exit(pass ? 0 : 1)
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
