const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac268f493154e21e628ed15?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    await page.locator('[data-tab-value="variable"], [data-tab-value="variables"], button:has-text("Variable")').first().click();
    await page.waitForTimeout(2000);

    const editRawBtn = page.locator('button:has-text("Edit Raw Variables")').first();
    await editRawBtn.click();
    await page.waitForTimeout(1500);

    const ta = page.locator('textarea').first();
    const currentVal = await ta.inputValue();
    console.log('Current backend raw vars:\n', currentVal);

    let newVal;
    if (currentVal.includes('FRONTEND_ORIGIN=')) {
      newVal = currentVal.replace(/FRONTEND_ORIGIN=.*/g, 'FRONTEND_ORIGIN=https://tayooli-frontend.zeabur.app');
    } else {
      newVal = currentVal.trim() + '\nFRONTEND_ORIGIN=https://tayooli-frontend.zeabur.app\n';
    }

    await ta.click();
    await ta.fill(newVal);
    await page.waitForTimeout(500);

    const val = await ta.inputValue();
    console.log('New backend raw vars:\n', val);

    const saveBtn = page.locator('button:has-text("Save")').first();
    await saveBtn.click();
    console.log('Clicked Save for backend FRONTEND_ORIGIN');
    await page.waitForTimeout(2500);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After save:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
