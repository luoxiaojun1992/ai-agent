# AI Agent System - Deployment Guide

## 🚀 Quick Deployment

### Manual Deployment Steps
```bash
# 1. Start all services
docker-compose up --build -d

# 2. Access the system
open http://localhost:3000
```

## 📋 Pre-Deployment Checklist

### ✅ System Requirements
- [ ] Docker 20.10+
- [ ] Docker Compose 2.0+
- [ ] 8GB+ RAM recommended
- [ ] 10GB+ disk space
- [ ] Internet connection for initial setup

### ✅ Port Availability
- [ ] Port 3000 (Frontend)
- [ ] Port 3001 (UI Backend)
- [ ] Port 8080 (AI Agent Service)
- [ ] Port 8443 (VSCode IDE / code-server)
- [ ] Port 19530 (Milvus)
- [ ] Port 11434 (Ollama)

## 🐳 Service Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Docker Compose Stack                      │
├─────────────────────────────────────────────────────────────┤
│  🌐 Frontend (Nginx)     │  🔌 UI Backend (Node.js)        │
│  Port: 3000              │  Port: 3001                     │
│  Web Interface           │  API Gateway                    │
├─────────────────────────┼─────────────────────────────────┤
│  🤖 AI Agent SVC (Go)   │  📊 Milvus (Vector DB)          │
│  Port: 8080              │  Port: 19530                    │
│  Core Logic              │  Memory Storage                 │
├─────────────────────────┼─────────────────────────────────┤
│  🧠 Ollama (AI Models)  │  🧪 Test Runner (Optional)      │
│  Port: 11434             │  Automated Testing              │
│  LLM & Embeddings        │  CI/CD Integration              │
└─────────────────────────┴─────────────────────────────────┘
```

## 🔧 Configuration

### Environment Files
Create `.env` files for each service:

#### UI Backend (.env)
```env
PORT=3001
CORS_ORIGIN=http://localhost:3000
AI_AGENT_SVC_URL=http://ai-agent-svc:8080
LOG_LEVEL=info
```

#### AI Agent Service (Environment)
```env
PORT=8080
CORS_ORIGINS=*
CHAT_MODEL=qwen3:4b
EMBEDDING_MODEL=nomic-embed-text
SUPERVISOR_MODEL=qwen3:4b
OLLAMA_HOST=http://ollama:11434
MILVUS_HOST=milvus:19530
MILVUS_COLLECTION=ai_agent_memory
AGENT_CHARACTER=You are a helpful AI assistant
AGENT_ROLE=AI Assistant and Tool User
```

## 🔍 Monitoring and Health Checks

### Service Health
```bash
# Check all services
docker-compose ps

# Health check endpoints
curl http://localhost:3001/health    # UI Backend
curl http://localhost:8080/health    # AI Agent SVC
curl http://localhost:11434/api/tags # Ollama
```

### Logs and Debugging
```bash
# View all logs
docker-compose logs

# View specific service logs
docker-compose logs -f ai-agent-svc
docker-compose logs -f ui-backend
```

## 🧪 Testing After Deployment

### Quick Health Check
```bash
# Test API endpoints
curl http://localhost:3001/api/agent/status
curl http://localhost:3001/api/agent/skills

# Test chat functionality
curl -X POST http://localhost:3001/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello, are you working?"}'

# Test skill execution
curl -X POST http://localhost:3001/api/agent/skill \
  -H "Content-Type: application/json" \
  -d '{"skillName":"sleep","parameters":{"duration":"100ms"}}'
```

## 🔧 Troubleshooting

### Common Issues

1. **Services not starting**
   ```bash
   # Check port conflicts
   netstat -tulpn | grep :3000
   
   # Check Docker status
   docker system info
   
   # Restart services
   docker-compose down && docker-compose up -d
   ```

2. **AI Agent not responding**
   ```bash
   # Check Ollama models
   curl http://localhost:11434/api/tags
   
   # Pull required models
   curl -X POST http://localhost:11434/api/pull \
     -d '{"name":"qwen3:4b"}'
   
   # Check Milvus connection
   docker-compose logs milvus
   ```

3. **Memory issues**
   ```bash
   # Check memory usage
   docker stats
   
   # Increase Docker memory limit
   # Docker Desktop > Settings > Resources > Memory
   ```

### Performance Optimization
```bash
# Scale services
docker-compose up -d --scale ui-backend=2
docker-compose up -d --scale ai-agent-svc=2
```

## 🚪 Cleanup

### Stop Services
```bash
# Stop all services
docker-compose down

# Remove volumes
docker-compose down -v

# Remove everything
docker-compose down -v --rmi all
```

### Reset Environment
```bash
# Complete cleanup
docker-compose down -v --rmi all
docker system prune -f
docker volume prune -f
```

### Access URLs
- **Frontend**: http://localhost:3000
- **VSCode IDE (code-server)**: http://localhost:8443/?folder=/workspace-root/default
- **API**: http://localhost:3001
- **Health**: http://localhost:3001/health
- **Agent Status**: http://localhost:8080/health

### File Locations
- **Logs**: `docker-compose logs <service>`
- **Data**: Docker volumes (milvus_data, ollama_data)
- **Config**: Environment files and docker-compose.yml

**Happy deploying!** 🚀✨
