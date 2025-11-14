const axios = require('axios');

describe('Health and Connectivity Tests', () => {
  const UI_BACKEND_URL = process.env.TEST_BASE_URL || 'http://localhost:3001';
  const AI_AGENT_SVC_URL = process.env.AI_AGENT_SVC_URL || 'http://localhost:8080';

  describe('TC001: Service Health Check', () => {
    test('UI Backend service should be healthy', async () => {
      const response = await axios.get(`${UI_BACKEND_URL}/health`);
      
      expect(response.status).toBe(200);
      expect(response.data.status).toBe('OK');
      expect(response.data.timestamp).toBeDefined();
    }, 10000);

    test('AI Agent service should be healthy', async () => {
      const response = await axios.get(`${AI_AGENT_SVC_URL}/health`);
      
      expect(response.status).toBe(200);
      expect(response.data.status).toBe('healthy');
      expect(response.data.timestamp).toBeDefined();
    }, 10000);

    test('Service response time should be reasonable', async () => {
      const startTime = Date.now();
      await axios.get(`${UI_BACKEND_URL}/health`);
      const responseTime = Date.now() - startTime;
      
      expect(responseTime).toBeLessThan(5000); // Should respond within 5 seconds
    });
  });

  describe('TC002: CORS Configuration', () => {
    test('Should handle OPTIONS preflight requests', async () => {
      const response = await axios.options(`${UI_BACKEND_URL}/api/agent/status`, {
        headers: {
          'Origin': 'http://localhost:3000',
          'Access-Control-Request-Method': 'GET'
        }
      });
      
      expect(response.status).toBe(200);
      expect(response.headers['access-control-allow-origin']).toBeDefined();
    });

    test('Should allow requests from configured origin', async () => {
      const response = await axios.get(`${UI_BACKEND_URL}/api/agent/status`, {
        headers: {
          'Origin': 'http://localhost:3000'
        }
      });
      
      expect(response.status).toBe(200);
      expect(response.headers['access-control-allow-origin']).toBeDefined();
    });
  });
});