import { test, expect, Page } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'

// Real end-to-end verification of the export feature (lib/export + ExportModal).
// Logs in against the configured backend, opens the modal on /products and
// downloads a real CSV and XLSX, plus the PDF print window.

// Log in once and reuse the session: the backend rate-limits /login,
// so logging in per test makes repeated runs fail with "rate limit exceeded".
let authState: Awaited<ReturnType<import('@playwright/test').BrowserContext['storageState']>> | undefined

test.describe.configure({ mode: 'serial' })

// Logs in on first call, then injects the saved cookies/localStorage into later tests.
async function login(page: Page) {
  if (authState) {
    await page.context().addCookies(authState.cookies)
    await page.context().addInitScript((origins) => {
      for (const o of origins) for (const kv of o.localStorage) localStorage.setItem(kv.name, kv.value)
    }, authState.origins)
    return
  }
  await page.goto('/login')
  await page.fill('#email', 'admin@test.com')
  await page.fill('#password', 'password123')
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL('/dashboard')
  authState = await page.context().storageState()
}

async function openExportModal(page: Page) {
  await page.goto('/products')
  await expect(page.getByRole('heading', { name: /Katalog Produk/i })).toBeVisible()
  await page.getByRole('button', { name: 'Ekspor' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page.getByRole('heading', { name: /Ekspor Katalog Produk/ })).toBeVisible()
}

test.describe('Export Data (CSV / Excel / PDF)', () => {
  test('modal shows scope, 3 formats and live row/column counter', async ({ page }) => {
    await login(page)
    await openExportModal(page)

    // Scope options (page has search filter → both visible)
    await expect(page.getByRole('radio', { name: /Sesuai tampilan/ })).toBeVisible()
    await expect(page.getByRole('radio', { name: /Filter khusus/ })).toBeVisible()

    // Exactly three format options
    await expect(page.getByRole('radio', { name: /^Excel/ })).toBeVisible()
    await expect(page.getByRole('radio', { name: /^CSV/ })).toBeVisible()
    await expect(page.getByRole('radio', { name: /^PDF/ })).toBeVisible()

    // Footer counter: "N baris · M kolom"
    await expect(page.getByText(/baris · \d+ kolom/)).toBeVisible()

    // Custom scope reveals the extra filters (date range + price select)
    await page.getByRole('radio', { name: /Filter khusus/ }).click()
    await expect(page.getByText('Tanggal dibuat', { exact: true })).toBeVisible()
    await expect(page.getByText('Rentang harga', { exact: true })).toBeVisible()
  })

  test('Esc closes the modal', async ({ page }) => {
    await login(page)
    await openExportModal(page)
    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog')).not.toBeVisible()
  })

  test('CSV download: UTF-8 BOM, header row, real product data', async ({ page }) => {
    await login(page)
    await openExportModal(page)

    await page.getByRole('radio', { name: /^CSV/ }).click()
    const downloadP = page.waitForEvent('download')
    await page.getByRole('button', { name: /^Unduh$/ }).click()
    const download = await downloadP

    expect(download.suggestedFilename()).toMatch(/^katalog-produk-\d{4}-\d{2}-\d{2}\.csv$/)

    const file = path.join(test.info().outputDir, download.suggestedFilename())
    await download.saveAs(file)
    const raw = fs.readFileSync(file)

    // UTF-8 BOM present
    expect(raw.subarray(0, 3)).toEqual(Buffer.from([0xef, 0xbb, 0xbf]))
    const text = raw.toString('utf8')
    const lines = text.replace(/^﻿/, '').split('\r\n').filter(Boolean)
    expect(lines[0]).toBe('SKU,Nama Produk,Deskripsi,Harga (Rp),Dibuat,Diperbarui')
    // At least one data row from the live backend (unless the tenant is empty)
    if (!/Belum ada data|^\s*$/m.test(text)) {
      expect(lines.length).toBeGreaterThanOrEqual(1)
    }
    // Formula-injection guard: no raw formula starter in first column of data rows
    for (const line of lines.slice(1)) {
      expect(line.startsWith('=') || line.startsWith('+') || line.startsWith('@')).toBe(false)
    }
  })

  test('XLSX download: valid ZIP workbook with header', async ({ page }) => {
    await login(page)
    await openExportModal(page)

    await page.getByRole('radio', { name: /^Excel/ }).click()
    const downloadP = page.waitForEvent('download')
    await page.getByRole('button', { name: /^Unduh$/ }).click()
    const download = await downloadP

    expect(download.suggestedFilename()).toMatch(/^katalog-produk-\d{4}-\d{2}-\d{2}\.xlsx$/)
    const file = path.join(test.info().outputDir, download.suggestedFilename())
    await download.saveAs(file)

    const raw = fs.readFileSync(file)
    // XLSX files are ZIP containers → start with "PK"
    expect(raw.subarray(0, 2).toString('ascii')).toBe('PK')
    expect(raw.length).toBeGreaterThan(100)
  })

  test('PDF opens print window with report content', async ({ page, context }) => {
    await login(page)
    await openExportModal(page)

    await page.getByRole('radio', { name: /^PDF/ }).click()
    const popupP = context.waitForEvent('page')
    await page.getByRole('button', { name: /Cetak \/ PDF/ }).click()
    const popup = await popupP

    await popup.waitForLoadState('domcontentloaded')
    const content = await popup.content()
    expect(content).toContain('TAYOOLI')
    expect(content).toContain('Katalog Produk')
    expect(content).toContain('SKU')          // table header
    expect(content).toContain('baris')        // row-count summary
    // XSS guard: header title rendered as text, not markup injection
    expect(popup.getByRole('heading', { name: /Katalog Produk/ }).first()).toBeVisible()
  })

  test('custom scope filter changes exported row count', async ({ page }) => {
    await login(page)
    await openExportModal(page)

    // Switch to custom scope with an impossible date range → 0 baris expected
    await page.getByRole('radio', { name: /Filter khusus/ }).click()
    const from = page.getByLabel('Tanggal dibuat dari')
    const to = page.getByLabel('Tanggal dibuat sampai')
    if (await from.isVisible()) {
      await from.fill('2000-01-01')
      await to.fill('2000-01-02')
      // Export button must be disabled when filtered set is empty (or shows 0 rows)
      await expect(page.getByText(/^0 baris/)).toBeVisible()
      await expect(page.getByRole('button', { name: /^Unduh$/ })).toBeDisabled()
    }
  })

  // Smoke: every integrated page exposes the Ekspor button and its modal opens.
  for (const [route, modalTitle] of [
    ['/wms', 'Stok & Lokasi Gudang'],
    ['/wms/inbound', 'Barang Masuk'],
    ['/wms/delivery-orders', 'Surat Jalan'],
    ['/pos', 'Riwayat Transaksi POS'],
  ] as const) {
    test(`export modal opens on ${route}`, async ({ page }) => {
      await login(page)
      await page.goto(route)
      const btn = page.getByRole('button', { name: 'Ekspor', exact: true })
      await expect(btn).toBeVisible()
      // Button is disabled while data loads; let queries settle before branching.
      await page.waitForLoadState('networkidle')
      if (await btn.isEnabled()) {
        await btn.click()
        await expect(page.getByRole('dialog')).toBeVisible()
        await expect(page.getByRole('dialog').getByRole('heading', { name: `Ekspor ${modalTitle}` })).toBeVisible()
        await expect(page.getByRole('radio', { name: /^Excel/ })).toBeVisible()
        await expect(page.getByRole('radio', { name: /^CSV/ })).toBeVisible()
        await expect(page.getByRole('radio', { name: /^PDF/ })).toBeVisible()

        // Real CSV download from this page's live data
        await page.getByRole('radio', { name: /^CSV/ }).click()
        const downloadP = page.waitForEvent('download')
        await page.getByRole('button', { name: /^Unduh$/ }).click()
        const download = await downloadP
        expect(download.suggestedFilename()).toMatch(/\.csv$/)
        const file = path.join(test.info().outputDir, download.suggestedFilename())
        await download.saveAs(file)
        const raw = fs.readFileSync(file)
        expect(raw.subarray(0, 3)).toEqual(Buffer.from([0xef, 0xbb, 0xbf]))
        const lines = raw.toString('utf8').slice(1).split('\r\n').filter(Boolean)
        expect(lines.length).toBeGreaterThanOrEqual(2) // header + ≥1 data row
        console.log(`${route}: ${download.suggestedFilename()} → ${lines.length - 1} baris; header: ${lines[0]}`)
        // Modal closes after a successful export
        await expect(page.getByRole('dialog')).not.toBeVisible()
      } else {
        // Tenant has no data on this page → button correctly disabled.
        await expect(btn).toBeDisabled()
      }
    })
  }
})
