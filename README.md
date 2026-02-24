# forge-ui

**A Go web dashboard that visualizes Forge workspaces, build systems, test results, and git metadata across Go module workspaces.**

> "I manage 12 repositories across 3 Go workspaces. I need to see which repos are
> dirty, which tests fail, and where meta-plans stand -- without opening each repo
> individually. forge-ui gives me that view in one browser tab."

## What problem does forge-ui solve?

Go workspaces (`go.work`) group related repositories into a single development
unit. A workspace with 10-20 repos forces developers to inspect git status, build
artifacts, and test results one repo at a time. Across workspaces in a portfolio,
this becomes unmanageable. forge-ui aggregates filesystem, git, and Forge data
into a 4-level hierarchy: Portfolio > Workspace > Repository > Forge. It renders
this data as a live HTTP dashboard or a static WebAssembly (WASM) demo, with
background git refresh and test heatmaps.

## Quick Start

**HTTP server** (live data from your workspaces):

```sh
forge build forge-ui
./build/bin/forge-ui -workspaces ~/workspaces
# Open http://localhost:8080
```

**WASM demo** (static demo data in the browser):

```sh
forge build
# Serve build/web/ with any static file server
python3 -m http.server -d build/web 8080
# Open http://localhost:8080
```

**Container**:

```sh
hack/run.sh
```

## How does it work?

```
+-------------------------------------------------+
|               INBOUND DRIVERS                   |
|  +-------------------+  +--------------------+  |
|  | HTTP Driver       |  | WASM Driver        |  |
|  | (GOOS=linux)      |  | (GOOS=js)          |  |
|  | Server-side HTML  |  | Client-side HTML   |  |
|  +--------+----------+  +---------+----------+  |
|           |                       |              |
|           v                       v              |
|              CONTROLLERS                         |
|  PortfolioService  WorkspaceService  ForgeService|
|  PageRenderer (WASM)   Refresher (HTTP)          |
|           |                       |              |
|           v                       v              |
|               ADAPTERS                           |
|  Git  Cache  Filesystem  ForgeLoader  Demo       |
|           |                       |              |
|           v                       v              |
|              TYPES (pure data)                   |
+-------------------------------------------------+
```

forge-ui uses hexagonal architecture with dual drivers. Drivers receive inbound
requests (HTTP or WASM hash routing). Controllers hold business logic: portfolio
listing, workspace aggregation, and forge detail retrieval. Adapters read data
from the filesystem, git, and cache. Types define the domain model with no
behavior. See [DESIGN.md](DESIGN.md) for full details.

## Table of Contents

- [How do I configure?](#how-do-i-configure)
- [How do I build and test?](#how-do-i-build-and-test)
- [What does the UI show?](#what-does-the-ui-show)
- [FAQ](#faq)
- [Documentation](#documentation)
- [Contributing](#contributing)

## How do I configure?

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `8080` | HTTP server port |
| `-workspaces` | `$WORKSPACES` or `$HOME/workspaces` | Base directory containing workspaces |
| `-refresh-interval` | `1m` | Background git refresh interval |
| `-refresh-workers` | `3` | Number of background git refresh workers |

Set the `WORKSPACES` environment variable to override the default base directory.

### Configuration Files

| File | Scope | Purpose |
|------|-------|---------|
| `forge-workspace.yaml` | Per workspace | Workspace description and meta-plan references |
| `forge-portfolio.yaml` | Per portfolio | Portfolio description and purpose |
| `forge.yaml` | Per repository | Build specs and test stages |
| `.forge-workspace-ignore` | Per workspace | Exclude directories from workspace discovery |

## How do I build and test?

```sh
forge build                    # Build all 4 targets
forge build forge-ui           # Build HTTP server binary
forge build forge-ui-wasm      # Build WASM binary
forge test-all                 # Build + lint + unit tests
forge test-run unit            # Unit tests only
forge test-run lint            # Lint only
```

Requirements: Go 1.25.0 and [forge](https://github.com/alexandremahdhaoui/forge).

## What does the UI show?

forge-ui renders 4 page types:

1. **Portfolios** -- Lists all portfolios with aggregate stats: repository count,
   dirty repo count, and test pass/fail counts.
2. **Portfolio detail** -- Lists workspaces within a portfolio with per-workspace
   repository summaries.
3. **Workspace detail** -- Shows per-repo git metadata (branch, status, commits,
   ahead/behind, diff) and a test result heatmap (repos x stages).
4. **Forge detail** -- Shows build specs, artifacts with dependencies, test
   reports with coverage, and test environments for a single repository.

The UI uses Material Design 3 styles with 4 light palettes and 1 dark palette
(Catppuccin Mocha). Theme selection persists via cookie (HTTP) or localStorage
(WASM).

## FAQ

**What is the WASM build for?**

The WASM build serves as a demo and static deployment option. It uses hardcoded
demo data instead of live git and filesystem access.

**How does background refresh work?**

On startup, the HTTP driver runs a synchronous initial refresh that blocks until
complete. After that, a scheduler triggers refreshes at the configured interval.
A worker pool (default: 3 workers) processes workspace refresh jobs concurrently.
Page loads read from the in-memory cache and never block on git operations.

**What is a portfolio?**

A portfolio is a named collection of workspaces. Directories under the base path
that contain subdirectories with `go.work` files are detected as portfolios.
Workspaces not inside a named portfolio belong to a default catch-all portfolio.

**What Go version is required?**

Go 1.25.0. See `go.mod` for the full dependency list (2 direct dependencies:
`sigs.k8s.io/yaml` and `github.com/stretchr/testify`).

**Can I run it in a container?**

Yes. Run `hack/run.sh` to build and start a Docker container.

**What are meta-plans?**

Meta-plans are cross-repo orchestration plans stored in `.forge-ai/meta-plan/`
directories within workspaces. forge-ui reads these plans and displays progress
tracking across repositories and workspaces.

**How is theme selection stored?**

The HTTP driver stores the selected theme in a cookie. The WASM driver stores it
in the browser's localStorage.

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
