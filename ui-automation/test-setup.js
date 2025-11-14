// Global test setup
beforeAll(() => {
  // Set longer timeout for integration tests
  jest.setTimeout(60000);
  
  console.log('Setting up test environment...');
  console.log(`UI Backend URL: ${process.env.TEST_BASE_URL || 'http://localhost:3001'}`);
  console.log(`AI Agent SVC URL: ${process.env.AI_AGENT_SVC_URL || 'http://localhost:8080'}`);
});

afterAll(() => {
  console.log('Cleaning up test environment...');
});

// Global error handler for unhandled promises
process.on('unhandledRejection', (reason, promise) => {
  console.error('Unhandled Rejection at:', promise, 'reason:', reason);
});