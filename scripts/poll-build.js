const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Check logs button
    const logsBtn = page.locator('button:has-text("Logs")').first();
    if (await logsBtn.isVisible()) {
      await logsBtn.click();
      console.log('Opened logs view');
    }
    
    // Poll logs for up to 90 seconds
    const start = Date.now();
    while (Date.now() - start < 90000) {
      await page.waitForTimeout(5000);
      const logText = await page.evaluate(() => {
        const pre = document.querySelector('pre, code, [class*="log"], [class*="terminal"]');
        return document.body.innerText;
      });
      
      const lines = logText.split('\n').filter(l => l.includes('STEP') || l.includes('INFO') || l.includes('ERROR') || l.includes('Running') || l.includes('Building') || l.includes('Success'));
      console.log('--- PROGRESS (' + Math.round((Date.now() - start) / 1000) + 's) ---');
      lines.slice(-6).forEach(l => console.log('  ', l));

      if (logText.includes('database migrations completed') || logText.includes('server listening') || logText.includes('Running')) {
        console.log('🎉 SUCCESS: Service is running!');
        break;
      }
      if (logText.includes('Build failed') || logText.includes('ERROR 🔴')) {
        console.log('⚠️ Error detected in logs');
        break;
      }
    }

    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/build-progress.png' });
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
