// @ts-check
const { test, expect } = require('@playwright/test');

test.describe('AI Agent Chat Interface', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the AI Agent UI
    await page.goto('/');
  });

  test('should display the correct title', async ({ page }) => {
    // Expect the page title to be "AI Agent UI"
    await expect(page).toHaveTitle('AI Agent UI');
  });

  test('should display initial welcome message', async ({ page }) => {
    // Check that the initial message from the agent is displayed
    await expect(page.locator('.message.agent .message-content')).toContainText("Hello! I'm your AI Agent. How can I help you today?");
  });

  test('should send a message and receive a response in blocking mode', async ({ page }) => {
    // Type a message in the input field
    await page.locator('#messageInput').fill('Hello, AI Agent!');
    
    // Click the send button
    await page.locator('#sendButton').click();
    
    // Wait for the response
    await page.waitForSelector('.message.agent:nth-child(3)');
    
    // Verify that both user and agent messages are displayed
    const userMessages = page.locator('.message.user');
    const agentMessages = page.locator('.message.agent');
    
    await expect(userMessages).toHaveCount(1);
    await expect(agentMessages).toHaveCount(2); // Initial + response
    
    // Check that the user message is displayed correctly
    await expect(userMessages.first()).toContainText('Hello, AI Agent!');
  });

  test('should switch between blocking and streaming modes', async ({ page }) => {
    // Initially, blocking mode should be selected
    await expect(page.locator('input[name="chatMode"][value="blocking"]')).toBeChecked();
    await expect(page.locator('input[name="chatMode"][value="streaming"]')).not.toBeChecked();
    
    // Switch to streaming mode
    await page.locator('input[name="chatMode"][value="streaming"]').check();
    
    // Verify that streaming mode is now selected
    await expect(page.locator('input[name="chatMode"][value="streaming"]')).toBeChecked();
    await expect(page.locator('input[name="chatMode"][value="blocking"]')).not.toBeChecked();
    
    // Check that the streaming indicator is visible
    await expect(page.locator('.stream-indicator')).toBeVisible();
    
    // Switch back to blocking mode
    await page.locator('input[name="chatMode"][value="blocking"]').check();
    
    // Verify that blocking mode is now selected
    await expect(page.locator('input[name="chatMode"][value="blocking"]')).toBeChecked();
    await expect(page.locator('input[name="chatMode"][value="streaming"]')).not.toBeChecked();
    
    // Check that the streaming indicator is hidden
    await expect(page.locator('.stream-indicator')).not.toBeVisible();
  });

  test('should clear chat history', async ({ page }) => {
    // Send a message first
    await page.locator('#messageInput').fill('Test message');
    await page.locator('#sendButton').click();
    
    // Wait for the response
    await page.waitForSelector('.message.agent:nth-child(3)');
    
    // Verify there are messages
    const messages = page.locator('.message');
    await expect(messages).toHaveCount(3); // Initial + user + agent response
    
    // Click the clear button (specifically the one in the chat container)
    await page.locator('.chat-container button:has-text("Clear")').click();
    
    // Verify that chat is cleared and only initial message remains
    await expect(messages).toHaveCount(1);
    await expect(page.locator('.message.agent .message-content')).toContainText("Chat cleared. How can I help you?");
  });
});

test.describe('AI Agent Configuration', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the AI Agent UI
    await page.goto('/');
  });

  test('should display agent configuration', async ({ page }) => {
    // Wait for configuration to load
    await page.waitForSelector('#configDisplay p');
    
    // Check that configuration information is displayed
    const configItems = page.locator('#configDisplay p');
    await expect(configItems).toHaveCount(4); // chatModel, embeddingModel, mode, character
    
    // Check that each config item has the expected labels
    const configTexts = await configItems.allTextContents();
    expect(configTexts.some(text => text.includes('Chat Model:'))).toBeTruthy();
    expect(configTexts.some(text => text.includes('Embedding Model:'))).toBeTruthy();
    expect(configTexts.some(text => text.includes('Mode:'))).toBeTruthy();
    expect(configTexts.some(text => text.includes('Character:'))).toBeTruthy();
  });

  test('should refresh agent configuration', async ({ page }) => {
    // Get initial config content
    const initialConfig = await page.locator('#configDisplay').textContent();
    
    // Click refresh button
    await page.getByRole('button', { name: 'Refresh Config' }).click();
    
    // Wait a bit for refresh
    await page.waitForTimeout(1000);
    
    // Get updated config content
    const updatedConfig = await page.locator('#configDisplay').textContent();
    
    // Content should be similar (not empty)
    expect(initialConfig).not.toBeNull();
    expect(updatedConfig).not.toBeNull();
  });
});

test.describe('AI Agent Memory Operations', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the AI Agent UI
    await page.goto('/');
  });

  test('should display memory information area', async ({ page }) => {
    // Check that memory display area exists
    await expect(page.locator('#memoryDisplay')).toBeVisible();
    await expect(page.locator('#memoryDisplay')).toContainText('Memory information will appear here...');
  });

  test('should refresh memory', async ({ page }) => {
    // Click refresh memory button
    await page.getByRole('button', { name: 'Refresh Memory' }).click();
    
    // Wait for some time
    await page.waitForTimeout(1000);
    
    // Check that memory display has been updated
    const memoryText = await page.locator('#memoryDisplay').textContent();
    expect(memoryText).toContain('Memory refreshed');
  });
});