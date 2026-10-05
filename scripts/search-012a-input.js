const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const searchInput = page.locator('input[placeholder*="Search Logs"], input[placeholder*="Search logs"]').first();
    await searchInput.waitFor({ state: 'visible', timeout: 10000 });
    await searchInput.fill('012a');
    await page.waitForTimeout(2000);
    const text = await page.evaluate(() => document.body.innerText);
    console.log('After searching 012a:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
