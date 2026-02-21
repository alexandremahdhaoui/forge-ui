# Plan: Fix WASM Implementation

## Problem
- Too much JavaScript: `app.js` (250 lines) owns all logic
- WASM is just a template renderer with no real value
- Templates use JS onclick handlers instead of standard HTML links

## Target Architecture

```
index.html
├── CSS (M3/Catppuccin design system)
├── <script> inline (~80 lines total)
│   ├── WASI shim (provide wasi_snapshot_preview1 to WASM)
│   ├── Load forge-ui.wasm
│   └── On hashchange: run WASM(route) → inject HTML into #content
└── <main id="content">...</main>

Go WASM module (wasip1)
├── Reads route string from stdin (e.g. "/portfolios/infra")
├── Owns all app logic: routing, state, sorting
├── Embeds demo data in Go
├── Renders HTML with hash-based <a href="#/..."> links
└── Writes complete page content to stdout
```

## Files to change

1. **DELETE** `web/app.js` — all logic moves to Go
2. **DELETE** `web/wasi_shim.js` — merge into inline script in index.html
3. **REWRITE** `web/index.html` — single inline `<script>` with WASI shim + loader
4. **REWRITE** `cmd/forge-ui-wasm/main.go` — parse route from stdin, route to page
5. **REWRITE** `internal/render/render.go` — accept route string, not JSON command
6. **REWRITE** `internal/render/command.go` — simplify to route-based input
7. **ADD** `internal/render/data.go` — embedded demo data (moved from app.js)
8. **REWRITE** `internal/render/templates/*.html` — replace onclick with `<a href="#/...">`
9. **UPDATE** `internal/render/render_test.go` — update tests for new API
10. **UPDATE** `forge.yaml` — remove web-assets build (no separate JS files)

## Data flow

```
User clicks <a href="#/portfolios/infra">
        │
        ▼
hashchange event (browser built-in)
        │
        ▼
inline <script>: extract hash, encode as stdin
        │
        ▼
instantiate WASM with stdin="/portfolios/infra&sort=time&theme=light"
        │
        ▼
Go main(): parse route → render.Execute(route) → stdout
        │
        ▼
inline <script>: document.getElementById('content').innerHTML = stdout
```

## JS budget: ~80 lines (all inline in index.html)
- WASI shim: ~60 lines (fd_write, fd_read, args, environ, clock, random, proc_exit)
- Loader + hashchange: ~20 lines
