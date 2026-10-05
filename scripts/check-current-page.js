const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    console.log('Current URL:', page.url());
    console.log('Current Title:', await page.title());
    const snippet = await page.evaluate(() => document.body.innerText.substring(0, 1000));
    console.log('Snippet:\n', snippet);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
