# Execution Plan: forge-ui WASM Refactor

## Overview

Refactor forge-ui WASM from wasip1 (stdin/stdout) to GOOS=js (direct DOM access),
restructure to adapter/controller/driver pattern following shaper conventions.

## Pre-Conditions

- Go 1.25.1 at `/usr/local/go1.25.1/bin`
- Forge CLI at `~/go/bin/forge`
- Existing passing `forge test-all`

---

## Phase 1: Create `internal/types/` (rename from model/)

### Step 1.1: Copy model to types

1. Create `internal/types/` directory
2. Copy `internal/model/model.go` to `internal/types/types.go`
3. Change `package model` to `package types`
4. Update ALL imports across the codebase:
   - `internal/cache/` → change `model.` to `types.`
   - `internal/forge/` → change `model.` to `types.`
   - `internal/git/` → change `model.` to `types.`
   - `internal/handler/` → change `model.` to `types.`
   - `internal/portfolio/` → change `model.` to `types.`
   - `internal/refresher/` → change `model.` to `types.`
   - `internal/workspace/` → change `model.` to `types.`
   - `cmd/forge-ui/main.go` (if it imports model)
5. Delete `internal/model/` directory
6. Run `forge test-all` — must pass

**Risk**: Many files import model. This is a mechanical find-and-replace.

---

## Phase 2: Create `internal/adapter/`

### Step 2.1: Define adapter interfaces

Create `internal/adapter/adapter.go`:

```go
package adapter

import "github.com/alexandremahdhaoui/forge-ui/internal/types"

// DataSource provides page data for rendering.
type DataSource interface {
    ListPortfolios(sort string) (types.PortfoliosPageData, error)
    GetPortfolio(name, sort string) (types.PortfolioPageData, error)
    GetWorkspace(portfolio, workspace, sort string) (types.WorkspacePageData, error)
    GetForge(portfolio, workspace, repo string) (types.ForgePageData, error)
}
```

### Step 2.2: Create demo adapter

Create `internal/adapter/demo.go`:

1. Move the demo data from `internal/render/data.go` into this file
2. Implement `DataSource` interface with methods that look up demo data by key
3. `NewDemoDataSource() DataSource` constructor

### Step 2.3: Verify build

Run `go build ./internal/adapter/...` — must compile.

---

## Phase 3: Create `internal/controller/`

### Step 3.1: Define controller interface and implementation

Create `internal/controller/controller.go`:

```go
package controller

// PageRenderer renders a route to HTML.
type PageRenderer interface {
    Render(route string) (string, error)
}
```

Create `internal/controller/renderer.go`:

1. Move route parsing logic from `internal/render/command.go` (ParseInput, Input struct)
2. Move route dispatch logic from `internal/render/render.go` (splitRoute, executeRoute)
3. Move template loading from `internal/render/render.go` (go:embed, template.ParseFS)
4. The renderer struct holds a `adapter.DataSource` and embedded templates
5. Constructor: `NewPageRenderer(ds adapter.DataSource) PageRenderer`

**Templates**: The controller embeds the templates via `//go:embed templates/*.html`
and uses them for rendering. The template files are placed in
`internal/controller/templates/`.

### Step 3.2: Move templates

1. Move `internal/render/templates/*.html` to `internal/controller/templates/`
2. Templates already use hash-based `<a href="#/...">` links — no changes needed

### Step 3.3: Verify build

Run `go build ./internal/controller/...` — must compile.
Run `go test ./internal/controller/...` — write tests (ported from render tests).

---

## Phase 4: Create `internal/driver/wasm/`

### Step 4.1: Create WASM driver

Create `internal/driver/wasm/dom.go`:

```go
package wasm

import "syscall/js"

// DOM helper functions
func getElementById(id string) js.Value { ... }
func setInnerHTML(el js.Value, html string) { ... }
func getHash() string { ... }
func setHash(h string) { ... }
func getLocalStorage(key, fallback string) string { ... }
func setLocalStorage(key, value string) { ... }
```

Create `internal/driver/wasm/router.go`:

```go
package wasm

// Router handles hash-based navigation
type Router struct {
    onNavigate func(route string)
    cb         js.Func
}

func NewRouter(onNavigate func(string)) *Router { ... }
func (r *Router) Start() { ... }     // Register hashchange listener
func (r *Router) Release() { ... }   // Release js.Func
```

