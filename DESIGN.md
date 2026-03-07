# forge-ui Design

forge-ui uses hexagonal architecture with four drivers (REST API, WASM dashboard, WASM terminal, TUI) to deliver a Go workspace dashboard with browser-based and terminal-based access to Kubernetes workspace containers.

## Problem Statement

Go workspaces (`go.work`) group related repositories into a single dependency graph. A team managing 5-20 repositories per workspace tracks git status, build artifacts, and test results across each repo individually. Portfolios add another layer: named collections of workspaces that span entire product areas. Inspecting each repository one by one does not scale. forge-ui aggregates filesystem, git, and forge data into a 4-level hierarchy -- Portfolio > Workspace > Repository > Forge -- and renders it via a REST API consumed by a WASM dashboard client. forge-ui also provides browser-based terminal access to workspace containers via a WebSocket-to-SSH proxy.

## Tenets

These tenets are prioritized. When two tenets conflict, the higher-ranked tenet wins.

1. **Read-only.** forge-ui reads data. It never modifies repositories, workspaces, or build artifacts.
2. **Hexagonal boundaries.** Drivers, controllers, and adapters communicate through interfaces. No layer imports from a layer above it.
3. **Dual-driver parity.** REST API and WASM dashboard drivers serve the same 4 page types with the same controllers. The terminal subsystem operates independently.
4. **Minimal dependencies.** 5 direct Go dependencies: `kin-openapi`, `oapi-codegen/runtime`, `testify`, `x/crypto`, `sigs.k8s.io/yaml`. xterm.js is the single JavaScript library, bundled via rollup. No CSS frameworks beyond hand-crafted Material Design 3 styles.
5. **Zero-config default.** Without flags, forge-frontend scans `$HOME/workspaces` and serves on port 8081.
6. **Background freshness.** The REST API driver refreshes git data in the background. Page loads never block on git operations after startup.
7. **Kubernetes-native deployment.** The full stack deploys to a Kubernetes cluster via Helm charts and Gateway API routing.

## Requirements

From the user's perspective:

- View all portfolios with aggregate stats: repo count, dirty count, test pass/fail
- View all workspaces within a portfolio with per-workspace repo summaries
- View workspace detail with per-repo git metadata: branch, status, commits, ahead/behind, diff
- View test result heatmap (repos x stages) per workspace
- View forge detail per repo: build specs, artifacts with dependencies, test reports with coverage, test environments
- Toggle dark/light theme; select among 4 light palettes
- Run as REST API server with live data or as WASM client with demo data
- Track meta-plan progress across repos and workspaces
- Open a browser-based terminal connected to a workspace container's tmux session
- Generate SSH keys in-browser and register them with the key provisioning service
- Deploy all services to a Kubernetes cluster via Helm charts

## Out of Scope

- Write operations: forge-ui does not create, modify, or delete files
- User authentication or authorization
- Real-time dashboard updates via WebSocket (dashboard uses polling; WebSocket is used only for terminal I/O)
- CI/CD integration (reads forge artifacts, does not trigger builds)
- Multi-user state (no database, in-memory cache only)

## Success Criteria

- REST API server responds to all 4 page data endpoints (portfolios, portfolio, workspace, forge) in under 50ms after initial refresh
- WASM client loads and renders the initial page in under 2 seconds
- Background refresh completes for N workspaces with configurable interval (default 1 minute) and worker count (default 3)
- `forge test-all` passes: lint + unit + e2e stages
- Terminal connects to workspace container within 3 seconds of initiating connection
- e2e tests validate terminal connectivity in a Kind cluster
- Zero external runtime dependencies beyond git (REST API mode) or a browser (WASM mode); Kubernetes cluster required for terminal and full deployment

## Proposed Design

### Architecture Overview

