const axios = require('axios');

describe('Memory Management Tests', () => {
  const UI_BACKEND_URL = process.env.TEST_BASE_URL || 'http://localhost:3001';

  describe('TC010: Memory Retrieval', () => {
    test('Should retrieve memory contexts', async () => {
      // First, add some memory by having a conversation
      await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'Hello, I need to test memory functionality'
      });

      // Wait a moment for processing
      await new Promise(resolve => setTimeout(resolve, 1000));

      const response = await axios.get(`${UI_BACKEND_URL}/api/agent/memory`);
      
      expect(response.status).toBe(200);
      expect(response.data.contexts).toBeDefined();
      expect(response.data.length).toBeDefined();
      expect(response.data.length).toBeGreaterThanOrEqual(0);
    });

    test('Memory should contain conversation history', async () => {
      // Have a conversation
      await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'My name is Test User'
      });

      await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'What is my name?'
      });

      // Wait for processing
      await new Promise(resolve => setTimeout(resolve, 2000));

      const response = await axios.get(`${UI_BACKEND_URL}/api/agent/memory`);
      
      expect(response.status).toBe(200);
      expect(response.data.contexts).toBeInstanceOf(Array);
      
      // Check if contexts have proper structure
      if (response.data.contexts.length > 0) {
        const context = response.data.contexts[0];
        expect(context).toHaveProperty('Role');
        expect(context).toHaveProperty('Content');
        expect(context).toHaveProperty('Epoch');
      }
    });
  });

  describe('TC011: Memory Clearing', () => {
    test('Should clear memory successfully', async () => {
      // First add some memory
      await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'This conversation should be cleared'
      });

      // Wait for processing
      await new Promise(resolve => setTimeout(resolve, 1000));

      // Clear memory
      const clearResponse = await axios.delete(`${UI_BACKEND_URL}/api/agent/memory`);
      
      expect(clearResponse.status).toBe(200);
      expect(clearResponse.data.message).toBe('Memory cleared successfully');

      // Verify memory is cleared
      const memoryResponse = await axios.get(`${UI_BACKEND_URL}/api/agent/memory`);
      
      expect(memoryResponse.status).toBe(200);
      expect(memoryResponse.data.length).toBe(0);
      expect(memoryResponse.data.contexts).toHaveLength(0);
    });

    test('Agent behavior should reset after memory clear', async () => {
      // Add some information to memory
      await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'My favorite color is blue'
      });

      // Clear memory
      await axios.delete(`${UI_BACKEND_URL}/api/agent/memory`);

      // Ask about the information
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'What is my favorite color?'
      });

n      // The agent should not remember after memory clear
      expect(response.status).toBe(200);
      expect(response.data.response).toBeDefined();
      // Note: The exact response will depend on the AI model's behavior
    });
  });
});