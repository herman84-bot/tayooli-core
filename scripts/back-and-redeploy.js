const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2da1493154e21e6291302?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const clicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button')).filter(
        el => el.innerText && el.innerText.trim() === 'Redeploy' && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return true; }
      return false;
    });
    console.log('Clicked Redeploy:', clicked);
    await page.waitForTimeout(1500);

    // Handle confirm dialog
    const confirmClicked = await page.evaluate(() => {
      const portal = document.getElementById('headlessui-portal-root');
      if (!portal) return 'no portal';
      const btns = Array.from(portal.querySelectorAll('button')).filter(b => b.innerText && b.innerText.trim() === 'Redeploy');
      if (btns.length > 0) { btns[0].click(); return 'confirmed'; }
      return 'no redeploy btn in portal';
    });
    console.log('Confirm result:', confirmClicked);
    await page.waitForTimeout(3000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After redeploy trigger:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
