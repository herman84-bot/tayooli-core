const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    
    // Click Settings tab
    const settingsTab = page.locator('button[data-tab-value="settings"], button:has-text("Settings")').last();
    await settingsTab.click();
    await page.waitForTimeout(2000);

    const textareaInfo = await page.evaluate(() => {
      const ta = document.querySelector('textarea');
      return {
        exists: !!ta,
        value: ta ? ta.value : '',
        placeholder: ta ? ta.placeholder : ''
      };
    });
    console.log('Textarea info:\n', textareaInfo);
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/backend-settings-view.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
