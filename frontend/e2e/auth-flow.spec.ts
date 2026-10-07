import { test, expect } from '@playwright/test'

test.describe('Authentication Flow (Tayooli Core)', () => {
  test('visit dashboard → redirect to login → login → redirect back', async ({ page }) => {
    // Attempt to access protected route unauthenticated
    await page.goto('/dashboard')

    // Should redirect to login
    await expect(page).toHaveURL('/login')
    await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible()

    // Fill login form with demo credentials
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
    await expect(page.getByTestId('login-error-text')).toHaveText(/Email atau kata sandi salah|Invalid credentials|invalid credentials/i)
    await expect(page).toHaveURL('/login')
  })

  test('logout clears session and redirects to login', async ({ page }) => {
    // Login first
    await page.goto('/login')
    await page.fill('#email', 'admin@test.com')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('/dashboard')

    // Logout via sidebar button (matches aria-label="Logout")
    await page.getByRole('button', { name: 'Logout' }).first().click()
    await expect(page).toHaveURL('/login')

    // Verify unauthenticated user cannot access protected routes
    await page.goto('/dashboard')
    await expect(page).toHaveURL('/login')
  })

  test('authenticated user can access core protected routes (/dashboard, /products, /pos, /settings)', async ({ page }) => {
    // Login with demo credentials
    await page.goto('/login')
    await page.fill('#email', 'admin@test.com')
    await page.fill('#password', 'password123')
    await page.click('button[type="submit"]')
    await expect(page).toHaveURL('/dashboard')

    // 1. /dashboard
    await page.goto('/dashboard')
    await expect(page).toHaveURL('/dashboard')
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()

    // 2. /products
    await page.goto('/products')
    await expect(page).toHaveURL('/products')
    await expect(page.getByRole('heading', { name: /Katalog Produk/i })).toBeVisible()

    // 3. /pos
    await page.goto('/pos')
    await expect(page).toHaveURL('/pos')
    await expect(page.getByRole('heading', { name: /Point of Sale/i })).toBeVisible()

    // 4. /settings
    await page.goto('/settings')
    await expect(page).toHaveURL('/settings')
    await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible()
  })
})
