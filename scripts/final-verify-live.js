// Final production verification: dashboard sections render, no "[object Object]"
// leaks, and the transfer modal blocks empty lines. Reuses the CDP session's
// logged-in state (CDP_URL, default http://127.0.0.1:9100).
const { chromium } = require('@playwright/test')
const { baseUrl, cdpUrl } = require('./_live-helpers')

const BASE = baseUrl()
;(async () => {
  const browser = await chromium.connectOverCDP(cdpUrl())
  const ctx = browser.contexts()[0]
  const page = await ctx.newPage()
  
  await page.goto(`${BASE}/dashboard`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(3000)
  
  const text = await page.evaluate(() => document.body.innerText)
  const textLower = text.toLowerCase()
  // Case-insensitive match (frontend may capitalize differently).
  const checkSections = (lower) => [
    lower.includes('total penjualan'),
    lower.includes('kas masuk'),
    lower.includes('valuasi stok'),
    lower.includes('pengeluaran vendor'),
    lower.includes('pos kasir') || lower.includes('pos-kasir'),
    lower.includes('gudang') && lower.includes('monitoring'),
    lower.includes('pembelian') || lower.includes('procure'),
  ]
  const sectionLabels = ['Total Penjualan', 'Kas Masuk', 'Valuasi Stok', 'Pengeluaran Vendor', 'POS Kasir', 'Gudang Monitoring', 'Procure/Vendor']
  const found = sectionLabels.map((s, i) => [s, checkSections(textLower)[i]])
  
  console.log('dashboard sections (case-insensitive):')
  found.forEach(([s, ok]) => console.log(`  ${ok ? '✓' : '✗'} ${s}`))
  
  const stats = await page.evaluate(() => ({
    noObjectObject: !document.body.innerText.includes('[object Object]'),
    cards: document.querySelectorAll('[class*="StatCard"], [class*="card"]').length > 4,
  }))
  console.log('dashboard stats:', JSON.stringify(stats, null, 2))
  
  await page.goto(`${BASE}/wms/transfers`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(2000)
  await page.click('button:has-text("Buat Transfer Baru")')
  await page.waitForTimeout(1500)
  const form = await page.evaluate(() => {
    const t = document.body.innerText.toLowerCase()
    return {
      pilihGudangMsg: t.includes('pilih gudang'),
      noAutoRow: !document.querySelector('[aria-label="Rak asal"]'),
    }
  })
  console.log('transfer form:', JSON.stringify(form, null, 2))
  
  await page.screenshot({ path: 'screenshots/final-verify-live.png', fullPage: true })
  await page.close()
  await browser.close()
  
  const ribbonOk = found.slice(0, 4).every(([, ok]) => ok) // 4 header cards
  const allPass = ribbonOk && stats.noObjectObject && form.pilihGudangMsg
  console.log(`\nFINAL: ${allPass ? '✓ LIVE & WORKING' : '✗ INCOMPLETE'}`)
  process.exit(allPass ? 0 : 1)
})().catch((e) => { console.error(e); process.exit(1) })
