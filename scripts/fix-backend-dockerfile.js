const { chromium } = require('@playwright/test');

(async () => {
  try {
    const browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    await page.locator('button:has-text("Settings")').first().click();
    await page.waitForTimeout(2000);

    // Find Dockerfile section
    const dockerfileSection = await page.evaluate(() => {
      const textarea = document.querySelector('textarea');
      return {
        value: textarea ? textarea.value : null,
        placeholder: textarea ? textarea.placeholder : null
      };
    });
    console.log('Dockerfile section:', dockerfileSection);

    // Let's read the backend Dockerfile
    const fs = require('fs');
    const backendDockerfile = fs.readFileSync('C:/Users/rehan/Desktop/project/tayooli-core/backend/go-core/Dockerfile', 'utf8');
    console.log('Backend Dockerfile content:\n', backendDockerfile);

    // Check if we can fill the textarea or paste Dockerfile
    const textarea = page.locator('textarea').first();
    await textarea.fill(backendDockerfile);
    console.log('Filled Dockerfile in Zeabur settings!');

    // Click Save button near Dockerfile
    const saveBtn = page.locator('button:has-text("Save")').first();
    await saveBtn.click();
    console.log('Clicked Save!');
    await page.waitForTimeout(3000);

    // Let's go to Overview and click Redeploy!
    await page.locator('button:has-text("Overview")').first().click();
    await page.waitForTimeout(2000);
    const redeployBtn = page.locator('button:has-text("Redeploy")').first();
    if (await redeployBtn.isVisible()) {
      await redeployBtn.click();
      console.log('Clicked Redeploy!');
    }

    await page.waitForTimeout(3000);
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/zeabur-after-redeploy.png' });
    await browser.close();
  } catch (err) {
    console.error('Error:', err);
  }
})();
