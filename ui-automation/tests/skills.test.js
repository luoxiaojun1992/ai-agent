const axios = require('axios');
const fs = require('fs');
const path = require('path');

describe('Skill Management Tests', () => {
  const UI_BACKEND_URL = process.env.TEST_BASE_URL || 'http://localhost:3001';
  const TEST_DIR = '/tmp/agent/test';
  const TEST_FILE = 'test.txt';

  beforeAll(async () => {
    // Ensure test directory exists
    try {
      await fs.promises.mkdir(TEST_DIR, { recursive: true });
    } catch (error) {
      console.log('Test directory already exists or cannot be created');
    }
  });

  describe('TC006: Get Available Skills', () => {
    test('Should retrieve all available skills', async () => {
      const response = await axios.get(`${UI_BACKEND_URL}/api/agent/skills`);
      
      expect(response.status).toBe(200);
      expect(response.data.skills).toBeDefined();
      expect(response.data.count).toBeGreaterThan(0);
      expect(response.data.skills.length).toBe(response.data.count);
      
      // Check for expected skills
      const skillNames = response.data.skills.map(skill => skill.name);
      expect(skillNames).toContain('file_reader');
      expect(skillNames).toContain('file_writer');
      expect(skillNames).toContain('http');
      expect(skillNames).toContain('sleep');
    });

    test('All skills should have descriptions', async () => {
      const response = await axios.get(`${UI_BACKEND_URL}/api/agent/skills`);
      
      response.data.skills.forEach(skill => {
        expect(skill.description).toBeDefined();
        expect(skill.description.length).toBeGreaterThan(0);
      });
    });
  });

  describe('TC007: File System Skills', () => {
    test('Should write content to file', async () => {
      const testContent = 'This is a test file content';
      
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'file_writer',
        parameters: {
          path: TEST_FILE,
          content: testContent
        }
      });
      
      expect(response.status).toBe(200);
      expect(response.data.skill).toBe('file_writer');
      expect(response.data.result).toBeDefined();
    });

    test('Should read content from file', async () => {
      const testContent = 'This is a test file content';
      
      // First write the file
      await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'file_writer',
        parameters: {
          path: TEST_FILE,
          content: testContent
        }
      });
      
      // Then read it
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'file_reader',
        parameters: {
          path: path.join('/tmp/agent', TEST_FILE)
        }
      });
      
      expect(response.status).toBe(200);
      expect(response.data.skill).toBe('file_reader');
      expect(response.data.result).toBeDefined();
    });

    test('Should create directory', async () => {
      const testDir = 'test_directory';
      
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'directory_writer',
        parameters: {
          path: testDir
        }
      });
      
      expect(response.status).toBe(200);
      expect(response.data.skill).toBe('directory_writer');
    });

    test('Should list directory contents', async () => {
      const testDir = 'list_test_dir';
      
      // Create directory first
      await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'directory_writer',
        parameters: {
          path: testDir
        }
      });
      
      // List directory
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'directory_reader',
        parameters: {
          path: path.join('/tmp/agent', testDir)
        }
      });
      
      expect(response.status).toBe(200);
      expect(response.data.skill).toBe('directory_reader');
      expect(response.data.result).toBeDefined();
    });
  });

  describe('TC008: HTTP Skill Execution', () => {
    test('Should make HTTP GET request', async () => {
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'http',
        parameters: {
          method: 'GET',
          path: 'https://httpbin.org/get',
          body: '',
          query_params: {},
          http_header: {}
        }
      });
      
      expect(response.status).toBe(200);
      expect(response.data.skill).toBe('http');
      expect(response.data.result).toBeDefined();
    });

    test('Should handle HTTP errors gracefully', async () => {
      try {
        await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
          skillName: 'http',
          parameters: {
            method: 'GET',
            path: 'https://httpbin.org/status/404',
            body: '',
            query_params: {},
            http_header: {}
          }
        });
      } catch (error) {
        // HTTP skill might return the error response rather than throwing
        expect(error.response).toBeDefined();
      }
    });
  });

  describe('TC009: Time-based Skills', () => {
    test('Should sleep for specified duration', async () => {
      const startTime = Date.now();
      
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'sleep',
        parameters: {
          duration: '1s'
        }
      });
      
      const endTime = Date.now();
      const elapsedTime = endTime - startTime;
      
      expect(response.status).toBe(200);
      expect(response.data.skill).toBe('sleep');
      expect(elapsedTime).toBeGreaterThan(900); // Should take at least 900ms
    }, 5000);

    test('Should handle different time formats', async () => {
      const response = await axios.post(`${UI_BACKEND_URL}/api/agent/skill`, {
        skillName: 'sleep',
        parameters: {
          duration: '100ms'
        }
      });
      
      expect(response.status).toBe(200);
      expect(response.data.skill).toBe('sleep');
    });
  });
});