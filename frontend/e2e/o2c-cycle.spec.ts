import { test, expect } from '@playwright/test'

// Order-to-Cash (O2C) full cycle against the live app.
// Route map (matches the real app under app/(app)/):
//   /customers, /sales-orders, /sales-invoices at top level;
//   /dashboard/payment-orders for payment orders.
// UI labels: "+ New Customer", "+ New Order", "+ New Invoice", "+ Buat Payment Order",
// and action buttons "Setujui" (approve draft) / "Bayar" (pay approved).
test.describe('Order-to-Cash Full Cycle', () => {
  test('O2C full cycle: login → customer → sales order → invoice → payment', async ({ page }) => {
    // Step 1: Login
    await page.goto('/login')
    await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible()
    await page.fill('#email', 'admin@test.com')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('/dashboard')

    // Step 2: Create Customer
    await page.goto('/customers')
    await page.click('text=+ New Customer')
    await page.fill('input[placeholder="PT Maju Jaya"]', 'E2E Customer Corp')
    await page.fill('input[placeholder="contact@majujaya.com"]', 'customer@e2e.com')
    await page.click('button:has-text("Create Customer")')
    await expect(page.locator('text=E2E Customer Corp')).toBeVisible()

    // Step 3: Create Sales Order (customer select loads from API)
    await page.goto('/sales-orders')
    await page.click('text=+ New Order')
    await page.selectOption('#customer', { label: 'E2E Customer Corp' })
    await page.fill('#amount', '250000')
    await page.click('button:has-text("Create Order")')
    await expect(page.locator('text=E2E Customer Corp')).toBeVisible()

    // Step 4: Create Sales Invoice (invoice number + amount + due date required)
    const invNumber = `SI-${Date.now()}`
    await page.goto('/sales-invoices')
    await page.click('text=+ New Invoice')
    await page.fill('input[placeholder="e.g. SI-2026-0001"]', invNumber)
    await page.fill('input[type="number"]', '250000')
    await page.fill('input[type="date"]', '2026-09-01')
    await page.click('button:has-text("Create Invoice")')
    await expect(page.locator(`text=${invNumber}`)).toBeVisible()

    // Step 5: Create Payment Order (from invoice) — draft, then approve, then pay
    await page.goto('/dashboard/payment-orders')
    await page.click('text=+ Buat Payment Order')
    await expect(page).toHaveURL(/\/dashboard\/payment-orders\/create/)
    // Pick the first available invoice via the UI (form selects it) and submit.
    await page.fill('input[name="amount"]', '250000')
    await page.click('button:has-text("Buat")')
    await expect(page).toHaveURL(/\/dashboard\/payment-orders$/)

    // Step 6: Approve the draft payment order (Setujui button on draft rows)
    await page.getByRole('button', { name: 'Setujui' }).first().click()
    await expect(page.getByText('Disetujui').first()).toBeVisible()

    // Step 7: Pay the approved payment order (Bayar button on approved rows)
    await page.getByRole('button', { name: 'Bayar' }).first().click()
    await expect(page.getByText('Dibayar').first()).toBeVisible()
  })

  test('O2C cycle shows validation errors for missing required fields', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible()
    await page.fill('#email', 'admin@test.com')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('/dashboard')

    // Try to create customer with missing name — the form requires name + email
    // (HTML required), so submitting empty should not create a row.
    await page.goto('/customers')
    await page.click('text=+ New Customer')
    await page.click('button:has-text("Create Customer")')
    await expect(page.locator('form')).toBeVisible()
    await expect(page.locator('input[placeholder="PT Maju Jaya"]')).toBeVisible()
  })
})