```
+-------------------------------------------------------------------------+
|                            forge-ui                                     |
|                                                                         |
|  DASHBOARD SUBSYSTEM                                                    |
|                                                                         |
|  +---------------------------+    +-----------------------------+       |
|  | REST API Driver           |    | WASM Dashboard Driver       |       |
|  | (GOOS=linux)              |    | (GOOS=js, GOARCH=wasm)      |       |
|  |                           |    |                             |       |
|  | cmd/forge-frontend/       |    | cmd/forge-ui-wasm/          |       |
|  | driver/rest/handler.go    |    | driver/wasm/driver.go       |       |
|  | driver/rest/zz_generated  |    | driver/wasm/router.go       |       |
|  |   .oapi-codegen.go        |    | driver/wasm/dom.go          |       |
|  |                           |    |                             |       |
|  | OpenAPI strict server     |    | syscall/js hash routing     |       |
|  | JSON responses            |    | Client-side HTML rendering  |       |
|  | api/forge-ui.v1.yaml      |    | Embedded templates (embed)  |       |
|  +------------+--------------+    +-------------+---------------+       |
|               |                                 |                       |
|               v                                 v                       |
|  CONTROLLER LAYER                                                       |
|  +---------------------+  +---------------------+  +---------------+   |
|  | PortfolioService     |  | WorkspaceService    |  | ForgeService  |   |
|  | ListPortfolios()     |  | GetWorkspace()      |  | GetForge()    |   |
|  | GetPortfolio()       |  |                     |  |               |   |
|  +----------+----------+  +----------+----------+  +-------+-------+   |
|             |                        |                      |           |
|  +----------+------------------------+----------------------+-------+   |
|  | PageRenderer (WASM only)          Refresher (REST API only)      |   |
|  | Render(route) -> HTML string      Start() -> initial refresh     |   |
|  +------------------------------------------------------------------+   |
|               |                                 |                       |
|               v                                 v                       |
|  ADAPTER LAYER                                                          |
|  +--------------------+  +------------------+  +-------------------+    |
|  | PortfolioDiscovery |  | WorkspaceDisc.   |  | GitInfo           |    |
|  | Cache              |  | ForgeLoader      |  | DataSource        |    |
|  | MetaPlanLoader     |  | RepoPlanLoader   |  | WsConfigLoader    |    |
|  | PortfolioConfig    |  | DemoDataSource   |  |                   |    |
|  +--------------------+  +------------------+  +-------------------+    |
|               |                                                         |
|               v                                                         |
|  DOMAIN MODEL                                                           |
|  +----------------------------------------------------------------+     |
|  | types/types.go (40 types, 364 lines)                           |     |
|  | PortfolioSummary  WorkspaceSummary  RepoOverview  RepoSummary  |     |
|  | PortfolioPageData WorkspacePageData ForgePageData ForgeSpec    |     |
|  | TestReport  TestEnv  Artifact  MetaPlan  WsConfig  Cache...   |     |
|  +----------------------------------------------------------------+     |
|                                                                         |
|  TERMINAL SUBSYSTEM                                                     |
|                                                                         |
|  +-----------------------------+                                        |
|  | WASM Terminal Driver        |                                        |
|  | (GOOS=js, GOARCH=wasm)     |                                        |
|  |                             |                                        |
|  | cmd/forge-terminal-wasm/    |                                        |
|  | terminal/driver/wasm/       |                                        |
|  | Exposes forgeTerminal.start |                                        |
|  | and forgeTerminal.stop to JS|                                        |
|  +-------------+---------------+                                        |
|                |                                                        |
|                v                                                        |
|  +-----------------------------+                                        |
|  | SessionController           |                                        |
|  | terminal/controller/        |                                        |
|  | Start(cfg, termIO) -> error |                                        |
|  | Stop() -> error             |                                        |
|  +-------------+---------------+                                        |
|                |                                                        |
|                v                                                        |
|  +-------------+--+  +-----------+  +-------------+  +-----------+      |
|  | TerminalIO     |  | SSHClient |  | KeyRegistrar|  | KeyStore  |      |
|  | (xterm.js      |  | (WebSocket|  | (HTTP POST  |  | (IndexedDB|      |
|  |  bridge)       |  |  + SSH)   |  |  to proxy)  |  |  storage) |      |
|  +----------------+  +-----------+  +-------------+  +-----------+      |
|                |                                                        |
|                v                                                        |
|  +-----------------------------+                                        |
|  | terminal/types/ (6 types)   |                                        |
|  | TerminalConfig  SSHKey      |                                        |
|  | TerminalEndpoint KnownHost  |                                        |
|  | TerminalSession             |                                        |
|  | SSHSessionConfig            |                                        |
|  +-----------------------------+                                        |
|                                                                         |
|  +-----------------------------+                                        |
|  | terminal/keygen/            |                                        |
|  | Generate() -> SSHKey        |                                        |
|  | (ed25519 key pair)          |                                        |
|  +-----------------------------+                                        |
|                                                                         |
|  WSS-PROXY SUBSYSTEM                                                    |
|                                                                         |
|  +-----------------------------+                                        |
|  | forge-wss-proxy             |                                        |
|  | cmd/forge-wss-proxy/        |                                        |
|  | internal/wssproxy/          |                                        |
|  |                             |                                        |
|  | WebSocket upgrade at        |                                        |
|  |   /ws/{workspace}           |                                        |
|  | TCP bridge to               |                                        |
|  |   forge-workspace-{ws}:22   |                                        |
|  | Key registration endpoint   |                                        |
|  | In-memory key store         |                                        |
|  +-----------------------------+                                        |
|                                                                         |
+-------------------------------------------------------------------------+

KUBERNETES DEPLOYMENT
+---------------------------------------------------------------------+
| Browser :8080 (Gateway API)                                         |
|   /         --> forge-ui-wasm (nginx: dashboard + terminal assets)  |
|   /api/     --> forge-frontend:8081 (REST API)                      |
|   /ws/{ws}  --> forge-wss-proxy:8080 --> forge-workspace-{ws}:22    |
+---------------------------------------------------------------------+
```

The architecture has 3 subsystems. The dashboard subsystem uses 4 layers: types hold pure data structs (40 types in 364 lines), adapters define outbound interfaces, controllers hold business logic (3 domain services, a page renderer, and a refresher), and drivers (REST API and WASM dashboard) handle inbound requests. The terminal subsystem follows the same hexagonal pattern with its own types, adapters, controller, and WASM driver. The wss-proxy subsystem bridges WebSocket connections to SSH backends.

### Dependency Direction

```
Driver --> Controller --> Adapter --> Types
  |             |            |          ^
  |             |            |          |
  |             |            +----------+
  |             +-------------------------+
  +-----------------------------------------+

All layers depend inward on types.
Adapters define interfaces; controllers consume them.
Drivers consume controller interfaces.
```

All layers depend inward on types. No layer imports from a layer above it.

### Build Targets

```
GOOS=linux GOARCH=amd64                          GOOS=js GOARCH=wasm
     |           |           |                       |            |
     v           v           v                       v            v
+-----------+ +-------------+ +------------+  +------------+ +----------------+
| cmd/      | | cmd/        | | cmd/       |  | cmd/       | | cmd/           |
| forge-    | | forge-wss-  | | forge-ui-  |  | forge-ui-  | | forge-terminal-|
| frontend/ | | proxy/      | | tui/       |  | wasm/      | | wasm/          |
|           | |             | |            |  |            | |                |
| REST API  | | WebSocket   | | TUI        |  | Dashboard  | | Terminal       |
| server    | | to SSH      | | dashboard  |  | WASM       | | WASM           |
| :8081     | | proxy :8080 | | (bubbletea)|  | client     | | emulator       |
+-----------+ +-------------+ +------------+  +------------+ +----------------+
```

forge.yaml defines 11 build targets:

| Target | Output | Engine |
|--------|--------|--------|
| `forge-frontend` | `build/bin/forge-frontend` | `go://go-build` |
| `forge-ui-wasm` | `build/web/forge-ui.wasm` | `alias://wasm-build` (GOOS=js GOARCH=wasm) |
| `forge-terminal-wasm` | `build/web/terminal.wasm` | `alias://wasm-build` (GOOS=js GOARCH=wasm) |
| `forge-terminal-xterm` | `build/web/` | `alias://web-assets` (npm + rollup) |
| `forge-ui-web-assets` | `build/web/` | `alias://web-assets` (cp web/ to build/) |
| `forge-ui-tui` | `build/bin/forge-ui-tui` | `go://go-build` |
| `forge-wss-proxy` | `build/bin/forge-wss-proxy` | `go://go-build` |
| `forge-wss-proxy-image` | container image | `go://container-build` |
| `forge-workspace-image` | container image | `go://container-build` |
| `generate-mocks` | `internal/util/mocks/` | `alias://generate-mocks` (mockery) |
| `generate-rest-api` | `internal/driver/rest/` | `go://go-gen-openapi` |

### REST API Request Flow

