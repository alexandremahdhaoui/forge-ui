# forge-ui

**A Go web dashboard and embedded terminal that visualizes Forge workspaces, build systems, test results, and git metadata -- with browser-based SSH access to workspace containers.**

> "I manage 12 repositories across 3 Go workspaces. I need to see which repos are
> dirty, which tests fail, and where meta-plans stand -- without opening each repo
> individually. forge-ui gives me that view in one browser tab. When I need to fix
> something, I open a terminal right there -- no local SSH setup required."

## What problem does forge-ui solve?

Go workspaces (`go.work`) group related repositories into a single development
unit. A workspace with 10-20 repos forces developers to inspect git status, build
artifacts, and test results one repo at a time. Across workspaces in a portfolio,
this becomes unmanageable. forge-ui aggregates filesystem, git, and Forge data
into a 4-level hierarchy: Portfolio > Workspace > Repository > Forge. It renders
this data as a WebAssembly (WASM) dashboard with background git refresh and test
heatmaps. forge-ui also provides browser-based terminal access to workspace
containers via a WebSocket-to-SSH proxy.

## Quick Start

**REST API server** (live data from your workspaces):

```sh
forge build forge-frontend
./build/bin/forge-frontend -workspaces ~/workspaces
# REST API at http://localhost:8081/api/v1/
```

**WASM demo** (dashboard-only static demo, no terminal):

```sh
forge build
# Serve build/web/ with any static file server
python3 -m http.server -d build/web 8080
# Open http://localhost:8080
```

**Kind cluster** (all services: dashboard + REST API + terminal + wss-proxy):

```sh
hack/run.sh
# Deploys to Kind cluster with Gateway API routing
# Open http://localhost:8080
```

## How does it work?

```
  Browser
  +---------------------------+     +-------------------------+
  | WASM Dashboard            |     | WASM Terminal (xterm.js)|
  | (forge-ui-wasm)           |     | (forge-terminal-wasm)   |
  +----------+----------------+     +----------+--------------+
             |  REST /api/                      |  WebSocket /ws/{ws}
             v                                  v
  +----------+----------------+     +-----------+-------------+
  | forge-frontend            |     | forge-wss-proxy         |
  | REST API (port 8081)      |     | WebSocket-to-SSH bridge |
  +----------+----------------+     +-----------+-------------+
             |                                  |  SSH
             v                                  v
  +----------+----------------+     +-----------+-------------+
  | Controllers               |     | forge-workspace         |
  | Portfolio / Workspace /   |     | SSH server + tmux       |
  | Forge / Refresher         |     +-------------------------+
  +----------+----------------+
             |
             v
  +---------------------------+
  | Adapters                  |
  | Git  Cache  Filesystem    |
  | ForgeLoader  Demo         |
  +----------+----------------+
             |
             v
  +---------------------------+
  | Types (pure data)         |
  +---------------------------+
```

forge-ui uses hexagonal architecture with 4 binaries. The WASM dashboard fetches
data from the forge-frontend REST API. Controllers hold business logic: portfolio
listing, workspace aggregation, and forge detail retrieval. The WASM terminal
connects via WebSocket through forge-wss-proxy to SSH sessions in forge-workspace
containers. See [DESIGN.md](DESIGN.md) for full details.

## Table of Contents

