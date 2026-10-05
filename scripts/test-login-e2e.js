const { chromium } = require('@playwright/test');
const { requireCredentials } = require('./_live-helpers');
// Credentials: set TAYOOLI_EMAIL and TAYOOLI_PASSWORD (see scripts/_live-helpers.js).

(async () => {
  const creds = requireCredentials();
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const context = browser.contexts()[0];
    const page = await context.newPage();

    await page.goto('https://tayooli-frontend.zeabur.app/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    console.log('Page title:', await page.title());
    console.log('URL:', page.url());

    // Try to find email/password inputs
    const emailInput = page.locator('input[type="email"], input[name="email"]').first();
    const passwordInput = page.locator('input[type="password"]').first();

    await emailInput.fill(creds.email);
    await passwordInput.fill(creds.password);
    await page.waitForTimeout(500);

    const submitBtn = page.locator('button[type="submit"]').first();
    await submitBtn.click();
    await page.waitForTimeout(4000);

    console.log('URL after login attempt:', page.url());
    const bodyText = await page.evaluate(() => document.body.innerText.substring(0, 800));
    console.log('Page content after login:\n', bodyText);

    await page.close();
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
