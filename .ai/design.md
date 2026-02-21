# System Design: forge-ui WASM (GOOS=js)

## 1. High-Level Architecture

The forge-ui project has two entry points sharing a common core:

```
┌─────────────────────────────────────────────────────────────────┐
│                        forge-ui monorepo                         │
│                                                                  │
│  cmd/forge-ui/           cmd/forge-ui-wasm/                      │
│  (HTTP server)           (Browser WASM)                          │
│       │                       │                                  │
│       ▼                       ▼                                  │
│  ┌─────────┐            ┌──────────┐                             │
│  │ driver/ │            │ driver/  │                             │
│  │ http    │            │ wasm     │                             │
│  └────┬────┘            └────┬─────┘                             │
│       │                      │                                   │
│       ▼                      ▼                                   │
│  ┌──────────────────────────────────┐                            │
│  │         controller/              │                            │
│  │   (shared business logic)        │                            │
│  └──────────────┬───────────────────┘                            │
│                 │                                                 │
│                 ▼                                                 │
│  ┌──────────────────────────────────┐                            │
│  │          adapter/                │                            │
│  │  (data access — git, demo, fs)   │                            │
│  └──────────────────────────────────┘                            │
│                                                                  │
│  internal/types/  (shared domain model — pure data)              │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## 2. Package Layout (Target State)

```
internal/
├── types/                     # Domain model (renamed from model/)
│   └── types.go               # All data types (portfolios, workspaces, repos, etc.)
│
├── adapter/                   # Outbound ports — data access
│   ├── adapter.go             # Interface definitions
│   ├── git.go                 # Git adapter (wraps existing git/ package)
│   ├── forge.go               # Forge adapter (wraps existing forge/ package)
│   ├── filesystem.go          # Filesystem adapter (wraps portfolio/ + workspace/)
│   └── demo.go                # Demo/stub adapter (static data for WASM)
│
├── controller/                # Business logic — page assembly
│   ├── controller.go          # Interface definitions
│   ├── portfolios.go          # Portfolios page controller
│   ├── portfolio.go           # Single portfolio controller
│   ├── workspace.go           # Workspace page controller
│   └── forge.go               # Forge page controller
│
├── driver/                    # Inbound ports — transport layer
│   ├── http/                  # HTTP server driver (existing handler/)
│   │   ├── driver.go          # HTTP handler setup
│   │   ├── routes.go          # Route registration
│   │   └── templates/         # Server-side HTML templates
│   │
│   └── wasm/                  # Browser WASM driver (NEW)
│       ├── driver.go          # WASM entry: init, router, DOM bridge
│       ├── router.go          # Hash-based router (hashchange listener)
│       ├── dom.go             # DOM helpers (getElementById, innerHTML, etc.)
│       └── templates/         # Embedded HTML templates (go:embed)
│           ├── portfolios.html
│           ├── portfolio.html
│           ├── workspace.html
│           └── forge.html
│
├── cache/                     # (keep) Thread-safe git data cache
├── git/                       # (keep) Git command execution
├── forge/                     # (keep) forge.yaml parsing
├── ignore/                    # (keep) .forge-workspace-ignore
├── portfolio/                 # (keep) Portfolio discovery
├── refresher/                 # (keep) Background refresher
└── workspace/                 # (keep) Workspace discovery
```

## 3. Layer Interactions

### Sequence Diagram: Browser Page Load

```
┌────────┐    ┌──────────┐    ┌──────────────┐    ┌──────────────┐
│Browser │    │driver/wasm│    │  controller/ │    │   adapter/   │
│        │    │          │    │              │    │              │
│  Load  │    │          │    │              │    │              │
│ index  │───>│          │    │              │    │              │
│ .html  │    │          │    │              │    │              │
│        │    │          │    │              │    │              │
│  Load  │    │          │    │              │    │              │
│ .wasm  │───>│ main()   │    │              │    │              │
│        │    │          │    │              │    │              │
│        │    │ Init()   │    │              │    │              │
│        │    │ ├ setup  │    │              │    │              │
│        │    │ │ DOM ref│    │              │    │              │
│        │    │ ├ listen │    │              │    │              │
│        │    │ │hashchg │    │              │    │              │
│        │    │ ├ listen │    │              │    │              │
│        │    │ │theme   │    │              │    │              │
│        │    │ └ call   │    │              │    │              │
│        │    │  navigate│    │              │    │              │
│        │    │    │     │    │              │    │              │
│        │    │    ▼     │    │              │    │              │
│        │    │ navigate()    │              │    │              │
│        │    │ ├ read   │    │              │    │              │
│        │    │ │ hash   │    │              │    │              │
│        │    │ ├ parse  │    │              │    │              │
│        │    │ │ route  │    │              │    │              │
│        │    │ └───────────>│ Render(route)│    │              │
│        │    │          │    │ ├───────────────>│ GetData()    │
│        │    │          │    │ │            │    │ └ return     │
│        │    │          │    │ │<───────────────│   demo data  │
│        │    │          │    │ ├ assemble   │    │              │
│        │    │          │    │ │ page data  │    │              │
│        │    │          │    │ ├ execute    │    │              │
│        │    │          │    │ │ template   │    │              │
│        │    │          │    │ └ return HTML│    │              │
│        │    │<─────────────│              │    │              │
│        │    │ set      │    │              │    │              │
│        │    │ innerHTML│    │              │    │              │
│<───────────│          │    │              │    │              │
│  Render │    │          │    │              │    │              │
│  page   │    │          │    │              │    │              │
└────────┘    └──────────┘    └──────────────┘    └──────────────┘
```

### Sequence Diagram: Hash Navigation (User clicks link)

```
┌────────┐    ┌──────────┐    ┌──────────────┐    ┌──────────────┐
│Browser │    │driver/wasm│    │  controller/ │    │   adapter/   │
│        │    │          │    │              │    │              │
│ Click  │    │          │    │              │    │              │
│<a href>│    │          │    │              │    │              │
│ "#/p/x"│    │          │    │              │    │              │
│        │    │          │    │              │    │              │
│hashchg │───>│ onHash() │    │              │    │              │
│ event  │    │ ├ read   │    │              │    │              │
│        │    │ │ hash   │    │              │    │              │
│        │    │ └───────────>│ Render(route)│    │              │
│        │    │          │    │ ├───────────────>│ GetData()    │
│        │    │          │    │ │<───────────────│              │
│        │    │          │    │ ├ template   │    │              │
│        │    │          │    │ └ return HTML│    │              │
│        │    │<─────────────│              │    │              │
│        │    │ innerHTML│    │              │    │              │
│<───────────│          │    │              │    │              │
│  Update │    │          │    │              │    │              │
│  page   │    │          │    │              │    │              │
└────────┘    └──────────┘    └──────────────┘    └──────────────┘
```

### Sequence Diagram: Theme Toggle

```
┌────────┐    ┌──────────┐
│Browser │    │driver/wasm│
│        │    │          │
│ Click  │───>│onTheme() │
│ toggle │    │├ flip    │
│  btn   │    ││ theme   │
│        │    │├ set     │
│        │    ││ data-   │
│        │    ││ theme   │
│        │    │├ save to │
│        │    ││ storage │
│        │    │└ call    │
│        │    │ navigate │
│        │    │ (re-     │
│        │    │  render) │
│<───────────│          │
│  Update │    │          │
│  theme  │    │          │
└────────┘    └──────────┘
```

## 4. Interface Definitions

### adapter/ — Outbound Port

```go
package adapter