```
WASM Dashboard (browser)
  |
  | GET /api/v1/portfolios/infrastructure/workspaces/platform?sort=time
  v
net/http ServeMux + oapi-codegen strict server
  |
  | route match: GET /api/v1/portfolios/{portfolio}/workspaces/{workspace}
  v
rest.APIHandler.GetWorkspace(ctx, request)
  |
  | 1. Extract typed params: portfolio="infrastructure", workspace="platform"
  | 2. Read sort mode from query param (default: "time")
  v
controller.WorkspaceService.GetWorkspace(baseDir, "infrastructure", "platform", "time")
  |
  | 1. Resolve wsBaseDir = baseDir + "/infrastructure"
  | 2. Call workspaceDisc.Get(wsBaseDir, "platform")
  |       --> scan filesystem for .git subdirs
  |       --> return WorkspacePageData with repo list
  |
  | 3. For each repo, read from cache:
  |       cache.GetRepoSummary("infrastructure/platform", repoName)
  |       --> merge branch, status, logs, diff, ahead/behind
  |
  | 4. Sort repos by LastCommitTime (descending)
  |
  | 5. For each repo with HasForge=true:
  |       forgeLoader.Load(repoPath)
  |       --> read forge.yaml + .forge/artifact-store.yaml
  |       --> build RepoForgeStats (heatmap row)
  |       --> accumulate test stats
  |
  | 6. Return enriched WorkspacePageData
  v
GetWorkspace200JSONResponse(data)
  |
  | Serialize WorkspacePageData to JSON
  | Write Content-Type: application/json
  v
WASM Dashboard renders HTML from JSON data
```

The REST API driver follows a stateless request-response model. The WASM dashboard sends a GET request. The oapi-codegen strict server deserializes path and query parameters into typed request objects. The handler calls the appropriate controller service and returns a typed JSON response. Controllers read from the in-memory cache (filled by the background refresher) and the filesystem (for forge.yaml data).

### Background Refresh Flow

```
cmd/forge-frontend/main.go
  |
  | controller.NewRefresher(cache, gitInfo, portfolioDisc, wsDisc, cfg)
  | r.Start()
  v
Refresher.Start()
  |
  | PHASE 1: Synchronous initial refresh (blocks startup)
  |
  | refreshAll()
  |   |
  |   | portfolioDisc.List(baseDir)
  |   |   --> discover all portfolios + their workspaces
  |   |
  |   | For each portfolio:
  |   |   For each workspace:
  |   |     refreshWorkspace(item)
  |   |       |
  |   |       | workspaceDisc.Get(wsBase, wsName)
  |   |       |   --> list repos (dirs with .git)
  |   |       |
  |   |       | For each repo:
  |   |       |   gitInfo.RepoInfo(repoPath)
  |   |       |     --> git fetch origin
  |   |       |     --> git branch --show-current
  |   |       |     --> git status --porcelain
  |   |       |     --> git log --oneline -10
  |   |       |     --> git diff --stat
  |   |       |     --> git rev-list --left-right --count @{upstream}...HEAD
  |   |       |     --> git log -1 --format=%cI
  |   |       |
  |   |       | Build RepoSummary + RepoOverview maps
  |   |       | cache.SetWorkspace("portfolio/workspace", data)
  |   |       v
  |   v
  | log: "refresher: initial refresh complete"
  |
  | PHASE 2: Background goroutines
  |
  | Spawn N worker goroutines (default: 3)
  | Spawn 1 scheduler goroutine
  v

+-----------+         +--------+         +-----------+
| Scheduler |         | Queue  |         | Worker(s) |
|           |  items  | chan   |  items  |           |
| ticker    +-------->+ 100    +-------->+ refresh   |
| (1 min)   |         | buffer |         | Workspace |
+-----------+         +--------+         +-----------+
      |                                        |
      | Every tick:                             | For each item:
      | 1. portfolioDisc.List(baseDir)          | 1. workspaceDisc.Get()
      | 2. For each portfolio+workspace:        | 2. gitInfo.RepoInfo() per repo
      |    send refreshItem to queue            | 3. cache.SetWorkspace()
      |                                        |
      | On done signal: return                  | On done signal: return
```

The refresh model has 2 phases. Phase 1 runs synchronously at startup: it blocks until all workspaces are refreshed, so the first page load always has data. Phase 2 spawns N worker goroutines (configurable, default 3) reading from a buffered channel (capacity 100). A scheduler goroutine ticks at the configured interval (default 1 minute) and enqueues all discovered workspaces for refresh.

### WASM Initialization and Navigation

**Initialization:**

```
Browser loads index.html
  |
  | <script src="wasm_exec.js">
  | WebAssembly.instantiateStreaming(fetch("main.wasm"))
  v
Go main() (GOOS=js GOARCH=wasm)
  |
  | adapter.NewDemoDataSource()
  |   --> static portfolios, workspaces, forge data
  |   --> hash-prefixed links (#/portfolios/...)
  |
  | controller.NewPageRenderer(demoDataSource)
  |   --> parse embedded templates (go:embed internal/controller/templates/*.html)
  |
  | wasm.New(renderer)
  | driver.Init()
  |   |
  |   | 1. content = document.getElementById("content")
  |   |
  |   | 2. theme = localStorage.getItem("forge-ui-theme") || "light"
  |   |    if dark: document.documentElement.setAttribute("data-theme", "dark")
  |   |
  |   | 3. Register click listener on #theme-btn
  |   |      --> toggleTheme()
  |   |
  |   | 4. router = NewRouter(navigate callback)
  |   |    router.Start()
  |   |      |
  |   |      | window.addEventListener("hashchange", callback)
  |   |      | if hash == "" or "#": set hash to "#/portfolios"
  |   |      | trigger initial navigate(getHash())
  |   v
  | select {}   (block forever, keep WASM alive)
```

**Navigation Flow (hash change):**

```
User clicks link: <a href="#/portfolios/infrastructure">
  |
  | Browser updates location.hash
  | Fires "hashchange" event
  v
Router.cb (js.FuncOf)
  |
  | hash = window.location.hash
  v
Driver.navigate(hash)
  |
  | 1. Strip leading "#"
  |    route = "/portfolios/infrastructure"
  |
  | 2. Default empty route to "/portfolios"
  |
  | 3. Append theme as query param
  |    route = "/portfolios/infrastructure?theme=dark"
  |
  | 4. renderer.Render(route)
  |      |
  |      | parseInput(route)
  |      |   --> input{Route: "/portfolios/infrastructure", Sort: "time", Theme: "dark"}
  |      |
  |      | executeRoute(input)
  |      |   |
  |      |   | splitRoute("/portfolios/infrastructure")
  |      |   |   --> ["portfolios", "infrastructure"]
  |      |   |
  |      |   | Match: len==2, parts[0]=="portfolios"
  |      |   |   --> renderPortfolio("infrastructure", input)
  |      |   |
  |      |   | ds.GetPortfolio("infrastructure", "time")
  |      |   |   --> DemoDataSource returns static PortfolioPageData
  |      |   |
  |      |   | data.DarkMode = true
  |      |   |
  |      |   | templates.ExecuteTemplate(buf, "portfolio", data)
  |      |   |   --> render HTML to buffer
  |      |   |
  |      |   | return buf.String()
  |      v
  |
  | 5. content.innerHTML = html
  v
Browser re-renders #content div
```

