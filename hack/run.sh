#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

WORKSPACES="${WORKSPACES:-$HOME/workspaces}"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-forge-ui-dev}"
NGF_VERSION="${NGF_VERSION:-v2.4.2}"

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
echo "==> Building forge artifacts..."
forge build --force

echo "==> Building container images..."
docker build -t forge-ui-wasm:dev     -f containers/forge-ui-wasm/Containerfile .
docker build -t forge-frontend:dev    -f containers/forge-frontend/Containerfile .
docker build -t forge-wss-proxy:dev   -f containers/forge-wss-proxy/Containerfile .
docker build -t forge-workspace:dev   -f containers/forge-workspace/Containerfile .

# ---------------------------------------------------------------------------
# Kind cluster
# ---------------------------------------------------------------------------
echo "==> Setting up Kind cluster..."
if kind get clusters 2>/dev/null | grep -q "^${KIND_CLUSTER_NAME}$"; then
  echo "    Cluster '${KIND_CLUSTER_NAME}' exists, reusing."
else
  cat <<KINDEOF | kind create cluster --name "$KIND_CLUSTER_NAME" --config -
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 31437
        hostPort: 8080
        protocol: TCP
KINDEOF
fi

echo "==> Loading images into Kind..."
kind load docker-image forge-ui-wasm:dev     --name "$KIND_CLUSTER_NAME"
kind load docker-image forge-frontend:dev    --name "$KIND_CLUSTER_NAME"
kind load docker-image forge-wss-proxy:dev   --name "$KIND_CLUSTER_NAME"
kind load docker-image forge-workspace:dev   --name "$KIND_CLUSTER_NAME"

# ---------------------------------------------------------------------------
# NGINX Gateway Fabric
# ---------------------------------------------------------------------------
echo "==> Installing Gateway API CRDs..."
kubectl kustomize "https://github.com/nginx/nginx-gateway-fabric/config/crd/gateway-api/standard?ref=${NGF_VERSION}" \
  | kubectl apply -f -

echo "==> Installing NGINX Gateway Fabric..."
helm upgrade --install ngf oci://ghcr.io/nginx/charts/nginx-gateway-fabric \
  --create-namespace -n nginx-gateway \
  --set nginx.service.type=NodePort \
  --set-json 'nginx.service.nodePorts=[{"port":31437,"listenerPort":80}]' \
  --wait --timeout 120s

# ---------------------------------------------------------------------------
# Application deployments
# ---------------------------------------------------------------------------
PUBLIC_KEY=$(cat test/e2e/fixtures/test_user_ed25519_key.pub)

echo "==> Deploying application..."
kubectl apply -f - <<'MANIFESTS'
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: forge-ui-wasm
spec:
  replicas: 1
  selector:
    matchLabels:
      app: forge-ui-wasm
  template:
    metadata:
      labels:
        app: forge-ui-wasm
    spec:
      containers:
        - name: nginx
          image: forge-ui-wasm:dev
          imagePullPolicy: Never
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: forge-ui-wasm
spec:
  selector:
    app: forge-ui-wasm
  ports:
    - port: 8080
      targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: forge-frontend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: forge-frontend
  template:
    metadata:
      labels:
        app: forge-frontend
    spec:
      containers:
        - name: api
          image: forge-frontend:dev
          imagePullPolicy: Never
          command: ["forge-frontend"]
          args: ["-port", "8081", "-workspaces", "/workspaces"]
          ports:
            - containerPort: 8081
          volumeMounts:
            - name: workspaces
              mountPath: /workspaces
      volumes:
        - name: workspaces
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: forge-frontend
spec:
  selector:
    app: forge-frontend
  ports:
    - port: 8081
      targetPort: 8081
MANIFESTS

echo "==> Deploying forge-workspace-default..."
helm upgrade --install forge-workspace-default charts/forge-workspace \
  --set image.repository=forge-workspace \
  --set image.tag=dev \
  --set image.pullPolicy=Never \
  --set config.workspaceName=default \
  --set "ssh.authorizedKeys[0]=${PUBLIC_KEY}"

echo "==> Deploying forge-wss-proxy..."
helm upgrade --install forge-wss-proxy charts/forge-wss-proxy \
  --set image.repository=forge-wss-proxy \
  --set image.tag=dev \
  --set image.pullPolicy=Never

# ---------------------------------------------------------------------------
# Gateway + HTTPRoutes
# ---------------------------------------------------------------------------
echo "==> Configuring Gateway routing..."
kubectl apply -f - <<'GATEWAY'
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: forge-gateway
spec:
  gatewayClassName: nginx
  listeners:
    - name: http
      port: 80
      protocol: HTTP
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: forge-api
spec:
  parentRefs:
    - name: forge-gateway
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /api/
      backendRefs:
        - name: forge-frontend
          port: 8081
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: forge-ws
spec:
  parentRefs:
    - name: forge-gateway
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /ws/
      backendRefs:
        - name: forge-wss-proxy
          port: 8080
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: forge-ui
spec:
  parentRefs:
    - name: forge-gateway
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /
      backendRefs:
        - name: forge-ui-wasm
          port: 8080
GATEWAY

# ---------------------------------------------------------------------------
# Wait for rollouts
# ---------------------------------------------------------------------------
echo "==> Waiting for pods..."
kubectl rollout status deployment/forge-ui-wasm --timeout=120s
kubectl rollout status deployment/forge-frontend --timeout=120s
kubectl rollout status deployment/forge-workspace-default --timeout=120s
kubectl rollout status deployment/forge-wss-proxy --timeout=120s

# ---------------------------------------------------------------------------
# Output
# ---------------------------------------------------------------------------
PRIVATE_KEY_B64=$(base64 -w0 < test/e2e/fixtures/test_user_ed25519_key)

cat <<EOF

forge-ui running at http://localhost:8080

  Kind cluster:  ${KIND_CLUSTER_NAME}
  Gateway:       NGINX Gateway Fabric
  Routing:
    /         -> forge-ui-wasm   (static dashboard + terminal assets)
    /api/     -> forge-frontend  (REST API)
    /ws/{ws}  -> forge-wss-proxy -> forge-workspace-{ws}:22

--- FIRST TIME SETUP ---
Open http://localhost:8080, click the terminal toggle, then paste in console (F12):

(async()=>{const r=indexedDB.open("forge-terminal",2);r.onupgradeneeded=()=>{if(!r.result.objectStoreNames.contains("store"))r.result.createObjectStore("store")};r.onsuccess=()=>{const t=r.result.transaction("store","readwrite");t.objectStore("store").put(JSON.stringify([{"Name":"default","Type":"ed25519","PublicKey":"${PUBLIC_KEY}","PrivateKey":"${PRIVATE_KEY_B64}","Encrypted":false}]),"keys");t.oncomplete=()=>console.log("Key imported. Reload the page.")}})()

--- CLEANUP ---
hack/cleanup.sh

EOF
