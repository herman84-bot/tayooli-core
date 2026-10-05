const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const info = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button')).filter(b => b.innerText && b.innerText.trim() === 'Restart');
      return btns.map(b => ({ visible: b.offsetParent !== null, html: b.outerHTML.substring(0, 150) }));
    });
    console.log(JSON.stringify(info, null, 2));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
