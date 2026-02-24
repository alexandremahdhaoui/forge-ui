# forge-ui Design

forge-ui uses hexagonal architecture with dual drivers to render a Go workspace dashboard as both a live HTTP server and a static WebAssembly (WASM) browser client.

## Problem Statement

Go workspaces (`go.work`) group related repositories into a single dependency graph. A team managing 5-20 repositories per workspace tracks git status, build artifacts, and test results across each repo individually. Portfolios add another layer: named collections of workspaces that span entire product areas. Inspecting each repository one by one does not scale. forge-ui aggregates filesystem, git, and forge data into a 4-level hierarchy -- Portfolio > Workspace > Repository > Forge -- and renders it via server-side HTML (HTTP) or client-side HTML (WASM).

## Tenets

These tenets are prioritized. When two tenets conflict, the higher-ranked tenet wins.

1. **Read-only.** forge-ui reads data. It never modifies repositories, workspaces, or build artifacts.
2. **Hexagonal boundaries.** Drivers, controllers, and adapters communicate through interfaces. No layer imports from a layer above it.
3. **Dual-driver parity.** HTTP and WASM drivers render the same 4 pages with the same controllers. Only the data source and rendering mechanism differ.
4. **Minimal dependencies.** 2 external dependencies: `sigs.k8s.io/yaml` and `stretchr/testify`. No JavaScript frameworks. No CSS frameworks beyond hand-crafted Material Design 3 styles.
5. **Zero-config default.** Without flags, forge-ui scans `$HOME/workspaces` and serves on port 8080.
6. **Background freshness.** The HTTP driver refreshes git data in the background. Page loads never block on git operations after startup.

## Requirements

From the user's perspective:

- View all portfolios with aggregate stats: repo count, dirty count, test pass/fail
- View all workspaces within a portfolio with per-workspace repo summaries
- View workspace detail with per-repo git metadata: branch, status, commits, ahead/behind, diff
- View test result heatmap (repos x stages) per workspace
- View forge detail per repo: build specs, artifacts with dependencies, test reports with coverage, test environments
- Toggle dark/light theme; select among 4 light palettes
- Run as HTTP server with live data or as WASM client with demo data
- Track meta-plan progress across repos and workspaces

## Out of Scope

- Write operations: forge-ui does not create, modify, or delete files
- User authentication or authorization
- Real-time WebSocket updates (uses polling via background refresh)
- CI/CD integration (reads forge artifacts, does not trigger builds)
- Multi-user state (no database, in-memory cache only)

## Success Criteria

- HTTP server renders all 4 page types (portfolios, portfolio, workspace, forge) in under 50ms after initial refresh
- WASM client loads and renders the initial page in under 2 seconds
- Background refresh completes for N workspaces with configurable interval (default 1 minute) and worker count (default 3)
- `forge test-all` passes: lint + unit stages
- Zero external runtime dependencies beyond git (HTTP mode) or a browser (WASM mode)

## Proposed Design

### Architecture Overview

