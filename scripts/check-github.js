const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('github.com/herman84-bot/tayooli-core'));
    if (!page) {
      console.log('GitHub tab not found');
      return;
    }
    console.log('Found GitHub page:', page.url());
    const title = await page.title();
    console.log('Title:', title);
    
    // Check if repo has "Public" or "Private" badge
    const badge = await page.evaluate(() => {
      const b = document.querySelector('#repo-title-component, .Label, span[class*="Label"]');
      return document.body.innerText.includes('Private') ? 'Private' : 'Public';
    });
    console.log('Repo visibility detected:', badge);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
