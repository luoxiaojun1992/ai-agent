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
- Available ports: `3000`, `3001`, `8080`, `8443`, `11434`, `19530`, `9091`, `4001`, `4002`, `9000`, `9001`

### Start full stack

```bash
docker compose up --build -d
```

### Access

- Web UI: http://localhost:3000
- VSCode IDE (code-server): http://localhost:8443/?folder=/workspace-root/default (default password: `vscode-workspace`)
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
| POST | `/api/agent/chat` | Proxy chat (`stream: true` supports SSE; optional `images: string[]` for multimodal image input) |
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
| POST | `/chat` | Chat (`stream: true` for SSE; optional `images: string[]` for multimodal image input) |
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
10. `mcp_workspace` (enabled when `MCP_WORKSPACE_HOST` is configured)

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
MCP_WORKSPACE_HOST=http://mcp-workspace-server:8080
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

## 📉 对比 OpenClaw（开源 Agent 工程范式）与 Claude Code：当前项目劣势

基于当前仓库的实际实现（`frontend/desktop -> ui-backend -> ai-agent-svc` 多服务分层），目前短板主要集中在以下方面：

1. **工程化“开发闭环”能力偏弱**
   - 当前以通用聊天与技能调用为主，缺少面向代码任务的端到端闭环（任务拆解、仓库理解、改动建议、自动验证、PR 辅助）。
   - 与 Claude Code 一类产品相比，代码开发场景的“从需求到可合并变更”的路径不够顺滑。

2. **默认技能集较少，复杂任务编排能力有限**
   - 运行时核心默认注册 9 个技能（文件/目录、搜索、sleep，另可按配置启用 `mcp_workspace`），复杂研发任务需要更多结构化工具链支持。
   - 缺乏统一的任务编排层（例如多步骤计划执行、失败重试、长任务状态机）。

3. **可观测性与评测体系不足**
   - 现有 CI 覆盖了测试执行，但缺少面向 Agent 质量的指标体系（任务成功率、工具调用成功率、响应延迟分位、上下文命中率）。
   - 与 OpenClaw 类强调评测/基准的平台相比，持续迭代数据闭环不够强。

4. **多租户与治理能力待完善**
   - 当前架构更偏单实例工程实践，缺少面向团队/组织的权限边界、审计追踪、配额与策略中心能力。
   - 在企业落地时，治理与合规扩展成本较高。

5. **生态扩展标准化程度有限**
   - 虽有 MCP 能力接入，但缺少更标准的“技能市场化”机制（版本管理、兼容性声明、质量分级、灰度发布）。
   - 第三方能力接入后，生命周期管理成本较高。

## 🗺️ 面向当前架构的研发迭代 Roadmap

目标：不推翻现有分层架构，在保持 API 兼容（`ui-backend` 的 `/api/agent/*` 与 `ai-agent-svc` 的 `/chat`、`/skill` 等）前提下，分阶段补齐能力。

### Phase 0（0-1 个月）：稳定性与可观测性补强
- `ai-agent-svc`
  - 增加请求级 trace ID、工具调用日志结构化、关键链路延迟指标。
  - 为 `/chat`（流式/非流式）建立统一错误码与可诊断字段。
- `ui-backend`
  - 补充 SSE 代理链路指标（连接数、断开原因、超时分布）。
- 交付结果
  - 建立可量化基线：成功率、P95 延迟、技能失败 TopN 原因。

### Phase 1（1-2 个月）：面向研发任务的能力层
- `ai-agent-svc`
  - 在现有 skill 框架上增加“任务执行器”抽象（计划 -> 执行 -> 校验 -> 总结）。
  - 补齐高频研发技能（命令执行沙箱、测试触发、差异读取）并保持最小权限策略。
- `ui-backend` + `desktop-client/frontend`
  - 增加任务状态可视化（排队/执行中/失败重试/完成）。
- 交付结果
  - 支持“需求 -> 改动建议 -> 验证结果”基础闭环，提升代码任务完成率。

### Phase 2（2-4 个月）：记忆与检索质量升级
- `ai-agent-svc` + Milvus
  - 引入分层记忆策略（短期会话记忆 / 项目记忆 / 团队知识记忆）。
  - 增加检索评测集与离线回放，持续优化召回与相关性。
- MCP 能力层
  - 强化代码仓检索与文档检索路由策略，减少无效工具调用。
- 交付结果
  - 降低上下文丢失导致的重复对话与错误操作，提升多轮任务稳定性。

### Phase 3（4-6 个月）：企业化与生态化
- 平台治理
  - 增加多租户隔离、权限策略、审计日志、资源配额。
- 技能生态
  - 建立技能元数据规范（版本、依赖、风险级别、回滚策略）。
  - 支持技能灰度发布与兼容性检查。
- 交付结果
  - 从“可用的 Agent 工具”演进到“可治理、可扩展的团队生产力平台”。

### 持续性原则（贯穿各阶段）
- 保持现有 API 契约稳定，优先做向后兼容扩展。
- 所有新增能力默认可观测、可测试、可回滚。
- 安全策略前置：文件系统边界、输入校验、最小权限执行不退化。

## 📄 License

MIT
