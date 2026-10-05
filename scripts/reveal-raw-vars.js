const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const editRawBtn = page.locator('button:has-text("Edit Raw Variables")').first();
    await editRawBtn.click();
    await page.waitForTimeout(2000);

    const ta = page.locator('textarea').first();
    const value = await ta.inputValue();
    console.log('Raw variables:\n', value);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
