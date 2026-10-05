const { chromium } = require('@playwright/test');

(async () => {
  try {
    const browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    if (!page) {
      console.log('Page not found');
      await browser.close();
      return;
    }
    await page.locator('button:has-text("Overview")').first().click();
    await page.waitForTimeout(2000);
    const text = await page.evaluate(() => document.body.innerText);
    console.log('Overview text:\n', text.substring(0, 1500));
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/zeabur-overview.png' });
    await browser.close();
  } catch (err) {
    console.error('Error:', err);
  }
})();
