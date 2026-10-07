import { test, expect, type Page } from '@playwright/test'

async function loginAsDemo(page: Page) {
  await page.goto('/login')
  await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible()
  await page.fill('#email', 'admin@test.com')
  await page.fill('#password', 'password123')
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL('/dashboard')
}

test.describe('Settings Payment Gateway Configuration', () => {
  test.beforeEach(async ({ page, context }) => {
    // Grant clipboard permissions for copy webhook URL test
    await context.grantPermissions(['clipboard-read', 'clipboard-write']).catch(() => {
      // Some environments may not support clipboard permission grant, non-fatal
    })
    await loginAsDemo(page)
  })

  test('navigate to /settings and switch to tab "Metode Pembayaran"', async ({ page }) => {
    await page.goto('/settings')
    await expect(page).toHaveURL('/settings')

    // Verify main Settings heading and available tabs
    await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Tim' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Perusahaan' })).toBeVisible()

    const paymentsTab = page.getByRole('button', { name: 'Metode Pembayaran' })
    await expect(paymentsTab).toBeVisible()

    // Switch to 'Metode Pembayaran' tab
    await paymentsTab.click()

    // Verify section header and BYO Settlement context
    await expect(
      page.getByRole('heading', { name: 'Integrasi Payment Gateway (BYO Settlement)' })
    ).toBeVisible()
    await expect(
      page.getByText('Hubungkan akun payment gateway Anda sendiri. Dana penjualan kasir (QRIS) langsung disalurkan')
    ).toBeVisible()
  })

  test('verify Midtrans and Pakasir options switching', async ({ page }) => {
    await page.goto('/settings')
    await page.getByRole('button', { name: 'Metode Pembayaran' }).click()

    // Verify provider toggle buttons exist
    const midtransBtn = page.getByRole('button', { name: /Midtrans \(QRIS & Core API\)/i })
    const pakasirBtn = page.getByRole('button', { name: /Pakasir \(Fallback UMKM\)/i })
    await expect(midtransBtn).toBeVisible()
    await expect(pakasirBtn).toBeVisible()

    // Midtrans is active by default: verify Midtrans configuration fields
    await expect(page.getByText('Server Key Midtrans')).toBeVisible()
    await expect(page.getByText('Client Key Midtrans')).toBeVisible()
    await expect(page.getByText('Buka Midtrans Dashboard')).toBeVisible()

    // Switch to Pakasir
    await pakasirBtn.click()

    // Verify Pakasir fields appear and Midtrans fields disappear
    await expect(page.getByText('Project Slug Pakasir')).toBeVisible()
    await expect(page.getByText('API Key Pakasir')).toBeVisible()
    await expect(page.getByText('Server Key Midtrans')).not.toBeVisible()

    // Switch back to Midtrans
    await midtransBtn.click()
    await expect(page.getByText('Server Key Midtrans')).toBeVisible()
    await expect(page.getByText('Client Key Midtrans')).toBeVisible()
  })

  test('verify active status banner and Webhook URL helper', async ({ page }) => {
    await page.goto('/settings')
    await page.getByRole('button', { name: 'Metode Pembayaran' }).click()

    // Verify Gateway Status Badge / Banner
    const statusBannerHeading = page.locator('div.rounded-xl.border').filter({
      hasText: /Gateway (Midtrans|Pakasir) Aktif|Mode Demo Aktif/,
    })
    await expect(statusBannerHeading.first()).toBeVisible()
    await expect(
      page.getByText(/pembayaran instan|tagihan riil/i)
    ).toBeVisible()

    // Verify Webhook URL Helper (active under Midtrans provider)
    await expect(page.getByText('Webhook Notification URL')).toBeVisible()
    await expect(
      page.getByText(/Tempel URL berikut pada dashboard Midtrans/i)
    ).toBeVisible()

    // Verify the Webhook URL container
    const webhookContainer = page
      .locator('div.font-mono')
      .filter({ hasText: /\/api\/v1\/webhooks\/midtrans/ })
    await expect(webhookContainer).toBeVisible()

    // Verify Copy Webhook URL button and feedback interaction
    const copyBtn = page.getByRole('button', { name: /Salin URL/i })
    await expect(copyBtn).toBeVisible()
    await copyBtn.click()

    // Verify button switches to "Disalin!" confirmation
    await expect(page.getByText('Disalin!')).toBeVisible()
  })
})
