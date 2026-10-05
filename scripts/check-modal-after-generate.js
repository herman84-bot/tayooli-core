const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    // Check for any dialog/modal elements
    const modals = await page.evaluate(() => {
      const dialogs = Array.from(document.querySelectorAll('[role="dialog"], [class*="modal"], [class*="Dialog"]'));
      return dialogs.map(d => ({ visible: d.offsetParent !== null, text: d.innerText.substring(0, 300) }));
    });
    console.log('Modals found:', JSON.stringify(modals, null, 2));

    // Also list all inputs now visible
    const inputs = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('input')).filter(i => i.offsetParent !== null).map(i => ({ placeholder: i.placeholder, value: i.value }));
    });
    console.log('Visible inputs:', JSON.stringify(inputs, null, 2));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
