#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

WORKSPACES="${WORKSPACES:-$HOME/workspaces}"

# Build all artifacts.
forge build --force

# Build container images.
docker build -t forge-frontend-dev:latest -f containers/forge-frontend/Containerfile .
docker build -t forge-ui-wasm-dev:latest  -f containers/forge-ui-wasm/Containerfile .

# Stop and remove any existing containers.
docker rm -f forge-frontend forge-ui-wasm 2>/dev/null || true

# Run the WASM frontend on port 8080.
docker run -d \
  --name forge-ui-wasm \
  -p 8080:8080 \
  forge-ui-wasm-dev:latest

# Run the API server on port 8081.
docker run -d \
  --name forge-frontend \
  -p 8081:8081 \
  -v "$WORKSPACES:/workspaces:ro" \
  forge-frontend-dev:latest \
  forge-frontend -port 8081 -workspaces /workspaces

echo ""
echo "forge-ui running:"
echo "  WASM frontend: http://localhost:8080"
echo "  REST API:      http://localhost:8081"
echo "  workspaces:    $WORKSPACES"