Create `internal/driver/wasm/driver.go`:

```go
package wasm

import (
    "github.com/alexandremahdhaoui/forge-ui/internal/controller"
    "syscall/js"
)

type Driver struct {
    renderer controller.PageRenderer
    router   *Router
    content  js.Value
    theme    string
    themeCb  js.Func
}

func New(renderer controller.PageRenderer) *Driver { ... }
func (d *Driver) Init() { ... }       // Setup DOM, events, initial render
func (d *Driver) navigate() { ... }   // Read hash, render, set innerHTML
func (d *Driver) toggleTheme() { ... }
```

### Step 4.2: Rewrite cmd/forge-ui-wasm/main.go

```go
package main

import (
    "github.com/alexandremahdhaoui/forge-ui/internal/adapter"
    "github.com/alexandremahdhaoui/forge-ui/internal/controller"
    "github.com/alexandremahdhaoui/forge-ui/internal/driver/wasm"
)

func main() {
    ds := adapter.NewDemoDataSource()
    renderer := controller.NewPageRenderer(ds)
    d := wasm.New(renderer)
    d.Init()
    select {} // block forever
}
```

### Step 4.3: Verify WASM build

```bash
GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/forge-ui-wasm
```

---

## Phase 5: Create `internal/driver/http/`

### Step 5.1: Move existing handler to driver/http

1. Create `internal/driver/http/` directory
2. Move files from `internal/handler/` to `internal/driver/http/`
3. Change `package handler` to `package httpdriver` (CANNOT use `http` — conflicts
   with stdlib `net/http` in imports)
4. Update imports in `cmd/forge-ui/main.go`:
   `httpdriver "github.com/alexandremahdhaoui/forge-ui/internal/driver/http"`
5. Existing handler continues to use cache/refresher directly (no adapter interface
   for now — HTTP driver is not being refactored deeply, just relocated)

**Pragmatic trade-off**: The HTTP driver does NOT go through the controller/adapter
layers yet. It continues to use cache/refresher/portfolio/workspace packages directly.
This is deliberate — the HTTP driver has a fundamentally different data flow (live
git data via background refresh + cache) vs. the WASM driver (static demo data).
Forcing them through the same adapter interface would require a complex adapter
wrapping cache+refresher, which is out of scope. The HTTP driver can be migrated
to use the adapter pattern in a future refactor.

### Step 5.2: Verify server still works

Run `go build ./cmd/forge-ui` — must compile.
Run `forge test-all` — must pass.

---

## Phase 6: Update web assets + forge.yaml

### Step 6.1: Copy wasm_exec.js

The `wasm_exec.js` file MUST come from the same Go version used to compile the WASM.
We copy it into `web/` so it gets version-controlled and copied by the existing
web-assets build step.

**Pre-build setup** (one-time, also in forge.yaml):

```bash
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./web/wasm_exec.js
```

**forge.yaml**: Add a build step using `sh -c` to resolve the GOROOT dynamically:

```yaml
  - name: forge-ui-wasm-exec
    src: ./web
    dest: ./build/web
    engine: alias://web-assets
    spec:
      command: "sh"
      args: ["-c", "cp $(go env GOROOT)/lib/wasm/wasm_exec.js ./build/web/wasm_exec.js"]
```

**Alternative (simpler)**: Just commit `web/wasm_exec.js` to git. The existing
web-assets step already copies `web/*` to `build/web/`. This avoids the dynamic
GOROOT resolution. Downside: must update when Go version changes.

### Step 6.2: Rewrite web/index.html

1. Remove all inline JS (WASI shim, router, theme toggle)
2. Add `<script src="wasm_exec.js"></script>`
3. Add 5-line WASM bootstrap:
   ```html
   <script>
       const go = new Go();
       WebAssembly.instantiateStreaming(fetch("forge-ui.wasm"), go.importObject)
           .then(result => go.run(result.instance));
   </script>
   ```
4. Keep all CSS intact
5. Keep the `<main id="content">` element
6. Keep the `<button id="theme-btn">` element
7. Change `<a class="top-app-bar__title" href="#/portfolios">` (already done)

### Step 6.3: Update forge.yaml WASM build

Change GOOS from `wasip1` to `js`:

```yaml
  - name: forge-ui-wasm
    spec:
      env:
        GOOS: "js"      # Changed from wasip1
        GOARCH: "wasm"
```

