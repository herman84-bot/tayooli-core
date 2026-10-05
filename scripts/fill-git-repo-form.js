const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // List all text inputs with their labels via preceding sibling/parent text
    const inputs = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('input')).map((el, i) => {
        return {
          index: i,
          placeholder: el.placeholder,
          type: el.type,
          name: el.name,
          value: el.value,
        };
      });
    });
    console.log('Inputs:', JSON.stringify(inputs, null, 2));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
