const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const inputs = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('input')).map(i => ({ placeholder: i.placeholder, type: i.type, visible: i.offsetParent !== null }));
    });
    console.log('Inputs found:', JSON.stringify(inputs, null, 2));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