- [How do I configure?](#how-do-i-configure)
- [How do I build and test?](#how-do-i-build-and-test)
- [What does the UI show?](#what-does-the-ui-show)
- [FAQ](#faq)
- [Documentation](#documentation)
- [Contributing](#contributing)

## How do I configure?

### forge-frontend flags

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8081` | REST API server port |
| `-workspaces` | `$WORKSPACES` or `$HOME/workspaces` | Base directory containing workspaces |
| `-refresh-interval` | `1m` | Background git refresh interval |
| `-refresh-workers` | `3` | Number of background git refresh workers |

Set the `WORKSPACES` environment variable to override the default base directory.

### forge-wss-proxy flags

| Flag | Default | Description |
|------|---------|-------------|
| `-listen` | `:8080` | Listen address |
| `-serve-dir` | `/web` | Directory with terminal web assets |
| `-workspace-service-pattern` | `forge-workspace-{name}:22` | Backend workspace service DNS pattern |
| `-config` | (none) | Path to JSON config file (overrides flags) |

### Configuration Files

| File | Scope | Purpose |
|------|-------|---------|
| `forge-workspace.yaml` | Per workspace | Workspace description and meta-plan references |
| `forge-portfolio.yaml` | Per portfolio | Portfolio description and purpose |
| `forge.yaml` | Per repository | Build specs and test stages |
| `.forge-workspace-ignore` | Per workspace | Exclude directories from workspace discovery |

## How do I build and test?

```sh
forge build                      # Build all 10 targets
forge build forge-frontend       # Build REST API server binary
forge build forge-ui-wasm        # Build WASM dashboard binary
forge build forge-terminal-wasm  # Build WASM terminal binary
forge build forge-wss-proxy      # Build WebSocket-to-SSH proxy binary
forge test-all                   # Build + lint + unit + e2e tests
forge test-run unit              # Unit tests only
forge test-run lint              # Lint only
forge test-run e2e               # End-to-end tests (requires Kind cluster)
```

Requirements: Go 1.25.0, Node.js (for xterm.js build), Docker + Kind (for e2e
tests), and [forge](https://github.com/alexandremahdhaoui/forge). 5 direct Go
dependencies: `kin-openapi`, `oapi-codegen/runtime`, `testify`, `x/crypto`,
`sigs.k8s.io/yaml`.

## What does the UI show?

forge-ui renders 4 page types plus an embedded terminal:

1. **Portfolios** -- Lists all portfolios with aggregate stats: repository count,
   dirty repo count, and test pass/fail counts.
2. **Portfolio detail** -- Lists workspaces within a portfolio with per-workspace
   repository summaries.
3. **Workspace detail** -- Shows per-repo git metadata (branch, status, commits,
   ahead/behind, diff) and a test result heatmap (repos x stages).
4. **Forge detail** -- Shows build specs, artifacts with dependencies, test
   reports with coverage, and test environments for a single repository.
5. **Terminal** -- Browser-based SSH terminal connected to workspace containers.
   Uses xterm.js rendered by a WASM terminal emulator. SSH keys are generated
   in-browser (ed25519) and registered with the key provisioning service.
   Connects via WebSocket through forge-wss-proxy to a tmux session in the
   forge-workspace container.

The UI uses Material Design 3 styles with 4 light palettes and 1 dark palette
(Catppuccin Mocha). Theme selection persists via localStorage.

## FAQ

**What is the WASM build for?**

The WASM build serves as a demo and static deployment option. It uses hardcoded
demo data instead of live git and filesystem access.

**How does background refresh work?**

On startup, the REST API server runs a synchronous initial refresh that blocks
until complete. After that, a scheduler triggers refreshes at the configured
interval. A worker pool (default: 3 workers) processes workspace refresh jobs
concurrently. Page loads read from the in-memory cache and never block on git
operations.

**What is a portfolio?**

A portfolio is a named collection of workspaces. Directories under the base path
that contain subdirectories with `go.work` files are detected as portfolios.
Workspaces not inside a named portfolio belong to a default catch-all portfolio.

**What Go version is required?**

Go 1.25.0. See `go.mod` for the full dependency list (5 direct dependencies:
`kin-openapi`, `oapi-codegen/runtime`, `testify`, `x/crypto`,
`sigs.k8s.io/yaml`).

**Can I run it in a container?**

Yes. Run `hack/run.sh` to deploy all services to a Kind cluster with Gateway API
routing. The script builds container images, creates the cluster, and installs
Helm charts for forge-wss-proxy and forge-workspace.

**What are meta-plans?**

Meta-plans are cross-repo orchestration plans stored in `.forge-ai/meta-plan/`
directories within workspaces. forge-ui reads these plans and displays progress
tracking across repositories and workspaces.

**How does the terminal work?**

The terminal uses xterm.js rendered by a WASM terminal emulator
(forge-terminal-wasm). SSH keys (ed25519) are generated in-browser and stored in
IndexedDB. The WASM client opens a WebSocket connection through forge-wss-proxy,
which bridges it to an SSH session on the target forge-workspace container. The
container runs tmux, providing a persistent terminal session.

**What is forge-wss-proxy?**

forge-wss-proxy is a WebSocket-to-SSH bridge proxy. It runs as a Kubernetes
deployment and routes WebSocket connections to forge-workspace pods by DNS name
pattern (default: `forge-workspace-{name}:22`).

**How is theme selection stored?**

The WASM dashboard stores the selected theme in the browser's localStorage.

## Documentation

| Document | Description |
|----------|-------------|
| [DESIGN.md](DESIGN.md) | Architecture, data model, and technical design |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributor guide: build, test, and commit conventions |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, commit
conventions, and project structure.

## License

See LICENSE.
