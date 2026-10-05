const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    
    const buildLogsTab = page.locator('button:has-text("Build logs")').first();
    if (await buildLogsTab.isVisible()) {
      await buildLogsTab.click();
      await page.waitForTimeout(2000);
      const text = await page.evaluate(() => document.body.innerText);
      console.log('Build logs content:\n', text.substring(0, 1500));
    }
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
