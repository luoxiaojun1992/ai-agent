const axios = require('axios');

describe('Configuration Tests', () => {
  const UI_BACKEND_URL = process.env.TEST_BASE_URL || 'http://localhost:3001';

  describe('TC012: Configuration Retrieval', () => {
    test('Should retrieve agent configuration', async () => {
      const response = await axios.get(`${UI_BACKEND_URL}/api/agent/config`);
      
      expect(response.status).toBe(200);
      expect(response.data.chatModel).toBeDefined();
      expect(response.data.embeddingModel).toBeDefined();
      expect(response.data.ollamaHost).toBeDefined();
      expect(response.data.milvusHost).toBeDefined();
      expect(response.data.character).toBeDefined();
      expect(response.data.role).toBeDefined();
    });

    test('Configuration should have default values', async () => {
      const response = await axios.get(`${UI_BACKEND_URL}/api/agent/config`);
      
      expect(response.data.chatModel).toBeTruthy();
      expect(response.data.embeddingModel).toBeTruthy();
      expect(response.data.ollamaHost).toBeTruthy();
      expect(response.data.milvusHost).toBeTruthy();
    });
  });

  describe('TC013: Configuration Update', () => {
    test('Should update configuration successfully', async () => {
      const newConfig = {
        character: 'Test Character Profile',
        role: 'Test Role Definition'
      };

      const response = await axios.put(`${UI_BACKEND_URL}/api/agent/config`, newConfig);
      
      expect(response.status).toBe(200);
      expect(response.data.message).toBe('Configuration updated successfully');
      expect(response.data.updated).toEqual(newConfig);
    });

    test('Should handle partial configuration updates', async () => {
      const partialConfig = {
        character: 'Updated Character Only'
      };

      const response = await axios.put(`${UI_BACKEND_URL}/api/agent/config`, partialConfig);
      
      expect(response.status).toBe(200);
      expect(response.data.message).toBe('Configuration updated successfully');
      expect(response.data.updated.character).toBe('Updated Character Only');
    });

    test('Should handle empty configuration update', async () => {
      const emptyConfig = {};

      const response = await axios.put(`${UI_BACKEND_URL}/api/agent/config`, emptyConfig);
      
      expect(response.status).toBe(200);
      expect(response.data.message).toBe('Configuration updated successfully');
      expect(response.data.updated).toEqual(emptyConfig);
    });
  });
});