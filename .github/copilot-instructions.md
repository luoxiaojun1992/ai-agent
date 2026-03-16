# AI Agent Instructions for This Repository

## Scope and Goal
- This repository contains a multi-service AI Agent system.
- Main stacks:
  - Go core agent library and service
  - Node.js UI backend
  - Electron desktop client
  - Docker Compose for local full-stack deployment
- Prioritize safe, minimal, and testable changes.

## Repository Map
- Go core library: `agent.go`, `skill/`, `pkg/`, `util/`, `checkpoint.go`
- Go HTTP service: `ai-agent-svc/main.go`
- Node.js API proxy backend: `ui-backend/server.js`
- Desktop client (Electron): `desktop-client/src/`
- Web frontend static app: `frontend/`
- Deployment and environment docs: `README.md`, `ARCHITECTURE.md`, `DEPLOYMENT_GUIDE.md`, `docker-compose.yml`

## Working Rules
- Keep edits focused and avoid unrelated refactors.
- Preserve existing API behavior unless explicitly requested.
- Maintain backward compatibility for existing endpoints:
  - AI service: `/health`, `/status`, `/chat`, `/skill`, `/config`, `/memory`
  - UI backend proxy: `/api/agent/*`
- For user-facing behavior changes, update related docs.

## Code Style and Conventions
- Go:
  - Keep code idiomatic and context-aware (`context.Context` propagation where relevant).
  - Return explicit errors with actionable messages.
  - Do not introduce global mutable state unless necessary.
- Node.js (`ui-backend`):
  - Keep handlers small and resilient.
  - Preserve streaming chat behavior (`text/event-stream`) when touching `/api/agent/chat`.
- Electron (`desktop-client`):
  - Respect separation between `main`, `preload`, and `renderer` responsibilities.

## Security Constraints
- Treat all external inputs as untrusted.
- Never weaken path traversal protections in filesystem skills.
- Avoid logging secrets or sensitive payloads.
- Keep CORS changes explicit and environment-driven.

## Validation Checklist After Changes
- If Go files changed:
  - Run `go test ./...` at repository root.
  - Run `go test ./...` inside `ai-agent-svc/` when service code changes.
- If `ui-backend` changed:
  - Run `npm test` in `ui-backend/`.
  - Sanity check `POST /api/agent/chat` non-stream and stream modes.
- If `desktop-client` changed:
  - Run `npm run build` or `npm run start` in `desktop-client/` as appropriate.
- If cross-service contracts changed:
  - Verify with Docker Compose integration run.

## Preferred Development Commands
- Full stack (Docker):
  - `docker-compose up --build -d`
- Go root:
  - `go test ./...`
- AI service:
  - `cd ai-agent-svc && go test ./...`
  - `cd ai-agent-svc && go run main.go`
- UI backend:
  - `cd ui-backend && npm install && npm run dev`
- Desktop client:
  - `cd desktop-client && npm install && npm run start`

## Change Guidance by Area
- Skill system (`skill/` and `skill/impl/`):
  - Keep skill names and payload contracts stable.
  - If adding a skill, include description updates in docs and service registration.
- MCP integration (`pkg/mcp/`, service bootstrap):
  - Handle initialization failures gracefully.
  - Keep timeout and retry behavior explicit.
- Milvus and Ollama clients (`pkg/milvus/`, `pkg/ollama/`):
  - Preserve model and host configurability via env vars.

## Documentation Policy
- Update `README.md` and/or deployment docs when introducing:
  - New environment variables
  - New ports, endpoints, or services
  - New build or runtime prerequisites

## Non-Goals
- Do not migrate frameworks or replace architecture without explicit request.
- Do not silently change default models, ports, or service URLs.
