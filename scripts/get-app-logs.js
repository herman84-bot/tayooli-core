const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Get all text in log table/rows
    const logRows = await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll('tr, [role="row"], div[class*="log-row"], div[class*="item"]')).map(r => r.innerText);
      return rows.filter(r => r.length > 5 && !r.includes('[Zeabur] Pod/'));
    });
    console.log('App log lines:\n', logRows.slice(-15).join('\n---\n'));

    // Also check if there's any search filter or severity dropdown
    const allText = await page.evaluate(() => document.body.innerText);
    const nonK8sLines = allText.split('\n').filter(l => !l.includes('[Zeabur]') && !l.includes('Claude Opus') && l.trim().length > 3);
    console.log('Non-K8s lines:\n', nonK8sLines.slice(-25).join('\n'));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
