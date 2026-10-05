const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    async function clearAndType(selector, text) {
      await page.evaluate((sel) => {
        const el = document.querySelector(sel);
        el.focus();
      }, selector);
      await page.keyboard.press('Control+A');
      await page.keyboard.press('Backspace');
      await page.waitForTimeout(150);
      if (text) {
        await page.keyboard.type(text, { delay: 15 });
      }
      await page.waitForTimeout(300);
      const val = await page.evaluate((sel) => document.querySelector(sel).value, selector);
      console.log(`${selector} = "${val}"`);
    }

    await clearAndType('#git-service-name', 'tayooli-frontend');
    await clearAndType('#git-branch', 'main');

    // Check root directory current value - should stay "/" since frontend is at repo root
    const rootVal = await page.evaluate(() => document.querySelector('#git-root-directory').value);
    console.log('git-root-directory current value:', rootVal);

    await page.waitForTimeout(500);
    const text = await page.evaluate(() => document.body.innerText);
    console.log('Final form state:\n', text.substring(0, 1800));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
