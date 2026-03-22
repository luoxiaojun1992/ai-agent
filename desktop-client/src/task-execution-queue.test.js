const test = require('node:test');
const assert = require('node:assert/strict');
const { createTaskExecutionQueue } = require('./task-execution-queue');

test('executes enqueued tasks sequentially in enqueue order', async () => {
  const enqueue = createTaskExecutionQueue();
  const events = [];

  const first = enqueue(async () => {
    events.push('start-1');
    await new Promise((resolve) => setTimeout(resolve, 20));
    events.push('end-1');
    return 'first';
  });

  const second = enqueue(async () => {
    events.push('start-2');
    await new Promise((resolve) => setTimeout(resolve, 5));
    events.push('end-2');
    return 'second';
  });

  const [firstResult, secondResult] = await Promise.all([first, second]);

  assert.equal(firstResult, 'first');
  assert.equal(secondResult, 'second');
  assert.deepEqual(events, ['start-1', 'end-1', 'start-2', 'end-2']);
});

test('continues processing queue after a task failure', async () => {
  const errors = [];
  const enqueue = createTaskExecutionQueue({
    error: (...args) => errors.push(args)
  });

  await assert.rejects(
    enqueue(async () => {
      throw new Error('boom');
    }),
    /boom/
  );

  const result = await enqueue(async () => 'recovered');

  assert.equal(result, 'recovered');
  assert.equal(errors.length, 1);
});
