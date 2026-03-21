#!/bin/sh
set -eu

cleanup() {
  pids="$(pgrep -f '/app/desktop-client/node_modules/electron/dist/electron' || true)"
  if [ -n "${pids}" ]; then
    for pid in ${pids}; do
      kill -TERM "${pid}" 2>/dev/null || true
    done
  fi
}

trap cleanup EXIT INT TERM

rm -rf /app/desktop-client
cp -a /app/workspace/desktop-client /app/desktop-client

cd /app/desktop-client
npm ci

cd /app
exec xvfb-run -a npx playwright test --reporter=line
