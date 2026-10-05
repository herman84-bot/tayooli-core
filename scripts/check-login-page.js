const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const context = browser.contexts()[0];
    const pages = context.pages();
    const page = pages.find(p => p.url().includes('tayooli-frontend.zeabur.app')) || pages[pages.length - 1];

    console.log('URL:', page.url());
    const text = await page.evaluate(() => document.body.innerText.substring(0, 1000));
    console.log('Body text:\n', text);

    const inputs = await page.evaluate(() => Array.from(document.querySelectorAll('input')).map(i => ({ type: i.type, name: i.name, placeholder: i.placeholder })));
    console.log('Inputs:', JSON.stringify(inputs));

    const buttons = await page.evaluate(() => Array.from(document.querySelectorAll('button')).map(b => b.innerText.trim()).filter(Boolean));
    console.log('Buttons:', JSON.stringify(buttons));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