```
+-----------------------------------------------------------------------+
|                         forge-ui                                      |
|                                                                       |
|  INBOUND DRIVERS                                                      |
|  (how requests enter)                                                 |
|                                                                       |
|  +---------------------------+    +-----------------------------+     |
|  | HTTP Driver               |    | WASM Driver                 |     |
|  | (GOOS=linux)              |    | (GOOS=js, GOARCH=wasm)      |     |
|  |                           |    |                             |     |
|  | cmd/forge-ui/main.go      |    | cmd/forge-ui-wasm/main.go   |     |
|  | driver/http/handler.go    |    | driver/wasm/driver.go       |     |
|  | driver/http/portfolios.go |    | driver/wasm/router.go       |     |
|  | driver/http/workspace.go  |    | driver/wasm/dom.go          |     |
|  | driver/http/forge.go      |    |                             |     |
|  | driver/http/redirect.go   |    |                             |     |
|  |                           |    |                             |     |
|  | net/http server           |    | syscall/js hash routing     |     |
|  | Server-side rendering     |    | Client-side rendering       |     |
|  | HTML templates on disk    |    | Embedded templates (embed)  |     |
|  | Cookie-based theme        |    | localStorage theme          |     |
|  +------------+--------------+    +-------------+---------------+     |
|               |                                 |                     |
|               v                                 v                     |
|  CONTROLLER LAYER                                                     |
|  (business logic)                                                     |
|                                                                       |
|  +---------------------+  +---------------------+  +---------------+ |
|  | PortfolioService     |  | WorkspaceService    |  | ForgeService  | |
|  | portfolio.go         |  | workspace.go        |  | forge.go      | |
|  |                      |  |                     |  |               | |
|  | ListPortfolios()     |  | GetWorkspace()      |  | GetForge()    | |
|  | GetPortfolio()       |  |                     |  |               | |
|  +----------+----------+  +----------+----------+  +-------+-------+ |
|             |                        |                      |         |
|  +----------+------------------------+----------------------+-------+ |
|  |                                                                  | |
|  |  PageRenderer (WASM only)          Refresher (HTTP only)         | |
|  |  renderer.go                       refresher.go                  | |
|  |                                                                  | |
|  |  Render(route) -> HTML string      Start() -> initial refresh    | |
|  |  Parses route, calls DataSource,   scheduler -> ticker -> queue  | |
|  |  executes embedded templates        N workers -> refreshWorkspace | |
|  +------------------------------------------------------------------+ |
|               |                                 |                     |
|               v                                 v                     |
|  ADAPTER LAYER                                                        |
|  (outbound ports -- interfaces + implementations)                     |
|                                                                       |
|  +--------------------+  +------------------+  +-------------------+  |
|  | PortfolioDiscovery |  | WorkspaceDisc.   |  | GitInfo           |  |
|  | portfolio.go       |  | workspace.go     |  | git.go            |  |
|  |                    |  |                  |  |                   |  |
|  | List(baseDir)      |  | List(basedir)    |  | RepoInfo(path)    |  |
|  | Get(baseDir, name) |  | Get(basedir,name)|  |  -> branch        |  |
|  |                    |  |                  |  |  -> status         |  |
|  | Reads filesystem   |  | Scans for dirs   |  |  -> log           |  |
|  | Detects portfolios |  | with go.work +   |  |  -> diff          |  |
|  | vs loose workspaces|  | .git children    |  |  -> ahead/behind  |  |
|  +--------------------+  +------------------+  +-------------------+  |
|                                                                       |
|  +--------------------+  +------------------+  +-------------------+  |
|  | Cache              |  | ForgeLoader      |  | DataSource        |  |
|  | cache.go           |  | forge.go         |  | (interface only)  |  |
|  |                    |  |                  |  |                   |  |
|  | SetWorkspace()     |  | Load(repoPath)   |  | ListPortfolios()  |  |
|  | GetRepoSummary()   |  |  -> forge.yaml   |  | GetPortfolio()    |  |
|  | GetRepoOverview()  |  |  -> artifact-    |  | GetWorkspace()    |  |
|  |                    |  |     store.yaml   |  | GetForge()        |  |
|  | sync.RWMutex map   |  |                  |  |                   |  |
|  +--------------------+  +------------------+  +-------------------+  |
|                                                                       |
|  +--------------------+  +------------------+  +-------------------+  |
|  | MetaPlanLoader     |  | RepoPlanLoader   |  | WsConfigLoader    |  |
|  | metaplan.go        |  | repoplan.go      |  | wsconfig.go       |  |
|  |                    |  |                  |  |                   |  |
|  | LoadAll(wsPath)    |  | LoadAll(repoPath)|  | Load(wsPath)      |  |
|  | Load(path)         |  | LoadSummary()    |  |  -> forge-        |  |
|  |  -> .forge-ai/     |  |  -> .forge-ai/   |  |    workspace.yaml |  |
|  |     meta-plan/*.yml|  |     plan/*/tasks  |  |                   |  |
|  +--------------------+  +------------------+  +-------------------+  |
|                                                                       |
|  +--------------------+  +------------------+                         |
|  | PortfolioConfig    |  | DemoDataSource   |                         |
|  | portfolioconfig.go |  | demo.go          |                         |
|  |                    |  |                  |                         |
|  | Load(portfolioPath)|  | Implements       |                         |
|  |  -> forge-         |  | DataSource with  |                         |
|  |    portfolio.yaml  |  | static demo data |                         |
|  |                    |  | (WASM build only)|                         |
|  +--------------------+  +------------------+                         |
|                                                                       |
|  DOMAIN MODEL                                                         |
|  (pure data, no behavior)                                             |
|                                                                       |
|  +----------------------------------------------------------------+   |
|  | types/types.go                                                 |   |
|  |                                                                |   |
|  | PortfolioSummary  WorkspaceSummary  RepoOverview  RepoSummary  |   |
|  | PortfolioPageData WorkspacePageData ForgePageData ForgeSpec    |   |
|  | TestReport  TestEnv  Artifact  MetaPlan  WsConfig  Cache...   |   |
|  +----------------------------------------------------------------+   |
+-----------------------------------------------------------------------+
```

