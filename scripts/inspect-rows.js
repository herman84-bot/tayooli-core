const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    
    // Find all rows in the log table
    const rows = await page.locator('[role="row"], table tbody tr, div[class*="table"] > div').all();
    console.log('Total rows found:', rows.length);
    for (let i = Math.max(0, rows.length - 8); i < rows.length; i++) {
      console.log(`Row ${i}:`, (await rows[i].innerText()).replace(/\n/g, ' | '));
    }
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
