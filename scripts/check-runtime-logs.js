const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Click Runtime logs tab
    const runtimeTab = page.locator('button:has-text("Runtime logs")').first();
    if (await runtimeTab.isVisible()) {
      await runtimeTab.click();
      await page.waitForTimeout(2000);
      const text = await page.evaluate(() => document.body.innerText);
      console.log('Runtime logs:\n', text.substring(0, 1500));
    }
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
