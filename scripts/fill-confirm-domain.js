const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const inputs = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('input')).filter(i => i.offsetParent !== null).map(i => ({ placeholder: i.placeholder, value: i.value }));
    });
    console.log('Visible inputs before fill:', JSON.stringify(inputs));

    const domainInput = page.locator('input').filter({ hasNot: page.locator('[type="file"]') }).first();
    await domainInput.click();
    await domainInput.fill('tayooli-frontend');
    await page.waitForTimeout(500);

    const val = await domainInput.inputValue();
    console.log('Domain input value:', val);

    const confirmed = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button')).filter(
        el => el.innerText && el.innerText.trim() === 'Confirm' && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return true; }
      return false;
    });
    console.log('Clicked Confirm:', confirmed);
    await page.waitForTimeout(3000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After Confirm:\n', text.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
