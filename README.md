# AI Agent System - Enhanced Version

![Version](https://img.shields.io/badge/version-2.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)

## 🚀 Overview

This is an enhanced version of the AI Agent system with comprehensive skill documentation, security improvements, modern microservices architecture.

## ✨ New Features

### 🎯 Enhanced Skills
- **Complete Documentation**: All 12 skills now have comprehensive descriptions
- **Security Improvements**: Path traversal protection and input validation
- **Better Error Handling**: Enhanced error messages and type safety

### 🌐 Modern Web Interface
- **Real-time Chat**: Interactive conversation with the AI agent
- **Skill Explorer**: Browse and execute available skills
- **Configuration Management**: View and update agent settings
- **Memory Visualization**: Monitor conversation history and context

### 🔧 Microservices Architecture
- **UI Backend Service**: Node.js Express server with CORS support
- **AI Agent Microservice**: Go-based Gin server with all skill integrations
- **Containerized Deployment**: Docker Compose orchestration
- **Health Monitoring**: Service health checks and status endpoints

## 📁 Project Structure

```
├── ai-agent-main/              # Original AI Agent core library
│   ├── skill/                  # Enhanced skill implementations
│   └── ...
├── ui-backend/                 # Node.js UI backend service
├── ai-agent-svc/              # Go AI Agent microservice
├── frontend/                   # Web interface
├── docker-compose.yml          # Complete service stack
```

## 🚀 Quick Start

### Prerequisites
- Docker and Docker Compose
- Ports 3000, 3001, 8080, 19530, 11434 available

### Start All Services
```bash
# Clone the enhanced repository
git clone <repository-url>
cd ai-agent

# Start all services with Docker Compose
docker-compose up --build -d
```

### Access the System
- **🌐 Web Interface**: http://localhost:3000
- **🔌 API Backend**: http://localhost:3001
- **🤖 AI Agent Service**: http://localhost:8080
- **📊 Health Check**: http://localhost:3001/health

## 🧪 Testing

### Test Categories
- ✅ **Health & Connectivity**: Service health and CORS
- ✅ **Agent Communication**: Chat and status APIs
- ✅ **Skill Management**: All 12 skill executions
- ✅ **Memory Management**: Context and persistence
- ✅ **Configuration**: Settings management
- ✅ **Error Handling**: Invalid inputs and edge cases
- ✅ **Integration**: End-to-end workflows

## 🔧 Available Skills

| Skill | Description |
|-------|-------------|
| `file_reader` | Read content from files |
| `file_writer` | Write content to files |
| `file_remover` | Delete files and directories |
| `directory_reader` | List directory contents |
| `directory_writer` | Create directories |
| `directory_remover` | Remove directories |
| `http` | Make HTTP requests |
| `sleep` | Pause execution |
| `milvus_insert` | Insert vectors into Milvus |
| `milvus_search` | Search vectors in Milvus |
| `ollama_embedding` | Generate text embeddings |
| `mcp` | Call MCP tools |

## 🔐 Security Features

- **Path Traversal Protection**: Secure file system operations
- **Input Validation**: Comprehensive parameter validation
- **CORS Configuration**: Cross-origin request handling
- **Error Handling**: Secure error messages without sensitive data
- **System Protection**: Prevents access to critical system directories

## 📊 Performance

- **Response Time**: < 5 seconds average
- **Concurrent Requests**: Handles 5+ simultaneous requests
- **Memory Usage**: Optimized for container deployment
- **Scalability**: Microservices architecture supports horizontal scaling

## 🔧 Configuration

### Environment Variables

#### UI Backend
```env
PORT=3001
CORS_ORIGIN=http://localhost:3000
AI_AGENT_SVC_URL=http://ai-agent-svc:8080
```

#### AI Agent Service
```env
PORT=8080
CORS_ORIGINS=*
CHAT_MODEL=qwen3:0.6b
EMBEDDING_MODEL=nomic-embed-text
OLLAMA_HOST=http://ollama:11434
MILVUS_HOST=milvus:19530
```

## 📚 API Documentation

### Core Endpoints

#### Chat with Agent
```http
POST /api/agent/chat
Content-Type: application/json

{
  "message": "Hello, how are you?",
  "agentConfig": {
    "character": "Optional custom character"
  }
}
```

#### Execute Skill
```http
POST /api/agent/skill
Content-Type: application/json

{
  "skillName": "file_writer",
  "parameters": {
    "path": "test.txt",
    "content": "Hello World"
  }
}
```

#### Get Available Skills
```http
GET /api/agent/skills
```

#### Manage Memory
```http
GET /api/agent/memory      # View memory
DELETE /api/agent/memory   # Clear memory
```

## 🐳 Docker Services

| Service | Port | Description |
|---------|------|-------------|
| `frontend` | 3000 | Web interface |
| `ui-backend` | 3001 | API backend |
| `ai-agent-svc` | 8080 | AI Agent service |
| `milvus` | 19530 | Vector database |
| `ollama` | 11434 | AI models |

## 🔄 Development Workflow

### Local Development
```bash
# Start infrastructure services
docker-compose up milvus ollama

# Run UI Backend locally
cd ui-backend
npm install
npm run dev

# Run AI Agent Service locally
cd ai-agent-svc
go run main.go
```

## 📈 Monitoring

### Health Checks
- **UI Backend**: `GET /health`
- **AI Agent SVC**: `GET /health`
- **Milvus**: Built-in health endpoint
- **Ollama**: `GET /api/tags`

### Logs
```bash
# View all service logs
docker-compose logs

# View specific service logs
docker-compose logs ai-agent-svc
docker-compose logs ui-backend
```

## 🚨 Troubleshooting

### Common Issues

1. **Services not starting**
   - Check port availability
   - Verify Docker daemon is running
   - Check Docker Compose version

2. **AI Agent not responding**
   - Verify Ollama models are loaded
   - Check Milvus connection
   - Review service logs

3. **Tests failing**
   - Ensure all services are healthy
   - Check network connectivity
   - Verify environment configuration

### Debug Commands
```bash
# Check service status
docker-compose ps

# View service logs
docker-compose logs -f <service-name>

# Test API directly
curl http://localhost:3001/api/agent/status

# Test skill execution
curl -X POST http://localhost:3001/api/agent/skill \
  -H "Content-Type: application/json" \
  -d '{"skillName":"sleep","parameters":{"duration":"1s"}}'
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add/update tests
5. Run the test suite
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Original AI Agent framework by @luoxiaojun1992
- Ollama for AI model serving
- Milvus for vector database
- Gin framework for Go web services
- Express.js for Node.js backend

---

## 📞 Support

For issues, questions, or contributions:
- Create an issue in the repository
- Check the troubleshooting section
- Review the API documentation
- Run the test suite for validation

**Enjoy your enhanced AI Agent system!** 🤖✨