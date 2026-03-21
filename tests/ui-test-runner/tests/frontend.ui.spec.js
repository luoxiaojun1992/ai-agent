const { test, expect } = require('@playwright/test');

test('frontend can send blocking chat message', async ({ page }) => {
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

  await page.screenshot({ path: 'artifacts/frontend-ui.png', fullPage: true });
});
