const { test, expect } = require('@playwright/test');

test('frontend can send blocking chat message', async ({ page }) => {
  // Frontend loads marked.js from CDN; provide a fallback in CI/network-restricted environments.
  await page.addInitScript(() => {
    if (!window.marked) {
      window.marked = { parse: (value) => value };
    }
  });

  await page.goto('/');

  await expect(page.getByRole('heading', { name: 'AI Agent UI' })).toBeVisible();
  await expect(page.locator('#messageInput')).toBeVisible();

  const message = 'playwright ui test';
  await page.locator('#messageInput').fill(message);
  await page.locator('#sendButton').click();

  await expect(page.locator('.message.user .message-content').last()).toContainText(message);
  await expect(page.locator('.message.agent .message-content').last()).toContainText(`mock response: ${message}`);

  await page.screenshot({ path: 'artifacts/frontend-ui.png', fullPage: true });
});
