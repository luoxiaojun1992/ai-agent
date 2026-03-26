const test = require('node:test');
const assert = require('node:assert/strict');
const { parseSSEEvents } = require('./sse');

test('parseSSEEvents parses message and complete events and returns remainder', () => {
  const chunk = [
    'event: message',
    'data: {"content":"hello"}',
    '',
    'event: complete',
    'data: {"done":true}',
    '',
    'event: message',
    'data: {"content":"partial"}'
  ].join('\n');

  const parsed = parseSSEEvents(chunk);

  assert.equal(parsed.events.length, 2);
  assert.deepEqual(parsed.events[0], {
    eventType: 'message',
    data: { content: 'hello' }
  });
  assert.deepEqual(parsed.events[1], {
    eventType: 'complete',
    data: { done: true }
  });
  assert.equal(parsed.remainder, 'event: message\ndata: {"content":"partial"}');
});

test('parseSSEEvents ignores malformed JSON payloads', () => {
  const chunk = [
    'event: message',
    'data: {"content":"ok"}',
    '',
    'event: message',
    'data: {invalid json}',
    '',
    ''
  ].join('\n');

  const parsed = parseSSEEvents(chunk);

  assert.equal(parsed.events.length, 1);
  assert.deepEqual(parsed.events[0], {
    eventType: 'message',
    data: { content: 'ok' }
  });
  assert.equal(parsed.remainder, '');
});
