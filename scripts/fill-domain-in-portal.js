const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const result = await page.evaluate(() => {
      const portal = document.getElementById('headlessui-portal-root');
      if (!portal) return 'portal not found';
      const input = portal.querySelector('input');
      if (!input) return 'input not found in portal';
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
      nativeInputValueSetter.call(input, 'tayooli-frontend');
      input.dispatchEvent(new Event('input', { bubbles: true }));
      input.dispatchEvent(new Event('change', { bubbles: true }));
      return input.value;
    });
    console.log('Result:', result);
    await page.waitForTimeout(800);

    const text = await page.evaluate(() => {
      const portal = document.getElementById('headlessui-portal-root');
      return portal ? portal.innerText.substring(0, 1000) : 'no portal';
    });
    console.log('Portal text:\n', text);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
