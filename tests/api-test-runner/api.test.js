const axios = require('axios');

const BASE_URL = process.env.API_BASE_URL || 'http://ui-backend:3001';
const RETRY_COUNT = Number(process.env.API_TEST_RETRY_COUNT || 20);
const RETRY_INTERVAL_MS = Number(process.env.API_TEST_RETRY_INTERVAL_MS || 1000);

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function run() {
  console.log(`[api-test] base url: ${BASE_URL}`);

  const health = await requestWithRetry(() => axios.get(`${BASE_URL}/health`), 'health');
  assert(health.status === 200, 'health status should be 200');
  assert(health.data && health.data.status === 'OK', 'health status body should be OK');

  const status = await requestWithRetry(() => axios.get(`${BASE_URL}/api/agent/status`), 'status');
  assert(status.status === 200, 'status endpoint should be 200');
  assert(status.data && status.data.status === 'running', 'agent status should be running');

  const chat = await requestWithRetry(() => axios.post(`${BASE_URL}/api/agent/chat`, {
    message: 'hello from api test',
    stream: false,
  }), 'chat');
  assert(chat.status === 200, 'chat endpoint should be 200');
  assert(
    typeof chat.data.response === 'string' && chat.data.response.includes('mock response'),
    'chat response should contain mock response'
  );

  const imageChat = await requestWithRetry(() => axios.post(`${BASE_URL}/api/agent/chat`, {
    message: 'hello with image',
    images: ['aGVsbG8='],
    stream: false,
  }), 'image chat');
  assert(imageChat.status === 200, 'image chat endpoint should be 200');
  assert(
    typeof imageChat.data.response === 'string' && imageChat.data.response.includes('mock response'),
    'image chat response should contain mock response'
  );

  const skill = await requestWithRetry(() => axios.post(`${BASE_URL}/api/agent/skill`, {
    skillName: 'sleep',
    parameters: { duration: '1s' },
  }), 'skill');
  assert(skill.status === 200, 'skill endpoint should be 200');
  assert(skill.data && skill.data.skill === 'sleep', 'skill name should be sleep');

  const config = await requestWithRetry(() => axios.get(`${BASE_URL}/api/agent/config`), 'config');
  assert(config.status === 200, 'config endpoint should be 200');
  assert(config.data && config.data.chatModel, 'config should include chatModel');

  const update = await requestWithRetry(() => axios.put(`${BASE_URL}/api/agent/config`, {
    chatModel: 'mock-ci-model',
  }), 'update config');
  assert(update.status === 200, 'update config should be 200');

  const memory = await requestWithRetry(() => axios.get(`${BASE_URL}/api/agent/memory`), 'memory');
  assert(memory.status === 200, 'memory endpoint should be 200');
  assert(memory.data && typeof memory.data.length === 'number', 'memory length should be number');

  const clearMemory = await requestWithRetry(() => axios.delete(`${BASE_URL}/api/agent/memory`), 'clear memory');
  assert(clearMemory.status === 200, 'clear memory should be 200');

  console.log('[api-test] all API checks passed');
}

run().catch((error) => {
  console.error('[api-test] failed:', error.message);
  process.exit(1);
});

async function requestWithRetry(fn, label) {
  let lastError;
  for (let i = 0; i < RETRY_COUNT; i++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
      if (i < RETRY_COUNT - 1) {
        await new Promise((resolve) => setTimeout(resolve, RETRY_INTERVAL_MS));
      }
    }
  }
  throw new Error(`${label} failed after retries: ${lastError?.message || 'unknown error'}`);
}
