const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    
    // 1. Verify GitHub visibility
    const ghPage = browser.contexts()[0].pages().find(p => p.url().includes('github.com/herman84-bot/tayooli-core'));
    if (ghPage) {
      await ghPage.reload({ waitUntil: 'domcontentloaded' });
      const isPublic = await ghPage.evaluate(() => document.body.innerText.includes('Public'));
      console.log('GitHub repo is now Public:', isPublic);
    }

    // 2. Navigate Zeabur tab to backend service overview
    const zeaburPage = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));
    if (!zeaburPage) {
      console.log('Zeabur page not found');
      return;
    }

    console.log('Navigating Zeabur to backend overview...');
    await zeaburPage.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac268f493154e21e628ed15?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await zeaburPage.waitForTimeout(3000);

    // Click Redeploy
    const redeployBtn = zeaburPage.locator('button:has-text("Redeploy")').first();
    if (await redeployBtn.isVisible()) {
      await redeployBtn.click();
      console.log('🚀 Clicked Redeploy for tayooli-backend!');
    } else {
      console.log('Redeploy button not visible');
    }

    await zeaburPage.waitForTimeout(4000);
    const statusText = await zeaburPage.evaluate(() => document.body.innerText);
    console.log('Status snippet:\n', statusText.substring(0, 800));

  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
