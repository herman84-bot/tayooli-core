const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    if (!page) {
      console.log('No Zeabur page found');
      return;
    }
    console.log('Found Zeabur page:', page.url());
    // Click Integration
    const integrationBtn = page.locator('button:has-text("Integration"), a:has-text("Integration")').first();
    if (await integrationBtn.isVisible()) {
      await integrationBtn.click();
      await page.waitForTimeout(2000);
      console.log('Integration URL:', page.url());
      const text = await page.evaluate(() => document.body.innerText);
      console.log('Integration text:\n', text.substring(0, 1000));
    } else {
      console.log('Integration button not visible');
    }
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
