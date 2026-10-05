const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    if (!page) throw new Error('Zeabur page not found in BrowserOS');

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const addServiceBtn = page.locator('button:has-text("Add Service"), a:has-text("Add Service")').first();
    await addServiceBtn.click();
    await page.waitForTimeout(2000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After clicking Add Service:\n', text.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
