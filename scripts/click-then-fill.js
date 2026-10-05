const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Click Anonymous radio first (may be required to enable other fields)
    const anonBtn = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button,div,span,label')).filter(
        el => el.innerText && el.innerText.trim() === 'Anonymous' && el.offsetParent !== null
      );
      if (els.length > 0) { els[0].click(); return true; }
      return false;
    });
    console.log('Clicked Anonymous:', anonBtn);
    await page.waitForTimeout(1000);

    const gitUrlInput = page.locator('#git-url');
    const isEnabled = await gitUrlInput.isEnabled();
    console.log('git-url enabled now:', isEnabled);

    if (isEnabled) {
      await gitUrlInput.click();
      await gitUrlInput.fill('https://github.com/herman84-bot/tayooli-core.git');
      console.log('Filled git-url');
    } else {
      // Try JS-level value set + input event dispatch
      await page.evaluate(() => {
        const el = document.querySelector('#git-url');
        const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
        nativeInputValueSetter.call(el, 'https://github.com/herman84-bot/tayooli-core.git');
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
      console.log('Set git-url via JS');
    }
    await page.waitForTimeout(1000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After attempt:\n', text.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
