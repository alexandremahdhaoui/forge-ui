# Research: Refactor to Strict Layered Architecture

## 1. Current Package Inventory

| Package | Files | Purpose | Depends On (internal) |
|---------|-------|---------|----------------------|
| `types` | types.go | 25+ domain DTOs | (none) |
| `adapter` | adapter.go, demo.go, demo_test.go | `DataSource` interface + demo impl | types |
| `controller` | controller.go, renderer.go, renderer_test.go | `PageRenderer` interface + template renderer | adapter |
| `driver/http` | 7 files | HTTP handlers (portfolio, workspace, forge pages) | cache, forge, portfolio, workspace, types |
| `driver/wasm` | 3 files (build-tagged js+wasm) | WASM DOM driver with hash routing | controller |
| `cache` | cache.go, cache_test.go | Thread-safe RWMutex store for git data | types |
| `forge` | forge.go | Parses forge.yaml + artifact-store.yaml | types |
| `git` | git.go, git_test.go, testutil_test.go | Runs git commands, extracts repo metadata | types |
| `ignore` | ignore.go, ignore_test.go | Reads .forge-workspace-ignore files | (none) |
| `portfolio` | portfolio.go, portfolio_test.go | Discovers portfolios from filesystem | ignore, workspace, types |
| `workspace` | workspace.go, workspace_test.go | Discovers workspaces + repos from filesystem | ignore, types |
| `refresher` | refresher.go, refresher_test.go | Background worker: discover workspaces, run git, populate cache | cache, git, portfolio, workspace, types |

## 2. Current Cross-Layer Dependency Graph

```
types           <- foundational, no deps
ignore          <- pure utility, no internal deps
git             <- types
cache           <- types
forge           <- types
workspace       <- types, ignore
portfolio       <- types, ignore, workspace
refresher       <- types, cache, git, portfolio, workspace
adapter         <- types
controller      <- adapter
driver/wasm     <- controller
driver/http     <- types, cache, forge, portfolio, workspace (BYPASSES controller/adapter)
```

### Problem: driver/http LEAKS through layers
`driver/http` directly calls `portfolio.List`, `portfolio.Get`, `workspace.Get`, `forgepkg.Load`,
and `cache.GetRepoSummary/GetRepoOverview`. It completely bypasses the adapter/controller abstraction.
It also contains significant business logic (enrichment, sorting, heatmap computation).

## 3. Current Test Coverage

| Package | Coverage | Notes |
|---------|----------|-------|
| types | N/A | No test files (DTOs only) |
| adapter | 100.0% | Demo data source fully tested |
| controller | 83.6% | Template rendering tested |
| driver/http | 11.4% | Only helpers tested (rewriteRepoLinks, enrichWorkspaces, maxCommitTime) |
| driver/wasm | N/A | Build-tagged, cannot test on host |
| cache | 94.4% | Good concurrent access tests |
| forge | 0.0% | ZERO tests |
| git | 97.8% | Excellent, uses real git repos |
| ignore | 100.0% | Complete |
| portfolio | 85.7% | Good filesystem-based tests |
| workspace | 83.1% | Good filesystem-based tests |
| refresher | 71.2% | Tested but tightly coupled |
| **Total** | **51.6%** | Target: 80%+ |

### Coverage Gaps (to reach 80%)
1. **forge**: 0% -> needs tests for Load(), convertSpec, convertArtifacts, convertTestReports, convertTestEnvs
2. **driver/http**: 11.4% -> needs mock-based tests for all HTTP handlers
3. **refresher**: 71.2% -> needs mock-based tests to avoid real git/filesystem

## 4. Shaper Architecture Reference (github.com/alexandremahdhaoui/shaper)

### Directory Structure
```
internal/
  adapter/       <- interfaces + implementations for external services
  controller/    <- business logic interfaces + implementations
  driver/        <- inbound drivers (server, tftp, webhook)
  types/         <- shared domain types
  k8s/           <- k8s-specific helpers
  util/
    mocks/       <- auto-generated mocks (mockery/testify)
      mockadapter/    <- zz_generated.{InterfaceName}.go
      mockcontroller/ <- zz_generated.{InterfaceName}.go
      mockclient/     <- zz_generated.{InterfaceName}.go
    testutil/    <- test helpers
    certutil/    <- certificate utilities
    httputil/    <- http utilities
    ...
```

