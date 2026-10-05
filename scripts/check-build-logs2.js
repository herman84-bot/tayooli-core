const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const text1 = await page.evaluate(() => document.body.innerText);
    console.log('Current state:\n', text1.substring(0, 1000));

    const logsClicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button')).filter(
        el => el.innerText && el.innerText.trim() === 'Logs' && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return true; }
      return false;
    });
    console.log('Clicked Logs:', logsClicked);
    await page.waitForTimeout(2000);

    const text2 = await page.evaluate(() => document.body.innerText);
    console.log('Logs page:\n', text2.substring(0, 5000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
