const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  timeout: 30000,
  expect: {
    timeout: 10000,
  },
  reporter: [
    ['line'],
    ['allure-playwright', { outputFolder: 'allure-results' }],
  ],
  outputDir: 'test-results',
  use: {
    baseURL: process.env.UI_BASE_URL || 'http://frontend',
    video: 'on',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
});
