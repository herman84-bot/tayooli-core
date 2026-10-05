const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
  const context = browser.contexts()[0];
  const pages = context.pages();

  console.log('=== OPEN PAGES IN BROWSER ===');
  for (let i = 0; i < pages.length; i++) {
    const p = pages[i];
    console.log(`[${i}] ${await p.title()} - ${p.url()}`);
  }

  // Find Zeabur frontend page
  const zeaburAppPage = pages.find(p => p.url().includes('tayooli-frontend.zeabur.app'));
  if (zeaburAppPage) {
    console.log('\n--- ZEABUR FRONTEND CURRENT STATE ---');
    console.log('URL:', zeaburAppPage.url());
    await zeaburAppPage.reload({ waitUntil: 'networkidle' }).catch(() => {});
    await zeaburAppPage.waitForTimeout(2000);
    const navText = await zeaburAppPage.evaluate(() => {
      const aside = document.querySelector('aside');
      return aside ? aside.innerText : 'NO ASIDE';
    });
    console.log('Sidebar on Zeabur Frontend:\n', navText);
  } else {
    console.log('No tab found for tayooli-frontend.zeabur.app');
  }

  // Find GCP page
  const gcpPage = pages.find(p => p.url().includes('tayooli.my.id') || p.url().includes('104.197.178.237'));
  if (gcpPage) {
    console.log('\n--- GCP CURRENT STATE ---');
    console.log('URL:', gcpPage.url());
    const gcpNavText = await gcpPage.evaluate(() => {
      const aside = document.querySelector('aside');
      return aside ? aside.innerText : 'NO ASIDE';
    });
    console.log('Sidebar on GCP:\n', gcpNavText);
  }

  // Find Zeabur Console page
  const zeaburConsole = pages.find(p => p.url().includes('zeabur.com'));
  if (zeaburConsole) {
    console.log('\n--- ZEABUR CONSOLE STATE ---');
    const statusText = await zeaburConsole.evaluate(() => {
      return document.body.innerText.substring(0, 1500);
    });
    console.log('Zeabur console snippet:\n', statusText);
  }

  await browser.close();
})().catch(e => { console.error('ERROR:', e); process.exit(1); });
