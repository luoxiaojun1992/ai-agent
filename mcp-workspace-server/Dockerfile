FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/mcp-workspace-server ./cmd/mcp-workspace-server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates su-exec

WORKDIR /app

COPY --from=builder /out/mcp-workspace-server /usr/local/bin/mcp-workspace-server
COPY mcp-workspace-server/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/mcp-workspace-server \
	&& addgroup -S workspace -g 1001 \
	&& adduser -S workspace -u 1001 -G workspace \
	&& mkdir -p /workspace-root/default \
	&& chown -R workspace:workspace /workspace-root \
	&& /usr/local/bin/mcp-workspace-server --validate-config >/dev/null

ENV PORT=8080
ENV WORKSPACE_DIR=default

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
