const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const context = browser.contexts()[0];
    const pages = context.pages();
    const page = pages.find(p => p.url().includes('tayooli-frontend.zeabur.app'));

    const emailVal = await page.evaluate(() => document.querySelector('input[type="email"]').value);
    const passVal = await page.evaluate(() => document.querySelector('input[type="password"]').value);
    console.log('email value:', emailVal, '| password filled:', !!passVal);

    if (!emailVal) {
      await page.fill('input[type="email"]', 'admin@test.com');
      await page.fill('input[type="password"]', 'password123');
      await page.waitForTimeout(500);
    }

    const clicked = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button')).filter(b => b.innerText.trim() === 'Masuk');
      if (btns.length > 0) { btns[0].click(); return true; }
      return false;
    });
    console.log('Clicked Masuk:', clicked);
    await page.waitForTimeout(4000);

    console.log('URL after:', page.url());
    const text = await page.evaluate(() => document.body.innerText.substring(0, 1000));
    console.log('Page content:\n', text);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
