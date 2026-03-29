const path = require('path');
const { test, expect, _electron: electron } = require('@playwright/test');

async function launchDesktopApp(testInfo) {
  const desktopClientPath = testInfo.project.use.desktopClientPath;
  const electronExecutablePath = path.join(desktopClientPath, 'node_modules', 'electron', 'dist', 'electron');
  return electron.launch({
    executablePath: electronExecutablePath,
    args: [desktopClientPath],
    env: {
      ...process.env,
      DISPLAY: process.env.DISPLAY || ':99',
      ELECTRON_DISABLE_SANDBOX: '1',
    },
  });
}

test('desktop client can send blocking chat message', async ({}, testInfo) => {
  const app = await launchDesktopApp(testInfo);
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

test('desktop client supports scheduled task CRUD and enable toggle', async ({}, testInfo) => {
  const app = await launchDesktopApp(testInfo);
  const taskName = `scheduled-task-${Date.now()}`;
  const taskMessage = `scheduled task message ${Date.now()}`;

  try {
    const page = await app.firstWindow();

    await page.evaluate(() => {
      if (!window.marked) {
        window.marked = { parse: (value) => value };
      }
    });

    await page.locator('.nav-item', { hasText: 'Scheduled Tasks' }).click();
    await expect(page.locator('#scheduledTasksPanel')).toHaveClass(/active/);

    await page.locator('#taskName').fill(taskName);
    await page.locator('#taskMessage').fill(taskMessage);
    await page.locator('#taskScheduleType').selectOption('interval');
    await page.locator('#taskInterval').fill('60');
    await page.locator('#taskExecuteImmediately').uncheck();
    await page.getByRole('button', { name: 'Add Task' }).click();

    const taskItem = page.locator('.task-item').filter({ hasText: taskName });
    await expect(taskItem).toBeVisible();
    await expect(taskItem).toContainText(taskMessage);
    await expect(taskItem).toContainText('Active');

    await taskItem.getByRole('button', { name: 'Pause' }).click();
    await expect(taskItem).toContainText('Paused');
    await expect(taskItem.getByRole('button', { name: 'Resume' })).toBeVisible();

    await taskItem.getByRole('button', { name: 'Resume' }).click();
    await expect(taskItem).toContainText('Active');
    await expect(taskItem.getByRole('button', { name: 'Pause' })).toBeVisible();

    page.once('dialog', (dialog) => dialog.accept());
    await taskItem.getByRole('button', { name: 'Delete' }).click();
    await expect(page.locator('.task-item').filter({ hasText: taskName })).toHaveCount(0);

    await page.screenshot({ path: 'artifacts/desktop-scheduled-task-ui.png', fullPage: true });
  } finally {
    await app.close();
  }
});

test('desktop client supports image upload with text message', async ({}, testInfo) => {
  const app = await launchDesktopApp(testInfo);
  let patchedElectronApi = false;
  try {
    const page = await app.firstWindow();

    await page.evaluate(() => {
      if (!window.marked) {
        window.marked = { parse: (value) => value };
      }
      const tinyPngBase64 =
        'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO5nY6UAAAAASUVORK5CYII=';
      const originalShowOpenDialog = window.electronAPI.showOpenDialog;
      const originalReadFileAsBase64 = window.electronAPI.readFileAsBase64;
      window.__originalShowOpenDialog = originalShowOpenDialog;
      window.__originalReadFileAsBase64 = originalReadFileAsBase64;
      window.electronAPI.showOpenDialog = async () => ({ canceled: false, filePaths: ['/tmp/tiny.png'] });
      window.electronAPI.readFileAsBase64 = async () => ({ success: true, data: tinyPngBase64 });
    });
    patchedElectronApi = true;

    await page.locator('#uploadImageBtn').click();
    await expect(page.locator('#uploadHint')).toContainText('tiny.png');

    const message = 'playwright desktop image test';
    await page.locator('#messageInput').fill(message);
    await page.locator('#sendBtn').click();

    await expect(page.locator('.message.user .message-content').last()).toContainText('[1 image(s) uploaded]');
    await expect(page.locator('.message.agent .message-content').last()).toContainText(`mock response: ${message}`);

  } finally {
    if (patchedElectronApi) {
      const windows = app.windows();
      if (windows.length > 0) {
        await windows[0].evaluate(() => {
          if (window.__originalShowOpenDialog) {
            window.electronAPI.showOpenDialog = window.__originalShowOpenDialog;
          }
          if (window.__originalReadFileAsBase64) {
            window.electronAPI.readFileAsBase64 = window.__originalReadFileAsBase64;
          }
        });
      }
    }
    await app.close();
  }
});
