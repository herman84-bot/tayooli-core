const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    await page.locator('[data-tab-value="settings"]').click();
    await page.waitForTimeout(2000);
    const text = await page.evaluate(() => document.body.innerText);
    console.log(text.substring(0, 2800));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
