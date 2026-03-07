# Contributing to forge-ui

**Contribute to forge-ui -- a Go web dashboard and embedded terminal for Forge workspaces and portfolios.**

## Quick start

```sh
git clone https://github.com/alexandremahdhaoui/forge-ui.git
cd forge-ui
forge build
forge test-all
./build/bin/forge-frontend -workspaces ~/workspaces
# REST API at http://localhost:8081/api/v1/
```

Requirements: Go 1.25.0, Node.js (for xterm.js build), [forge](https://github.com/alexandremahdhaoui/forge) CLI, git. For e2e tests: Docker, Kind.

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
| `forge test-all` | Build all targets, then run lint + unit + e2e |
| `forge test-run lint` | Run golangci-lint |
| `forge test-run unit` | Run Go unit tests |
| `forge test-run e2e` | Run end-to-end tests (requires Kind cluster) |
| `forge build` | Build all 11 targets |
| `forge build forge-frontend` | Build REST API server binary |
| `forge build forge-ui-wasm` | Build WASM dashboard binary |
| `forge build forge-terminal-wasm` | Build WASM terminal binary |
| `forge build forge-wss-proxy` | Build WebSocket-to-SSH proxy binary |
| `forge build forge-ui-tui` | Build TUI dashboard binary (bubbletea) |
| `forge build generate-mocks` | Regenerate test mocks |
| `forge build generate-rest-api` | Regenerate OpenAPI REST handler |

Always run `forge test-all` before submitting a PR. Running `go build` alone is not sufficient.

## How is the project structured?

```
forge-ui/
  api/                       OpenAPI 3.0.3 spec (forge-ui.v1.yaml)
  cmd/
    forge-frontend/          REST API server entry point
    forge-ui-wasm/           WASM dashboard browser entry point
    forge-ui-tui/            TUI dashboard entry point (bubbletea)
    forge-terminal-wasm/     WASM terminal emulator entry point
    forge-wss-proxy/         WebSocket-to-SSH proxy entry point
  internal/
    adapter/                 Dashboard outbound ports: git, filesystem, cache, api (11 files)
    controller/              Dashboard business logic: portfolio, workspace, forge, refresher (6 files)
      templates/             Embedded HTML templates for WASM renderer (5 files)
    driver/
      rest/                  REST API inbound driver: OpenAPI codegen + handler (2 files)
      wasm/                  WASM dashboard inbound driver: DOM, router, driver (3 files)
    terminal/
      adapter/               Terminal outbound ports: TerminalIO, SSHClient, KeyStore (7 files)
      controller/            Terminal business logic: SSH session controller (2 files)
      driver/wasm/           WASM terminal inbound driver (1 file)
      keygen/                Ed25519 SSH key generation (1 file)
      types/                 Terminal domain model (1 file, 6 types)
    types/                   Dashboard domain model: pure data structs (1 file)
    util/
      ignoreutil/            .forge-workspace-ignore parser
      mocks/                 Generated test mocks (mockery)
    wssproxy/                WebSocket-to-SSH proxy server + key store (2 files)
  web/                       Static assets: index.html, wasm_exec.js, terminal/
  xterm/                     xterm.js source + rollup config
  charts/
    forge-workspace/         Helm chart for workspace container
    forge-wss-proxy/         Helm chart for wss-proxy
  containers/
    forge-frontend/          Containerfile for REST API server
    forge-ui-wasm/           Containerfile + nginx.conf for WASM dashboard
    forge-workspace/         Containerfile + entrypoint + authorized-keys script
    forge-wss-proxy/         Multi-stage Containerfile for wss-proxy
  test/e2e/                  End-to-end tests + SSH fixtures
  hack/                      Development scripts (run.sh, cleanup.sh)
  forge.yaml                 Build and test configuration
```

Dependency direction: `driver` -> `controller` -> `adapter` -> `types`. Never import a higher layer.

## What does each package do?

### `internal/adapter/` -- Outbound Ports (11 files)

| File | Interface / Impl | Purpose |
|------|-----------------|---------|
| `adapter.go` | DataSource, MetaPlanLoader, RepoPlanLoader, WsConfigLoader, PortfolioConfigLoader | Core interface definitions |
| `cache.go` | Cache | In-memory repo data cache (`sync.RWMutex`) |
| `api.go` | APIDataSource | REST API client for WASM build (`js/wasm` only) |
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

### `internal/driver/rest/` -- REST API Inbound Driver (2 files)

| File | Purpose |
|------|---------|
| `handler.go` | REST API handler: implements StrictServerInterface |
| `zz_generated.oapi-codegen.go` | Generated OpenAPI strict server (do not edit) |

### `internal/driver/wasm/` -- WASM Dashboard Inbound Driver (3 files)

| File | Purpose |
|------|---------|
| `driver.go` | WASM entry point, initializes DOM and router |
| `dom.go` | DOM manipulation helpers (`innerHTML`, element access) |
| `router.go` | Hash-based URL routing for single-page navigation |

### `internal/terminal/` -- Terminal Subsystem

| Package | Files (non-test) | Purpose |
|---------|-----------------|---------|
| `adapter/` | 7 | TerminalIO, SSHClient, KeyRegistrar, KeyStore, IndexedDB, WebSocket |
| `controller/` | 2 | SSH session controller |
| `driver/wasm/` | 1 | WASM terminal entry point |
| `keygen/` | 1 | Ed25519 key pair generation |
| `types/` | 1 | Terminal domain types (6 types) |

### `internal/wssproxy/` -- WebSocket-to-SSH Proxy

| File | Purpose |
|------|---------|
| `proxy.go` | WebSocket upgrade, SSH connection, bidirectional pipe |
| `keystore.go` | In-memory key store for SSH key provisioning |

## What conventions must I follow?

- **Build tags**: WASM files use `//go:build js && wasm`. Do not add WASM-specific code to files without this tag.
- **Dependency direction**: `driver` -> `controller` -> `adapter` -> `types`. Never import a higher layer.
- **Interfaces in adapter package**: Define new outbound interfaces in `internal/adapter/adapter.go` or a dedicated adapter file.
- **Mock generation**: Run `forge build generate-mocks` after changing interfaces. Mocks live in `internal/util/mocks/`. Configuration is in `.mockery.yml`.
- **OpenAPI codegen**: The REST API handler is generated from `api/forge-ui.v1.yaml`. Run `forge build generate-rest-api` after modifying the spec. Do not edit `zz_generated.oapi-codegen.go` manually.
- **Terminal subsystem**: Terminal packages live in `internal/terminal/`. They follow the same hexagonal pattern (types -> adapter -> controller -> driver) but are independent from the dashboard packages.
- **Templates (WASM)**: Client-side templates go in `internal/controller/templates/` (5 files, embedded via `go:embed`).
- **No `go generate`**: Use `forge build generate-mocks` instead.
- **Signed-off-by**: Required on every commit. Use `git commit -s`.
- **Five direct dependencies**: `kin-openapi`, `oapi-codegen/runtime`, `testify`, `x/crypto`, `sigs.k8s.io/yaml`. One JavaScript library: xterm.js (bundled via rollup). Propose new dependencies only when necessary.
