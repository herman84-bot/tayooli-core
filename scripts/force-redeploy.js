const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac268f493154e21e628ed15?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    const text1 = await page.evaluate(() => document.body.innerText);
    console.log('Before click:\n', text1.substring(0, 500));

    const clicked = await page.evaluate(() => {
      const elements = Array.from(document.querySelectorAll('button, a, div, span')).filter(
        el => el.innerText && el.innerText.trim() === 'Redeploy' && el.offsetParent !== null
      );
      console.log('Found', elements.length, 'Redeploy elements');
      if (elements.length > 0) {
        elements[0].click();
        return true;
      }
      return false;
    });
    console.log('Clicked:', clicked);
    await page.waitForTimeout(3000);

    // Check if a confirm dialog appeared
    const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes"), button:has-text("Deploy")').first();
    if (await confirmBtn.isVisible().catch(() => false)) {
      await confirmBtn.click();
      console.log('Clicked confirm dialog');
    }

    await page.waitForTimeout(3000);
    const text2 = await page.evaluate(() => document.body.innerText);
    console.log('After click:\n', text2.substring(0, 500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
