const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const info = await page.evaluate(() => {
      const el = document.querySelector('#git-url');
      if (!el) return 'not found';
      return {
        disabled: el.disabled,
        readOnly: el.readOnly,
        outerHTML: el.outerHTML.substring(0, 500),
      };
    });
    console.log(JSON.stringify(info, null, 2));

    // Try clicking directly on the git-url field (maybe it opens a selection UI)
    await page.locator('#git-url').click({ force: true });
    await page.waitForTimeout(1000);
    const text = await page.evaluate(() => document.body.innerText);
    console.log('After clicking git-url field:\n', text.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
