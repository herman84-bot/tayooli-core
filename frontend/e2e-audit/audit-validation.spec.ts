import { test, expect, type Page } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

// Production validation audit through the real UI.
// SAFETY: every input here MUST be rejected. If the backend ever accepts one,
// the test deletes the created product immediately (cleanup) and FAILS,
// because that acceptance is itself the finding.

const EMAIL = process.env.TAYOOLI_EMAIL || 'admin@test.com';
const PASSWORD = process.env.TAYOOLI_PASSWORD || 'password123';
const OUT = path.resolve(__dirname, '../../test-results/audit');

let authState: Awaited<ReturnType<import('@playwright/test').BrowserContext['storageState']>> | undefined;

async function login(page: Page) {
  if (authState) {
    await page.context().addCookies(authState.cookies);
    await page.context().addInitScript((origins) => {
      for (const o of origins) for (const kv of o.localStorage) localStorage.setItem(kv.name, kv.value);
    }, authState.origins);
    return;
  }
  await page.goto('/login');
  await page.fill('#email', EMAIL);
  await page.fill('#password', PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/dashboard/, { timeout: 30_000 });
  authState = await page.context().storageState();
}

type Case = { name: string; product: string; sku: string; price: string; cost?: string };

const CASES: Case[] = [
  { name: 'price 0', product: 'Audit Harga Nol', sku: `AUD-P0-${Date.now()}`, price: '0' },
  { name: 'price negative', product: 'Audit Harga Minus', sku: `AUD-PN-${Date.now()}`, price: '-500' },
  { name: 'empty name', product: '', sku: `AUD-EN-${Date.now()}`, price: '1000' },
  { name: 'empty SKU', product: 'Audit SKU Kosong', sku: '', price: '1000' },
  { name: 'whitespace-only name', product: '    ', sku: `AUD-WS-${Date.now()}`, price: '1000' },
  { name: 'negative cost price', product: 'Audit HPP Minus', sku: `AUD-CN-${Date.now()}`, price: '1000', cost: '-1' },
  { name: 'duplicate existing SKU (lowercase of 275424)', product: 'Audit SKU Kembar', sku: '275424', price: '1000' },
];

const results: { case: string; uiError: string; apiStatus: number | null; accepted: boolean; screenshot: string }[] = [];

test.describe.configure({ mode: 'default' });

for (const c of CASES) {
  test(`product form rejects: ${c.name}`, async ({ page }) => {
    await login(page);
    await page.goto('/products');
    await page.getByRole('button', { name: /Tambah Produk/ }).first().click();

    await page.getByPlaceholder('Contoh: Kertas HVS A4 80gsm').fill(c.product);
    await page.getByPlaceholder('Contoh: HVS-A4-80').fill(c.sku);
    await page.getByPlaceholder('Contoh: 55000').fill(c.price);
    if (c.cost !== undefined) await page.locator('#create-cost-price').fill(c.cost);

    let apiStatus: number | null = null;
    let createdId: string | null = null;
    page.on('response', async (r) => {
      if (r.request().method() === 'POST' && /\/api\/v1\/products\/?$/.test(new URL(r.url()).pathname)) {
        apiStatus = r.status();
        if (r.ok()) {
          try { const j = await r.json(); createdId = (j.data || j).id || null; } catch { /* ignore */ }
        }
      }
    });

    await page.getByRole('button', { name: 'Simpan Produk' }).click();
    await page.waitForTimeout(2_500);

    const errBox = page.locator('form .bg-red-50').first();
    const uiError = (await errBox.isVisible().catch(() => false)) ? (await errBox.innerText()).trim() : '';
    const shot = path.join(OUT, `validation-${c.name.replace(/[^a-z0-9]+/gi, '_')}.png`);
    await page.screenshot({ path: shot, fullPage: false });

    const accepted = apiStatus !== null && apiStatus >= 200 && apiStatus < 300;
    if (accepted && createdId) {
      // cleanup: never leave audit junk in production
      await page.evaluate(async (id) => { await fetch(`/api/v1/products/${id}`, { method: 'DELETE', credentials: 'include' }); }, createdId);
    }

    results.push({ case: c.name, uiError, apiStatus, accepted, screenshot: path.basename(shot) });
    fs.mkdirSync(OUT, { recursive: true });
    fs.writeFileSync(path.join(OUT, `validation-${c.name.replace(/[^a-z0-9]+/gi, '_')}.json`), JSON.stringify(results[results.length - 1], null, 2));

    expect(accepted, `backend ACCEPTED invalid input (${c.name}); API status ${apiStatus}`).toBe(false);
    expect(uiError, 'user must see a clear error message').not.toBe('');
  });
}
