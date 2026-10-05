const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Look for all elements containing "Container Port" text
    const html = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('*')).filter(e => e.textContent.includes('Container Port') && e.children.length < 10);
      if (els.length === 0) return 'not found';
      // smallest matching element
      els.sort((a,b) => a.innerHTML.length - b.innerHTML.length);
      return els[0].outerHTML;
    });
    console.log(html);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