### Pattern Rules
1. **Interfaces live in the package that DEFINES the contract** (adapter defines adapter interfaces, controller defines controller interfaces)
2. **Each interface file has**: INTERFACES section, CONSTRUCTORS section, CONCRETE IMPLEMENTATION section
3. **Dependency direction**: driver -> controller -> adapter -> types
4. **Drivers depend on controller interfaces** (e.g., `server.New(ipxe controller.IPXE, config controller.Content)`)
5. **Controllers depend on adapter interfaces** (e.g., `NewContent(profile adapter.Profile, mux ResolveTransformerMux)`)
6. **Mocks go in `util/mocks/mock{layer}/`** with `zz_generated.{InterfaceName}.go` naming
7. **Mock library**: mockery/testify (`github.com/vektra/mockery` + `github.com/stretchr/testify/mock`)

### Key Architectural Insight
The pattern is **hexagonal architecture** (ports & adapters):
- **Adapter** = outbound port (how the app talks to external world: DB, filesystem, APIs)
- **Controller** = application service / use case (business logic)
- **Driver** = inbound port (how the external world talks to the app: HTTP, WASM, CLI)
- **Types** = shared domain model (DTOs, value objects)
- **Util** = cross-cutting utilities

## 5. Package Reclassification Analysis

### Current packages -> target layer:

| Current Package | Target Layer | Rationale |
|----------------|-------------|-----------|
| `types` | **types** | Already correct |
| `adapter` | **adapter** | Already correct, but `DataSource` is WASM-only |
| `controller` | **controller** | Already correct, but `PageRenderer` is WASM-only |
| `cache` | **adapter** (as `adapter.Cache` interface) | Cache is an outbound port — drivers/controllers read/write through it |
| `forge` | **adapter** (as `adapter.ForgeLoader` interface) | Reads forge.yaml files from filesystem — outbound I/O |
| `git` | **adapter** (as `adapter.GitInfo` interface) | Runs git commands — outbound I/O |
| `ignore` | **util/ignoreutil** | Pure utility (pattern matching) — no domain logic |
| `portfolio` | **adapter** (as `adapter.PortfolioDiscovery` interface) | Discovers portfolios from filesystem — outbound I/O |
| `workspace` | **adapter** (as `adapter.WorkspaceDiscovery` interface) | Discovers workspaces from filesystem — outbound I/O |
| `refresher` | **controller** (as `controller.Refresher` interface) | Orchestrates: discover -> fetch git info -> cache. This is business logic |
| `driver/http` | **driver/http** | Already correct, but needs to use controller/adapter interfaces |
| `driver/wasm` | **driver/wasm** | Already correct |

### New adapter interfaces needed:

```go
// adapter/cache.go
type Cache interface {
    SetWorkspace(name string, data CacheWorkspaceData)
    GetRepoSummary(workspace, repo string) (types.RepoSummary, bool)
    GetRepoOverview(workspace, repo string) (types.RepoOverview, bool)
}

// adapter/forge.go
type ForgeLoader interface {
    Load(repoPath string) (types.ForgePageData, error)
}

// adapter/git.go
type GitInfo interface {
    RepoInfo(repoPath string) (types.RepoSummary, error)
}

// adapter/portfolio.go
type PortfolioDiscovery interface {
    List(baseDir string) ([]types.PortfolioSummary, error)
    Get(baseDir, name string) (types.PortfolioPageData, error)
}

// adapter/workspace.go
type WorkspaceDiscovery interface {
    List(basedir string) ([]types.WorkspaceSummary, error)
    Get(basedir, name string) (types.WorkspacePageData, error)
}
```

### New controller interfaces/services needed:

The HTTP driver currently has business logic inline. This needs to move to controller:

