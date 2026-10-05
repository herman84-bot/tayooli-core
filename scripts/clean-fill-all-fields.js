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
        el.select();
      }, selector);
      await page.keyboard.press('Control+A');
      await page.keyboard.press('Backspace');
      await page.waitForTimeout(200);
      await page.keyboard.type(text, { delay: 15 });
      await page.waitForTimeout(300);
      const val = await page.evaluate((sel) => document.querySelector(sel).value, selector);
      console.log(`${selector} = "${val}"`);
    }

    await clearAndType('#git-url', 'https://github.com/herman84-bot/tayooli-core.git');

    // Find Service Name, Branch, Root Directory input ids/selectors
    const allInputs = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('input[type="text"], input:not([type])')).map(el => ({
        id: el.id,
        placeholder: el.placeholder,
      }));
    });
    console.log('All text inputs:', JSON.stringify(allInputs));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
