const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac268f493154e21e628ed15?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(4000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Overview:\n', text.substring(0, 1200));
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/recheck-overview.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
