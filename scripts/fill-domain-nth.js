const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const domainInput = page.locator('input[type="text"]').nth(0);
    await domainInput.click({ force: true });
    await page.keyboard.type('tayooli-frontend', { delay: 15 });
    await page.waitForTimeout(500);

    const val = await domainInput.inputValue();
    console.log('Domain input value after typing:', val);

    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/domain-dialog.png' });

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Current text:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
