const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const clicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button')).filter(
        el => el.innerText && el.innerText.trim() === 'Expose Port' && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return true; }
      return false;
    });
    console.log('Clicked Expose Port:', clicked);
    await page.waitForTimeout(1500);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After Expose Port click:\n', text.substring(0, 2000));

    const inputs = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('input')).filter(i => i.offsetParent !== null).map(i => ({ placeholder: i.placeholder, value: i.value, type: i.type }));
    });
    console.log('Visible inputs:', JSON.stringify(inputs));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
