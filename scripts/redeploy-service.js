// Usage: node scripts/redeploy-service.js <backend|frontend>
// Opens (or reuses) a Zeabur tab in the CDP browser and clicks Redeploy.
const { chromium } = require('@playwright/test')
const SERVICES = { backend: '6ac268f493154e21e628ed15', frontend: '6ac2da1493154e21e6291302' }
const which = process.argv[2] || 'backend'

;(async () => {
  const browser = await chromium.connectOverCDP('http://127.0.0.1:9100')
  const ctx = browser.contexts()[0]
  const page = ctx.pages().find((p) => p.url().includes('zeabur.com')) || (await ctx.newPage())
  await page.goto(`https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/${SERVICES[which]}?envID=6ac2683a6a873116ad572b40`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(4000)
  console.log('url:', page.url())
  if (/login|signin/i.test(page.url())) throw new Error('Zeabur session not logged in')

  const btn = page.locator('button:has-text("Redeploy")').first()
  await btn.waitFor({ timeout: 15000 })
  await btn.click()
  await page.waitForTimeout(1500)
  const confirm = page.locator('[role="dialog"] button:has-text("Redeploy"), [role="alertdialog"] button:has-text("Redeploy"), [role="dialog"] button:has-text("Confirm")').first()
  if (await confirm.isVisible().catch(() => false)) { await confirm.click(); console.log('confirmed dialog') }
  await page.waitForTimeout(3000)
  const txt = await page.evaluate(() => document.body.innerText)
  const m = txt.match(/(Building|Deploying|Running|Pending|Queued|Failed)[^\n]{0,60}/)
  console.log(`Redeploy clicked (${which}). status hint:`, m ? m[0] : '(none)')
  await page.screenshot({ path: `screenshots/zeabur-${which}-redeploy.png` })
  await browser.close()
})().catch((e) => { console.error('ERROR:', e.message); process.exit(1) })
