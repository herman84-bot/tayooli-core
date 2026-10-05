const { chromium } = require('@playwright/test');

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac268f493154e21e628ed15?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const restartBtn = page.locator('button:has-text("Restart")').first();
    await restartBtn.click();
    console.log('Clicked Restart');
    await page.waitForTimeout(2000);

    // Go to Logs -> Runtime logs immediately
    const logsBtn = page.locator('button:has-text("Logs")').first();
    await logsBtn.click();
    await page.waitForTimeout(2000);

    // Poll for up to 60s capturing non-K8s lines
    const start = Date.now();
    let lastText = '';
    while (Date.now() - start < 60000) {
      await page.waitForTimeout(4000);
      const allText = await page.evaluate(() => document.body.innerText);
      const nonK8sLines = allText.split('\n').filter(l => !l.includes('[Zeabur]') && !l.includes('Claude Opus') && l.trim().length > 3 && !l.includes('Agent'));
      const joined = nonK8sLines.join('\n');
      if (joined !== lastText) {
        console.log(`--- t=${Math.round((Date.now()-start)/1000)}s ---`);
        console.log(joined.slice(-1000));
        lastText = joined;
      }
      if (allText.includes('FTL') || allText.includes('server listening') || allText.includes('migrations completed')) {
        console.log('STOP CONDITION REACHED');
        break;
      }
    }
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