The architecture has 4 layers. Types (center) hold pure data structs with no behavior -- 36 types in 329 lines. Adapters (outbound ports) define interfaces for filesystem, git, and cache access. Controllers hold business logic: 3 domain services, a page renderer, and a refresher. Drivers (inbound ports) receive requests via HTTP or WASM and delegate to controllers.

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
GOOS=linux GOARCH=amd64          GOOS=js GOARCH=wasm
         |                                |
         v                                v
+------------------+            +--------------------+
| cmd/forge-ui/    |            | cmd/forge-ui-wasm/ |
| main.go          |            | main.go            |
|                  |            |                    |
| Uses:            |            | Uses:              |
|  httpdriver.New  |            |  wasm.New          |
|  Refresher.Start |            |  PageRenderer      |
|  PortfolioSvc    |            |  DemoDataSource    |
|  WorkspaceSvc    |            |                    |
|  ForgeSvc        |            | Output: main.wasm  |
|                  |            | Served by static   |
| Output: binary   |            | HTML + wasm_exec.js|
| Serves on :8080  |            +--------------------+
+------------------+
```

forge.yaml defines 4 build targets:

| Target | Output | Engine |
|--------|--------|--------|
| `forge-ui` | `build/bin/forge-ui` | `go://go-build` |
| `forge-ui-wasm` | `build/web/forge-ui.wasm` | `alias://wasm-build` (GOOS=js GOARCH=wasm) |
| `forge-ui-web-assets` | `build/web/` | `alias://web-assets` (cp web/ to build/) |
| `generate-mocks` | `internal/util/mocks/` | `alias://generate-mocks` (mockery) |

### HTTP Request Flow

```
Browser
  |
  | GET /portfolios/infrastructure/workspaces/platform?sort=time
  v
net/http ServeMux
  |
  | route match: GET /portfolios/{p}/workspaces/{w}
  v
httpdriver.Handler.HandleWorkspace(w, r)
  |
  | 1. Extract path values: p="infrastructure", w="platform"
  | 2. Read sort mode from query (default: "time")
  | 3. Read dark mode from "theme" cookie
  | 4. Read light palette from "light-palette" cookie
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
httpdriver.Handler.render(w, "workspace", data)
  |
  | Execute template "workspace" with layout
  | Write Content-Type: text/html
  v
Browser renders HTML
```

The HTTP driver follows a direct request-response model. Browser sends a GET request. The ServeMux routes it to the correct handler. The handler reads cookies and path values, calls the appropriate controller service, and renders an HTML template. Controllers read from the in-memory cache (filled by the background refresher) and the filesystem (for forge.yaml data).

### Background Refresh Flow