import "github.com/alexandremahdhaoui/forge-ui/internal/types"

// DataSource provides page data for rendering.
// Implementations:
//   - DemoDataSource: static data for WASM demo (initial implementation)
//   - (future) LiveDataSource: wraps cache/refresher/portfolio/workspace for HTTP server
type DataSource interface {
    ListPortfolios(sort string) (types.PortfoliosPageData, error)
    GetPortfolio(name, sort string) (types.PortfolioPageData, error)
    GetWorkspace(portfolio, workspace, sort string) (types.WorkspacePageData, error)
    GetForge(portfolio, workspace, repo string) (types.ForgePageData, error)
}
```

**Current scope**: Only `DemoDataSource` is implemented. The HTTP driver continues
to use cache/refresher directly (not through this interface). This is a pragmatic
choice — migrating the HTTP driver to use `DataSource` requires wrapping the
cache/refresher/portfolio/workspace packages, which is out of scope for this refactor.

### controller/ — Core Business Logic

```go
package controller

// PageRenderer renders HTML for a given route.
type PageRenderer interface {
    Render(route string) (string, error)
}
```

### driver/wasm — Inbound Port (Browser)

```go
package wasm

// Driver is the WASM browser driver.
// It owns the DOM references, event listeners, and router.
// Constructed in main(), blocks forever.
type Driver struct {
    renderer controller.PageRenderer
    content  js.Value   // document.getElementById("content")
    theme    string     // "light" or "dark"
}
```

## 5. Build & Serve

### forge.yaml Changes

```yaml
build:
  - name: forge-ui-wasm
    engine: alias://wasm-build
    spec:
      command: "go"
      args: ["build", "-o", "./build/web/forge-ui.wasm", "./cmd/forge-ui-wasm"]
      env:
        GOOS: "js"          # <-- Changed from wasip1
        GOARCH: "wasm"

  - name: forge-ui-wasm-exec
    engine: alias://web-assets
    spec:
      command: "cp"
      args: ["GOROOT/lib/wasm/wasm_exec.js", "./build/web/wasm_exec.js"]
