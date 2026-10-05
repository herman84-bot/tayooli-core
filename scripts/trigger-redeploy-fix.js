const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Go to service overview
    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac268f493154e21e628ed15?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Click Redeploy
    const clicked = await page.evaluate(() => {
      const elements = Array.from(document.querySelectorAll('button, a, div, span')).filter(
        el => el.innerText && el.innerText.trim() === 'Redeploy'
      );
      if (elements.length > 0) {
        elements[0].click();
        return true;
      }
      return false;
    });
    console.log('🚀 Triggered Redeploy for commit b98cba5:', clicked);

    await page.waitForTimeout(4000);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
