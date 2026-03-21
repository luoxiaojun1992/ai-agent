#!/bin/sh
set -eu

rm -rf /app/desktop-client
cp -a /app/workspace/desktop-client /app/desktop-client

cd /app/desktop-client
npm ci

cd /app
TEST_TIMEOUT_SECONDS="${DESKTOP_TEST_TIMEOUT_SECONDS:-900}"
exec xvfb-run -a timeout --foreground --signal=TERM --kill-after=30s "${TEST_TIMEOUT_SECONDS}" npx playwright test --reporter=line
