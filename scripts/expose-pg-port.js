const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2686e93154e21e628ecd1?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const netTab = page.locator('button:has-text("Networking")').first();
    await netTab.click();
    await page.waitForTimeout(2000);

    // Find the Port Forwarding section input for Container Port
    const inputs = await page.locator('input').all();
    for (let i = 0; i < inputs.length; i++) {
      const placeholder = await inputs[i].getAttribute('placeholder');
      const type = await inputs[i].getAttribute('type');
      console.log(`Input ${i}: placeholder="${placeholder}" type="${type}"`);
    }
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
