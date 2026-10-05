const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Click on tayooli-backend service in the left sidebar
    const backendBtn = page.locator('button:has-text("tayooli-backend"), a:has-text("tayooli-backend")').first();
    await backendBtn.click();
    await page.waitForTimeout(2000);
    console.log('Clicked tayooli-backend. Current URL:', page.url());

    // Check Overview text
    const text = await page.evaluate(() => document.body.innerText);
    console.log('Service Overview text:\n', text.substring(0, 1200));

    // Also check if there is an error message displayed
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/backend-overview-detail.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
