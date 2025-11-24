#!/bin/bash
set -e

echo "🚀 Starting migration..."

echo "⏳ Waiting for ollama"
while ! curl -s http://ollama:11434/api/tags > /dev/null; do
    echo "Ollama is not ready，wait for 5s..."
    sleep 5
done
echo "✅ Ollama is ready"

echo "⏳ Waiting for Milvus..."
while ! python3 -c "from pymilvus import connections; connections.connect(alias='default', host='milvus', port='19530')" 2>/dev/null; do
    echo "Milvus is not ready，wait for 5s..."
    sleep 5
done
echo "✅ Milvus is ready"

echo "📥 Pulling Ollama models..."
echo "Pulling deepseek-r1:1.5b..."
curl -X POST http://ollama:11434/api/pull -d '{"name": "deepseek-r1:1.5b"}'
echo "Pulling nomic-embed-text..."
curl -X POST http://ollama:11434/api/pull -d '{"name": "nomic-embed-text"}'
echo "✅ Ollama models pulled successfully"

echo "📦 Creating Milvus collection..."
milvus_cli << EOF
connect --uri http://milvus:19530
create collection -c ai_agent_memory -f id:INT64:primary_field -f content:VARCHAR:128 -f content_embedding:FLOAT_VECTOR:128 -p id -A -d 'ai_agent_memory_collection'
create index -c ai_agent_memory -f content_embedding -t IVF_FLAT -m L2 -n content_embedding -p '{"nlist": 2}'
load collection -c ai_agent_memory
exit
EOF

echo "✅ Milvus collection created and loaded successfully"
echo "🎉 Migration is finished!"
