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
echo "Pulling qwen3:8b..."
curl -X POST http://ollama:11434/api/pull -d '{"name": "qwen3:8b"}'
echo "Pulling nomic-embed-text..."
curl -X POST http://ollama:11434/api/pull -d '{"name": "nomic-embed-text"}'
echo "✅ Ollama models pulled successfully"

echo "📦 Creating Milvus collection..."
python3 << 'EOF'
from pymilvus import connections, Collection, FieldSchema, CollectionSchema, DataType, utility

try:
    # Connect to Milvus
    connections.connect(alias="default", host="milvus", port="19530")
    print("✅ Connected to Milvus")
    
    # Define collection name
    collection_name = "ai_agent_memory"
    
    # Drop collection if exists
    if utility.has_collection(collection_name):
        print(f"⚠️ Collection {collection_name} already exists, dropping it...")
        utility.drop_collection(collection_name)
    
    # Define fields
    fields = [
        FieldSchema(name="id", dtype=DataType.INT64, is_primary=True, auto_id=False),
        FieldSchema(name="content", dtype=DataType.VARCHAR, max_length=128),
        FieldSchema(name="content_embedding", dtype=DataType.FLOAT_VECTOR, dim=128)
    ]
    
    # Create collection
    schema = CollectionSchema(fields, description="ai_agent_memory_collection")
    collection = Collection(name=collection_name, schema=schema)
    print(f"✅ Collection '{collection_name}' created successfully")
    
    # Create index
    index_params = {
        "index_type": "IVF_FLAT",
        "params": {"nlist": 2},
        "metric_type": "L2"
    }
    collection.create_index("content_embedding", index_params, index_name="content_embedding")
    print("✅ Index created successfully")
    
    # Load collection
    collection.load()
    print("✅ Collection loaded successfully")
    
except Exception as e:
    print(f"❌ Error: {e}")
    exit(1)
finally:
    connections.disconnect("default")
EOF

echo "✅ Milvus collection created and loaded successfully"
echo "🎉 Migration is finished!"
