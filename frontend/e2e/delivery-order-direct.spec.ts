import { test, expect } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'

// Full journey for a direct Surat Jalan (no Sales Order), which used to fail
// with HTTP 400 because the form sent a non-UUID "SO-DIRECT":
//   login → open form → invalid SO ref is rejected client-side
//   → submit without SO → verify via API (persisted, sales_order_id null, DRAFT)
//   → reload → row visible → export CSV contains the new DO.
// Creates one DRAFT DO (no stock movement; dispatch is not performed).

test.describe.configure({ mode: 'serial' })

const CREATE_URL = /\/wms\/delivery-orders(\?|$)/

test('direct Surat Jalan: create without Sales Order, persist, export', async ({ page }) => {
  test.setTimeout(120_000)

  await page.goto('/login')
  await page.fill('#email', 'admin@test.com')
  await page.fill('#password', 'password123')
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL('/dashboard')

  await page.goto('/wms/delivery-orders')
  await page.getByRole('button', { name: /Buat Surat Jalan Baru/ }).click()
  const form = page.locator('form').filter({ hasText: 'Terbitkan Surat Jalan' })
  await expect(form).toBeVisible()

  // SO field is optional and starts empty (no more fake "so-xxxx" value)
  const soInput = page.getByLabel(/Ref\. Sales Order/)
  await expect(soInput).toHaveValue('')
  await expect(soInput).not.toHaveAttribute('required', /.*/)

  const doNumber = `DO/E2E/${Date.now()}`
  await form.getByPlaceholder('e.g. DO/2026/09/0001').fill(doNumber)
  await form.getByPlaceholder('e.g. PT Nusantara Retail Makmur').fill('Uji E2E Surat Jalan Langsung')

  // Add one line item and pick the first real rack location
  await form.getByRole('button', { name: /Tambah Baris/ }).click()
  const locSelect = form.locator('select').filter({ has: page.locator('option', { hasText: 'Pilih Rak/Bin...' }) }).first()
  const locOptions = await locSelect.locator('option').evaluateAll((os) =>
    os.map((o) => (o as HTMLOptionElement).value).filter(Boolean),
  )
  test.skip(locOptions.length === 0, 'tenant has no rack location in the default warehouse')
  await locSelect.selectOption(locOptions[0])

  // Negative: a non-UUID SO reference is rejected before hitting the API
  let createCalls = 0
  page.on('request', (r) => {
    if (r.method() === 'POST' && CREATE_URL.test(r.url())) createCalls++
  })
  await soInput.fill('SO-2026-0045')
  await form.getByRole('button', { name: 'Terbitkan Surat Jalan' }).click()
  await expect(form.getByText(/Ref\. Sales Order harus berupa ID Sales Order yang valid/)).toBeVisible()
  expect(createCalls).toBe(0)

  // Happy path: leave SO empty → direct Surat Jalan
  await soInput.fill('')
  const respP = page.waitForResponse(
    (r) => r.request().method() === 'POST' && CREATE_URL.test(r.url()),
  )
  await form.getByRole('button', { name: 'Terbitkan Surat Jalan' }).click()
  const resp = await respP
  expect(resp.status(), await resp.text()).toBe(201)
  const created = await resp.json()
  expect(created.do_number).toBe(doNumber)
  expect(created.sales_order_id).toBeNull()
  expect(created.status).toBe('DRAFT')
  await expect(form).not.toBeVisible()

  // Persisted state via API (not just the toast)
  const detail = await page.request.get(`/api/v1/wms/delivery-orders/${created.id}`)
  expect(detail.status()).toBe(200)
  const detailBody = await detail.json()
  const header = detailBody.delivery_order ?? detailBody.data ?? detailBody
  expect(header.do_number).toBe(doNumber)
  expect(header.sales_order_id ?? null).toBeNull()

  // Reload → row visible in the list
  await page.reload()
  await expect(page.getByText(doNumber)).toBeVisible()

  // Export CSV from the Surat Jalan page contains the new DO
  const exportBtn = page.getByRole('button', { name: 'Ekspor', exact: true })
  await expect(exportBtn).toBeEnabled()
  await exportBtn.click()
  await expect(page.getByRole('dialog').getByRole('heading', { name: 'Ekspor Surat Jalan' })).toBeVisible()
  await page.getByRole('radio', { name: /^CSV/ }).click()
  const downloadP = page.waitForEvent('download')
  await page.getByRole('button', { name: /^Unduh$/ }).click()
  const download = await downloadP
  expect(download.suggestedFilename()).toMatch(/^surat-jalan-\d{4}-\d{2}-\d{2}\.csv$/)
  const file = path.join(test.info().outputDir, download.suggestedFilename())
  await download.saveAs(file)
  const csv = fs.readFileSync(file, 'utf8')
  expect(csv).toContain(doNumber)
  expect(csv).toContain('Uji E2E Surat Jalan Langsung')
  console.log(`created ${doNumber} (id ${created.id}); CSV rows: ${csv.trim().split('\r\n').length - 1}`)
})
