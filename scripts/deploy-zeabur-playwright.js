const { chromium } = require('@playwright/test');
const path = require('path');
const fs = require('fs');

async function run() {
  console.log('🚀 Membuka Microsoft Edge via Playwright...');
  
  const browser = await chromium.launch({
    channel: 'msedge',
    headless: false,
    args: ['--start-maximized']
  });

  const context = await browser.newContext({ viewport: null });
  const page = await context.newPage();

  console.log('🌐 Membuka https://dash.zeabur.com ...');
  await page.goto('https://dash.zeabur.com', { waitUntil: 'domcontentloaded' });

  console.log('⏳ Memeriksa status login...');
  
  // Tunggu sampai user masuk ke halaman project dashboard
  let loggedIn = false;
  const maxWaitMs = 300000; // 5 menit untuk login jika belum login
  const startTime = Date.now();

  while (Date.now() - startTime < maxWaitMs) {
    const currentUrl = page.url();
    // Jika URL sudah di /projects atau ada elemen dashboard/project card
    if (currentUrl.includes('/projects') || currentUrl.includes('/project/')) {
      loggedIn = true;
      break;
    }

    // Cek apakah ada avatar atau tombol profile
    const hasDashboard = await page.$('text=Projects').catch(() => null);
    if (hasDashboard) {
      loggedIn = true;
      break;
    }

    await page.waitForTimeout(2000);
  }

  if (!loggedIn) {
    console.log('⚠️ Waktu tunggu login habis.');
    await browser.close();
    return;
  }

  console.log('✅ Berhasil terdeteksi di Dashboard Zeabur:', page.url());
  await page.waitForTimeout(3000);

  // Ambil screenshot halaman dashboard terkini
  const screenshotPath = path.join(__dirname, 'zeabur-dashboard.png');
  await page.screenshot({ path: screenshotPath, fullPage: false });
  console.log(`📸 Screenshot tersimpan di: ${screenshotPath}`);

  // Simpan halaman HTML untuk analisis selector
  const htmlPath = path.join(__dirname, 'zeabur-dashboard.html');
  fs.writeFileSync(htmlPath, await page.content());
  console.log(`📄 Snapshot DOM tersimpan di: ${htmlPath}`);

  console.log('🎉 Browser tetap terbuka. Silakan pilih project Anda jika belum dipilih.');
}

run().catch(err => {
  console.error('❌ Terjadi kesalahan:', err);
});
