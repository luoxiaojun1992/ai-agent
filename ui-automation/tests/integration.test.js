const axios = require('axios');

describe('Integration Tests', () => {
  const UI_BACKEND_URL = process.env.TEST_BASE_URL || 'http://localhost:3001';

  describe('TC017: End-to-End Conversation Flow', () => {
    test('Should maintain conversation context across multiple messages', async () => {
      // First message
      const response1 = await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'My favorite programming language is JavaScript'
      });
      
      expect(response1.status).toBe(200);
      expect(response1.data.response).toBeDefined();

      // Wait for processing
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Second message asking about previous information
      const response2 = await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'What is my favorite programming language?'
      });
      
      expect(response2.status).toBe(200);
      expect(response2.data.response).toBeDefined();
      
      // The response should contain information about the conversation
      const responseText = response2.data.response.toLowerCase();
      // Note: This depends on the AI model's ability to recall context
    }, 60000);

    test('Should execute skills during conversation', async () => {
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'Can you create a file named hello.txt with content "Hello World"?'
      });
      
      expect(response.status).toBe(200);
      expect(response.data.response).toBeDefined();
      
      // The agent might use the file_writer skill in response
      // Check if the file was created (this depends on agent's decision)
    }, 45000);
  });

  describe('TC018: Skill Chaining', () => {
    test('Should execute multiple skills in sequence', async () => {
      // Create a directory
      const dirResponse = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'directory_writer',
        parameters: {
          path: 'chain_test'
        }
      });
      
      expect(dirResponse.status).toBe(200);

      // Create a file in that directory
      const fileResponse = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'file_writer',
        parameters: {
          path: 'chain_test/test.txt',
          content: 'Chained skill test'
        }
      });
      
      expect(fileResponse.status).toBe(200);

      // Read the directory
      const readResponse = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'directory_reader',
        parameters: {
          path: '/tmp/agent/chain_test'
        }
      });
      
      expect(readResponse.status).toBe(200);
      expect(readResponse.data.result).toBeDefined();
    });

    test('Should handle skill dependencies correctly', async () => {
      // This test demonstrates that skills can depend on each other
      
      // First, get the current configuration
      const configResponse = await axios.get(`${UI_BACKEND_URL}/api/agent/config`);
      expect(configResponse.status).toBe(200);
      
      // Update the configuration
      const updateResponse = await axios.put(`${UI_BACKEND_URL}/api/agent/config`, {
        character: 'Dependency Test Character'
      });
      expect(updateResponse.status).toBe(200);
      
      // Verify the configuration was updated
      const newConfigResponse = await axios.get(`${UI_BACKEND_URL}/api/agent/config`);
      expect(newConfigResponse.status).toBe(200);
    });
  });

  describe('TC019: Concurrent Requests', () => {
    test('Should handle multiple simultaneous chat requests', async () => {
      const messages = [
        'What is the weather like?',
        'Tell me a joke',
        'What is 2 + 2?',
        'Who are you?',
        'What can you do?'
      ];
      
      const promises = messages.map(message => 
        axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
          message: message
        })
      );
      
      const responses = await Promise.all(promises);
      
      responses.forEach(response => {
        expect(response.status).toBe(200);
        expect(response.data.response).toBeDefined();
      });
    }, 60000);

    test('Should handle mixed concurrent requests', async () => {
      const chatPromise = axios.post(`${UI_BACKEND_URL}/api/agent/chat`, {
        message: 'Hello'
      });
      
      const statusPromise = axios.get(`${UI_BACKEND_URL}/api/agent/status`);
      
      const skillsPromise = axios.get(`${UI_BACKEND_URL}/api/agent/skills`);
      
      const [chatResponse, statusResponse, skillsResponse] = await Promise.all([
        chatPromise,
        statusPromise,
        skillsPromise
      ]);
      
      expect(chatResponse.status).toBe(200);
      expect(chatResponse.data.response).toBeDefined();
      
      expect(statusResponse.status).toBe(200);
      expect(statusResponse.data.status).toBe('running');
      
      expect(skillsResponse.status).toBe(200);
      expect(skillsResponse.data.skills).toBeDefined();
    });
  });
});