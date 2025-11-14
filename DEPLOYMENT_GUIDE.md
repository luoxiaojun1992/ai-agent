# AI Agent System - Deployment Guide

## 🚀 Quick Deployment

### One-Command Deployment
```bash
# Download and deploy the complete system
curl -L <deployment-url> | tar -xz && cd ai-agent-enhanced && ./test-runner.sh
```

### Manual Deployment Steps
```bash
# 1. Extract the deployment package
tar -xzf ai-agent-enhanced.tar.gz
cd ai-agent-enhanced

# 2. Validate the system
./validate-simple.sh

# 3. Start all services
docker-compose up -d

# 4. Wait for services to start
sleep 30

# 5. Run comprehensive tests
./test-runner.sh

# 6. Access the system
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
- [ ] Port 19530 (Milvus)
- [ ] Port 11434 (Ollama)

### ✅ Environment Setup
- [ ] Create deployment directory
- [ ] Extract deployment package
- [ ] Verify all components

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
CHAT_MODEL=deepseek-r1:1.5b
EMBEDDING_MODEL=nomic-embed-text
SUPERVISOR_MODEL=deepseek-r1:1.5b
OLLAMA_HOST=http://ollama:11434
MILVUS_HOST=milvus:19530
MILVUS_COLLECTION=ai_agent_memory
AGENT_CHARACTER=You are a helpful AI assistant
AGENT_ROLE=AI Assistant and Tool User
```

## 🏃‍♂️ Deployment Scenarios

### Development Deployment
```bash
# Start with hot reload
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

# Or start specific services
docker-compose up ui-backend ai-agent-svc
```

### Production Deployment
```bash
# Production configuration
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# With reverse proxy
docker-compose -f docker-compose.yml -f docker-compose.traefik.yml up -d
```

### Testing Deployment
```bash
# Run tests
docker-compose --profile test up ui-test-runner

# Or manually
cd ui-automation && npm test
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

# Debug mode
docker-compose -f docker-compose.yml -f docker-compose.debug.yml up
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

### Complete Test Suite
```bash
# Run all tests
./test-runner.sh

# Or run specific test suite
cd ui-automation
npm test -- tests/health.test.js
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
     -d '{"name":"deepseek-r1:1.5b"}'
   
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

# Optimize containers
docker-compose -f docker-compose.yml -f docker-compose.optimized.yml up -d
```

## 🔐 Security Considerations

### Production Security
- [ ] Change default passwords
- [ ] Configure firewall rules
- [ ] Use HTTPS with proper certificates
- [ ] Implement rate limiting
- [ ] Monitor access logs
- [ ] Regular security updates

### Network Security
```bash
# Restrict container network access
docker network create --internal ai-agent-internal

# Use specific network configuration
docker-compose -f docker-compose.yml -f docker-compose.secure.yml up -d
```

## 📊 Performance Monitoring

### Metrics Collection
```bash
# Add monitoring stack
docker-compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d

# Access monitoring dashboard
# Grafana: http://localhost:3001
# Prometheus: http://localhost:9090
```

### Performance Tuning
```bash
# Optimize for production
docker-compose -f docker-compose.yml -f docker-compose.optimized.yml up -d

# Scale based on load
docker-compose up -d --scale ai-agent-svc=3
```

## 🔄 Maintenance

### Regular Updates
```bash
# Update images
docker-compose pull

# Restart with new images
docker-compose up -d

# Clean up old images
docker image prune -f
```

### Backup and Recovery
```bash
# Backup data volumes
docker run --rm -v ai-agent_milvus_data:/source:ro \
  -v $(pwd)/backup:/backup alpine tar czf /backup/milvus_backup.tar.gz /source

# Restore data
docker run --rm -v ai-agent_milvus_data:/target \
  -v $(pwd)/backup:/backup alpine tar xzf /backup/milvus_backup.tar.gz -C /target
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

## 📞 Support

### Getting Help
1. Check the troubleshooting section
2. Review service logs: `docker-compose logs <service>`
3. Run health checks: `./validate-simple.sh`
4. Create an issue with:
   - Deployment method used
   - Service logs
   - Error messages
   - System information

### Community Resources
- 📚 Documentation: Check `/docs` directory
- 🐛 Issues: Report bugs and feature requests
- 💬 Discussions: Community support and questions
- 🔄 Updates: Follow for new releases

---

## 🎯 Quick Reference

### Essential Commands
```bash
# Start everything
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# Run tests
./test-runner.sh

# Stop everything
docker-compose down
```

### Access URLs
- **Frontend**: http://localhost:3000
- **API**: http://localhost:3001
- **Health**: http://localhost:3001/health
- **Agent Status**: http://localhost:8080/health

### File Locations
- **Logs**: `docker-compose logs <service>`
- **Data**: Docker volumes (milvus_data, ollama_data)
- **Config**: Environment files and docker-compose.yml
- **Tests**: `ui-automation/tests/`

**Happy deploying!** 🚀✨