The WASM driver loads in the browser, creates a DemoDataSource with static data, and routes via hash-change events. Navigation renders HTML fragments into the `#content` div. No server round-trips occur after the initial WASM binary load.

### Terminal Architecture

```
Browser (xterm.js)    forge-terminal-wasm    forge-wss-proxy    forge-workspace
      |                      |                     |                  |
      |--keystroke---------->|                     |                  |
      |                      |--WebSocket frame--->|                  |
      |                      |                     |--TCP (raw)------>|
      |                      |                     |                  | sshd -> tmux
      |                      |                     |<-TCP (raw)-------|
      |                      |<-WebSocket frame----|                  |
      |<-terminal output-----|                     |                  |
      |                                                               |
      |<============= SSH protocol (end-to-end) =====================>|
```

xterm.js captures keystrokes in the browser. The WASM terminal driver reads and writes via the TerminalIO adapter, which wraps the xterm.js terminal object. The SSHClient adapter opens a WebSocket to forge-wss-proxy and runs the SSH protocol over it (the WebSocket connection implements `net.Conn`). The wss-proxy bridges WebSocket frames to raw TCP transparently -- it does not speak SSH. The forge-workspace container runs OpenSSH with tmux. The SSH protocol runs end-to-end between the WASM client and the workspace sshd.

**SSH Key Provisioning Flow:**

```
forge-terminal-wasm (browser)              forge-wss-proxy           forge-workspace
      |                                          |                        |
      | 1. keygen.Generate()                     |                        |
      |    -> ed25519 key pair                   |                        |
      |                                          |                        |
      | 2. keyStore.SaveKey(key)                 |                        |
      |    -> IndexedDB (private key stays       |                        |
      |       in browser)                        |                        |
      |                                          |                        |
      | 3. POST /ws/{ws}/register-key            |                        |
      |    (public key only)                     |                        |
      |----------------------------------------->|                        |
      |                                          | 4. store in memory     |
      |                                          |    keystore.Add(ws,key)|
      |                                          |                        |
      |                                          |     5. sshd auth       |
      |                                          |<-----------------------|
      |                                          | GET /internal/         |
      |                                          |   authorized-keys/{ws} |
      |                                          |----------------------->|
      |                                          |  (AuthorizedKeysCommand|
      |                                          |   get-authorized-      |
      |                                          |   keys.sh)             |
```

The browser generates an ed25519 key pair using the `keygen` package. The private key remains in IndexedDB (origin-scoped, never leaves the browser). The public key is sent to the wss-proxy key registration endpoint. When sshd authenticates a connection, it calls `AuthorizedKeysCommand` (`get-authorized-keys.sh`), which queries the wss-proxy for registered keys.

### WebSocket-to-SSH Proxy (forge-wss-proxy)

The wss-proxy receives a WebSocket upgrade at `/ws/{workspace}`, resolves the backend address via the workspace service pattern (`forge-workspace-{name}:22`), establishes a TCP connection, and pipes data bidirectionally. The SSH protocol runs end-to-end through this tunnel between the WASM client and the workspace sshd. The proxy does not interpret SSH frames.

The key store holds an in-memory map of workspace names to authorized public keys. Two endpoints manage keys:

- `POST /ws/{workspace}/register-key` -- register a public key for a workspace
- `GET /internal/authorized-keys/{workspace}` -- list registered keys (called by sshd `AuthorizedKeysCommand`)

The `get-authorized-keys.sh` script in the forge-workspace container reads the workspace name from `/etc/forge-workspace-name` and queries the wss-proxy endpoint via `wget`.

### Kubernetes Deployment

```
Browser :8080 (NGINX Gateway Fabric)
  |
  +-- /         --> forge-ui-wasm (nginx: dashboard + terminal assets)
  +-- /api/     --> forge-frontend:8081 (REST API)
  +-- /ws/{ws}  --> forge-wss-proxy:8080 --> forge-workspace-{ws}:22
```

The full stack deploys to a Kubernetes cluster. 4 containers run as separate deployments:

- **forge-frontend** -- REST API server (Alpine + Go binary, port 8081)
- **forge-ui-wasm** -- nginx serving static WASM dashboard and terminal assets (port 8080)
- **forge-wss-proxy** -- WebSocket-to-SSH proxy + key store (port 8080)
- **forge-workspace** -- OpenSSH + tmux workspace container (port 22)

2 Helm charts (`charts/forge-workspace`, `charts/forge-wss-proxy`) configure sshd, authorized keys, and service discovery. NGINX Gateway Fabric routes requests by path prefix to the correct backend service.

### Driver Comparison

```
Aspect            REST API Driver             WASM Dashboard Driver       WASM Terminal Driver        TUI Driver
------            ---------------             ---------------------       --------------------        ----------
Entry point       cmd/forge-frontend/         cmd/forge-ui-wasm/          cmd/forge-terminal-wasm/    cmd/forge-ui-tui/
Build tag         GOOS=linux                  GOOS=js GOARCH=wasm         GOOS=js GOARCH=wasm         GOOS=linux
Routing           net/http + oapi-codegen     hashchange event listener   JS global (forgeTerminal)   bubbletea key/mouse
URL format        /api/v1/portfolios/infra    #/portfolios/infra          N/A (programmatic)          N/A (interactive)
Data source       Live filesystem + git       Static demo data            WebSocket + SSH             REST API client
Cache             In-memory (sync.RWMutex)    None (static)               IndexedDB (SSH keys)        None (fetches on nav)
Background        Refresher (N workers)       None                        Persistent WebSocket        None
Templates         None (JSON responses)       Embedded (go:embed)         None                        None (bubbletea views)
Rendering         JSON (application/json)     Client-side (innerHTML)     xterm.js                    bubbletea (terminal)
Output            JSON response               HTML fragment               Terminal I/O stream         Terminal UI
```

## Technical Design

### Data Model

**Portfolio Hierarchy:**

```
Filesystem Layout                      Domain Model
==================                     ============

$WORKSPACES/                           PortfolioSummary
  |                                      Name: "default"
  +-- forge-ai/        (go.work)         IsDefault: true
  |     +-- forge/     (.git)            Workspaces[]
  |     +-- forge-ui/  (.git)              |
  |                                        +-- WorkspaceSummary
  +-- infrastructure/  (named portfolio)        Name: "forge-ai"
  |     +-- platform/  (go.work)                RepoCount: 2
  |     |    +-- svc-a/ (.git)                  Repos[]
  |     |    +-- svc-b/ (.git)                    |
  |     +-- networking/ (go.work)                 +-- RepoOverview
  |          +-- proxy/ (.git)                         Name: "forge"
  |                                                    Branch: "main"
  +-- .forge-workspace-ignore                          IsDirty: false
                                                       HasForge: true
```

