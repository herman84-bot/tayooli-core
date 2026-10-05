const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Find number-type input near "Port" label under Port Forwarding
    const numberInputs = await page.locator('input[type="number"]').all();
    console.log('Number inputs found:', numberInputs.length);
    for (let i = 0; i < numberInputs.length; i++) {
      const val = await numberInputs[i].inputValue();
      console.log(`Input ${i}: value="${val}"`);
    }

    if (numberInputs.length > 0) {
      await numberInputs[numberInputs.length - 1].fill('5432');
      console.log('Filled port 5432');
    }

    const saveBtn = page.locator('button:has-text("Save")').first();
    if (await saveBtn.isVisible().catch(() => false)) {
      await saveBtn.click();
      console.log('Clicked Save');
      await page.waitForTimeout(3000);
    }

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After save:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
