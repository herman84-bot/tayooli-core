const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const context = browser.contexts()[0];
    const pages = context.pages();

    for (const url of ['tayooli-frontend.zeabur.app', '104.197.178.237']) {
      const page = pages.find(p => p.url().includes(url));
      if (!page) {
        console.log(`No open page found for ${url}`);
        continue;
      }
      console.log(`\n=== ${url} === current URL: ${page.url()}`);
      const text = await page.evaluate(() => document.body.innerText.substring(0, 2500));
      console.log(text);
    }

    console.log('\nAll open page URLs:');
    for (const p of pages) console.log(' -', p.url());
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
