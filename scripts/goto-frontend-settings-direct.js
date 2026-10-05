const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2da1493154e21e6291302?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    // Within the service's own tab bar (Overview/Variable/Usage/Networking/Volumes/Settings),
    // click the LAST "Settings" occurrence which belongs to the service tab bar, not sidebar.
    const clicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button,a')).filter(
        el => el.innerText && el.innerText.trim() === 'Settings' && el.offsetParent !== null
      );
      console.log('Found', els.length, 'Settings elements');
      if (els.length > 0) {
        // The service tab bar Settings is usually the LAST one (sidebar's is first/global)
        els[els.length - 1].click();
        return els.length;
      }
      return 0;
    });
    console.log('Settings elements found & clicked last:', clicked);
    await page.waitForTimeout(2000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After clicking service Settings:\n', text.substring(0, 2500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
