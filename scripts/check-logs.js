const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    
    // Click on Logs button
    const logsBtn = page.locator('button:has-text("Logs")').first();
    await logsBtn.click();
    await page.waitForTimeout(3000);

    const logText = await page.evaluate(() => document.body.innerText);
    console.log('Logs text:\n', logText.substring(0, 1500));
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/backend-logs.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
