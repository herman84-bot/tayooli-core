const { chromium } = require('@playwright/test');

const DOCKERFILE_CONTENT = `# syntax=docker/dockerfile:1
# Tayooli ERP frontend — Next.js 15 standalone build.
FROM node:22-alpine AS base
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1

FROM base AS deps
COPY package.json package-lock.json ./
RUN npm ci

FROM base AS builder
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ARG BACKEND_URL
ARG NEXT_PUBLIC_API_URL
ENV BACKEND_URL=$BACKEND_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
RUN npm run build

FROM base AS runner
ENV NODE_ENV=production \\
    PORT=3000 \\
    HOSTNAME=0.0.0.0
RUN addgroup --system --gid 1001 nodejs \\
    && adduser --system --uid 1001 nextjs
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
CMD ["node", "server.js"]
`;

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9100');
    const page = browser.contexts()[0].pages().find(p => p.url().includes('zeabur.com'));

    const dockerfileTextarea = page.locator('textarea[placeholder^="FROM node:18-alpine"]').first();
    await dockerfileTextarea.click();
    await dockerfileTextarea.fill(DOCKERFILE_CONTENT);
    await page.waitForTimeout(500);

    const val = await dockerfileTextarea.inputValue();
    console.log('Dockerfile textarea value length:', val.length);
    console.log('First 100 chars:', val.substring(0, 100));

    const saveBtn = page.locator('button:has-text("Save")').first();
    await saveBtn.click();
    console.log('Clicked Save');
    await page.waitForTimeout(2500);

    const text = await page.evaluate(() => document.body.innerText);
    console.log('After save:\n', text.substring(0, 1500));
  } catch (err) {
    console.error('Error:', err);
  } finally {
    if (browser) await browser.close();
  }
})();