```
cmd/forge-ui/main.go
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
  |   --> parse embedded templates (go:embed templates/*.html)
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

### HTTP vs WASM Comparison

```
Aspect              HTTP Driver                 WASM Driver
------              -----------                 -----------
Entry point         cmd/forge-ui/main.go        cmd/forge-ui-wasm/main.go
Build tag           (none / GOOS=linux)         GOOS=js GOARCH=wasm
Routing             net/http ServeMux           hashchange event listener
URL format          /portfolios/infra           #/portfolios/infra
Data source         Live filesystem + git       Static demo data
Cache               In-memory (sync.RWMutex)    None (static)
Background refresh  Refresher (N workers)       None
Templates           Parsed from disk            Embedded (go:embed)
Rendering           Server-side (template exec) Client-side (innerHTML)
Theme storage       HTTP cookie                 localStorage
Output              Full HTML document          HTML fragment (inner content)
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

**Core Types and Relationships** (36 types in `internal/types/types.go`, 329 lines):

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

### Route Table

**HTTP Routes (7 routes):**

```
Method  Pattern                                      Handler
------  -------                                      -------
GET     /{$}                                         HandleRedirect -> /portfolios
GET     /portfolios                                  HandlePortfolios
GET     /portfolios/{name}                           HandlePortfolio
GET     /portfolios/{p}/workspaces/{w}               HandleWorkspace
GET     /portfolios/{p}/workspaces/{w}/repos/{r}     HandleForge
GET     /theme/toggle                                HandleToggleTheme
GET     /light-palette/{n}                           HandleSetLightPalette
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

### Controller Services

**`internal/controller/` -- 6 non-test files, 5 services:**

| Interface | File | Methods | Driver |
|-----------|------|---------|--------|
| `PortfolioService` | `portfolio.go` | `ListPortfolios(baseDir, sortMode)`, `GetPortfolio(baseDir, name, sortMode)` | HTTP |
| `WorkspaceService` | `workspace.go` | `GetWorkspace(baseDir, portfolio, workspace, sortMode)` | HTTP |
| `ForgeService` | `forge.go` | `GetForge(baseDir, portfolio, workspace, repo)` | HTTP |
| `PageRenderer` | `controller.go` (interface), `renderer.go` (impl) | `Render(route)` | WASM |
| `Refresher` | `refresher.go` | `Start()`, `Stop()` | HTTP |

`PortfolioService`, `WorkspaceService`, and `ForgeService` consume adapter interfaces (`PortfolioDiscovery`, `WorkspaceDiscovery`, `Cache`, `ForgeLoader`). `PageRenderer` consumes `DataSource`. `Refresher` consumes `Cache`, `GitInfo`, `PortfolioDiscovery`, and `WorkspaceDiscovery`.

### Package Catalog

| Package | Files (non-test) | Purpose |
|---------|-------------------|---------|
| `cmd/forge-ui` | 1 | HTTP server entry point |
| `cmd/forge-ui-wasm` | 1 | WASM browser entry point |
| `internal/types` | 1 | Domain model: 36 pure data structs |
| `internal/adapter` | 11 | Outbound ports: 10 interfaces, 10 implementations |
| `internal/controller` | 6 | Business logic: 5 services |
| `internal/driver/http` | 5 | HTTP inbound driver: 5 handlers |
| `internal/driver/wasm` | 3 | WASM inbound driver: DOM, router, driver |
| `internal/util/ignoreutil` | 1 | `.forge-workspace-ignore` parser |
| `internal/util/mocks` | -- | Generated test mocks (mockery) |
| `templates` | 5 | HTML templates for HTTP server: layout, portfolios, portfolio, workspace, forge |
| `web` | 2 | Static assets for WASM: `index.html`, `wasm_exec.js` |
| `containers/forge-ui` | 1 | Containerfile for Docker deployment |
| `hack` | 1 | Development scripts (`run.sh`) |

## Design Patterns

**Hexagonal Architecture (Ports and Adapters).** Types sit at the center with no behavior. Adapters define outbound interfaces. Controllers hold logic and consume adapter interfaces. Drivers handle inbound requests and consume controller interfaces. Each layer depends only on layers closer to the center.

**Dual-Driver Pattern.** Two independent drivers (HTTP and WASM) consume the same controller interfaces. Build tags (`GOOS=js GOARCH=wasm`) select the driver at compile time. Both drivers render the same 4 pages: portfolios, portfolio, workspace, and forge.

**Cache-Aside with Background Refresh.** The HTTP driver's `Refresher` fills an in-memory cache in the background. Controllers read from cache. Page rendering never blocks on git operations after the initial synchronous startup refresh.

**Interface-Based Testing.** All 10 adapter interfaces have generated mocks (mockery). Controller tests use mock adapters. Driver tests use mock controllers. This isolates each layer for unit testing.

## Alternatives Considered

| Alternative | Why rejected |
|-------------|-------------|
| Do nothing (inspect repos manually) | Does not scale beyond 5 repos. No aggregated view. |
| Single-driver (HTTP only) | No static demo deployment. WASM demo enables showcasing without server infrastructure. |
| React/Vue frontend with REST API | Adds JavaScript toolchain and npm dependencies. Conflicts with minimal-dependency tenet. |
| Database-backed state | Adds operational complexity. In-memory cache rebuilds in seconds from git. |

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Git operations are slow for large repos | Background refresh with worker pool. Page loads read from cache. |
| WASM binary size grows with features | Only DemoDataSource compiled in. No live git/filesystem adapters in WASM build. |
| `go.work` detection fails for non-standard layouts | `.forge-workspace-ignore` support and explicit `forge-workspace.yaml` config. |
| Cache becomes stale between refresh ticks | Configurable refresh interval (default 1 minute) and worker count (default 3). |

## Testing Strategy

- **Unit tests:** Controller and adapter packages use generated mocks. Run via `forge test-run unit`.
- **Lint:** golangci-lint via `forge test-run lint`.
- **Full validation:** `forge test-all` builds all 4 targets, then runs lint + unit stages.
- **No integration tests currently.** WASM testing requires a browser environment, which is out of scope.

## FAQ

**Why hexagonal architecture?**

Hexagonal architecture enables dual-driver compilation from one codebase. Controllers and adapters are driver-agnostic. Adding a third driver (e.g., a CLI) requires no changes to controllers or adapters.

**Why not use a JavaScript framework for the WASM UI?**

Go templates produce HTML directly. No JavaScript build toolchain required. The WASM binary replaces `innerHTML` on navigation. This keeps the dependency count at 2 external Go modules.

**How does the cache handle concurrent access?**

`sync.RWMutex` protects the in-memory map. Refresher workers call `SetWorkspace` (write lock). HTTP handlers call `GetRepoSummary` and `GetRepoOverview` (read lock). Reads do not block each other.

**Why `sigs.k8s.io/yaml` instead of `gopkg.in/yaml.v3`?**

`sigs.k8s.io/yaml` unmarshals YAML into Go structs using JSON struct tags. This avoids maintaining duplicate `yaml:` and `json:` tags on every struct field.

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

build:
  - name: forge-ui
    src: ./cmd/forge-ui
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

  - name: forge-ui-web-assets
    src: ./web
    dest: ./build/web
    engine: alias://web-assets
    spec:
      command: "cp"
      args: ["-r", "./web/.", "./build/web/"]

  - name: generate-mocks
    src: .
    engine: alias://generate-mocks

test:
  - name: lint
    runner: "go://go-lint"

  - name: unit
    runner: "go://go-test"
```

### Containerfile

```dockerfile
FROM docker.io/alpine:3.20.1

RUN apk add --no-cache git

COPY build/bin/forge-ui /bin/forge-ui
COPY templates /templates

EXPOSE 8080
CMD [ "forge-ui", "-port", "8080" ]
```
