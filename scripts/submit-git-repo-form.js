const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const inputs = await page.locator('input[type="text"]').all();
    console.log('Count of text inputs:', inputs.length);
    for (let i = 0; i < inputs.length; i++) {
      const ph = await inputs[i].getAttribute('placeholder');
      console.log(`[${i}] placeholder="${ph}"`);
    }

    // Fill by placeholder match directly (more robust than index)
    const gitUrlInput = page.locator('input[placeholder*="gitlab.com/user/repo"]').first();
    await gitUrlInput.fill('https://github.com/herman84-bot/tayooli-core.git');
    await page.waitForTimeout(300);

    const serviceNameInput = page.locator('input[placeholder="git-service"]').first();
    await serviceNameInput.fill('tayooli-frontend');
    await page.waitForTimeout(300);

    const branchInput = page.locator('input[placeholder*="main (optional)"]').first();
    await branchInput.fill('main');
    await page.waitForTimeout(500);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Form filled:\n', text.substring(0, 2000));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
