import { test, expect } from '@playwright/test'

test.describe('Order-to-Pay pages', () => {
  test.beforeEach(async ({ page }) => {
    // Login as admin (seeded in migration 012, password123)
    await page.goto('/login')
    await page.fill('#email', 'admin@test.com')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('/dashboard')
  })

  test('O2P module pages render for an authenticated admin', async ({ page }) => {
    await page.goto('/dashboard/purchase-orders')
    await expect(page.locator('h1', { hasText: 'Purchase Orders' })).toBeVisible()

    await page.goto('/dashboard/goods-receipts')
    await expect(page.locator('h1', { hasText: 'Goods Receipts' })).toBeVisible()

    await page.goto('/dashboard/payment-orders')
    await expect(page.locator('h1', { hasText: 'Payment Orders' })).toBeVisible()

    await page.goto('/dashboard/invoices')
    await expect(page.locator('h1', { hasText: 'Invoices' })).toBeVisible()
  })

  test('new-entity forms are reachable', async ({ page }) => {
    await page.goto('/dashboard/purchase-orders/new')
    await expect(page.locator('form')).toBeVisible()

    await page.goto('/dashboard/invoices/new')
    await expect(page.locator('form')).toBeVisible()
  })
})
