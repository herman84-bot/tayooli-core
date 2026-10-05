// Flows 2-14 dengan login di CDP session yang sudah open.
//
// Credentials come from the environment (TAYOOLI_EMAIL / TAYOOLI_PASSWORD) so no
// secret lives in the repo. CDP endpoint: CDP_URL (default http://127.0.0.1:9100).
const { chromium } = require('@playwright/test')
const { baseUrl, cdpUrl, requireCredentials } = require('./_live-helpers')

;(async () => {
  const browser = await chromium.connectOverCDP(cdpUrl())
  const context = browser.contexts()[0]
  const pages = context.pages()
  let page = pages.find((p) => p.url().includes('tayooli')) || pages[0]

  const BASE = baseUrl()
  if (!page) {
    page = await context.newPage()
    await page.goto(`${BASE}/login`)
  }

  const results = []

  // Check if logged in; if not, login
  const currentUrl = page.url()
  if (!currentUrl.includes('/dashboard') && !currentUrl.includes('/products')) {
    console.log('[login] Navigating to login...')
    const { email, password } = requireCredentials()
    await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 })
    const emailInput = await page.$('input[type="email"]')
    if (emailInput) {
      console.log('[login] Filling credentials...')
      await page.fill('input[type="email"]', email)
      await page.fill('input[type="password"]', password)
      await page.click('button:has-text("Masuk")')
      await page.waitForNavigation({ timeout: 10000 })
    }
  }

  const flows = [
    { num: 2, path: '/dashboard', title: 'Dashboard' },
    { num: 3, path: '/products', title: 'Produk' },
    { num: 4, path: '/wms/inbound', title: 'Barang Masuk' },
    { num: 5, path: '/wms', title: 'Gudang & Stok' },
    { num: 6, path: '/wms/delivery-orders', title: 'Surat Jalan' },
    { num: 7, path: '/wms/marketplace', title: 'Marketplace' },
    { num: 8, path: '/wms/transfers', title: 'Transfer Stok' },
    { num: 9, path: '/wms/opname', title: 'Opname' },
    { num: 10, path: '/wms/scrap', title: 'Barang Rusak' },
    { num: 11, path: '/wms/scanner', title: 'Scanner' },
    { num: 12, path: '/pos', title: 'POS' },
    { num: 13, path: '/settings', title: 'Pengaturan' },
    { num: 14, path: '/help', title: 'Bantuan' },
  ]

  for (const { num, path, title } of flows) {
    try {
      await page.goto(`${BASE}${path}`, { waitUntil: 'domcontentloaded', timeout: 8000 })
      const h1 = await page.evaluate(() => document.querySelector('h1')?.innerText.trim() || '(no h1)').catch(() => '(error)')
      const url = page.url()
      const status = url.includes(path) && !url.includes('/login') ? 'PASS' : 'FAIL'
      results.push({ num, title, status, h1, url })
      console.log(`[${status}] Flow ${num}: ${title.padEnd(18)} ${h1} (${url})`)
    } catch (e) {
      results.push({ num, title, status: 'ERROR', error: e.message })
      console.log(`[ERROR] Flow ${num}: ${title} -> ${e.message}`)
    }
  }

  console.log(`\n=== SUMMARY: ${results.filter((r) => r.status === 'PASS').length}/${results.length} PASS ===`)
  await browser.close()
  process.exit(results.every((r) => r.status === 'PASS') ? 0 : 1)
})().catch((e) => { console.error(e); process.exit(1) })
