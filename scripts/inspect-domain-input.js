const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const info = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('input')).map(i => ({
        visible: i.offsetParent !== null,
        type: i.type,
        placeholder: i.placeholder,
        id: i.id,
        value: i.value,
        outerHTML: i.outerHTML.substring(0, 200),
      }));
    });
    console.log(JSON.stringify(info, null, 2));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
