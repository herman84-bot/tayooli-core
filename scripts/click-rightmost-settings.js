const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Navigate back to the frontend service first
    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2da1493154e21e6291302?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const clicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button,a')).filter(
        el => el.innerText && el.innerText.trim() === 'Settings' && el.offsetParent !== null
      );
      if (els.length === 0) return false;
      // Pick the one furthest to the right (the service tab bar's Settings tab)
      els.sort((a, b) => b.getBoundingClientRect().left - a.getBoundingClientRect().left);
      els[0].click();
      return true;
    });
    console.log('Clicked rightmost Settings:', clicked);
    await page.waitForTimeout(2000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Result:\n', text.substring(0, 2500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