**Detection Rules:**

```
Directory has go.work?
    |
    YES --> It is a Workspace
    |       Contains repos (subdirs with .git/)
    |
    NO  --> Does it contain subdirs with go.work?
                |
                YES --> It is a Named Portfolio
                |       Contains workspaces
                |
                NO  --> Ignored
```

```
Loose workspaces (go.work at $WORKSPACES level)
    --> grouped into "default" portfolio (IsDefault: true)

Named portfolios (subdirs containing workspaces)
    --> each becomes a PortfolioSummary (IsDefault: false)
```

**Core Types and Relationships** (40 types in `internal/types/types.go`, 364 lines):

```
PortfoliosPageData
  |-- Portfolios: []PortfolioSummary
  |-- Stats: PortfoliosStats
  |
  +-- PortfolioSummary
        |-- Name          string        ("infrastructure" or "default")
        |-- Path          string        (absolute path)
        |-- IsDefault     bool
        |-- Description   string        (from forge-portfolio.yaml)
        |-- Workspaces    []WorkspaceSummary
        |-- Stats         WorkspacesStats
        |
        +-- WorkspaceSummary
              |-- Name        string    ("platform")
              |-- Path        string
              |-- Description string    (from forge-workspace.yaml)
              |-- RepoCount   int
              |-- Repos       []RepoOverview     (lightweight, for listing)
              |-- AllStages   []string            (heatmap columns)
              |-- RepoForge   []RepoForgeStats   (heatmap rows)
              |-- Progress    WorkspaceProgress
              |-- MetaPlans   []MetaPlan
              |
              +-- RepoOverview
                    |-- Name           string
                    |-- Branch         string
                    |-- IsDirty        bool
                    |-- Ahead/Behind   int
                    |-- HasUpstream    bool
                    |-- HasForge       bool
                    |-- LastCommitTime time.Time
```

**Workspace Detail Page:**

```
WorkspacePageData
  |-- Name            string
  |-- PortfolioName   string
  |-- Path            string
  |-- Description     string             (from forge-workspace.yaml)
  |-- Repos           []RepoSummary      (full detail per repo)
  |-- Stats           WorkspaceStats
  |-- AllStages       []string
  |-- RepoForge       []RepoForgeStats
  |-- RepoRoles       map[string]string   (repo name -> role description)
  |-- MetaPlans       []MetaPlan
  |-- RepoPlanSummaries []RepoPlanSummary
  |
  +-- RepoSummary (extends RepoOverview)
        |-- Name          string
        |-- Branch        string
        |-- IsDirty       bool
        |-- StatusFiles   []StatusEntry    {Code, FilePath}
        |-- DiffStat      string
        |-- RecentLogs    []LogEntry       {Hash, Message}
        |-- HasForge      bool
        |-- Ahead/Behind  int
        |-- HasUpstream   bool
        |-- LastCommitTime time.Time
```

**Forge (Repo Detail) Page:**

```
ForgePageData
  |-- WorkspaceName    string
  |-- RepoName         string
  |-- PortfolioName    string
  |-- Spec             ForgeSpec
  |     |-- Name       string
  |     |-- Build[]    {Name, Src, Dest, Engine}
  |     |-- Test[]     {Name, Testenv, Runner}
  |
  |-- Artifacts[]      Artifact
  |     |-- Name, Type ("binary"/"container")
  |     |-- Location, Timestamp, Version
  |     |-- Dependencies[] {Type, FilePath, ExternalPackage, Semver}
  |
  |-- TestReports[]    TestReport
  |     |-- ID, Stage, Status ("passed"/"failed")
  |     |-- StartTime, Duration
  |     |-- Stats      {Total, Passed, Failed, Skipped}
  |     |-- Coverage   {Enabled, Percentage, FilePath}
  |
  |-- TestEnvs[]       TestEnv
  |     |-- ID, Name, Status
  |     |-- CreatedAt, UpdatedAt
  |     |-- ManagedResources[]
  |
  |-- RepoPlans[]      RepoPlan
  |     |-- Name, TasksTotal, TasksDone
  |
  |-- Stats            ForgeStats
  |-- StageStatusMap   map[string]string
```

**Cache Data Model:**

```
inMemoryCache
  |
  +-- mu: sync.RWMutex
  |
  +-- workspaces: map[string]CacheWorkspaceData
        |
        key = "portfolioName/workspaceName"
        |
        +-- CacheWorkspaceData
              |-- Summaries: map[string]RepoSummary   (keyed by repo name)
              |-- Overviews: map[string]RepoOverview   (keyed by repo name)
              |-- UpdatedAt: time.Time
```

**Test Heatmap Data:**

```
WorkspaceSummary                  Rendered as heatmap grid
  |                               =========================
  |-- AllStages: [lint, unit, e2e]     lint   unit   e2e
  |                               forge   [OK]   [OK]   [OK]
  |-- RepoForge[]                 forge-ui[OK]   [OK]   [ - ]
        |-- RepoName              svc-a   [OK]   [FAIL] [ - ]
        |-- StageResults
              map[stage] -> "passed"/"failed"/""
```

**Terminal Types** (6 types in `internal/terminal/types/types.go`, 52 lines):

```
TerminalConfig
  |-- Workspace     string
  |-- Endpoints     []TerminalEndpoint
  |-- AutoConnect   bool
  |-- Theme         string
  |-- Persist       bool

TerminalEndpoint
  |-- Name     string
  |-- URL      string
  |-- Default  bool

SSHKey
  |-- Name       string
  |-- Type       string       ("ed25519")
  |-- PublicKey   string
  |-- PrivateKey  []byte
  |-- Encrypted   bool

KnownHost
  |-- Name  string
  |-- Key   string

TerminalSession
  |-- Workspace    string
  |-- SessionName  string
  |-- Connected    bool
  |-- Endpoint     string
  |-- Username     string

SSHSessionConfig
  |-- Endpoint  string
  |-- Username  string
  |-- Hostname  string
  |-- Command   string
  |-- Cols      int
  |-- Rows      int
```

### Route Table

**REST API Routes (4 routes):**

