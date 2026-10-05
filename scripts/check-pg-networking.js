const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2686e93154e21e628ecd1?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const netTab = page.locator('button:has-text("Networking")').first();
    await netTab.click();
    await page.waitForTimeout(2000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Networking tab:\n', text.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
