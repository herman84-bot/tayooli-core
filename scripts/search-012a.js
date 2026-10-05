const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    const text = await page.evaluate(() => document.body.innerText);
    console.log('Contains "012a":', text.includes('012a'));
    const idx = text.indexOf('012a');
    if (idx >= 0) console.log(text.substring(Math.max(0,idx-200), idx+200));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
