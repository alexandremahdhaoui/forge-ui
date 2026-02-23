# Execution Plan: Refactor to Strict Layered Architecture

## Pre-requisites
- Branch: `claude/go-wasm-forge-ui-kF4cF`
- Validate baseline: `forge test-all` passes (78 tests, 0 lint issues)

## Critical Review Findings (addressed in plan)

1. **CRITICAL: Test helper collision** — `writeIgnoreFile` is defined in both `portfolio_test.go`
   (variadic `...string`) and `workspace_test.go` (single `string`). After merging into `package adapter`,
   the compiler rejects duplicate definitions. **Fix**: Rename to `writePortfolioIgnoreFile` and
   `writeWorkspaceIgnoreFile` respectively during Phase 3d/3e.

2. **CRITICAL: `enrichWorkspaces` must be rewritten** — Currently calls `forgepkg.Load()` (package-level
   function) and takes concrete `*cache.Cache`. Must be rewritten to accept `adapter.Cache` and
   `adapter.ForgeLoader` interfaces. **Fix**: Explicit rewrite in Phase 4e (not just "move").

3. **HIGH: Transitional architecture violation** — Between Phase 3 (packages merged into adapter) and
   Phase 4-5 (controller created, handlers simplified), driver/http temporarily imports adapter directly.
   **Fix**: Phases 3-5 are treated as a single atomic refactoring block. `forge test-all` only required
   at Phase 3f (with temporary adapter imports in driver/http accepted) and Phase 5 step 7 (clean).

4. **HIGH: Dual interface architecture** — `adapter.DataSource` (WASM, no baseDir param) vs new controller
   services (HTTP, with baseDir param). **Fix**: Intentional design. DataSource is for WASM demo path,
   controller services are for HTTP live path. Document this clearly.

5. **MEDIUM: Test files also import `ignore`** — Phase 2 must also update imports in `portfolio_test.go`
   and `workspace_test.go`, not just the production files.

6. **MEDIUM: `helpers_test.go` references `cache.WorkspaceData`** — Phase 1 step 5 is NOT optional.
   The test file uses `cache.WorkspaceData` on 2 lines.

7. **MEDIUM: `RefreshItem` and `Config` types** — Rename `Config` to `RefresherConfig` when moving to
   controller package to avoid ambiguity.

---

## Phase 1: Move types.CacheWorkspaceData into types/

**Goal**: Move `cache.WorkspaceData` into `types/` so it can be referenced by adapter interfaces.

1. Add `CacheWorkspaceData` struct to `internal/types/types.go`:
   ```go
   type CacheWorkspaceData struct {
       Summaries map[string]RepoSummary
       Overviews map[string]RepoOverview
       UpdatedAt time.Time
   }
   ```
2. Update `internal/cache/cache.go`: replace `WorkspaceData` with `types.CacheWorkspaceData`
3. Update `internal/cache/cache_test.go`: replace `cache.WorkspaceData` with `types.CacheWorkspaceData`
4. Update `internal/refresher/refresher.go`: replace `cache.WorkspaceData` with `types.CacheWorkspaceData`
5. Update `internal/driver/http/helpers_test.go`: replace `cache.WorkspaceData` with `types.CacheWorkspaceData`
   (REQUIRED — lines 122, 231 reference `cache.WorkspaceData` directly)
6. Run `forge test-all` -> must pass

---

## Phase 2: Move `ignore/` to `util/ignoreutil/`

**Goal**: Reclassify pure utility package.

1. Create `internal/util/ignoreutil/`
2. Copy `internal/ignore/ignore.go` -> `internal/util/ignoreutil/ignore.go`, change `package ignore` to `package ignoreutil`
3. Copy `internal/ignore/ignore_test.go` -> `internal/util/ignoreutil/ignore_test.go`, change package
4. Update all imports referencing `internal/ignore` (BOTH production AND test files):
   - `internal/portfolio/portfolio.go`
   - `internal/portfolio/portfolio_test.go` (references `ignore.FileName`)
   - `internal/workspace/workspace.go`
   - `internal/workspace/workspace_test.go` (references `ignore.FileName`)
   - Change `ignore.Load` -> `ignoreutil.Load`, `ignore.IsIgnored` -> `ignoreutil.IsIgnored`
   - Change `ignore.FileName` -> `ignoreutil.FileName`
5. Delete `internal/ignore/`
6. Run `forge test-all` -> must pass

---

