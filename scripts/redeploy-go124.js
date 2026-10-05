const { chromium } = require('@playwright/test');

const standardBackendDockerfile = `# syntax=docker/dockerfile:1

# ---- Builder stage: compile a static binary ----
FROM golang:1.24-alpine AS builder
WORKDIR /src

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy full source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 go build \\
    -trimpath \\
    -ldflags="-s -w" \\
    -o /out/api ./cmd/api

# ---- Runtime stage ----
FROM alpine:3.19
RUN apk add --no-cache ca-certificates \\
    && addgroup -S -g 10001 app \\
    && adduser -S -D -H -u 10001 -G app app

WORKDIR /app

COPY --from=builder /out/api /app/api

USER 10001:10001
EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \\
    CMD wget -q -O /dev/null http://127.0.0.1:8081/health || exit 1

ENTRYPOINT ["/app/api"]
`;

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac2683a93154e21e628ecbc'));
    
    // Go to service page
    await page.goto('https://zeabur.com/projects/6ac2683a93154e21e628ecbc/services/6ac268f493154e21e628ed15?envID=6ac2683a6a873116ad572b40', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Click Settings
    const settingsTab = page.locator('button[data-tab-value="settings"], button:has-text("Settings")').last();
    await settingsTab.click();
    await page.waitForTimeout(2000);

    // Fill textarea
    const ta = page.locator('textarea').first();
    await ta.fill(standardBackendDockerfile);
    console.log('Updated Dockerfile to golang:1.24-alpine');

    // Click Save
    const saveBtn = page.locator('button:has-text("Save")').first();
    await saveBtn.click();
    console.log('Saved Dockerfile');
    await page.waitForTimeout(3000);

    // Go to Overview
    const overviewTab = page.locator('button[data-tab-value="overview"], button:has-text("Overview")').first();
    await overviewTab.click();
    await page.waitForTimeout(2000);

    // Click Redeploy
    const clicked = await page.evaluate(() => {
      const elements = Array.from(document.querySelectorAll('button, a, div, span')).filter(
        el => el.innerText && el.innerText.trim() === 'Redeploy'
      );
      if (elements.length > 0) {
        elements[0].click();
        return true;
      }
      return false;
    });
    console.log('Redeploy triggered:', clicked);

    await page.waitForTimeout(5000);
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
