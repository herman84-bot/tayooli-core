const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Find all "Settings"-text elements and their y-position; the service tab
    // bar one sits right after "Volumes" tab, lower in the DOM / page than
    // the sidebar's "Settings" (which is near the top, next to "Integration").
    const positions = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button,a')).filter(
        el => el.innerText && el.innerText.trim() === 'Settings' && el.offsetParent !== null
      );
      return els.map(el => {
        const rect = el.getBoundingClientRect();
        return { top: rect.top, left: rect.left, text: el.innerText };
      });
    });
    console.log('Settings elements positions:', JSON.stringify(positions));

    // Click the one with the larger `top` (lower on page = service tab bar)
    const clicked = await page.evaluate(() => {
      const els = Array.from(document.querySelectorAll('button,a')).filter(
        el => el.innerText && el.innerText.trim() === 'Settings' && el.offsetParent !== null
      );
      if (els.length === 0) return false;
      els.sort((a, b) => b.getBoundingClientRect().top - a.getBoundingClientRect().top);
      els[0].click();
      return true;
    });
    console.log('Clicked lower Settings:', clicked);
    await page.waitForTimeout(2000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Result:\n', text.substring(0, 2500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
