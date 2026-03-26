function parseSSEEvents(chunkBuffer) {
  const frames = chunkBuffer.split('\n\n');
  const remainder = frames.pop() || '';
  const events = [];

  for (const frame of frames) {
    const lines = frame.split('\n');
    let eventType = 'message';
    const dataLines = [];

    for (const rawLine of lines) {
      const line = rawLine.trim();
      if (!line || line.startsWith(':')) {
        continue;
      }
      if (line.startsWith('event:')) {
        eventType = line.slice(6).trim();
        continue;
      }
      if (line.startsWith('data:')) {
        dataLines.push(line.slice(5).trim());
      }
    }

    if (dataLines.length === 0) {
      continue;
    }

    try {
      events.push({
        eventType,
        data: JSON.parse(dataLines.join('\n'))
      });
    } catch (error) {
      // ignore malformed event payloads
    }
  }

  return { events, remainder };
}

module.exports = {
  parseSSEEvents
};
