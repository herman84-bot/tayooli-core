const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Click on the most recent "Removed" deployment to view its logs
    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac268f493154e21e628ed15?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const logsBtn = page.locator('button:has-text("Logs")').first();
    await logsBtn.click();
    await page.waitForTimeout(3000);

    const fullText = await page.evaluate(() => document.body.innerText);
    console.log('FULL TEXT LENGTH:', fullText.length);
    console.log('Contains 012a:', fullText.includes('012a'));
    console.log('Contains "021_wms":', fullText.includes('021_wms'));
    console.log(fullText);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
