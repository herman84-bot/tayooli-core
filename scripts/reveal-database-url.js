const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    let page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    if (!page) {
      page = await browser.contexts()[0].newPage();
    }

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2686e93154e21e628ecd1?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    const varTab = page.locator('button[data-tab-value="variables"], button:has-text("Variable")').first();
    await varTab.click();
    await page.waitForTimeout(2000);

    const revealed = await page.evaluate(() => {
      const maskedEls = Array.from(document.querySelectorAll('*')).filter(el => el.textContent.trim() === '******' && el.children.length === 0);
      maskedEls.forEach(el => el.click());
      return maskedEls.length;
    });
    console.log('Clicked', revealed, 'masked elements');
    await page.waitForTimeout(2000);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('Variables page after reveal attempt:\n', text.substring(0, 2000));
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/pg-vars-revealed.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
