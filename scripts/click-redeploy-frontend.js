const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const clicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button')).filter(
        el => el.innerText && el.innerText.trim() === 'Redeploy' && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return true; }
      return false;
    });
    console.log('Clicked Redeploy:', clicked);
    await page.waitForTimeout(2000);

    // Handle confirmation dialog if present
    const confirmClicked = await page.evaluate(() => {
      const portal = document.getElementById('headlessui-portal-root');
      if (!portal) return 'no portal';
      const btns = Array.from(portal.querySelectorAll('button')).filter(b => b.innerText && (b.innerText.trim() === 'Redeploy' || b.innerText.trim() === 'Confirm'));
      if (btns.length > 0) { btns[0].click(); return btns[0].innerText.trim(); }
      return 'no confirm button';
    });
    console.log('Confirm click result:', confirmClicked);
    await page.waitForTimeout(3000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After redeploy:\n', text.substring(0, 1800));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
