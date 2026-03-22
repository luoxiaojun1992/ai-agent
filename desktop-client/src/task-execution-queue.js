function createTaskExecutionQueue(logger = console) {
  let queueTail = Promise.resolve();

  return function enqueue(taskExecutor) {
    const executeTask = () => taskExecutor();
    // Continue with the next queued task even if the previous task failed.
    // This keeps queue availability while still returning each task's own result/error.
    const executePromise = queueTail.then(executeTask, executeTask);

    queueTail = executePromise.catch((error) => {
      if (logger && typeof logger.error === 'function') {
        logger.error('Scheduled task queue execution failed:', error);
      }
    });

    return executePromise;
  };
}

module.exports = {
  createTaskExecutionQueue
};
