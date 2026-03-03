#!/usr/bin/env bash
set -euo pipefail

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-forge-ui-dev}"

echo "==> Deleting Kind cluster '${KIND_CLUSTER_NAME}'..."
kind delete cluster --name "$KIND_CLUSTER_NAME" 2>/dev/null || true

echo "==> Done."
