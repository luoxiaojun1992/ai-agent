const request = require('supertest');
const { PassThrough } = require('stream');

jest.mock('axios', () => ({
  get: jest.fn(),
  post: jest.fn(),
  put: jest.fn(),
  delete: jest.fn()
}));

const axios = require('axios');
const app = require('./server');

describe('UI Backend API tests (按测试场景组织)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('场景一：健康检查与状态查询', () => {
    test('GET /health 返回服务可用状态和时间戳', async () => {
      const response = await request(app).get('/health');

      expect(response.status).toBe(200);
      expect(response.body.status).toBe('OK');
      expect(typeof response.body.timestamp).toBe('string');
    });

    test('GET /api/agent/status 成功代理到 AI Agent Service', async () => {
      axios.get.mockResolvedValueOnce({
        data: { status: 'running', character: 'assistant' }
      });

      const response = await request(app).get('/api/agent/status');

      expect(response.status).toBe(200);
      expect(response.body).toEqual({ status: 'running', character: 'assistant' });
      expect(axios.get).toHaveBeenCalledWith('http://localhost:8080/status');
    });

    test('GET /api/agent/status 在下游失败时返回 500', async () => {
      axios.get.mockRejectedValueOnce(new Error('network error'));

      const response = await request(app).get('/api/agent/status');

      expect(response.status).toBe(500);
      expect(response.body).toEqual({ error: 'Failed to fetch agent status' });
    });
  });

  describe('场景二：聊天接口（普通与流式）', () => {
    test('POST /api/agent/chat 非流式模式透传请求并返回结果', async () => {
      axios.post.mockResolvedValueOnce({
        data: { response: 'hello', timestamp: 12345 }
      });

      const payload = {
        message: 'Hello',
        agentConfig: { character: 'tester' }
      };
      const response = await request(app).post('/api/agent/chat').send(payload);

      expect(response.status).toBe(200);
      expect(response.body).toEqual({ response: 'hello', timestamp: 12345 });
      expect(axios.post).toHaveBeenCalledWith('http://localhost:8080/chat', {
        message: 'Hello',
        images: undefined,
        agentConfig: { character: 'tester' },
        stream: false
      });
    });

    test('POST /api/agent/chat 流式模式设置 SSE 并转发流数据', async () => {
      const stream = new PassThrough();
      axios.post.mockResolvedValueOnce({ data: stream });

      const responsePromise = request(app)
        .post('/api/agent/chat')
        .send({ message: 'streaming', stream: true });

      await new Promise((resolve) => setImmediate(resolve));
      stream.write('event: message\ndata: {"content":"hello"}\n\n');
      stream.end('event: complete\ndata: {"done":true}\n\n');

      const response = await responsePromise;

      expect(response.status).toBe(200);
      expect(response.headers['content-type']).toContain('text/event-stream');
      expect(response.text).toContain('event: message');
      expect(response.text).toContain('event: complete');
      expect(axios.post).toHaveBeenCalledWith(
        'http://localhost:8080/chat',
        {
          message: 'streaming',
          images: undefined,
          agentConfig: undefined,
          stream: true
        },
        {
          responseType: 'stream',
          headers: { Accept: 'text/event-stream' }
        }
      );
    });

    test('POST /api/agent/chat 流式初始化失败时返回 500', async () => {
      axios.post.mockRejectedValueOnce(new Error('stream unavailable'));

      const response = await request(app)
        .post('/api/agent/chat')
        .send({ message: 'streaming', stream: true });

      expect(response.status).toBe(500);
      // 流式接口会先设置 SSE 响应头，失败时只保证 HTTP 状态码为 500
      expect(response.body).toEqual({});
    });

    test('POST /api/agent/chat 透传 images 字段', async () => {
      axios.post.mockResolvedValueOnce({
        data: { response: 'image ok', timestamp: 12345 }
      });

      const payload = {
        message: 'describe this image',
        images: ['aGVsbG8='],
        stream: false
      };
      const response = await request(app).post('/api/agent/chat').send(payload);

      expect(response.status).toBe(200);
      expect(response.body).toEqual({ response: 'image ok', timestamp: 12345 });
      expect(axios.post).toHaveBeenCalledWith('http://localhost:8080/chat', {
        message: 'describe this image',
        images: ['aGVsbG8='],
        agentConfig: undefined,
        stream: false
      });
    });
  });

  describe('场景三：技能调用与配置管理', () => {
    test('POST /api/agent/skill 成功执行技能', async () => {
      axios.post.mockResolvedValueOnce({
        data: { result: { done: true }, skill: 'sleep' }
      });

      const response = await request(app)
        .post('/api/agent/skill')
        .send({ skillName: 'sleep', parameters: { duration: '1s' } });

      expect(response.status).toBe(200);
      expect(response.body).toEqual({ result: { done: true }, skill: 'sleep' });
      expect(axios.post).toHaveBeenCalledWith('http://localhost:8080/skill', {
        skillName: 'sleep',
        parameters: { duration: '1s' }
      });
    });

    test('GET /api/agent/config 获取配置', async () => {
      axios.get.mockResolvedValueOnce({
        data: { chatModel: 'qwen', role: 'assistant' }
      });

      const response = await request(app).get('/api/agent/config');

      expect(response.status).toBe(200);
      expect(response.body).toEqual({ chatModel: 'qwen', role: 'assistant' });
      expect(axios.get).toHaveBeenCalledWith('http://localhost:8080/config');
    });

    test('PUT /api/agent/config 更新配置', async () => {
      axios.put.mockResolvedValueOnce({
        data: { message: 'Configuration updated', config: { chatModel: 'new-model' } }
      });

      const response = await request(app)
        .put('/api/agent/config')
        .send({ chatModel: 'new-model' });

      expect(response.status).toBe(200);
      expect(response.body.message).toBe('Configuration updated');
      expect(axios.put).toHaveBeenCalledWith('http://localhost:8080/config', {
        chatModel: 'new-model'
      });
    });
  });

  describe('场景四：记忆管理接口', () => {
    test('GET /api/agent/memory 获取记忆内容', async () => {
      axios.get.mockResolvedValueOnce({
        data: { contexts: ['a', 'b'], length: 2 }
      });

      const response = await request(app).get('/api/agent/memory');

      expect(response.status).toBe(200);
      expect(response.body).toEqual({ contexts: ['a', 'b'], length: 2 });
      expect(axios.get).toHaveBeenCalledWith('http://localhost:8080/memory');
    });

    test('DELETE /api/agent/memory 清空记忆', async () => {
      axios.delete.mockResolvedValueOnce({
        data: { message: 'Memory cleared successfully' }
      });

      const response = await request(app).delete('/api/agent/memory');

      expect(response.status).toBe(200);
      expect(response.body).toEqual({ message: 'Memory cleared successfully' });
      expect(axios.delete).toHaveBeenCalledWith('http://localhost:8080/memory');
    });
  });
});
