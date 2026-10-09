import { test, expect, type Page } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

// READ-ONLY production audit of the 12 core modules (CLAUDE.md).
// For every page we record: HTTP failures (>=400) from the app's own API,
// console errors, uncaught page errors, and visible "junk" text
// (undefined / NaN / [object Object] / raw JSON / raw markdown).
// No form is submitted here; mutation probes live in audit-validation.spec.ts.

const EMAIL = process.env.TAYOOLI_EMAIL || 'admin@test.com';
const PASSWORD = process.env.TAYOOLI_PASSWORD || 'password123';

const MODULES: { name: string; route: string }[] = [
  { name: '01 Dashboard', route: '/dashboard' },
  { name: '02 Products', route: '/products' },
  { name: '03 Barang Masuk & Keluar', route: '/wms/arus-barang' },
  { name: '03b Inbound (legacy menu)', route: '/wms/inbound' },
  { name: '03c Surat Jalan (legacy menu)', route: '/wms/delivery-orders' },
  { name: '04 Warehouse & Stock', route: '/wms' },
  { name: '05 Marketplace', route: '/wms/marketplace' },
  { name: '06 Transfers', route: '/wms/transfers' },
  { name: '07 Opname', route: '/wms/opname' },
  { name: '08 Scrap', route: '/wms/scrap' },
  { name: '09 Scanner', route: '/wms/scanner' },
  { name: '10 POS', route: '/pos' },
  { name: '11 Settings', route: '/settings' },
  { name: '12 Help', route: '/help' },
];

type Finding = {
  module: string;
  route: string;
  finalUrl: string;
  httpFailures: string[];
  consoleErrors: string[];
  pageErrors: string[];
  junkText: string[];
  heading: string;
  screenshot: string;
};

const OUT = path.resolve(__dirname, '../../test-results/audit');
const findings: Finding[] = [];

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('#email', EMAIL);
  await page.fill('#password', PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/dashboard/, { timeout: 30_000 });
}

const JUNK: { label: string; re: RegExp }[] = [
  { label: 'undefined', re: /\bundefined\b/ },
  { label: 'NaN', re: /\bNaN\b/ },
  { label: '[object Object]', re: /\[object Object\]/ },
  { label: 'raw JSON', re: /\{"(error|data|code|message)"\s*:/ },
  { label: 'raw markdown **', re: /\*\*[^*\n]{2,40}\*\*/ },
  { label: 'Rp NaN / Rp undefined', re: /Rp\s*(NaN|undefined)/ },
  { label: 'Invalid Date', re: /Invalid Date/ },
];

test.describe.configure({ mode: 'default' });

test.beforeAll(() => fs.mkdirSync(OUT, { recursive: true }));

for (const m of MODULES) {
  test(`audit ${m.name}`, async ({ page }) => {
    const httpFailures: string[] = [];
    const consoleErrors: string[] = [];
    const pageErrors: string[] = [];

    // Login first; only errors AFTER authentication count as findings
    // (the login page's own /auth/me 401 is expected behaviour).
    await login(page);

    page.on('response', (r) => {
      const u = r.url();
      if (r.status() >= 400 && /\/api\/v1\//.test(u)) httpFailures.push(`${r.status()} ${r.request().method()} ${u.replace(/^https?:\/\/[^/]+/, '')}`);
    });
    page.on('console', (msg) => {
      if (msg.type() === 'error') consoleErrors.push(msg.text().slice(0, 300));
    });
    page.on('pageerror', (e) => pageErrors.push(String(e.message).slice(0, 300)));

    await page.goto(m.route);
    await page.waitForLoadState('networkidle', { timeout: 30_000 }).catch(() => {});
    await page.waitForTimeout(2_500);

    const body = (await page.locator('main').first().innerText().catch(async () => page.locator('body').innerText())) || '';
    const junkText: string[] = [];
    for (const j of JUNK) {
      const mm = body.match(j.re);
      if (mm) {
        const i = body.indexOf(mm[0]);
        junkText.push(`${j.label}: "…${body.slice(Math.max(0, i - 50), i + 60).replace(/\s+/g, ' ')}…"`);
      }
    }
    const heading = (await page.locator('h1').first().innerText().catch(() => '')).trim();
    const shot = path.join(OUT, `${m.name.replace(/[^a-z0-9]+/gi, '_')}.png`);
    await page.screenshot({ path: shot, fullPage: true });

    const f: Finding = {
      module: m.name,
      route: m.route,
      finalUrl: page.url().replace(/^https?:\/\/[^/]+/, ''),
      httpFailures: [...new Set(httpFailures)],
      consoleErrors: [...new Set(consoleErrors)],
      pageErrors: [...new Set(pageErrors)],
      junkText,
      heading,
      screenshot: path.relative(path.resolve(__dirname, '../..'), shot),
    };
    findings.push(f);
    // Persist per-module so results survive worker restarts after failures.
    fs.writeFileSync(path.join(OUT, `module-${m.name.replace(/[^a-z0-9]+/gi, '_')}.json`), JSON.stringify(f, null, 2));

    // Soft expectations so every module is audited even if one fails.
    expect.soft(pageErrors, 'uncaught page errors').toEqual([]);
    expect.soft(httpFailures, 'API calls failing').toEqual([]);
  });
}

test.afterAll(() => {
  fs.writeFileSync(path.join(OUT, 'modules.json'), JSON.stringify(findings, null, 2));
});
