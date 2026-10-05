const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/pg-networking.png', fullPage: true });
    console.log('Screenshot saved');
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