---

## Phase 7: Clean up old code

### Step 7.1: Delete obsolete packages

1. Delete `internal/render/` (replaced by controller/ + driver/wasm/)
2. Delete `internal/model/` (replaced by types/)
3. Delete `internal/handler/` (replaced by driver/http/)
4. Delete `PLAN.md` (old plan, superseded by this plan)

### Step 7.2: Verify nothing references deleted packages

```bash
grep -r "internal/render" --include="*.go" .
grep -r "internal/model" --include="*.go" .
grep -r "internal/handler" --include="*.go" .
```

All should return no results.

---

## Phase 8: Tests

### Step 8.1: Controller tests

Create `internal/controller/renderer_test.go`:
- Port existing render tests to test the controller's Render() method
- Test all 4 routes: /portfolios, /portfolios/{name}, /portfolios/{p}/workspaces/{w},
  /portfolios/{p}/workspaces/{w}/repos/{r}
- Test default route, unknown route fallback, sort params, hash prefix

### Step 8.2: Adapter tests

Create `internal/adapter/demo_test.go`:
- Test DemoDataSource returns correct data for each method
- Test error cases (unknown portfolio name, etc.)

### Step 8.3: WASM driver tests

The WASM driver uses `syscall/js` which requires a browser/WASM environment.
Unit tests for driver/wasm are NOT possible with standard `go test`.
Instead, test the controller layer which contains the actual logic.

The DOM helpers and router in driver/wasm are thin wrappers around syscall/js
and are tested indirectly via the browser.

### Step 8.4: Existing tests

Ensure all existing tests still pass:
- `internal/cache/` tests
- `internal/git/` tests
- `internal/forge/` tests (if any)
- `internal/workspace/` tests
- `internal/portfolio/` tests
- `internal/refresher/` tests
- `internal/ignore/` tests
- `internal/driver/http/` tests (moved from handler/)

---

## Phase 9: Final validation

### Step 9.1: Run forge test-all

```bash
forge test-all
```

Must pass: build (all 3 targets), lint, unit tests.

### Step 9.2: Verify WASM binary

```bash
file build/web/forge-ui.wasm
# Should show WebAssembly module
```

### Step 9.3: Verify web assets

```bash
ls build/web/
# Should contain: forge-ui.wasm index.html wasm_exec.js
```

---

## Phase 10: Commit and push

```bash
git add -A
git commit -m "refactor: restructure to adapter/controller/driver with GOOS=js WASM"
git push -u origin claude/go-wasm-forge-ui-kF4cF
```

---

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| `syscall/js` not available on non-js targets | Only import in files under `driver/wasm/` and `cmd/forge-ui-wasm/` — use build tags if needed |
| Lint may flag `syscall/js` imports | WASM driver files may need `//go:build js && wasm` build tags to exclude from standard lint |
| Template embed paths change | Use relative `//go:embed templates/*.html` in the controller package |
| import cycle between packages | Strict dependency rule: types ← adapter ← controller ← driver |
| Existing handler tests break | Tests move with handler to driver/http/, imports updated |
| wasm_exec.js version mismatch | Copy from same GOROOT used for compilation |

## Build Tag Strategy

Files that import `syscall/js` MUST have this as the FIRST line:
```go
//go:build js && wasm
```

**Affected files** (exhaustive list):
- `internal/driver/wasm/driver.go` — `//go:build js && wasm`
- `internal/driver/wasm/dom.go` — `//go:build js && wasm`
- `internal/driver/wasm/router.go` — `//go:build js && wasm`
- `cmd/forge-ui-wasm/main.go` — `//go:build js && wasm`

**Why**: `go test ./...` runs on GOOS=linux GOARCH=amd64 and will fail to compile
`syscall/js` (which only exists for GOOS=js). The build tag excludes these files
from host compilation. `go-lint` (golangci-lint) also runs on the host, so the
same exclusion applies.

**Not affected** (no build tags needed):
- `internal/controller/` — pure Go, no JS dependencies
- `internal/adapter/` — pure Go, no JS dependencies
- `internal/types/` — pure Go, no JS dependencies

**Verification**: After adding build tags, run:
```bash
go build ./...                          # Host — must skip wasm files
GOOS=js GOARCH=wasm go build ./cmd/forge-ui-wasm  # WASM — must compile
```
