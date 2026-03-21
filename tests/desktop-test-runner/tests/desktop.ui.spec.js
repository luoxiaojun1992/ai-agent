const path = require('path');
const { test, expect, _electron: electron } = require('@playwright/test');

test('desktop client can send blocking chat message', async ({}, testInfo) => {
  const desktopClientPath = testInfo.project.use.desktopClientPath;
  const electronExecutablePath = path.join(desktopClientPath, 'node_modules', 'electron', 'dist', 'electron');
  const app = await electron.launch({
    executablePath: electronExecutablePath,
    args: [desktopClientPath],
    env: {
      ...process.env,
      DISPLAY: process.env.DISPLAY || ':99',
      ELECTRON_DISABLE_SANDBOX: '1',
    },
  });
  try {
    const page = await app.firstWindow();

    await page.evaluate(() => {
      if (!window.marked) {
        window.marked = { parse: (value) => value };
      }
    });

    await expect(page.locator('.sidebar-header h1')).toContainText('AI Agent Desktop');
    await expect(page.locator('#messageInput')).toBeVisible();

    const message = 'playwright desktop ui test';
    await page.locator('#messageInput').fill(message);
    await page.locator('#sendBtn').click();

    await expect(page.locator('.message.user .message-content').last()).toContainText(message);
    await expect(page.locator('.message.agent .message-content').last()).toContainText(`mock response: ${message}`);

    await page.screenshot({ path: 'artifacts/desktop-ui.png', fullPage: true });
  } finally {
    await app.close();
  }
});
