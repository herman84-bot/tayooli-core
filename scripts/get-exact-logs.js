const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Ensure we are on Runtime logs
    const runtimeBtn = page.locator('button:has-text("Runtime logs")').first();
    if (await runtimeBtn.isVisible()) {
      await runtimeBtn.click();
      await page.waitForTimeout(2000);
    }

    const logLines = await page.evaluate(() => {
      // Find all log table cells or rows
      const items = Array.from(document.querySelectorAll('[data-index], tr, div[class*="log"]')).map(el => el.innerText);
      return items.filter(t => t.includes('INF') || t.includes('FTL') || t.includes('WRN') || t.includes('error') || t.includes('migration'));
    });

    console.log('Filtered log lines:\n', [...new Set(logLines)].join('\n---\n'));
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/detailed-log.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
