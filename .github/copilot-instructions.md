# AI Agent Instructions for This Repository

## Scope and Goal
- This repository implements a multi-service AI Agent system.
- Core stacks:
  - Go core agent library and HTTP service (`ai-agent-svc`)
  - Node.js API proxy backend (`ui-backend`)
  - Electron desktop client (`desktop-client`)
  - Static frontend (`frontend`) and Docker Compose deployment
- Prefer focused, minimal, testable changes.

## Repository Map
- Core agent and memory logic: `agent.go`, `checkpoint.go`, `util/`
- Skill system: `skill/` and `skill/impl/`
- AI service entrypoint: `ai-agent-svc/main.go`
- UI backend entrypoint: `ui-backend/server.js`
- Desktop app: `desktop-client/src/`
- Architecture and operations docs: `README.md`, `ARCHITECTURE.md`, `DEPLOYMENT_GUIDE.md`
- CI workflow: `.github/workflows/ci.yml`

## API Compatibility Requirements
Preserve existing externally used endpoints unless explicitly requested:

- AI service (`ai-agent-svc`):
  - `GET /health`
  - `GET /status`
  - `POST /chat`
  - `POST /skill`
  - `GET /config`
  - `PUT /config`
  - `GET /memory`
  - `DELETE /memory`

- UI backend proxy (`ui-backend`):
  - `GET /health`
  - `GET /api/agent/status`
  - `POST /api/agent/chat`
  - `POST /api/agent/skill`
  - `GET /api/agent/config`
  - `PUT /api/agent/config`
  - `GET /api/agent/memory`
  - `DELETE /api/agent/memory`

## Skills and Runtime Behavior
- Skill names and payload contracts are part of runtime behavior; do not change casually.
- `ai-agent-svc` currently registers these skills by default:
  - `file_reader`, `file_writer`, `file_remover`
  - `directory_reader`, `directory_writer`, `directory_remover`
  - `mcp_web_search`, `mcp_code_repo_search`
  - `sleep`
- If you add or remove registered skills, update both `README.md` and `ARCHITECTURE.md`.

## Code Conventions
### Go
- Keep code idiomatic and explicit.
- Propagate `context.Context` for request-scoped work.
- Return actionable errors.
- Avoid introducing global mutable state unless necessary.

### Node.js (`ui-backend`)
- Keep handlers small and resilient.
- Preserve streaming (`text/event-stream`) behavior in `/api/agent/chat`.
- Keep proxy contracts stable (`message`, `stream`, `agentConfig`).

### Electron (`desktop-client`)
- Respect process boundaries (`main`, `preload`, `renderer`).
- Avoid pushing backend logic into renderer.

## Security Constraints
- Treat all incoming payloads as untrusted.
- Do not weaken filesystem path traversal protections in filesystem skills.
- Do not log secrets or sensitive user payloads.
- Keep CORS configuration explicit and environment-driven.

## Validation Checklist
Run only what is relevant to touched areas:

- If Go code changed:
  - `go test ./...` (repo root)
  - `cd ai-agent-svc && go test ./...` when service code changed
- If `ui-backend` changed:
  - `cd ui-backend && npm ci && npm test -- --runInBand`
  - Sanity check `/api/agent/chat` non-stream + stream
- If `desktop-client` changed:
  - `cd desktop-client && npm ci && npm test`
  - `npm run build` when packaging/build behavior is touched
- If contracts or end-to-end behavior changed:
  - run compose-based integration tests from `tests/` compose files

## Documentation Policy
Update docs when behavior changes:
- `README.md`: developer-facing usage and APIs
- `ARCHITECTURE.md`: component topology and data flow
- `DEPLOYMENT_GUIDE.md`: environment variables, ports, operations

## Non-Goals
- Do not migrate frameworks or redesign architecture unless requested.
- Do not silently change default service URLs, ports, or model names.
- Do not make broad refactors unrelated to the task.
