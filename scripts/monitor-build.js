const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    
    // Check current status
    const text = await page.evaluate(() => document.body.innerText);
    console.log('Status snippet:\n', text.substring(0, 800));

    // Click on Logs
    const logsBtn = page.locator('button:has-text("Logs")').first();
    if (await logsBtn.isVisible()) {
      await logsBtn.click();
      await page.waitForTimeout(3000);
      console.log('Logs:\n', (await page.evaluate(() => document.body.innerText)).substring(0, 1500));
    }
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
