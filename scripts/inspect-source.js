const { chromium } = require('@playwright/test');

(async () => {
  try {
    const browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    await page.locator('button:has-text("Settings")').first().click();
    await page.waitForTimeout(2000);

    const sourceSection = await page.evaluate(() => {
      // Find elements containing 'Source'
      const headers = Array.from(document.querySelectorAll('h1, h2, h3, h4, div')).filter(el => el.innerText === 'Source');
      if (!headers.length) return 'Source header not found';
      const container = headers[0].parentElement;
      return container ? container.innerText : 'No parent';
    });
    console.log('Source section content:\n', sourceSection);
    await browser.close();
  } catch (err) {
    console.error('Error:', err);
  }
})();