```go
// controller/portfolio.go
type PortfolioService interface {
    ListPortfolios(baseDir, sort string) (types.PortfoliosPageData, error)
    GetPortfolio(baseDir, name, sort string) (types.PortfolioPageData, error)
}

// controller/workspace.go
type WorkspaceService interface {
    GetWorkspace(baseDir, portfolio, workspace, sort string) (types.WorkspacePageData, error)
}

// controller/forge.go
type ForgeService interface {
    GetForge(baseDir, portfolio, workspace, repo string) (types.ForgePageData, error)
}

// controller/refresher.go
type Refresher interface {
    Start()
    Stop()
}
```

## 6. forge go-gen-mocks

### What it is
`go://go-gen-mocks` is a forge build engine that wraps **mockery** (`github.com/vektra/mockery`) with **testify/mock** (`github.com/stretchr/testify/mock`).

### How it works in shaper
1. Added as the LAST step in `generate-all` engine alias
2. Before running, old mocks are deleted: `rm -rf ./internal/util/mocks`
3. Engine is invoked with no `spec:` (auto-discovers interfaces)
4. Generates to `internal/util/mocks/mock{package}/zz_generated.{InterfaceName}.go`

### forge.yaml configuration
```yaml
engines:
  - alias: generate-all
    type: builder
    builder:
      # ... other generation steps ...
      - engine: go://generic-builder
        spec:
          command: "rm"
          args: ["-rf", "./internal/util/mocks"]
      - engine: go://go-gen-mocks

build:
  - name: generate-all
    src: .
    engine: alias://generate-all
```

### Dependencies needed in go.mod
```
github.com/stretchr/testify  (for mock.Mock, assert, require)
github.com/vektra/mockery    (for //go:generate if needed)
```

### Generated mock pattern
```go
// Code generated by mockery; DO NOT EDIT.
package mockadapter

import (
    mock "github.com/stretchr/testify/mock"
)

type MockInterfaceName struct {
    mock.Mock
}

func NewMockInterfaceName(t interface{ mock.TestingT; Cleanup(func()) }) *MockInterfaceName {
    mock := &MockInterfaceName{}
    mock.Mock.Test(t)
    t.Cleanup(func() { mock.AssertExpectations(t) })
    return mock
}

// Each interface method gets:
// 1. A mock method implementation
// 2. An Expecter type
// 3. A Call type with Run/Return/RunAndReturn
```

### Output structure
```
internal/util/mocks/
  mockadapter/
    zz_generated.Cache.go
    zz_generated.ForgeLoader.go
    zz_generated.GitInfo.go
    zz_generated.PortfolioDiscovery.go
    zz_generated.WorkspaceDiscovery.go
    zz_generated.DataSource.go
  mockcontroller/
    zz_generated.PortfolioService.go
    zz_generated.WorkspaceService.go
    zz_generated.ForgeService.go
    zz_generated.PageRenderer.go
    zz_generated.Refresher.go
```

## 7. Key Risks and Constraints

1. **driver/http is tightly coupled** — it directly calls 5 packages. Refactoring requires extracting business logic into controller services
2. **Build tags** — driver/wasm files have `//go:build js && wasm`. Cannot test on host
3. **Real I/O in tests** — portfolio, workspace, git tests use real filesystems/git repos. Mock-based tests will be additive (not replacing existing tests)
4. **forge go-gen-mocks requires mockery+testify deps** — need to add to go.mod
5. **Refresher coupling** — refresher directly calls portfolio.List, workspace.Get, git.RepoInfo. Needs adapter interfaces
6. **Cache types** — cache.WorkspaceData is a concrete type containing types.RepoSummary maps. May need to stay in types or become part of the adapter interface
7. **Template embedding** — controller uses `//go:embed templates/*.html` for WASM path. driver/http uses `template.ParseFiles` from filesystem for HTTP path. These are TWO separate template systems that must NOT be unified. Controller services return data, not rendered HTML.
8. **Portfolio->workspace dependency** — portfolio.List() calls workspace.List(). After merge into adapter/, the portfolioDiscovery struct should take a WorkspaceDiscovery interface (interface injection) to avoid tight coupling within the package
