const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Click "Back" first to leave Project Settings
    const backClicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button,a')).filter(
        el => el.innerText && el.innerText.trim() === 'Back' && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return true; }
      return false;
    });
    console.log('Clicked Back:', backClicked);
    await page.waitForTimeout(2000);

    console.log('URL now:', page.url());
    const text = await page.evaluate(() => document.body.innerText);
    console.log('After Back:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