## Phase 3: Create adapter interfaces + merge packages

**Goal**: Define interfaces in adapter/, move implementations from standalone packages.

### 3a: adapter/cache.go — Cache interface + impl

1. Create `internal/adapter/cache.go`:
   - Define `Cache` interface with `SetWorkspace`, `GetRepoSummary`, `GetRepoOverview`
   - Move implementation from `internal/cache/cache.go` (struct `inMemoryCache`)
   - Constructor `NewCache() Cache`
2. Move `internal/cache/cache_test.go` -> `internal/adapter/cache_test.go`, update package + imports
3. Delete `internal/cache/`
4. Update imports in: `internal/refresher/refresher.go`, `internal/driver/http/handler.go`, `internal/driver/http/helpers.go`, `internal/driver/http/helpers_test.go`

### 3b: adapter/git.go — GitInfo interface + impl

1. Create `internal/adapter/git.go`:
   - Define `GitInfo` interface: `RepoInfo(repoPath string) (types.RepoSummary, error)`
   - Move implementation from `internal/git/git.go` (functions become methods on struct)
   - Constructor `NewGitInfo() GitInfo`
2. Move test files `internal/git/git_test.go` + `testutil_test.go` -> `internal/adapter/git_test.go` + `git_testutil_test.go`, update package + imports
3. Delete `internal/git/`
4. Update imports in: `internal/refresher/refresher.go`

### 3c: adapter/forge.go — ForgeLoader interface + impl

1. Create `internal/adapter/forge.go`:
   - Define `ForgeLoader` interface: `Load(repoPath string) (types.ForgePageData, error)`
   - Move implementation from `internal/forge/forge.go` (function + all private types/converters)
   - Constructor `NewForgeLoader() ForgeLoader`
2. Delete `internal/forge/`
3. Update imports in: `internal/driver/http/helpers.go`, `internal/driver/http/workspace.go`, `internal/driver/http/forge.go`

### 3d: adapter/workspace.go — WorkspaceDiscovery interface + impl (MUST come before portfolio)

1. Create `internal/adapter/workspace.go`:
   - Define `WorkspaceDiscovery` interface: `List(basedir string)`, `Get(basedir, name string)`
   - Move implementation from `internal/workspace/workspace.go`
   - Constructor `NewWorkspaceDiscovery() WorkspaceDiscovery`
2. Move `internal/workspace/workspace_test.go` -> `internal/adapter/workspace_test.go`, update package + imports
   - **CRITICAL**: Rename `writeIgnoreFile` -> `writeWorkspaceIgnoreFile` (collides with portfolio test helper)
   - Rename `mkdirWithFile` -> `workspaceMkdirWithFile` (preventive disambiguation)
3. Delete `internal/workspace/`
4. Update imports in: `internal/refresher/refresher.go`, `internal/driver/http/workspace.go`, `internal/portfolio/portfolio.go`

### 3e: adapter/portfolio.go — PortfolioDiscovery interface + impl

1. Create `internal/adapter/portfolio.go`:
   - Define `PortfolioDiscovery` interface: `List(baseDir string)`, `Get(baseDir, name string)`
   - Move implementation from `internal/portfolio/portfolio.go`
   - Constructor `NewPortfolioDiscovery(ws WorkspaceDiscovery) PortfolioDiscovery`
   - NOTE: portfolio.List calls workspace.List — use interface injection:
     the `portfolioDiscovery` struct takes a `WorkspaceDiscovery` interface field
   - Update internal imports: `ignore` -> `ignoreutil`, `workspace.List(...)` -> `pd.ws.List(...)`
2. Move `internal/portfolio/portfolio_test.go` -> `internal/adapter/portfolio_test.go`, update package + imports
   - **CRITICAL**: Rename `writeIgnoreFile` -> `writePortfolioIgnoreFile` (collides with workspace test helper)
   - Rename `createWorkspace` -> `portfolioCreateWorkspace` (preventive disambiguation)
3. Delete `internal/portfolio/`
4. Update imports in: `internal/refresher/refresher.go`, `internal/driver/http/portfolios.go`

### 3f: Validate
Run `forge test-all` -> must pass. All old tests now run from adapter/ package.

**NOTE**: At this point, `driver/http` temporarily imports `adapter` directly (architectural violation).
This is an accepted transitional state. The violation is fixed in Phases 4-5 when controller services
are created and driver/http is simplified to only import controller. Phases 3-5 form an atomic block.

---

