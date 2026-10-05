const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
  const context = browser.contexts()[0];
  const pages = context.pages();
  const zeaburAppPage = pages.find(p => p.url().includes('tayooli-frontend.zeabur.app'));
  if (zeaburAppPage) {
    console.log('Current URL:', zeaburAppPage.url());
    console.log('Page title:', await zeaburAppPage.title());
    const bodySnippet = await zeaburAppPage.evaluate(() => document.body.innerText.substring(0, 1000));
    console.log('Body snippet:\n', bodySnippet);
  }
  await browser.close();
})().catch(e => { console.error('ERROR:', e); process.exit(1); });
