const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    
    // Close any modal if visible
    const closeBtn = page.locator('button:has-text("Close")').first();
    if (await closeBtn.isVisible()) {
      await closeBtn.click();
      console.log('Closed modal');
      await page.waitForTimeout(1000);
    }

    // Click Back
    const backBtn = page.locator('button:has-text("Back"), a:has-text("Back")').first();
    if (await backBtn.isVisible()) {
      await backBtn.click();
      console.log('Clicked Back');
      await page.waitForTimeout(1000);
    }

    // Now click Variable tab
    const varTab = page.locator('button[data-tab-value="variables"], button:has-text("Variable")').first();
    await varTab.click({ force: true });
    await page.waitForTimeout(2000);

    const varText = await page.evaluate(() => document.body.innerText);
    console.log('Variables page text:\n', varText.substring(0, 1500));
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/backend-variables.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