## Phase 4: Create controller services (extract business logic from driver/http)

**Goal**: Move enrichment, sorting, stat computation from driver/http into controller services.

**IMPORTANT**: Controller services return DATA ONLY (types.XxxPageData structs). They do NOT render
templates. Template rendering stays in driver/http (filesystem templates) and controller/renderer.go
(embedded templates for WASM). The controller services encapsulate business logic: enrichment, sorting,
stat computation, heatmap building.

### 4a: controller/portfolio.go — PortfolioService

1. Create `internal/controller/portfolio.go`:
   - Define `PortfolioService` interface: `ListPortfolios(baseDir, sort string)`, `GetPortfolio(baseDir, name, sort string)`
   - Implement `portfolioService` struct holding `adapter.PortfolioDiscovery`, `adapter.Cache`, `adapter.ForgeLoader`
   - Move logic from `driver/http/portfolios.go` HandlePortfolios + HandlePortfolio:
     - Call portfolio.List/Get
     - Call enrichWorkspaces (cache enrichment + forge heatmap)
     - Call rewriteRepoLinks
     - Sort by time/name
     - Compute stats
   - Constructor `NewPortfolioService(...) PortfolioService`
   - Note: DarkMode and HomeURL are NOT set here — those are HTTP driver concerns (cookie/config)

### 4b: controller/workspace.go — WorkspaceService

1. Create `internal/controller/workspace.go`:
   - Define `WorkspaceService` interface: `GetWorkspace(baseDir, portfolio, workspace, sort string)`
   - Implement `workspaceService` struct holding `adapter.WorkspaceDiscovery`, `adapter.Cache`, `adapter.ForgeLoader`
   - Move logic from `driver/http/workspace.go` HandleWorkspace:
     - Call workspace.Get
     - Enrich repos with cached git data
     - Build forge heatmap
     - Sort, compute stats
   - Constructor `NewWorkspaceService(...) WorkspaceService`

### 4c: controller/forge.go — ForgeService

1. Create `internal/controller/forge.go`:
   - Define `ForgeService` interface: `GetForge(baseDir, portfolio, workspace, repo string)`
   - Implement `forgeService` struct holding `adapter.ForgeLoader`
   - Move logic from `driver/http/forge.go` HandleForge:
     - Call forge.Load
     - Compute test statistics
     - Build stage status map
   - Constructor `NewForgeService(...) ForgeService`

### 4d: controller/refresher.go — Refresher

1. Create `internal/controller/refresher.go`:
   - Define `Refresher` interface: `Start()`, `Stop()`
   - Move implementation from `internal/refresher/refresher.go`
   - Change direct package calls to adapter interface calls:
     - `portfolio.List(...)` -> `r.portfolioDisc.List(...)`
     - `workspace.Get(...)` -> `r.workspaceDisc.Get(...)`
     - `gitpkg.RepoInfo(...)` -> `r.gitInfo.RepoInfo(...)`
     - `r.cache.SetWorkspace(...)` -> `r.cache.SetWorkspace(...)`
   - Constructor `NewRefresher(cache adapter.Cache, git adapter.GitInfo, portfolio adapter.PortfolioDiscovery, workspace adapter.WorkspaceDiscovery, cfg RefresherConfig) Refresher`
   - Rename `Config` -> `RefresherConfig` and `RefreshItem` -> `refreshItem` (unexport, internal detail)
2. Move `internal/refresher/refresher_test.go` -> `internal/controller/refresher_test.go`
   - Rename `runGitCmd` helper -> `refresherRunGitCmd` (avoid potential future collisions)
3. Delete `internal/refresher/`

### 4e: Rewrite and move helper functions (NOT a simple move)
- **REWRITE** `enrichWorkspaces`: change signature from `enrichWorkspaces(workspaces, *cache.Cache, cacheKeyFn)`
  to `enrichWorkspaces(workspaces, adapter.Cache, adapter.ForgeLoader, cacheKeyFn)`.
  Replace `forgepkg.Load(repo.Path)` calls with `forgeLoader.Load(repo.Path)` calls.
  Move rewritten function into `controller/portfolio.go` (used by both ListPortfolios and GetPortfolio).
