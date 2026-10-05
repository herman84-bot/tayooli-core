const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Press Escape to go back to the main Add Service menu
    await page.keyboard.press('Escape');
    await page.waitForTimeout(1000);

    const text1 = await page.evaluate(() => document.body.innerText);
    console.log('After Escape:\n', text1.substring(0, 1500));

    // Re-open Add Service if needed
    if (!text1.includes('Git Repository')) {
      const addServiceBtn = page.locator('button:has-text("Add Service"), a:has-text("Add Service")').first();
      await addServiceBtn.click();
      await page.waitForTimeout(1500);
    }

    const clicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button,a,div,span,li')).filter(
        el => el.innerText && el.innerText.trim() === 'Git Repository' && el.offsetParent !== null
      );
      if (els.length > 0) {
        els[0].click();
        return true;
      }
      return false;
    });
    console.log('Clicked Git Repository:', clicked);
    await page.waitForTimeout(2000);

    const text2 = await page.evaluate(() => document.body.innerText);
    console.log('After selecting Git Repository:\n', text2.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
