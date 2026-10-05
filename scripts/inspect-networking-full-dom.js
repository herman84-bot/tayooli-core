const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const html = await page.evaluate(() => {
      // Find the main content area (usually a specific container, not html/head)
      const main = document.querySelector('main') || document.querySelector('[class*="content"]');
      return main ? main.innerHTML.substring(0, 6000) : 'main not found';
    });
    console.log(html);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
