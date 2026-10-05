const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac2da1493154e21e6291302?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    await page.locator('[data-tab-value="variable"], [data-tab-value="variables"], button:has-text("Variable")').first().click();
    await page.waitForTimeout(2000);

    const editRawBtn = page.locator('button:has-text("Edit Raw Variables")').first();
    await editRawBtn.click();
    await page.waitForTimeout(1500);

    const ta = page.locator('textarea').first();
    const rawVars = [
      'BACKEND_URL=https://tayooli-backend.zeabur.app',
      'NEXT_PUBLIC_API_URL=https://tayooli-backend.zeabur.app',
      'NODE_ENV=production',
      'AUTH_DEMO=false',
    ].join('\n');

    await ta.click();
    await ta.fill(rawVars);
    await page.waitForTimeout(500);

    const val = await ta.inputValue();
    console.log('Textarea value:\n', val);

    const saveBtn = page.locator('button:has-text("Save")').first();
    await saveBtn.click();
    console.log('Clicked Save for env vars');
    await page.waitForTimeout(2500);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After save:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
