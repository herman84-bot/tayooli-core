const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    
    const logs = await page.evaluate(() => {
      // Find the element with text "SEVERITY"
      const el = Array.from(document.querySelectorAll('*')).find(e => e.innerText && e.innerText.startsWith('SEVERITY'));
      if (!el) return 'SEVERITY element not found';
      return el.innerText;
    });
    console.log('Logs text block:\n', logs);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
