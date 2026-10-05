const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    
    // Click Restart button
    const restartBtn = page.locator('button:has-text("Restart")').first();
    if (await restartBtn.isVisible()) {
      await restartBtn.click();
      console.log('Clicked Restart!');
      await page.waitForTimeout(3000);
    } else {
      console.log('Restart button not visible');
    }

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Status snippet after restart:\n', text.substring(0, 800));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
