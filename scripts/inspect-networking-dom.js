const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const html = await page.evaluate(() => {
      const el = Array.from(document.querySelectorAll('*')).find(e => e.innerText && e.innerText.includes('Port Forwarding'));
      let parent = el;
      while (parent && parent.parentElement && !parent.parentElement.innerText.includes('Private')) {
        parent = parent.parentElement;
      }
      return parent ? parent.outerHTML.substring(0, 5000) : 'not found';
    });
    console.log(html);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