```
Method  Pattern                                                      Handler
------  -------                                                      -------
GET     /api/v1/portfolios                                           ListPortfolios
GET     /api/v1/portfolios/{name}                                    GetPortfolio
GET     /api/v1/portfolios/{portfolio}/workspaces/{workspace}        GetWorkspace
GET     /api/v1/portfolios/{portfolio}/workspaces/{workspace}/repos/{repo}  GetRepo
```

**WASM Route Matching (PageRenderer):**

```
Pattern                                        Handler
-------                                        -------
["portfolios"]                                 renderPortfolios
["portfolios", name]                           renderPortfolio
["portfolios", p, "workspaces", w]             renderWorkspace
["portfolios", p, "workspaces", w, "repos", r] renderForge
(anything else)                                renderPortfolios (fallback)
```

**WebSocket and Key Provisioning Routes (forge-wss-proxy):**

```
Method  Pattern                                   Handler
------  -------                                   -------
GET     /ws/{workspace}                           WebSocket upgrade -> TCP bridge to forge-workspace-{workspace}:22
POST    /ws/{workspace}/register-key              Register SSH public key for a workspace
GET     /internal/authorized-keys/{workspace}     List registered keys (called by sshd AuthorizedKeysCommand)
GET     /healthz                                  Health check (200 OK)
GET     /readyz                                   Readiness check (200 OK)
```

### Adapter Interfaces

**`internal/adapter/adapter.go` -- Composite interfaces (5 interfaces, 10 methods):**

| Interface | Methods | Purpose |
|-----------|---------|---------|
| `DataSource` | `ListPortfolios(sort)`, `GetPortfolio(name, sort)`, `GetWorkspace(portfolio, workspace, sort)`, `GetForge(portfolio, workspace, repo)` | Page data for rendering |
| `MetaPlanLoader` | `LoadAll(wsPath)`, `Load(path)` | Read `.forge-ai/meta-plan/*.yml` |
| `RepoPlanLoader` | `LoadAll(repoPath)`, `LoadSummary(repoPath, repoName)` | Read `.forge-ai/plan/*/tasks.md` |
| `WsConfigLoader` | `Load(wsPath)` | Read `forge-workspace.yaml` |
| `PortfolioConfigLoader` | `Load(portfolioPath)` | Read `forge-portfolio.yaml` |

**Per-file adapter interfaces (5 interfaces, 9 methods):**

| Interface | File | Methods | Purpose |
|-----------|------|---------|---------|
| `Cache` | `cache.go` | `SetWorkspace(name, data)`, `GetRepoSummary(workspace, repo)`, `GetRepoOverview(workspace, repo)` | In-memory repo data cache (`sync.RWMutex`) |
| `GitInfo` | `git.go` | `RepoInfo(repoPath)` | Execute git commands, return `RepoSummary` |
| `PortfolioDiscovery` | `portfolio.go` | `List(baseDir)`, `Get(baseDir, name)` | Detect portfolios from filesystem |
| `WorkspaceDiscovery` | `workspace.go` | `List(basedir)`, `Get(basedir, name)` | Detect workspaces (dirs with `go.work`) |
| `ForgeLoader` | `forge.go` | `Load(repoPath)` | Read `forge.yaml` + `.forge/artifact-store.yaml` |

Total: 10 interfaces, 19 methods across 11 non-test adapter files.

**`internal/terminal/adapter/adapter.go` -- Terminal interfaces (4 interfaces + 1 sub-interface):**

| Interface | Methods | Purpose |
|-----------|---------|---------|
| `TerminalIO` | `Read`, `Write`, `OnResize`, `Cols`, `Rows`, `Close` | Bridge xterm.js UI with Go domain |
| `SSHClient` | `Connect(cfg, signers, hostCb)` -> `SSHSession` | Establish SSH sessions over WebSocket |
| `SSHSession` | `Stdin`, `Stdout`, `Resize`, `Close` | Represents an active SSH session |
| `KeyRegistrar` | `RegisterKey(workspace, publicKey)` | Register SSH public keys with provisioning service |
| `KeyStore` | `ListKeys`, `GetKey`, `SaveKey`, `DeleteKey`, `ListEndpoints`, `SaveEndpoint`, `ListHosts`, `SaveHost`, `GetParams`, `SaveParams` | Persist SSH keys and config in IndexedDB |

Total: 5 interfaces, 22 methods across 7 non-test terminal adapter files.

### Controller Services

**`internal/controller/` -- 6 non-test files, 5 dashboard services:**

| Interface | File | Methods | Driver |
|-----------|------|---------|--------|
| `PortfolioService` | `portfolio.go` | `ListPortfolios(baseDir, sortMode)`, `GetPortfolio(baseDir, name, sortMode)` | REST API |
| `WorkspaceService` | `workspace.go` | `GetWorkspace(baseDir, portfolio, workspace, sortMode)` | REST API |
| `ForgeService` | `forge.go` | `GetForge(baseDir, portfolio, workspace, repo)` | REST API |
| `PageRenderer` | `controller.go` (interface), `renderer.go` (impl) | `Render(route)` | WASM Dashboard |
| `Refresher` | `refresher.go` | `Start()`, `Stop()` | REST API |

`PortfolioService`, `WorkspaceService`, and `ForgeService` consume adapter interfaces (`PortfolioDiscovery`, `WorkspaceDiscovery`, `Cache`, `ForgeLoader`). `PageRenderer` consumes `DataSource`. `Refresher` consumes `Cache`, `GitInfo`, `PortfolioDiscovery`, and `WorkspaceDiscovery`.

**`internal/terminal/controller/` -- 2 non-test files, 1 terminal service:**

| Interface | File | Methods | Driver |
|-----------|------|---------|--------|
| `SessionController` | `controller.go` (interface), `session.go` (impl) | `Start(cfg, termIO)`, `Stop()` | WASM Terminal |

`SessionController` consumes `SSHClient`, `KeyStore`, and `KeyRegistrar` adapter interfaces. It manages the SSH session lifecycle: key generation, key registration, SSH connection, and bidirectional I/O bridging.

### Package Catalog

