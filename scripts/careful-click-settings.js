const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2da1493154e21e6291302?envID=6ac2683a6a873116ad572b40', { waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);

    const beforeText = await page.evaluate(() => document.body.innerText);
    console.log('BEFORE click, page state:\n', beforeText.substring(0, 1000));
    console.log('---');

    const positions = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button,a')).filter(
        el => el.innerText && el.innerText.trim() === 'Settings' && el.offsetParent !== null
      );
      return els.map((el, i) => {
        const rect = el.getBoundingClientRect();
        return { i, top: rect.top, left: rect.left, tag: el.tagName, html: el.outerHTML.substring(0, 150) };
      });
    });
    console.log('Settings candidates:', JSON.stringify(positions, null, 2));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
