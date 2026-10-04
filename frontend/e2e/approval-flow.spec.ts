import { test, expect } from '@playwright/test'

// Approval Workflow — aligns with the live app:
//   - Invoice creation is a separate flow (dashboard/invoices/new); it does NOT
//     auto-create an approval request (the backend approval engine has no such
//     wiring). Approval requests are created by the approval engine directly.
//   - The approvals page (/approvals) lists pending requests; opening a row
//     opens a drawer with Approve / Reject.
//   - Approving/rejecting an approval request updates that request's status;
//     the invoice itself is approved via the invoice detail page.
//
// Because there is no UI for creating approval requests, each test seeds the
// underlying data through the same REST API the app uses:
//   - POST /api/v1/invoices (role admin/accountant) to create the invoice —
//     the approver role CANNOT create invoices (403), so invoice creation is
//     always done as accountant,
//   - INSERT the approval_requests row directly via psql (no API exists), and
//     the app's list/approve/reject endpoints surface it.
//
// Seed users (013_seed_approval_users.sql): accountant@test.com / password123,
// approver@test.com / password123.

const API = 'http://localhost:8081/api/v1'
const TENANT_ID = '550e8400-e29b-41d4-a716-446655440000'

async function login(page: import('@playwright/test').Page, email: string, password: string) {
  await page.goto('/login')
  await page.fill('#email', email)
  await page.fill('#password', password)
  await page.click('button[type="submit"]')
  await expect(page).toHaveURL('/dashboard')
}

async function apiLogin(email: string, password: string): Promise<{ userId: string; token: string }> {
  const res = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) throw new Error(`login failed: ${res.status}`)
  const body = (await res.json()) as { user: { id: string } }
  // Token is in the Set-Cookie header (HttpOnly). Parse it for API auth.
  const setCookie = res.headers.get('set-cookie') ?? ''
  const token = setCookie.split(';')[0].split('=')[1] ?? ''
  return { userId: body.user.id, token }
}

// Create an AP invoice directly via the API (role: admin/accountant).
async function createInvoice(token: string, invoiceNumber: string, amount: number) {
  const res = await fetch(`${API}/invoices`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      vendor_id: 'VENDOR001',
      invoice_number: invoiceNumber,
      amount: amount.toString(),
      currency: 'IDR',
    }),
  })
  if (!res.ok) throw new Error(`create invoice failed: ${res.status}`)
  const inv = (await res.json()) as { id: string; invoice_number: string }
  return inv
}

// Seed an approval request for the invoice (there is no API for this).
// NOTE: approval_requests.entity_type/entity_id are NOT NULL — they are the
// actual "what is being approved" columns (target_type/target_id are optional
// legacy fields). Both are seeded here.
async function seedApprovalRequest(userId: string, invoiceId: string) {
  const { execSync } = await import('node:child_process')
  const id = crypto.randomUUID()
  const sql = `SET app.current_tenant_id = '${TENANT_ID}';
INSERT INTO approval_requests (id, tenant_id, entity_type, entity_id, workflow_id, target_type, target_id, status, current_step_index, requested_by)
VALUES ('${id}', '${TENANT_ID}', 'invoice', '${invoiceId}', '00000000-0000-0000-0000-000000000000', 'invoice', '${invoiceId}', 'pending', 0, '${userId}');`
  execSync(`"C:/Program Files/PostgreSQL/17/bin/psql.exe" -h localhost -p 5432 -U tayooli -d tayooli -v ON_ERROR_STOP=1 -c "${sql.replace(/"/g, '\\"')}"`, {
    env: { ...process.env, PGPASSWORD: 'tayooli' },
  })
  return id
}

test.describe('Approval Workflow', () => {
  test('pending approval → approve via approvals page', async ({ page }) => {
    // Login as accountant (seeded) — can create invoices.
    await login(page, 'accountant@test.com', 'password123')

    // Create invoice via API and seed a pending approval request.
    const { userId, token } = await apiLogin('accountant@test.com', 'password123')
    const inv = await createInvoice(token, `INV-APPROVAL-${Date.now()}`, 150000000)
    await seedApprovalRequest(userId, inv.id)

    // Approvals page shows the pending request (target_id = invoice id).
    await page.goto('/approvals')
    await expect(page.getByRole('heading', { name: 'Approvals' })).toBeVisible()
    await expect(page.locator(`text=${inv.id.slice(0, 8)}`).first()).toBeVisible()

    // Open the request and approve it.
    await page.locator('tr', { hasText: inv.id.slice(0, 8) }).first().click()
    await page.getByRole('button', { name: 'Approve' }).click()
    await expect(page.getByText('Approved', { exact: true })).toBeVisible()
  })

  test('pending approval → reject with reason', async ({ page }) => {
    await login(page, 'approver@test.com', 'password123')

    // Invoice creation is admin/accountant-only (approver gets 403), so create
    // the invoice as accountant, then seed the request for the approver to see.
    const accountant = await apiLogin('accountant@test.com', 'password123')
    const inv = await createInvoice(accountant.token, `INV-REJECT-${Date.now()}`, 120000000)
    await seedApprovalRequest(accountant.userId, inv.id)

    await page.goto('/approvals')
    await expect(page.getByRole('heading', { name: 'Approvals' })).toBeVisible()
    await expect(page.locator(`text=${inv.id.slice(0, 8)}`).first()).toBeVisible()

    // Reject with a reason.
    await page.locator('tr', { hasText: inv.id.slice(0, 8) }).first().click()
    await page.getByRole('button', { name: 'Reject' }).click()
    await page.locator('input[placeholder="Why are you rejecting this?"]').fill('Rejected - Missing documentation')
    await page.getByRole('button', { name: 'Confirm Rejection' }).click()
    await expect(page.getByText('Rejected', { exact: true })).toBeVisible()
  })

  test('invoice detail approve button approves invoice directly', async ({ page }) => {
    // Low-value invoice: no manual approval needed — invoice detail approve works.
    await login(page, 'accountant@test.com', 'password123')

    const { token } = await apiLogin('accountant@test.com', 'password123')
    const inv = await createInvoice(token, `INV-LOW-${Date.now()}`, 5000000)

    // Invoice detail page shows approve action for pending invoice.
    await page.goto(`/dashboard/invoices/${inv.id}`)
    await expect(page.getByRole('button', { name: 'Setujui Invoice' })).toBeVisible()
    await page.getByRole('button', { name: 'Setujui Invoice' }).click()
    await expect(page.getByText('Invoice berhasil disetujui. Data diperbarui.')).toBeVisible()
  })
})
