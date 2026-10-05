const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2da1493154e21e6291302?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const text1 = await page.evaluate(() => document.body.innerText);
    console.log('Frontend service overview:\n', text1.substring(0, 1500));

    const clicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button')).filter(
        el => el.innerText && (el.innerText.trim() === 'Redeploy' || el.innerText.trim() === 'Deploy') && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return els[0].innerText.trim(); }
      return null;
    });
    console.log('Clicked button:', clicked);
    await page.waitForTimeout(2000);

    const text2 = await page.evaluate(() => document.body.innerText);
    console.log('After click:\n', text2.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
