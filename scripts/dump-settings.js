const { chromium } = require('@playwright/test');

(async () => {
  try {
    const browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    await page.locator('button:has-text("Settings")').first().click();
    await page.waitForTimeout(2000);

    const headings = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('h1, h2, h3, h4, h5, label')).map(el => ({
        tag: el.tagName,
        text: el.innerText.trim()
      })).filter(x => x.text);
    });
    console.log('Headings/Labels:', JSON.stringify(headings, null, 2));
    await browser.close();
  } catch (err) {
    console.error('Error:', err);
  }
})();
