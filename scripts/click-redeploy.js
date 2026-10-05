const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Find Redeploy element
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

    console.log('Clicked Redeploy via evaluate:', clicked);
    await page.waitForTimeout(4000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Text after clicking:\n', text.substring(0, 700));

    // If a modal or confirmation popped up, check it
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/redeploy-clicked.png' });

  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
