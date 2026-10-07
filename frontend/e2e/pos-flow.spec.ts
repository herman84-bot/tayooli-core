import { test, expect, type Page } from '@playwright/test'

async function loginAsDemo(page: Page) {
  await page.goto('/login')
  await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible()
  await page.fill('#email', 'admin@test.com')
  await page.fill('#password', 'password123')
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL('/dashboard')
}

test.describe('POS Flow (Point of Sale)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsDemo(page)
  })

  test('navigate to /pos, verify product catalog and empty cart state', async ({ page }) => {
    await page.goto('/pos')
    await expect(page).toHaveURL('/pos')

    // Verify main header
    await expect(page.getByRole('heading', { name: /Point of Sale \(POS Kasir\)/i })).toBeVisible()
    await expect(page.getByText('Mode Penjualan Odoo/OCA pattern')).toBeVisible()

    // Verify search bar and category filters
    await expect(page.getByPlaceholder('Cari nama barang atau SKU...')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Semua' })).toBeVisible()

    // Verify initial cart state
    await expect(page.getByRole('heading', { name: /Keranjang Kasir/i })).toBeVisible()
    await expect(page.getByText('Keranjang Masih Kosong')).toBeVisible()
    await expect(page.getByText('Scan barcode barang atau klik produk di katalog')).toBeVisible()

    // Verify checkout button is disabled when cart is empty
    const payBtn = page.getByRole('button', { name: /Bayar Sekarang/i })
    await expect(payBtn).toBeVisible()
    await expect(payBtn).toBeDisabled()
  })

  test('add product to cart, verify total calculation, and complete CASH checkout with receipt', async ({ page }) => {
    await page.goto('/pos')
    await expect(page).toHaveURL('/pos')
    await expect(page.getByRole('heading', { name: /Point of Sale \(POS Kasir\)/i })).toBeVisible()

    // Locate product cards in catalog grid
    const productCard = page.locator('div.grid button').filter({ hasText: 'Rp' }).first()
    await expect(productCard).toBeVisible()

    // Capture product name to verify cart inclusion
    const productName = await productCard.locator('h3').innerText()

    // Add product to cart
    await productCard.click()

    // Verify item is in cart
    await expect(page.getByText('Keranjang Masih Kosong')).not.toBeVisible()
    await expect(page.getByRole('heading', { name: /Keranjang Kasir \([1-9]\d*\)/i })).toBeVisible()
    await expect(page.locator('.divide-y').getByText(productName)).toBeVisible()

    // Verify total calculation breakdown
    await expect(page.getByText('Subtotal:')).toBeVisible()
    await expect(page.getByText('PPN 11%:', { exact: true })).toBeVisible()
    await expect(page.getByText('Total Belanja:')).toBeVisible()

    // Verify Pay Button is now enabled with total amount
    const payBtn = page.getByRole('button', { name: /Bayar Sekarang \(Rp .+\)/i })
    await expect(payBtn).toBeEnabled()

    // Open Payment Modal
    await payBtn.click()

    // Verify Payment Modal
    await expect(page.getByRole('heading', { name: 'Pembayaran Kasir' })).toBeVisible()
    await expect(page.getByText('Total Yang Harus Dibayar')).toBeVisible()

    // Verify payment methods
    const cashMethodBtn = page.getByRole('button', { name: /Uang Tunai \(Cash\)/i })
    const qrisMethodBtn = page.getByRole('button', { name: /QRIS Dinamis/i })
    await expect(cashMethodBtn).toBeVisible()
    await expect(qrisMethodBtn).toBeVisible()

    // Cash mode is default: verify exact cash quick button ("Pas")
    const exactCashBtn = page.getByRole('button', { name: 'Pas' })
    await expect(exactCashBtn).toBeVisible()
    await exactCashBtn.click()

    // Verify change calculation displays Rp 0 when paying exact amount
    await expect(page.getByText('Kembalian (Change):')).toBeVisible()

    // Submit cash payment
    const confirmPaymentBtn = page.getByRole('button', { name: /Konfirmasi Pembayaran Tunai/i })
    await expect(confirmPaymentBtn).toBeEnabled()
    await confirmPaymentBtn.click()

    // Verify Receipt Modal visibility
    await expect(page.getByRole('heading', { name: 'Pembayaran Berhasil!' })).toBeVisible()
    await expect(page.getByText('Transaksi telah dicatat & stok terpotong')).toBeVisible()

    // Verify thermal receipt elements
    const receipt = page.locator('#thermal-receipt')
    await expect(receipt).toBeVisible()
    await expect(receipt.getByText('TAYOOLI RETAIL & WMS')).toBeVisible()
    await expect(receipt.getByText(/No:\s*POS-/i)).toBeVisible()
    await expect(receipt.getByText(productName)).toBeVisible()
    await expect(receipt.getByText('TOTAL:', { exact: true })).toBeVisible()
    await expect(receipt.getByText(/Bayar \(CASH\):/i)).toBeVisible()
    await expect(receipt.getByText(/Kembali:\s*Rp\s*0/i)).toBeVisible()

    // Verify receipt actions
    await expect(page.getByRole('button', { name: /Cetak Struk/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /Kirim WhatsApp/i })).toBeVisible()

    // Close receipt modal via "Selesai" button
    const finishBtn = page.getByRole('button', { name: 'Selesai' })
    await expect(finishBtn).toBeVisible()
    await finishBtn.click()

    // Modal closed and cart reset
    await expect(page.getByRole('heading', { name: 'Pembayaran Berhasil!' })).not.toBeVisible()
    await expect(page.getByText('Keranjang Masih Kosong')).toBeVisible()
  })

  test('open and verify Rekonsiliasi Kasir (Z-report) modal', async ({ page }) => {
    await page.goto('/pos')
    await expect(page).toHaveURL('/pos')

    // Click "Rekonsiliasi Kas" button in the top action bar
    const reconButton = page.getByRole('button', { name: /Rekonsiliasi Kas/i })
    await expect(reconButton).toBeVisible()
    await reconButton.click()

    // Verify Z-report modal heading and description
    await expect(page.getByRole('heading', { name: 'Rekonsiliasi Kasir (Z-Report)' })).toBeVisible()
    await expect(
      page.getByText('Laporan ringkasan kas, penjualan, dan penutupan shift kasir')
    ).toBeVisible()

    // Verify summary metric cards
    await expect(page.getByText('Total Omzet')).toBeVisible()
    await expect(page.getByText('Kas Fisik (CASH)')).toBeVisible()
    await expect(page.getByText('QRIS / Midtrans')).toBeVisible()

    // Verify reconciliation instructions and audit notice
    await expect(page.getByText('Instruksi Pencocokan Kasir:')).toBeVisible()
    await expect(page.getByText(/Hitung uang fisik di laci kasir/i)).toBeVisible()

    // Close the reconciliation modal
    const closeBtn = page
      .locator('div.fixed')
      .filter({ hasText: 'Rekonsiliasi Kasir (Z-Report)' })
      .locator('button')
      .filter({ has: page.locator('svg.lucide-x') })
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()

    // Verify modal is dismissed
    await expect(page.getByRole('heading', { name: 'Rekonsiliasi Kasir (Z-Report)' })).not.toBeVisible()
  })
})
