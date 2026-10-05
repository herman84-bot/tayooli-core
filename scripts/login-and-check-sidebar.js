const { chromium } = require('@playwright/test');

(async () => {
  const browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
  const context = browser.contexts()[0];
  const pages = context.pages();
  const page = pages.find(p => p.url().includes('tayooli-frontend.zeabur.app'));
  if (!page) {
    console.error('Tab not found');
    await browser.close();
    return;
  }

  console.log('Logging in on:', page.url());
  await page.fill('input[type="email"]', 'admin@test.com');
  await page.fill('input[type="password"]', 'password123');
  await page.waitForTimeout(500);

  await page.evaluate(() => {
    const btn = Array.from(document.querySelectorAll('button')).find(b => b.innerText.trim() === 'Masuk');
    if (btn) btn.click();
  });

  await page.waitForNavigation({ waitUntil: 'networkidle' }).catch(() => {});
  await page.waitForTimeout(3000);

  console.log('Current URL after login:', page.url());
  const sidebarText = await page.evaluate(() => {
    const aside = document.querySelector('aside');
    return aside ? aside.innerText : 'NO ASIDE';
  });
  console.log('=== SIDEBAR ON TAYOOLI-FRONTEND.ZEABUR.APP ===\n', sidebarText);

  await browser.close();
})().catch(e => { console.error('ERROR:', e); process.exit(1); });
