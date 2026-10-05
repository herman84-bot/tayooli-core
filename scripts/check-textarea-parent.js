const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    
    const contextInfo = await page.evaluate(() => {
      const ta = document.querySelector('textarea');
      if (!ta) return 'no textarea';
      let parent = ta.parentElement;
      while (parent && !parent.innerText.includes('Dockerfile') && parent !== document.body) {
        parent = parent.parentElement;
      }
      return parent ? parent.innerText : 'no parent with Dockerfile text';
    });
    console.log('Parent container text:\n', contextInfo);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
