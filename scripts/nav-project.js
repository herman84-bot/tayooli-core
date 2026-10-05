const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    if (!page) {
      console.log('Project page not found');
      return;
    }
    console.log('Navigating to main project dashboard...');
    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Project dashboard text:\n', text.substring(0, 1000));
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/project-dashboard.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
