const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const settingsTab = page.locator('button:has-text("Settings")').first();
    await settingsTab.click();
    await page.waitForTimeout(2000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Settings tab content:\n', text.substring(0, 2500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
