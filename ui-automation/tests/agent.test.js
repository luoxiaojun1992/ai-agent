const axios = require('axios');

describe('Agent Communication Tests', () => {
  const UI_BACKEND_URL = process.env.TEST_BASE_URL || 'http://localhost:3001';

  describe('TC003: Agent Status Retrieval', () => {
    test('Should retrieve agent status successfully', async () => {
      const response = await axios.get(`${UI_BACKEND_URL}/api/agent/status`);
      
      expect(response.status).toBe(200);
      expect(response.data.status).toBe('running');
      expect(response.data.character).toBeDefined();
      expect(response.data.skills).toBeGreaterThan(0);
      expect(response.data.timestamp).toBeDefined();
    });
  });

  describe('TC004: Basic Chat Functionality', () => {
    test('Should send message and receive response', async () => {
      const message = 'Hello, can you help me?';
      
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: message
      });
      
      expect(response.status).toBe(200);
      expect(response.data.response).toBeDefined();
      expect(response.data.response.length).toBeGreaterThan(0);
      expect(response.data.timestamp).toBeDefined();
    }, 30000);

    test('Should handle empty message gracefully', async () => {
      try {
        await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
          message: ''
        });
        fail('Should have thrown an error');
      } catch (error) {
        expect(error.response.status).toBe(400);
        expect(error.response.data.error).toBeDefined();
      }
    });

    test('Should handle missing message field', async () => {
      try {
        await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {});
        fail('Should have thrown an error');
      } catch (error) {
        expect(error.response.status).toBe(400);
        expect(error.response.data.error).toBeDefined();
      }
    });
  });

  describe('TC005: Chat with Configuration', () => {
    test('Should send message with custom configuration', async () => {
      const message = 'What is your role?';
      const agentConfig = {
        character: 'You are a technical expert',
        role: 'Software Engineer'
      };
      
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: message,
        agentConfig: agentConfig
      });
      
      expect(response.status).toBe(200);
      expect(response.data.response).toBeDefined();
      expect(response.data.response.length).toBeGreaterThan(0);
    }, 30000);
  });
});