```

### index.html (Minimal Shell)

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <title>forge-ui</title>
    <!-- M3/Catppuccin CSS (inline) -->
    <style>/* ... existing CSS kept as-is ... */</style>
</head>
<body>
    <header class="top-app-bar">
        <a class="top-app-bar__title" href="#/portfolios">forge-ui</a>
        <span class="top-app-bar__badge">wasm</span>
        <span class="top-app-bar__spacer"></span>
        <button class="theme-toggle" id="theme-btn"><!-- SVGs --></button>
    </header>
    <main id="content" class="page-content">
        <div class="loading">Loading WASM...</div>
    </main>
    <script src="wasm_exec.js"></script>
    <script>
        const go = new Go();
        WebAssembly.instantiateStreaming(fetch("forge-ui.wasm"), go.importObject)
            .then(result => go.run(result.instance));
    </script>
</body>
</html>
```

**Key difference from previous approach:**
- NO WASI shim (wasm_exec.js replaces it)
- NO inline application JS (all logic is in Go)
- Only 5 lines of JS to bootstrap WASM
- Go `main()` takes over: sets up DOM listeners, handles routing, renders pages

## 6. WASM Driver Internals

### main.go Entry Point

```go
package main

import (
    "github.com/alexandremahdhaoui/forge-ui/internal/adapter"
    "github.com/alexandremahdhaoui/forge-ui/internal/controller"
    "github.com/alexandremahdhaoui/forge-ui/internal/driver/wasm"
)

func main() {
    // 1. Build adapter (demo data for now)
    ds := adapter.NewDemoDataSource()

    // 2. Build controller
    renderer := controller.NewPageRenderer(ds)

    // 3. Build driver (sets up DOM, events, router)
    d := wasm.New(renderer)
    d.Init()

    // 4. Block forever (Go callbacks stay alive)
    select {}
}
```

### driver/wasm/driver.go

