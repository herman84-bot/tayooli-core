const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const contexts = browser.contexts();
    const pages = contexts[0].pages();
    const page = pages.find(p => p.url().includes('zeabur.com/projects/6ac2683a93154e21e628ecbc'));
    if (!page) {
      console.log('Project page not found among open pages:');
      pages.forEach(p => console.log(' - ', p.url()));
      return;
    }
    console.log('Target page URL:', page.url());
    
    // Capture screenshot
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/current-project-view.png' });
    console.log('Screenshot saved to current-project-view.png');

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Current page text preview:\n', text.substring(0, 1000));
  } catch (err) {
    console.error('Error in script:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
