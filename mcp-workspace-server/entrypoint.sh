#!/bin/sh
set -eu

WORKSPACE_INPUT="${WORKSPACE_DIR:-default}"
if [ -z "${WORKSPACE_INPUT}" ]; then
  WORKSPACE_INPUT="default"
fi

VALIDATED_PATH="$(/usr/local/bin/mcp-workspace-server --validate-config)"

case "$VALIDATED_PATH" in
  /workspace-root/*) ;;
  *)
    echo "workspace path validation failed: must resolve under /workspace-root, got: $VALIDATED_PATH" >&2
    exit 1
    ;;
esac

if [ "$VALIDATED_PATH" = "/workspace-root" ]; then
  echo "workspace path validation failed: /workspace-root is not allowed as workspace dir" >&2
  exit 1
fi

if [ "$VALIDATED_PATH" = "/app" ] || [ "${VALIDATED_PATH#"/app/"}" != "$VALIDATED_PATH" ]; then
  echo "workspace path validation failed: /app (server code dir) is not allowed" >&2
  exit 1
fi

mkdir -p "$VALIDATED_PATH"
chown -R workspace:workspace "$VALIDATED_PATH"
chmod 775 "$VALIDATED_PATH"

exec su-exec workspace:workspace /usr/local/bin/mcp-workspace-server