- Move `rewriteRepoLinks` to `controller/portfolio.go` (only used for portfolio pages) or make shared.
- Move `maxCommitTime` to `controller/portfolio.go`.
- Delete `driver/http/helpers.go`
- **REWRITE** `driver/http/helpers_test.go` tests -> `internal/controller/helpers_test.go`:
  Replace `cache.New()` with `adapter.NewCache()`, replace `cache.WorkspaceData` with `types.CacheWorkspaceData`.
  The enrichment tests that exercise forge heatmap logic should use `adapter.NewForgeLoader()` or mock.

### 4f: Validate
Run `forge test-all` -> must pass.

---

## Phase 5: Simplify driver/http to delegate to controller

**Goal**: HTTP handlers become thin: parse request, call controller, render template.

1. Rewrite `driver/http/handler.go`:
   ```go
   type Handler struct {
       BaseDir          string
       Templates        map[string]*template.Template
       HomeURL          string
       PortfolioService controller.PortfolioService
       WorkspaceService controller.WorkspaceService
       ForgeService     controller.ForgeService
   }
   func New(baseDir, templateDir string, ps controller.PortfolioService, ws controller.WorkspaceService, fs controller.ForgeService) (*Handler, error)
   ```
   - Remove `Cache` field (no longer needed)
   - Add controller service fields

2. Simplify `driver/http/portfolios.go`:
   - HandlePortfolios: parse sort from query, call `h.PortfolioService.ListPortfolios(h.BaseDir, sortMode)`, set DarkMode/HomeURL, render
   - HandlePortfolio: parse name+sort, call `h.PortfolioService.GetPortfolio(h.BaseDir, name, sortMode)`, set DarkMode/HomeURL, render

3. Simplify `driver/http/workspace.go`:
   - HandleWorkspace: parse params, call `h.WorkspaceService.GetWorkspace(...)`, set DarkMode/HomeURL, render

4. Simplify `driver/http/forge.go`:
   - HandleForge: parse params, call `h.ForgeService.GetForge(...)`, set DarkMode/HomeURL, render

5. Delete `driver/http/helpers.go` (already moved to controller in 4e)

6. Update `cmd/forge-ui/main.go`:
   - Wire adapters -> controllers -> driver
   - Remove direct cache/refresher imports

7. Run `forge test-all` -> must pass

---

## Phase 6: Add testify dependency + configure forge go-gen-mocks

**Goal**: Set up mock generation infrastructure.

1. Add testify dependency:
   ```bash
   go get github.com/stretchr/testify@latest
   go mod tidy
   ```

2. Update `forge.yaml` — add mock generation engine + build step:
   ```yaml
   engines:
     - alias: generate-mocks
       type: builder
       builder:
         - engine: go://generic-builder
           spec:
             command: "rm"
             args: ["-rf", "./internal/util/mocks"]
         - engine: go://go-gen-mocks

   build:
     # ... existing entries ...
     - name: generate-mocks
       src: .
       engine: alias://generate-mocks
   ```

3. Run `forge build generate-mocks`
4. Verify generated files in `internal/util/mocks/mockadapter/` and `internal/util/mocks/mockcontroller/`
5. Run `forge test-all` -> must pass

---

## Phase 7: Write mock-based tests for controller services

**Goal**: Test controller business logic using generated mocks. Target: 80%+ coverage per controller file.

### 7a: controller/portfolio_test.go
Tests using mockadapter.Cache, mockadapter.PortfolioDiscovery, mockadapter.ForgeLoader:
- TestListPortfolios_SortByTime
- TestListPortfolios_SortByName
- TestListPortfolios_ErrorFromDiscovery
- TestGetPortfolio_Success
- TestGetPortfolio_NotFound
- TestGetPortfolio_WithCacheEnrichment
- TestGetPortfolio_WithForgeHeatmap

### 7b: controller/workspace_test.go
Tests using mockadapter.WorkspaceDiscovery, mockadapter.Cache, mockadapter.ForgeLoader:
- TestGetWorkspace_Success
- TestGetWorkspace_WithCachedGitData
- TestGetWorkspace_WithForgeHeatmap
- TestGetWorkspace_SortByTime
- TestGetWorkspace_NotFound

### 7c: controller/forge_test.go
Tests using mockadapter.ForgeLoader:
- TestGetForge_Success
- TestGetForge_WithTestReports
- TestGetForge_WithCoverage
- TestGetForge_LoadError
- TestGetForge_DefaultPortfolio

### 7d: controller/refresher_test.go (refactored)
Replace real git/filesystem calls with mocks:
- TestRefresher_Start_InitialRefresh
- TestRefresher_Stop
- TestRefresher_RefreshPopulatesCache
- TestRefresher_PortfolioListError
- TestRefresher_WorkspaceGetError

