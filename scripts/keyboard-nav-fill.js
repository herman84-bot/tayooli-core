const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Focus via JS then type with keyboard (real keydown events) instead of fill()
    await page.evaluate(() => {
      const el = document.querySelector('#git-url');
      el.focus();
    });
    await page.waitForTimeout(300);

    const activeEl = await page.evaluate(() => document.activeElement.id);
    console.log('Active element id:', activeEl);

    await page.keyboard.type('https://github.com/herman84-bot/tayooli-core.git', { delay: 20 });
    await page.waitForTimeout(500);

    const val = await page.evaluate(() => document.querySelector('#git-url').value);
    console.log('git-url value after typing:', val);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