```go
func (d *Driver) Init() {
    // Cache DOM references
    d.content = js.Global().Get("document").Call("getElementById", "content")

    // Restore theme from localStorage
    d.theme = getLocalStorage("forge-ui-theme", "light")
    applyTheme(d.theme)

    // Listen for hash changes
    d.hashCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        d.navigate()
        return nil
    })
    js.Global().Get("window").Call("addEventListener", "hashchange", d.hashCb)

    // Listen for theme toggle
    d.themeCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        d.toggleTheme()
        return nil
    })
    js.Global().Get("document").Call("getElementById", "theme-btn").
        Call("addEventListener", "click", d.themeCb)

    // Initial navigation
    if js.Global().Get("window").Get("location").Get("hash").String() == "" {
        js.Global().Get("window").Get("location").Set("hash", "#/portfolios")
    }
    d.navigate()
}

func (d *Driver) navigate() {
    hash := js.Global().Get("window").Get("location").Get("hash").String()
    html, err := d.renderer.Render(hash)
    if err != nil {
        d.content.Set("innerHTML",
            `<div class="empty-state body-large">Error: `+err.Error()+`</div>`)
        return
    }
    d.content.Set("innerHTML", html)
}
```

## 7. Template Strategy

There are TWO sets of templates serving different purposes:

**WASM templates** (`internal/controller/templates/*.html`):
- Embedded via `//go:embed` in the controller package
- Fragment templates (no `<html>`, no `<head>`) — they produce content injected via
  `innerHTML` into the shell page's `<main id="content">` element
- Use hash-based `<a href="#/...">` links for navigation
- 4 templates: portfolios, portfolio, workspace, forge

**HTTP templates** (`templates/*.html`):
- Loaded at runtime by the HTTP driver
- Full page templates with `layout.html` wrapper (`<html>`, `<head>`, `<body>`)
- Use server-side URL paths (`/portfolios/...`) for links
- 5 templates: layout, portfolios, portfolio, workspace, forge

The WASM and HTTP templates have **identical table/card/stats markup** but differ
in their outer structure (fragment vs. full page) and link format (hash vs. path).
They are NOT shared — each driver has its own copy. This is deliberate: the HTTP
driver may evolve independently (server-side features), and the WASM driver has
different navigation needs.

No JavaScript onclick handlers, no framework-specific bindings. The WASM driver uses
pure hash-based navigation. When the user clicks a link, the browser fires
`hashchange`, Go catches it via `js.FuncOf`, re-renders, and sets innerHTML.

## 8. What Changes vs. Current Codebase

| Component | Current State | Target State |
|-----------|--------------|--------------|
| WASM target | `GOOS=wasip1` (stdin/stdout) | `GOOS=js` (DOM access) |
| JS shim | 80+ lines inline WASI polyfill | 5 lines: `new Go(); go.run()` |
| Go runtime | Per-invocation (new instance each nav) | Persistent (single instance, blocks forever) |
| DOM updates | JS captures WASM stdout, sets innerHTML | Go directly sets innerHTML via `syscall/js` |
| Event handling | JS hashchange listener calls WASM | Go registers hashchange listener via `js.FuncOf` |
| Architecture | Flat render/ package | adapter/controller/driver layers |
| model/ | `internal/model/` | `internal/types/` (renamed) |
| render/ | `internal/render/` | Split into controller + driver/wasm |
| handler/ | `internal/handler/` | `internal/driver/http/` |
| Data source | Hardcoded in render/data.go | adapter.DataSource interface |
| wasm_exec.js | Not used (custom WASI shim) | From Go stdlib, served as static asset |

## 9. File Serving Strategy

The `web/` directory contains:
- `index.html` — shell page with CSS + WASM bootstrap
- `wasm_exec.js` — copied from `$(go env GOROOT)/lib/wasm/wasm_exec.js` during build

The `forge-ui.wasm` binary is built into `build/web/`.

The forge build copies `web/*` to `build/web/`, and the WASM build outputs
`forge-ui.wasm` to `build/web/`. The HTTP server can optionally serve `build/web/`
as a static file directory.
