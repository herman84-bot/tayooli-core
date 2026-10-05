const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const editRawBtn = page.locator('button:has-text("Edit Raw Variables")').first();
    await editRawBtn.click();
    await page.waitForTimeout(1500);

    const ta = page.locator('textarea').first();
    const currentVal = await ta.inputValue();
    console.log('Current:', currentVal);

    const newVal = currentVal.trim() + '\nPORT=3000\nHOSTNAME=0.0.0.0\n';
    await ta.click();
    await ta.fill(newVal);
    await page.waitForTimeout(500);

    const val = await ta.inputValue();
    console.log('New:', val);

    const saveBtn = page.locator('button:has-text("Save")').first();
    await saveBtn.click();
    console.log('Saved PORT=3000');
    await page.waitForTimeout(2500);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After save:\n', text.substring(0, 1200));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