| Package | Files (non-test) | Purpose |
|---------|-------------------|---------|
| `cmd/forge-frontend` | 1 | REST API server entry point |
| `cmd/forge-ui-wasm` | 1 | WASM dashboard browser entry point |
| `cmd/forge-terminal-wasm` | 1 | WASM terminal emulator entry point |
| `cmd/forge-wss-proxy` | 1 | WebSocket-to-SSH proxy entry point |
| `internal/types` | 1 | Dashboard domain model: 40 pure data structs |
| `internal/adapter` | 11 | Dashboard outbound ports: interfaces + implementations |
| `internal/controller` | 6 | Dashboard business logic: 5 services |
| `internal/controller/templates` | 5 | Embedded HTML templates for WASM renderer |
| `internal/driver/rest` | 2 | REST API inbound driver: OpenAPI codegen + handler |
| `internal/driver/wasm` | 3 | WASM dashboard inbound driver: DOM, router, driver |
| `internal/terminal/types` | 1 | Terminal domain model: 6 types |
| `internal/terminal/adapter` | 7 | Terminal outbound ports: TerminalIO, SSHClient, KeyRegistrar, KeyStore |
| `internal/terminal/controller` | 2 | Terminal business logic: SSH session controller |
| `internal/terminal/driver/wasm` | 1 | WASM terminal inbound driver |
| `internal/terminal/keygen` | 1 | Ed25519 SSH key generation |
| `internal/wssproxy` | 2 | WebSocket-to-SSH proxy server + key store |
| `internal/util/ignoreutil` | 1 | `.forge-workspace-ignore` parser |
| `internal/util/mocks` | -- | Generated test mocks (mockery) |
| `api` | 1 | OpenAPI 3.0.3 spec for REST API |
| `web` | 2+ | Static assets: index.html, wasm_exec.js, terminal/ |
| `xterm` | 1 | xterm.js source + rollup bundler config |
| `containers/forge-frontend` | 1 | Containerfile for REST API server |
| `containers/forge-ui-wasm` | 2 | Containerfile + nginx.conf for WASM dashboard |
| `containers/forge-workspace` | 3 | Containerfile + entrypoint + authorized-keys script |
| `containers/forge-wss-proxy` | 1 | Multi-stage Containerfile for wss-proxy |
| `charts/forge-workspace` | -- | Helm chart for workspace deployment |
| `charts/forge-wss-proxy` | -- | Helm chart for wss-proxy deployment |
| `test/e2e` | 1+ | End-to-end tests + SSH fixtures |
| `hack` | 2 | Development scripts (run.sh, cleanup.sh) |

## Design Patterns

**Hexagonal Architecture (Ports and Adapters).** Types sit at the center with no behavior. Adapters define outbound interfaces. Controllers hold logic and consume adapter interfaces. Drivers handle inbound requests and consume controller interfaces. Each layer depends only on layers closer to the center.

**Multi-Driver Pattern.** Three independent drivers (REST API, WASM dashboard, WASM terminal) consume controller interfaces. Build tags (`GOOS=js GOARCH=wasm`) select the driver at compile time. The REST API and WASM dashboard drivers serve the same 4 page types. The terminal driver operates independently.

**Cache-Aside with Background Refresh.** The REST API driver's `Refresher` fills an in-memory cache in the background. Controllers read from cache. API responses never block on git operations after the initial synchronous startup refresh.

**Interface-Based Testing.** Dashboard and terminal adapter interfaces have generated mocks (mockery). Controller tests use mock adapters. Driver tests use mock controllers. This isolates each layer for unit testing.

**WebSocket-to-SSH Bridge.** The wss-proxy upgrades HTTP to WebSocket, resolves the target workspace service by name pattern, and pipes WebSocket frames bidirectionally to a raw TCP connection. The SSH protocol runs end-to-end through the tunnel. No persistent state beyond in-memory key registration.

**Browser-Side Key Generation.** The terminal WASM binary generates ed25519 key pairs in the browser using `golang.org/x/crypto/ssh`. Private keys remain in IndexedDB. Only the public key leaves the browser (sent to the key provisioning endpoint).

## Alternatives Considered

| Alternative | Why rejected |
|-------------|-------------|
| Do nothing (inspect repos manually) | Does not scale beyond 5 repos. No aggregated view. |
| Single-driver (REST API only, no WASM) | No static demo deployment. WASM demo enables showcasing without server infrastructure. |
| React/Vue frontend with REST API | Adds JavaScript toolchain and npm dependencies. Conflicts with minimal-dependency tenet. |
| Database-backed state | Adds operational complexity. In-memory cache rebuilds in seconds from git. |
| Cloud IDE (VS Code Server, code-server) | Adds 200+ MB runtime. Conflicts with minimal-dependency tenet. Terminal-only access via SSH covers the primary use case. |
| Direct SSH from browser (no proxy) | Browsers cannot open raw TCP sockets. WebSocket-to-SSH proxy is the standard browser-to-SSH bridge pattern. |

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Git operations are slow for large repos | Background refresh with worker pool. API responses read from cache. |
| WASM binary size grows with features | Only DemoDataSource compiled in. No live git/filesystem adapters in WASM build. Terminal is a separate binary. |
| `go.work` detection fails for non-standard layouts | `.forge-workspace-ignore` support and explicit `forge-workspace.yaml` config. |
| Cache becomes stale between refresh ticks | Configurable refresh interval (default 1 minute) and worker count (default 3). |
| SSH key leakage from IndexedDB | Private keys stored only in browser IndexedDB (origin-scoped). Keys never sent to server. Ed25519 keys are ephemeral -- users can regenerate at any time. |
| WebSocket connection drops | Terminal session reconnects on page reload. tmux preserves session state server-side. Users resume interrupted sessions. |

## Testing Strategy

- **Unit tests:** Controller and adapter packages use generated mocks. Run via `forge test-run unit`.
- **Lint:** golangci-lint via `forge test-run lint`.
- **e2e tests:** Deploy to Kind cluster (testenv-kind + testenv-lcr + testenv-helm-install). Validate terminal connectivity and REST API responses. Run via `forge test-run e2e`.
- **Full validation:** `forge test-all` builds all 11 targets, then runs lint + unit + e2e stages.

## FAQ

**Why hexagonal architecture?**

Hexagonal architecture enables dual-driver compilation from one codebase. Controllers and adapters are driver-agnostic. Adding a third driver (e.g., a CLI) requires no changes to controllers or adapters.

**Why not use a JavaScript framework for the WASM UI?**

Go templates produce HTML directly. The WASM binary replaces `innerHTML` on navigation. xterm.js is the single JavaScript library, bundled via rollup. The project has 5 direct Go dependencies.

**How does the cache handle concurrent access?**

`sync.RWMutex` protects the in-memory map. Refresher workers call `SetWorkspace` (write lock). REST API handlers call `GetRepoSummary` and `GetRepoOverview` (read lock). Reads do not block each other.

**Why `sigs.k8s.io/yaml` instead of `gopkg.in/yaml.v3`?**

`sigs.k8s.io/yaml` unmarshals YAML into Go structs using JSON struct tags. This avoids maintaining duplicate `yaml:` and `json:` tags on every struct field.

**Why a separate WASM binary for the terminal?**

