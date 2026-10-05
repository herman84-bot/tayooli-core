const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const portInput = page.locator('input[placeholder="8080"]').first();
    await portInput.click();
    await portInput.fill('3000');
    await page.waitForTimeout(500);

    const val = await portInput.inputValue();
    console.log('Port input value:', val);

    const exposeBtn = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button')).filter(
        el => el.innerText && el.innerText.trim() === 'Expose' && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return true; }
      return false;
    });
    console.log('Clicked Expose:', exposeBtn);
    await page.waitForTimeout(2500);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After Expose click:\n', text.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
