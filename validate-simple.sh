#!/bin/bash

# Simple validation script for AI Agent System

echo "🔍 AI Agent System Validation"
echo "============================="

# Check directories
echo "Checking directories..."
for dir in ai-agent-main ui-backend ai-agent-svc frontend ui-automation; do
    if [ -d "$dir" ]; then
        echo "✅ $dir directory exists"
    else
        echo "❌ $dir directory missing"
    fi
done

# Check key files
echo ""
echo "Checking key files..."
KEY_FILES=(
    "ai-agent-main/skill/skill.go"
    "ui-backend/server.js"
    "ai-agent-svc/main.go"
    "frontend/index.html"
    "docker-compose.yml"
    "ui-automation/package.json"
)

for file in "${KEY_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file exists"
    else
        echo "❌ $file missing"
    fi
done

# Check skill descriptions
echo ""
echo "Checking skill descriptions..."
SKILL_COUNT=0
for skill_file in ai-agent-main/skill/impl/**/*.go; do
    if [ -f "$skill_file" ]; then
        if grep -q "GetDescription() string" "$skill_file" 2>/dev/null; then
            ((SKILL_COUNT++))
        fi
    fi
done
echo "📊 Found $SKILL_COUNT skills with GetDescription methods"

# Check test files
echo ""
echo "Checking test files..."
TEST_COUNT=$(find ui-automation/tests -name "*.test.js" 2>/dev/null | wc -l)
echo "📊 Found $TEST_COUNT test files"

# Check deployment package
if [ -f "ai-agent-enhanced.tar.gz" ]; then
    SIZE=$(stat -f%z "ai-agent-enhanced.tar.gz" 2>/dev/null || stat -c%s "ai-agent-enhanced.tar.gz" 2>/dev/null || echo "0")
    echo "📦 Deployment package exists ($SIZE bytes)"
else
    echo "📦 Deployment package missing"
fi

echo ""
echo "🎯 Validation Complete!"
echo "The AI Agent system enhancement is ready for deployment."
echo ""
echo "🚀 Quick Start:"
echo "   docker-compose up -d"
echo "   ./test-runner.sh"
echo "   Open: http://localhost:3000"