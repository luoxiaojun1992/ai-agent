const axios = require('axios');

describe('Error Handling Tests', () => {
  const UI_BACKEND_URL = process.env.TEST_BASE_URL || 'http://localhost:3001';

  describe('TC014: Invalid Skill Execution', () => {
    test('Should handle non-existent skill gracefully', async () => {
      try {
        await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
          skillName: 'non_existent_skill',
          parameters: {}
        });
        fail('Should have thrown an error');
      } catch (error) {
        expect(error.response.status).toBe(500);
        expect(error.response.data.error).toBeDefined();
      }
    });

    test('Should handle skill execution with missing parameters', async () => {
      try {
        await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
          skillName: 'file_reader',
          parameters: {} // Missing required 'path' parameter
        });
        fail('Should have thrown an error');
      } catch (error) {
        expect(error.response.status).toBe(500);
        expect(error.response.data.error).toBeDefined();
      }
    });
  });

  describe('TC015: Invalid Parameters', () => {
    test('Should handle malformed chat requests', async () => {
      try {
        await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
          invalidField: 'test'
        });
        fail('Should have thrown an error');
      } catch (error) {
        expect(error.response.status).toBe(400);
        expect(error.response.data.error).toBeDefined();
      }
    });

    test('Should handle missing required parameters in skill execution', async () => {
      try {
        await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
          // Missing skillName
          parameters: {}
        });
        fail('Should have thrown an error');
      } catch (error) {
        expect(error.response.status).toBe(400);
        expect(error.response.data.error).toBeDefined();
      }
    });

    test('Should handle invalid JSON in request body', async () => {
      try {
        await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, 'invalid json', {
          headers: {
            'Content-Type': 'application/json'
          }
        });
        fail('Should have thrown an error');
      } catch (error) {
        expect(error.response.status).toBe(400);
      }
    });
  });

  describe('TC016: Timeout Scenarios', () => {
    test('Should handle long-running chat operations', async () => {
      // This test might take longer than usual
      jest.setTimeout(60000);
      
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'Please provide a detailed explanation of quantum computing'
      });
      
      expect(response.status).toBe(200);
      expect(response.data.response).toBeDefined();
    }, 60000);

    test('Should handle concurrent skill executions', async () => {
      const promises = [];
      
      // Send multiple skill execution requests simultaneously
      for (let i = 0; i < 5; i++) {
        promises.push(
          axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
            skillName: 'sleep',
            parameters: {
              duration: '100ms'
            }
          })
        );
      }
      
      const responses = await Promise.all(promises);
      
      responses.forEach(response => {
        expect(response.status).toBe(200);
        expect(response.data.skill).toBe('sleep');
      });
    });
  });
});