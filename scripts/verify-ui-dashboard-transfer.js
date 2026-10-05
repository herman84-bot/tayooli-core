// UI evidence for the dashboard sections + transfer form behaviour.
// Logs in through the real UI form using TAYOOLI_EMAIL / TAYOOLI_PASSWORD.
const { chromium } = require('@playwright/test')
const { baseUrl, cdpUrl, requireCredentials, sleep } = require('./_live-helpers')

const BASE = baseUrl()

;(async () => {
  const { email, password } = requireCredentials()
  const browser = await chromium.connectOverCDP(cdpUrl())
  const ctx = browser.contexts()[0]
  const page = await ctx.newPage()
  const out = {}

  // Login via the real UI form.
  await ctx.clearCookies().catch(() => {})
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded' })
  await page.fill('input[type="email"]', email)
  await page.fill('input[type="password"]', password)
  await page.click('button[type="submit"]')
  await page.waitForURL(/dashboard|onboarding/, { timeout: 20000 })

  // Dashboard: wait for the frontend deploy to serve the new sections.
  for (let i = 0; i < 24; i++) {
    await page.goto(`${BASE}/dashboard`, { waitUntil: 'domcontentloaded' })
    await sleep(4000)
    const t = (await page.evaluate(() => document.body.innerText)).toLowerCase()
    out.dashNew = [
      'total penjualan',
      'kas masuk',
      'valuasi stok',
      'pengeluaran vendor',
      'pos kasir',
      'gudang',
      'pembelian',
    ].map((s) => [s, t.includes(s)])
    if (out.dashNew.every(([, ok]) => ok)) break
    console.log(`[${i}] frontend not updated yet:`, out.dashNew.filter(([, ok]) => !ok).map(([s]) => s).join(', '))
    await sleep(16000)
  }
  out.dashObjectObject = (await page.evaluate(() => document.body.innerText)).includes('[object Object]')
  await page.screenshot({ path: 'screenshots/live-dashboard.png', fullPage: true })

  // Transfer form.
  await page.goto(`${BASE}/wms/transfers`, { waitUntil: 'domcontentloaded' })
  await sleep(3000)
  await page.click('button:has-text("Buat Transfer Baru")')
  await sleep(1500)
  const modalText = await page.evaluate(() => document.body.innerText)
  out.formNoAutoRow = modalText.includes('Pilih Gudang Asal terlebih dahulu')
  out.addRowDisabledBeforeSource = await page.locator('button:has-text("Tambah Baris")').isDisabled()

  // Choose source warehouse, then add a row: rack must auto-fill.
  const selects = page.locator('form select')
  await selects.nth(0).selectOption({ index: 1 })
  await sleep(2500)
  await page.click('button:has-text("Tambah Baris")')
  await sleep(1000)
  out.rackAutoFilled = await page.locator('select[aria-label="Rak asal"]').first().inputValue().catch(() => '')
  await page.screenshot({ path: 'screenshots/live-transfer-form.png', fullPage: true })

  console.log(JSON.stringify(out, null, 2))
  await page.close()
  await browser.close()
})().catch((e) => { console.error('ERROR:', e.message); process.exit(1) })
