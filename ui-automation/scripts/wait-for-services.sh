#!/bin/bash

# Script to wait for services to be ready before running tests

set -e

UI_BACKEND_URL="${TEST_BASE_URL:-http://localhost:3001}"
AI_AGENT_SVC_URL="${AI_AGENT_SVC_URL:-http://localhost:8080}"

echo "Waiting for services to be ready..."
echo "UI Backend URL: $UI_BACKEND_URL"
echo "AI Agent SVC URL: $AI_AGENT_SVC_URL"

# Function to check service health
check_service() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -f -s "$url/health" > /dev/null 2>&1; then
            echo "✓ $name is ready"
            return 0
        fi
        echo "Waiting for $name... (attempt $((attempt + 1))/$max_attempts)"
        attempt=$((attempt + 1))
        sleep 2
    done
    
    echo "✗ $name is not ready after $max_attempts attempts"
    return 1
}

# Check both services
if ! check_service "$UI_BACKEND_URL" "UI Backend"; then
    exit 1
fi

if ! check_service "$AI_AGENT_SVC_URL" "AI Agent Service"; then
    exit 1
fi

echo "All services are ready! Running tests..."
exec "$@"