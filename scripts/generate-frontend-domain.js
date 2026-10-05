const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.locator('[data-tab-value="networking"], button:has-text("Networking")').first().click();
    await page.waitForTimeout(2000);

    const text1 = await page.evaluate(() => document.body.innerText);
    console.log('Networking tab:\n', text1.substring(0, 1500));

    const genBtn = page.locator('button:has-text("Generate Domain")').first();
    if (await genBtn.isVisible().catch(() => false)) {
      await genBtn.click();
      await page.waitForTimeout(2000);
      const text2 = await page.evaluate(() => document.body.innerText);
      console.log('After Generate Domain click:\n', text2.substring(0, 1500));
    } else {
      console.log('Generate Domain button not visible');
    }
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
