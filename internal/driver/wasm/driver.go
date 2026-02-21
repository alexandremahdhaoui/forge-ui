//go:build js && wasm

package wasm

import (
	"syscall/js"

	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
)

// Driver wires up the WASM runtime: DOM rendering, hash routing, theme toggle.
type Driver struct {
	renderer controller.PageRenderer
	router   *Router
	content  js.Value
	theme    string
	themeCb  js.Func
}

// New creates a Driver with the given PageRenderer.
func New(renderer controller.PageRenderer) *Driver {
	return &Driver{renderer: renderer}
}

// Init sets up the DOM, registers event listeners, and performs the initial render.
func (d *Driver) Init() {
	d.content = getElementById("content")

	// Restore theme from localStorage.
	d.theme = getLocalStorage("forge-ui-theme", "light")
	if d.theme == "dark" {
		document().Get("documentElement").Call("setAttribute", "data-theme", "dark")
	}

	// Theme toggle button.
	d.themeCb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		d.toggleTheme()
		return nil
	})
	getElementById("theme-btn").Call("addEventListener", "click", d.themeCb)

	// Hash-based router.
	d.router = NewRouter(func(hash string) {
		d.navigate(hash)
	})
	d.router.Start()
}

// navigate reads the hash, renders the page, and updates the DOM.
func (d *Driver) navigate(hash string) {
	// Build the route with theme parameter.
	route := hash
	if len(route) > 0 && route[0] == '#' {
		route = route[1:]
	}
	if route == "" {
		route = "/portfolios"
	}

	// Append theme as query parameter.
	sep := "?"
	if containsQuery(route) {
		sep = "&"
	}
	route += sep + "theme=" + d.theme

	html, err := d.renderer.Render(route)
	if err != nil {
		setInnerHTML(d.content, `<div class="empty-state body-large">Error: `+err.Error()+`</div>`)
		return
	}
	setInnerHTML(d.content, html)
}

// toggleTheme switches between light and dark mode.
func (d *Driver) toggleTheme() {
	if d.theme == "light" {
		d.theme = "dark"
	} else {
		d.theme = "light"
	}
	setLocalStorage("forge-ui-theme", d.theme)

	docEl := document().Get("documentElement")
	if d.theme == "dark" {
		docEl.Call("setAttribute", "data-theme", "dark")
	} else {
		docEl.Call("removeAttribute", "data-theme")
	}

	// Re-render with new theme.
	d.navigate(getHash())
}

func containsQuery(s string) bool {
	for _, c := range s {
		if c == '?' {
			return true
		}
	}
	return false
}
