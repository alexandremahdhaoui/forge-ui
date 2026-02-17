#!/usr/bin/env bash
set -euo pipefail

CONTAINER_NAME="forge-ui"
IMAGE="forge-ui-container:latest"
PORT="${PORT:-8080}"
WORKSPACES="${WORKSPACES:-$HOME/workspaces}"

# Build the container image.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
docker build -f "$PROJECT_DIR/containers/forge-ui/Containerfile" -t "$IMAGE" "$PROJECT_DIR"

# Stop and remove any existing container.
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

# Run the container.
docker run -d \
  --name "$CONTAINER_NAME" \
  -p "$PORT:8080" \
  -v "$WORKSPACES:/workspaces:ro" \
  "$IMAGE" \
  forge-ui -port 8080 -workspaces /workspaces

echo "forge-ui running at http://localhost:$PORT"
echo "workspaces directory: $WORKSPACES"