### 7e: Validate
Run `forge test-all` -> verify coverage is >= 80%

---

## Phase 8: Write mock-based tests for driver/http

**Goal**: Test HTTP handlers using generated mockcontroller mocks. Target: 80%+ coverage.

### 8a: driver/http/handler_test.go
Tests using mockcontroller.PortfolioService, mockcontroller.WorkspaceService, mockcontroller.ForgeService:
- TestHandlePortfolios_Success
- TestHandlePortfolios_SortParam
- TestHandlePortfolios_Error
- TestHandlePortfolio_Success
- TestHandlePortfolio_MissingName
- TestHandlePortfolio_Error
- TestHandleWorkspace_Success
- TestHandleWorkspace_MissingParams
- TestHandleWorkspace_Error
- TestHandleForge_Success
- TestHandleForge_MissingParams
- TestHandleForge_Error
- TestHandleRedirect
- TestHandleToggleTheme_LightToDark
- TestHandleToggleTheme_DarkToLight
- TestIsDarkMode

### 8b: adapter/forge_test.go
Tests for ForgeLoader with temp YAML files:
- TestForgeLoad_SpecOnly
- TestForgeLoad_FullArtifactStore
- TestForgeLoad_NoForgeYaml
- TestForgeLoad_InvalidYaml
- TestForgeLoad_TestReportsSorted
- TestForgeLoad_TestEnvsSorted

### 8c: Validate
Run `forge test-all` -> verify total coverage >= 80%

---

## Phase 9: Verify strict layer isolation

**Goal**: Ensure no cross-layer imports violate the dependency rules.

1. Check driver/ does NOT import adapter/:
   ```bash
   grep -r '"github.com/alexandremahdhaoui/forge-ui/internal/adapter"' internal/driver/
   ```
   Must return NOTHING (driver only imports controller)

2. Check controller/ does NOT import driver/:
   ```bash
   grep -r '"github.com/alexandremahdhaoui/forge-ui/internal/driver' internal/controller/
   ```
   Must return NOTHING

3. Check adapter/ does NOT import controller/ or driver/:
   ```bash
   grep -r '"github.com/alexandremahdhaoui/forge-ui/internal/controller' internal/adapter/
   grep -r '"github.com/alexandremahdhaoui/forge-ui/internal/driver' internal/adapter/
   ```
   Must return NOTHING

4. Check types/ imports NOTHING from internal:
   ```bash
   grep -r '"github.com/alexandremahdhaoui/forge-ui/internal/' internal/types/
   ```
   Must return NOTHING

5. Check util/ does NOT import controller/adapter/driver:
   ```bash
   grep -r '"github.com/alexandremahdhaoui/forge-ui/internal/controller' internal/util/
   grep -r '"github.com/alexandremahdhaoui/forge-ui/internal/adapter' internal/util/
   grep -r '"github.com/alexandremahdhaoui/forge-ui/internal/driver' internal/util/
   ```
   Must return NOTHING

---

## Phase 10: Final validation + commit + push

1. Run `forge test-all` -> all tests pass, lint clean, all builds compile
2. Verify coverage >= 80%:
   ```bash
   go test -coverprofile=cover.out ./internal/...
   go tool cover -func=cover.out | grep total
   ```
3. Commit: `refactor: strict layered architecture with adapter/controller/driver + generated mocks`
4. Push to `origin/claude/go-wasm-forge-ui-kF4cF`

---

## Summary of Changes

| Phase | What Changes | Risk |
|-------|-------------|------|
| 1 | Move CacheWorkspaceData to types | Low — mechanical |
| 2 | Move ignore -> util/ignoreutil | Low — rename + imports |
| 3 | Create adapter interfaces, merge 5 packages into adapter | Medium — lots of file moves |
| 4 | Create controller services, move business logic from driver/http | High — logic extraction |
| 5 | Simplify driver/http to thin handlers | Medium — depends on phase 4 |
| 6 | Add testify, configure forge go-gen-mocks | Low — dependency + config |
| 7 | Write controller tests with mocks | Medium — new test code |
| 8 | Write driver/http + forge tests | Medium — new test code |
| 9 | Verify layer isolation | Low — grep checks |
| 10 | Commit + push | Low |

### Estimated file count
- ~15 files moved/renamed
- ~8 new files created
- ~10 files modified (imports)
- ~6 old directories deleted
