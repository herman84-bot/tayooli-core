const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Type FTL into Search Logs
    const searchInput = page.locator('input[placeholder*="Search"]').first();
    await searchInput.fill('FTL');
    await page.waitForTimeout(2000);

    const text = await page.evaluate(() => document.body.innerText);
    const lines = text.split('\n').filter(l => l.includes('FTL') || l.includes('migration'));
    console.log('FTL search results:\n', lines.join('\n'));

    // Clear search
    await searchInput.fill('');
    await page.waitForTimeout(1000);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
