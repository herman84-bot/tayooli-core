const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const result = await page.evaluate(() => {
      const inputs = Array.from(document.querySelectorAll('input[type="text"]')).filter(i => i.offsetParent !== null);
      if (inputs.length === 0) return 'no inputs';
      const el = inputs[0];
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
      nativeInputValueSetter.call(el, 'tayooli-frontend');
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
      return el.value;
    });
    console.log('Set domain input via JS, value now:', result);
    await page.waitForTimeout(500);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Current dialog text:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