The terminal emulator has different lifecycle requirements from the dashboard. It maintains a persistent WebSocket connection and SSH session. Separating it into forge-terminal-wasm keeps the dashboard WASM binary small and avoids importing `x/crypto` in the dashboard build.

**Why WebSocket-to-SSH instead of WebSocket-to-PTY?**

SSH provides authentication, encryption, and session multiplexing (tmux). A direct PTY approach would require a custom protocol and skip standard SSH security. The wss-proxy pattern is well-established in browser-based terminal tools.

## Appendix

### forge.yaml

```yaml
name: forge-ui

envFile: .envrc
artifactStorePath: .forge/artifact-store.yaml

engines:
  - alias: wasm-build
    type: builder
    builder:
      - engine: go://generic-builder

  - alias: generate-mocks
    type: builder
    builder:
      - engine: go://generic-builder
        spec:
          command: "rm"
          args: ["-rf", "./internal/util/mocks"]
      - engine: go://go-gen-mocks

  - alias: web-assets
    type: builder
    builder:
      - engine: go://generic-builder

  - alias: setup-e2e
    type: testenv
    testenv:
      - engine: go://testenv-kind
      - engine: go://testenv-lcr
        spec:
          enabled: true
          namespace: testenv-lcr
          imagePullSecretName: testenv-lcr-credentials
          imagePullSecretNamespaces: [default]
          images:
            - name: local://forge-wss-proxy-image:latest
            - name: local://forge-workspace-image:latest
      - engine: go://testenv-helm-install
        spec:
          charts:
            - name: forge-wss-proxy
              sourceType: local
              path: ./charts/forge-wss-proxy
              values:
                image:
                  repository: "{{.Env.TESTENV_LCR_FQDN}}/forge-wss-proxy-image"
                  tag: latest
                  pullPolicy: IfNotPresent
                imagePullSecrets:
                  - name: testenv-lcr-credentials
            - name: forge-workspace-test-ws
              sourceType: local
              path: ./charts/forge-workspace
              values:
                image:
                  repository: "{{.Env.TESTENV_LCR_FQDN}}/forge-workspace-image"
                  tag: latest
                  pullPolicy: IfNotPresent
                imagePullSecrets:
                  - name: testenv-lcr-credentials
                config:
                  workspaceName: "test-ws"
                ssh:
                  authorizedKeys:
                    - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFFbznCQzYI2RNWzWYp0H4RHVNRn4n9Z3ZlfGeIjKSIN test-user"

build:
  - name: forge-frontend
    src: ./cmd/forge-frontend
    dest: ./build/bin
    engine: go://go-build

  - name: forge-ui-wasm
    src: ./cmd/forge-ui-wasm
    dest: ./build/web
    engine: alias://wasm-build
    spec:
      command: "go"
      args: ["build", "-o", "./build/web/forge-ui.wasm", "./cmd/forge-ui-wasm"]
      env:
        GOOS: "js"
        GOARCH: "wasm"

  - name: forge-terminal-wasm
    src: ./cmd/forge-terminal-wasm
    dest: ./build/web
    engine: alias://wasm-build
    spec:
      command: "go"
      args: ["build", "-o", "./build/web/terminal.wasm", "./cmd/forge-terminal-wasm"]
      env:
        GOOS: "js"
        GOARCH: "wasm"

  - name: forge-terminal-xterm
    src: ./xterm
    dest: ./build/web
    engine: alias://web-assets
    spec:
      command: "bash"
      args: ["-c", "cd xterm && npm install && npx rollup -c && cp node_modules/@xterm/xterm/css/xterm.css ../web/terminal/"]

  - name: forge-ui-web-assets
    src: ./web
    dest: ./build/web
    engine: alias://web-assets
    spec:
      command: "cp"
      args: ["-r", "./web/.", "./build/web/"]

  - name: forge-wss-proxy
    src: ./cmd/forge-wss-proxy
    dest: ./build/bin
    engine: go://go-build

  - name: forge-wss-proxy-image
    src: ./containers/forge-wss-proxy/Containerfile
    engine: go://container-build
    spec:
      dependsOn:
        - engine: go://go-dependency-detector
          spec:
            filepath: ./cmd/forge-wss-proxy/main.go
            funcName: main

  - name: forge-workspace-image
    src: ./containers/forge-workspace/Containerfile
    engine: go://container-build

  - name: generate-mocks
    src: .
    engine: alias://generate-mocks

  - name: generate-rest-api
    src: ./api/forge-ui.v1.yaml
    dest: ./internal/driver/rest
    engine: go://go-gen-openapi
    spec:
      sourceFile: ./api/forge-ui.v1.yaml
      destinationDir: ./internal/driver
      server:
        enabled: true
        packageName: rest

test:
  - name: lint
    runner: "go://go-lint"

  - name: unit
    runner: "go://go-test"

  - name: e2e
    runner: go://go-test
    testenv: alias://setup-e2e
    spec:
      buildTags: [e2e]
```

### Containerfiles

**containers/forge-frontend/Containerfile** (REST API server):

```dockerfile
FROM docker.io/alpine:3.20.1
RUN apk add --no-cache git
COPY build/bin/forge-frontend /bin/forge-frontend
EXPOSE 8081
CMD [ "forge-frontend", "-port", "8081" ]
```

**containers/forge-ui-wasm/Containerfile** (WASM dashboard + terminal assets):

```dockerfile
FROM docker.io/nginx:1.27-alpine
COPY containers/forge-ui-wasm/nginx.conf /etc/nginx/conf.d/default.conf
COPY build/web/ /usr/share/nginx/html/
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
```

**containers/forge-workspace/Containerfile** (SSH + tmux workspace):

```dockerfile
FROM docker.io/alpine:3.20.1
RUN apk add --no-cache openssh-server tmux git ncurses
RUN adduser -D -s /bin/sh forge
COPY containers/forge-workspace/get-authorized-keys.sh /usr/local/bin/get-authorized-keys
COPY containers/forge-workspace/entrypoint.sh /entrypoint.sh
EXPOSE 22
CMD ["/entrypoint.sh"]
```

**containers/forge-wss-proxy/Containerfile** (multi-stage Go build):

```dockerfile
FROM docker.io/golang:1.25 AS downloader
WORKDIR /workdir
COPY go.mod go.sum ./
RUN go mod download

FROM downloader AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/forge-wss-proxy ./cmd/forge-wss-proxy

FROM docker.io/alpine:3.20.1
COPY --from=builder /bin/forge-wss-proxy /bin/forge-wss-proxy
EXPOSE 8080
CMD ["/bin/forge-wss-proxy"]
```
