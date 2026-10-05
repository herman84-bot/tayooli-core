const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const context = browser.contexts()[0];
    const pages = context.pages();
    const page = pages.find(p => p.url().includes('tayooli.my.id'));
    if (!page) { console.log('not found'); return; }
    console.log('URL:', page.url());
    await page.reload({ waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);
    const text = await page.evaluate(() => document.body.innerText.substring(0, 3000));
    console.log(text);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
