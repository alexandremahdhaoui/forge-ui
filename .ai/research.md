# Research: Go WASM with GOOS=js and Shaper Architecture

## 1. go-app Library (Reference — NOT a dependency)

[github.com/maxence-charriere/go-app](https://github.com/maxence-charriere/go-app)

go-app is a Go framework for building progressive web apps using `GOOS=js GOARCH=wasm`.
We will NOT use go-app as a dependency. We study it to understand the patterns and
re-implement only what we need.

### Key Architecture Decisions in go-app

1. **Compilation target**: `GOOS=js GOARCH=wasm` (NOT wasip1). This gives Go code full
   access to the browser DOM via `syscall/js`.

2. **Component model**: Every UI component embeds `app.Compo` and implements `Render()`:
   ```go
   type Hello struct {
       app.Compo
       Name string
   }

   func (h *Hello) Render() app.UI {
       return app.Div().Body(
           app.H1().Text("Hello, " + h.Name),
           app.Input().Value(h.Name).OnChange(h.onNameChange),
       )
   }

   func (h *Hello) OnMount(ctx app.Context) {
       // Called when component is mounted to DOM
   }
   ```

3. **Declarative HTML**: Fluent builder API — `app.Div().Class("foo").Body(app.P().Text("hi"))`.
   Every HTML element has a corresponding Go function. Attributes and event handlers
   are chained methods.

4. **Routing**: `app.Route("/path", &Component{})` maps URL paths to components.

5. **Events**: `func(ctx app.Context, e app.Event)` — typed event handlers attached to
   elements via `.OnClick()`, `.OnChange()`, etc.

6. **Server**: `app.Handler{}` is an `http.Handler` that serves:
   - The compiled `.wasm` binary
   - The Go `wasm_exec.js` support file
   - A shell HTML page that bootstraps everything
   - Static resources

7. **VDOM diffing**: go-app implements a virtual DOM — `Render()` returns a tree,
   go-app diffs it against the previous tree and patches the real DOM.
   **We do NOT need this.** Our approach is simpler: render full HTML in Go,
   set `innerHTML` directly.

### What We Take From go-app

- **GOOS=js GOARCH=wasm** compilation target (not wasip1)
- The idea that Go code runs persistently in the browser (not per-invocation)
- Event handling via `syscall/js.FuncOf()` for hashchange, click, etc.
- The concept of a single Go `main()` that initializes and blocks forever

### What We Don't Need

- VDOM diffing (overkill — we set innerHTML directly)
- The fluent HTML builder API (we use Go `html/template`)
- The component lifecycle (Mounter/Dismounter/etc.)
- PWA/service worker support
- The `app.Handler` server (we already have our own HTTP server)

---

## 2. Go `syscall/js` Package (GOOS=js GOARCH=wasm)

### Build Command

```bash
GOOS=js GOARCH=wasm go build -o forge-ui.wasm ./cmd/forge-ui-wasm
```

### wasm_exec.js Bootstrap

Go provides `wasm_exec.js` at `$(go env GOROOT)/lib/wasm/wasm_exec.js`.
This file must be served alongside the `.wasm` binary. It provides the `Go` class.

**Critical**: The `wasm_exec.js` version MUST match the Go compiler version.
Copy it during build.

```html
<script src="wasm_exec.js"></script>
<script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("forge-ui.wasm"), go.importObject)
        .then(result => go.run(result.instance));
</script>
```

### Core API

```go
import "syscall/js"

// Access global objects
document := js.Global().Get("document")
window   := js.Global().Get("window")
console  := js.Global().Get("console")

// Query DOM elements
el := document.Call("getElementById", "content")

// Set properties
el.Set("innerHTML", "<h1>Hello</h1>")

// Get properties
hash := window.Get("location").Get("hash").String()

// Create elements
div := document.Call("createElement", "div")
div.Set("className", "my-class")
div.Set("textContent", "Hello")
parent.Call("appendChild", div)

// Add event listeners
cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
    event := args[0]
    // handle event
    return nil
})
defer cb.Release() // Release when no longer needed
window.Call("addEventListener", "hashchange", cb)

// Keep main alive (block forever)
select {} // or <-make(chan struct{})
```

### Key Points

- `js.FuncOf()` creates a Go callback callable from JavaScript. Must call `.Release()`
  when no longer needed to avoid memory leaks.
- `js.Value` wraps any JavaScript value. Use `.String()`, `.Int()`, `.Float()`, `.Bool()`
  to extract Go types.
- `js.Null()` and `js.Undefined()` for null/undefined checks.
- The Go `main()` must NOT return — use `select {}` to block forever after setup.
  If main returns, the WASM instance exits and callbacks stop working.

### Pattern: Hash-Based Router in Go

```go
func main() {
    // Register hashchange listener
    cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
        navigate()
        return nil
    })
    js.Global().Get("window").Call("addEventListener", "hashchange", cb)

    // Initial render
    navigate()

    // Block forever
    select {}
}

func navigate() {
    hash := js.Global().Get("window").Get("location").Get("hash").String()
    // Parse route from hash, render HTML, set innerHTML
    html := renderRoute(hash)
    js.Global().Get("document").Call("getElementById", "content").Set("innerHTML", html)
}
```

---

## 3. Shaper Architecture: Adapter / Controller / Driver

[github.com/alexandremahdhaoui/shaper](https://github.com/alexandremahdhaoui/shaper)

Shaper implements **hexagonal architecture** (ports and adapters) with three named layers:

### Directory Structure

```
internal/
  adapter/          # Outbound ports — interface to external systems
  controller/       # Core business logic — orchestrates adapters
  driver/           # Inbound ports — accepts external input (HTTP, TFTP, webhook)
    server/         # HTTP driver
    tftp/           # TFTP driver
    webhook/        # Kubernetes webhook driver
  types/            # Shared domain model — pure data, no behavior
```

### The Three Layers

| Layer        | Direction | Responsibility |
|--------------|-----------|----------------|
| **Driver**   | Inbound   | Accepts external input, translates to controller calls |
| **Controller** | Core    | Business logic, orchestration, no I/O knowledge |
| **Adapter**  | Outbound  | Communicates with external systems, returns domain types |

### Dependency Rule

```
External Input → DRIVER → CONTROLLER → ADAPTER → External Systems
                    ↓           ↓           ↓
                 depends on  depends on  depends on
                 controller  adapter     infra libs
                 interfaces  interfaces
```

No layer imports from a layer above it. All share `types/` for domain model.

### Interface Design Pattern

Every layer exposes interfaces. Constructors return interfaces, not concrete types:

```go
// Adapter layer — outbound port interface
type Profile interface {
    Get(ctx context.Context, name string) (types.Profile, error)
}

// Controller layer — business logic interface
type IPXE interface {
    FindProfileAndRender(ctx context.Context, selectors types.IPXESelectors) ([]byte, error)
    Bootstrap() []byte
}

// Driver layer — accepts controller interfaces
func New(ipxe controller.IPXE, config controller.Content) ServerInterface {
    return &server{ipxe: ipxe, config: config}
}
```

### Composition Root (cmd/main.go)

All wiring happens in main. Dependencies are injected explicitly:

```go
func main() {
    // 1. Build adapters
    profileAdapter := adapter.NewProfile(kubeClient, namespace)
    // 2. Build controllers (inject adapters)
    ipxeCtrl := controller.NewIPXE(assignmentAdapter, profileAdapter, mux)
    // 3. Build drivers (inject controllers)
    srv := server.New(ipxeCtrl, contentCtrl)
}
```

### Key Patterns Used

- **Constructor injection** — no DI framework, explicit wiring
- **Strategy pattern** — resolver/transformer maps dispatched by kind
- **Interface segregation** — small, focused interfaces per consumer
- **Anti-corruption layer** — `types/` package insulates domain from external API shapes

---

## 4. Existing forge-ui Architecture (Current State)

### Package Structure

```
internal/
  cache/        # Thread-safe git data cache (sync.RWMutex)
  forge/        # Parses forge.yaml + artifact-store.yaml
  git/          # Executes git commands → model types
  handler/      # HTTP request handlers (server-side rendering)
  ignore/       # .forge-workspace-ignore pattern matching
  model/        # All data types (29 types)
  portfolio/    # Portfolio discovery (directories)
  refresher/    # Background git data refresher (goroutine pool)
  render/       # WASM template rendering (current broken wasip1 approach)
  workspace/    # Workspace discovery + repo scanning
```

### What Exists

- **Full HTTP server** (`cmd/forge-ui/main.go`) with server-side rendered templates
- **Model types** (`model/model.go`) — 29 types covering portfolios, workspaces, repos, forge data
- **Git integration** — real git commands, background refresh, caching
- **Template system** — 4 page templates (portfolios, portfolio, workspace, forge)
- **WASM entry point** (`cmd/forge-ui-wasm/main.go`) — broken wasip1 approach reading from stdin

### What Needs to Change

The entire WASM approach must be rewritten:
- Replace wasip1 → GOOS=js GOARCH=wasm
- Replace stdin/stdout I/O → direct DOM manipulation via `syscall/js`
- Replace per-invocation WASM → persistent Go runtime in browser
- Replace external JS router → Go-based hash router
- Replace WASI shim → standard wasm_exec.js from Go stdlib
- Restructure to adapter/controller/driver pattern
