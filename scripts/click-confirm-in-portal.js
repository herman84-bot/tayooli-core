const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const clicked = await page.evaluate(() => {
      const portal = document.getElementById('headlessui-portal-root');
      if (!portal) return false;
      const btns = Array.from(portal.querySelectorAll('button')).filter(b => b.innerText.trim() === 'Confirm');
      if (btns.length > 0) { btns[0].click(); return true; }
      return false;
    });
    console.log('Clicked Confirm:', clicked);
    await page.waitForTimeout(3000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After Confirm:\n', text.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
