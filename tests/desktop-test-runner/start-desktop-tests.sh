#!/bin/sh
set -eu

rm -rf /app/desktop-client
cp -a /app/workspace/desktop-client /app/desktop-client

cd /app/desktop-client
npm ci
npm run build:dir

cd /app
exec xvfb-run -a npx playwright test --reporter=line
