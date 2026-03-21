#!/bin/sh
set -eu

rm -rf /app/desktop-client
cp -a /app/workspace/desktop-client /app/desktop-client

cd /app/desktop-client
npm ci

cd /app
Xvfb :99 -screen 0 1280x1024x24 -nolisten tcp &
export DISPLAY=:99
exec npx playwright test --reporter=line
