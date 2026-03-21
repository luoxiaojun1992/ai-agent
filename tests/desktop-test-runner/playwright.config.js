const path = require('path');
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  timeout: 120000,
  expect: {
    timeout: 10000,
  },
  reporter: [
    ['line'],
    ['allure-playwright', { outputFolder: 'allure-results' }],
  ],
  outputDir: 'test-results',
  use: {
    screenshot: 'only-on-failure',
    video: 'on',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'electron',
      use: {
        desktopClientPath: path.join(__dirname, 'desktop-client'),
      },
    },
  ],
});
