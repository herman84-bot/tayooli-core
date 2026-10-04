import { test, expect } from '@playwright/test'

test.describe('Authentication Flow', () => {
  test('visit dashboard → redirect to login → login → redirect back', async ({ page }) => {
    // Attempt to access protected route
    await page.goto('/dashboard')

    // Should redirect to login
    await expect(page).toHaveURL('/login')
    await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible()

    // Fill login form
    await page.fill('#email', 'admin@test.com')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')

    // Should redirect to dashboard after successful login
    await expect(page).toHaveURL('/dashboard')
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  })

  test('invalid credentials shows error message', async ({ page }) => {
    await page.goto('/login')

    await page.fill('#email', 'invalid@test.com')
    await page.fill('#password', 'wrongpassword')
    await page.click('button[type="submit"]')

    // Should show error message
    await expect(page.getByTestId('login-error')).toBeVisible()
    await expect(page.getByTestId('login-error-text')).toHaveText(/Email atau kata sandi salah|Invalid credentials/)
    await expect(page).toHaveURL('/login')
  })

  test('logout clears session and redirects to login', async ({ page }) => {
    // Login first
    await page.goto('/login')
    await page.fill('#email', 'admin@test.com')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('/dashboard')

    // Logout
    await page.click('text=Logout')
    await expect(page).toHaveURL('/login')

    // Verify can't access protected routes
    await page.goto('/dashboard')
    await expect(page).toHaveURL('/login')
  })

  test('authenticated user can access all protected routes', async ({ page }) => {
    // Login
    await page.goto('/login')
    await page.fill('#email', 'admin@test.com')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('/dashboard')

    // Access various protected routes
    await page.goto('/dashboard/invoices')
    await expect(page.getByRole('heading', { name: 'Invoices' })).toBeVisible()

    await page.goto('/approvals')
    await expect(page.getByRole('heading', { name: 'Approvals' })).toBeVisible()

    await page.goto('/dashboard/payment-orders')
    await expect(page.getByRole('heading', { name: 'Payment Orders' })).toBeVisible()
  })
})
