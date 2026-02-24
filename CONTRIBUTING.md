# Contributing to forge-ui

**Contribute to forge-ui -- a Go web dashboard for Forge workspaces and portfolios.**

## Quick start

```sh
git clone https://github.com/alexandremahdhaoui/forge-ui.git
cd forge-ui
forge build
forge test-all
./build/bin/forge-ui -workspaces ~/workspaces
# Open http://localhost:8080
```

Requirements: Go 1.25.0, [forge](https://github.com/alexandremahdhaoui/forge) CLI, git.

## How do I structure commits?

Each commit uses an emoji prefix and a structured body.

| Emoji | Meaning |
|-------|---------|
| `✨` | New feature |
| `🐛` | Bug fix |
| `📖` | Documentation |
| `🌱` | Misc (chore, test, etc.) |
| `⚠` | Breaking change (requires maintainer approval) |

Commit body format:

```
<emoji> Short imperative summary (50 chars or less)

Why: Explain the motivation. What problem exists?

How: Describe the approach. What strategy did you choose?

What:

- path/to/file.go: description of change
- path/to/other.go: description of change

How changes were verified:

- Unit tests pass (forge test-run unit)
- forge test-all: all stages passed

Signed-off-by: Your Name <your@email.com>
```

Every commit must include `Signed-off-by`. Use `git commit -s` to add it automatically.

Example:

```
✨ Add workspace sorting by last commit time

Why: Users with 10+ repos per workspace cannot find recently modified repos.

How: Sort RepoSummary slice by LastCommitTime descending in WorkspaceService.

What:

- internal/controller/workspace.go: add sortByLastCommit function
- internal/controller/workspace_test.go: add test for sort order

How changes were verified:

- Unit tests pass (forge test-run unit)
- forge test-all: all stages passed

Signed-off-by: Your Name <your@email.com>
```

## How do I submit a pull request?

1. Create a feature branch from `main`.
2. Make changes and commit with the format above.
3. Run `forge test-all` -- all stages must pass.
4. Push your branch and open a PR against `main`.
5. PR title: imperative mood, under 70 characters.
6. PR body: describe what changed and why.

## How do I run tests?

| Command | What it does |
|---------|--------------|
| `forge test-all` | Build all targets, then run lint + unit |
| `forge test-run lint` | Run golangci-lint |
| `forge test-run unit` | Run Go unit tests |
| `forge build` | Build all 4 targets |
| `forge build forge-ui` | Build HTTP server binary |
| `forge build forge-ui-wasm` | Build WebAssembly (WASM) binary |
| `forge build generate-mocks` | Regenerate test mocks |

Always run `forge test-all` before submitting a PR. Running `go build` alone is not sufficient.

## How is the project structured?

```
forge-ui/
  cmd/
    forge-ui/              HTTP server entry point
    forge-ui-wasm/         WASM browser entry point
  internal/
    adapter/               Outbound ports: git, filesystem, cache, demo (11 files)
    controller/            Business logic: portfolio, workspace, forge, renderer, refresher (6 files)
      templates/           Embedded HTML templates for WASM renderer (4 files)
    driver/
      http/                HTTP inbound driver: handlers + route wiring (5 files)
      wasm/                WASM inbound driver: DOM, router, driver (3 files)
    types/                 Domain model: pure data structs (1 file, 30+ types)
    util/
      ignoreutil/          .forge-workspace-ignore parser
      mocks/               Generated test mocks (mockery)
  templates/               HTML templates for HTTP server (5 files)
  web/                     Static assets for WASM (index.html, wasm_exec.js)
  containers/forge-ui/     Containerfile for Docker deployment
  hack/                    Development scripts (run.sh)
  forge.yaml               Build and test configuration
```

Dependency direction: `driver` -> `controller` -> `adapter` -> `types`. Never import a higher layer.

## What does each package do?

### `internal/adapter/` -- Outbound Ports (11 files)

| File | Interface / Impl | Purpose |
|------|-----------------|---------|
| `adapter.go` | DataSource, MetaPlanLoader, RepoPlanLoader, WsConfigLoader, PortfolioConfigLoader | Core interface definitions |
| `cache.go` | Cache | In-memory repo data cache (`sync.RWMutex`) |
| `demo.go` | DemoDataSource | Static demo data for WASM build |
| `forge.go` | ForgeLoader | Read `forge.yaml` and `artifact-store.yaml` |
| `git.go` | GitInfo | Execute git commands, produce RepoSummary |
| `metaplan.go` | MetaPlanLoader | Read `.forge-ai/meta-plan/*.yml` |
| `portfolio.go` | PortfolioDiscovery | Detect portfolios from filesystem |
| `portfolioconfig.go` | PortfolioConfigLoader | Read `forge-portfolio.yaml` |
| `repoplan.go` | RepoPlanLoader | Read `.forge-ai/plan/*/tasks.md` |
| `workspace.go` | WorkspaceDiscovery | Detect workspaces (dirs with `go.work`) |
| `wsconfig.go` | WsConfigLoader | Read `forge-workspace.yaml` |

### `internal/controller/` -- Business Logic (6 files)

| File | Service | Purpose |
|------|---------|---------|
| `controller.go` | PageRenderer (interface) | Route-to-HTML rendering contract |
| `portfolio.go` | PortfolioService | List and get portfolios with aggregate stats |
| `workspace.go` | WorkspaceService | Get workspace detail with repo summaries |
| `forge.go` | ForgeService | Get forge detail for a single repo |
| `renderer.go` | PageRenderer (impl) | Parse routes, call DataSource, execute templates (WASM) |
| `refresher.go` | Refresher | Background git data refresh (HTTP) |

### `internal/driver/http/` -- HTTP Inbound Driver (5 files)

| File | Purpose |
|------|---------|
| `handler.go` | Route wiring, template loading, theme handling |
| `portfolios.go` | Portfolio list and detail handlers |
| `workspace.go` | Workspace detail handler |
| `forge.go` | Forge detail handler |
| `redirect.go` | Root redirect to `/portfolios` |

### `internal/driver/wasm/` -- WASM Inbound Driver (3 files)

| File | Purpose |
|------|---------|
| `driver.go` | WASM entry point, initializes DOM and router |
| `dom.go` | DOM manipulation helpers (`innerHTML`, element access) |
| `router.go` | Hash-based URL routing for single-page navigation |

## What conventions must I follow?

- **Build tags**: WASM files use `//go:build js && wasm`. Do not add WASM-specific code to files without this tag.
- **Dependency direction**: `driver` -> `controller` -> `adapter` -> `types`. Never import a higher layer.
- **Interfaces in adapter package**: Define new outbound interfaces in `internal/adapter/adapter.go` or a dedicated adapter file.
- **Mock generation**: Run `forge build generate-mocks` after changing interfaces. Mocks live in `internal/util/mocks/`. Configuration is in `.mockery.yml`.
- **Templates (HTTP)**: Server-side templates go in `templates/` (5 files, loaded at runtime).
- **Templates (WASM)**: Client-side templates go in `internal/controller/templates/` (4 files, embedded via `go:embed`).
- **No `go generate`**: Use `forge build generate-mocks` instead.
- **Signed-off-by**: Required on every commit. Use `git commit -s`.
- **Two external dependencies**: `sigs.k8s.io/yaml` and `github.com/stretchr/testify`. Propose new dependencies only when necessary.
