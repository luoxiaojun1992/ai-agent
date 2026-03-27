# 🤖 AI Agent

<p align="left">
  <img alt="License" src="https://img.shields.io/badge/license-MIT-green.svg" />
  <img alt="Workflow" src="https://img.shields.io/badge/CI-CI%20Tests-blue.svg" />
</p>

A multi-service AI Agent project that combines:

- **Go** core agent library and HTTP service
- **Node.js** API proxy backend for web/desktop clients
- **Static Web** frontend and **Electron** desktop client
- **Docker Compose** local deployment and integration testing

---

## ✨ Highlights

- Clear service layering: `frontend/desktop` → `ui-backend` → `ai-agent-svc`
- Built-in skills for file/directory operations, web/code search, and utility sleep
- Runtime config APIs and memory APIs for operational control
- Compose-based API/UI/Desktop integration tests

## 🧭 Project Layout

```text
.
├── agent.go                  # Core agent library
├── ai-agent-svc/             # Go HTTP service
├── ui-backend/               # Node.js API proxy
├── frontend/                 # Static web app (served by Nginx)
├── desktop-client/           # Electron desktop app
├── skill/                    # Skill definitions and implementations
├── tests/                    # API/UI/Desktop test runners
├── docker-compose.yml        # Full local stack
├── ARCHITECTURE.md           # Architecture diagram and data flow
└── DEPLOYMENT_GUIDE.md       # Deployment and operations guide
```

## 🏗️ Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full diagram.

Typical web request path:

```text
Browser -> frontend (3000) -> ui-backend (3001) -> ai-agent-svc (8080)
                                                    |- Ollama (11434)
                                                    |- Milvus (19530)
                                                    |- MCP Web Search
                                                    `- MCP Context7
```

Desktop app calls `ui-backend` directly (same `/api/agent/*` contract).

## 🚀 Quick Start (Docker Compose)

### Prerequisites

- Docker 20+
- Docker Compose v2+
- Available ports: `3000`, `3001`, `8080`, `11434`, `19530`, `9091`, `4001`, `4002`, `9000`, `9001`

### Start full stack

```bash
docker compose up --build -d
```

### Access

- Web UI: http://localhost:3000
- UI backend health: http://localhost:3001/health
- AI service health: http://localhost:8080/health

### Stop

```bash
docker compose down
# remove volumes if needed
# docker compose down -v
```

## 🔌 Services and Endpoints

### UI Backend (`ui-backend/server.js`)

Base URL: `http://localhost:3001`

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/health` | UI backend health |
| GET | `/api/agent/status` | Proxy agent status |
| POST | `/api/agent/chat` | Proxy chat (`stream: true` supports SSE) |
| POST | `/api/agent/skill` | Proxy skill execution |
| GET | `/api/agent/config` | Proxy config read |
| PUT | `/api/agent/config` | Proxy config update |
| GET | `/api/agent/memory` | Proxy memory snapshot |
| DELETE | `/api/agent/memory` | Proxy memory clear |

### AI Agent Service (`ai-agent-svc/main.go`)

Base URL: `http://localhost:8080`

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/health` | Service health |
| GET | `/status` | Runtime status and persona |
| POST | `/chat` | Chat (`stream: true` for SSE) |
| POST | `/skill` | Execute one skill |
| GET | `/config` | Read agent config |
| PUT | `/config` | Update runtime config |
| GET | `/memory` | Read in-memory contexts |
| DELETE | `/memory` | Reset memory |

## 🧩 Registered Skills (Current)

`ai-agent-svc` currently registers the following skills in `ai-agent-svc/main.go`:

1. `file_reader`
2. `file_writer`
3. `file_remover`
4. `directory_reader`
5. `directory_writer`
6. `directory_remover`
7. `mcp_web_search`
8. `mcp_code_repo_search`
9. `sleep`

> Note: other skill implementations exist under `skill/impl/`, but the list above is the runtime-registered set by default.

## ⚙️ Configuration

Configuration is loaded from environment variables (and `.env` files when present).
Defaults below follow the current runtime values in `ai-agent-svc/main.go` and `ui-backend/server.js`.

### UI Backend (`ui-backend/.env`)

```env
PORT=3001
CORS_ORIGIN=http://localhost:3000
AI_AGENT_SVC_URL=http://ai-agent-svc:8080
```

### AI Agent Service (`ai-agent-svc/.env`)

```env
PORT=8080
CORS_ORIGINS=*
CHAT_MODEL=qwen3:4b
EMBEDDING_MODEL=nomic-embed-text
SUPERVISOR_MODEL=qwen3:4b
OLLAMA_HOST=http://ollama:11434
MILVUS_HOST=milvus:19530
MILVUS_COLLECTION=ai_agent_memory
MCP_WEB_SEARCH_HOST=http://mcp-web-search:3000
MCP_CONTEXT_7_CLIENT_HOST=http://mcp-context7:8080
AGENT_MODE=loop
```

Key model variables:

- `CHAT_MODEL`: primary model for normal chat generation
- `SUPERVISOR_MODEL`: model used by supervisor/review logic when enabled
- `EMBEDDING_MODEL`: model used for embedding generation in memory/vector workflows

## 🛠️ Development

### Go tests (root)

```bash
go test ./...
```

### AI service tests

```bash
cd ai-agent-svc
go test ./...
```

### UI backend tests

```bash
cd ui-backend
npm ci
npm test -- --runInBand
```

### Desktop client tests

```bash
cd desktop-client
npm ci
npm test
```

## 🧪 Integration / E2E Testing (Docker Compose)

### API compose tests

```bash
docker compose -f docker-compose.api-test.yml up --build --abort-on-container-exit --exit-code-from api-test-runner
docker compose -f docker-compose.api-test.yml down -v
```

### Frontend UI tests (Playwright)

```bash
docker compose -f docker-compose.ui-test.yml up --build --abort-on-container-exit --exit-code-from ui-test-runner
docker compose -f docker-compose.ui-test.yml down -v
```

### Desktop UI tests (Playwright)

```bash
docker compose -f docker-compose.desktop-test.yml up --build --abort-on-container-exit --exit-code-from desktop-test-runner
docker compose -f docker-compose.desktop-test.yml down -v
```

## 🔁 CI

Main workflow: `.github/workflows/ci.yml` (`CI Tests`)

Pipeline includes:

- `ui-backend-tests`
- `api-compose-tests`
- `ui-compose-tests`
- `desktop-compose-tests`
- `go-root-tests` (with coverage gate)

## 🔒 Security Notes

- Filesystem skills resolve paths under `RootDir` and reject traversal outside root
- CORS is configured via environment variables
- Treat all external API payloads as untrusted input

## 📄 License

MIT
