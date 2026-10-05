const { chromium } = require('@playwright/test');

const backendDockerfile = `# syntax=docker/dockerfile:1

# ---- Builder stage: compile a static binary ----
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Copy backend go.mod and go.sum from repo root
COPY backend/go-core/go.mod backend/go-core/go.sum ./
RUN go mod download

# Copy full backend source code including migrations
COPY backend/go-core ./

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
    const page = browser.contexts()[0].pages().find(p => p.url().includes('6ac268f493154e21e628ed15'));
    
    // Ensure on Settings tab
    const settingsTab = page.locator('button[data-tab-value="settings"], button:has-text("Settings")').last();
    await settingsTab.click();
    await page.waitForTimeout(2000);

    // Fill textarea
    const ta = page.locator('textarea').first();
    await ta.fill(backendDockerfile);
    console.log('Filled Dockerfile in textarea');

    // Click Save button near textarea
    // Find the Save button that is inside or following the Dockerfile section
    const saveBtn = page.locator('button:has-text("Save")').first();
    await saveBtn.click();
    console.log('Clicked Save button');
    await page.waitForTimeout(3000);

    // Navigate to Overview tab
    const overviewTab = page.locator('button[data-tab-value="overview"], button:has-text("Overview")').first();
    await overviewTab.click();
    await page.waitForTimeout(2000);

    // Click Redeploy
    const redeployBtn = page.locator('button:has-text("Redeploy")').first();
    if (await redeployBtn.isVisible()) {
      await redeployBtn.click();
      console.log('🚀 Clicked Redeploy!');
    } else {
      console.log('Redeploy button not visible');
    }

    await page.waitForTimeout(5000);
    await page.screenshot({ path: 'C:/Users/rehan/Desktop/project/tayooli-core/scripts/after-redeploy-triggered.png' });
    console.log('Status preview:\n', (await page.evaluate(() => document.body.innerText)).substring(0, 800));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
