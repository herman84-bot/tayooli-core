const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const html = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button')).filter(
        el => el.innerText && el.innerText.trim() === 'Generate Domain'
      );
      return btns.map(b => ({
        visible: b.offsetParent !== null,
        disabled: b.disabled,
        outerHTML: b.outerHTML.substring(0, 300),
      }));
    });
    console.log(JSON.stringify(html, null, 2));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
