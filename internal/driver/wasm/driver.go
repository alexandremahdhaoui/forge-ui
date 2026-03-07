// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build js && wasm

package wasm

import (
	"strconv"
	"syscall/js"

	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
)

// Driver wires up the WASM runtime: DOM rendering, hash routing, theme toggle, palette selection.
type Driver struct {
	renderer       controller.PageRenderer
	router         *Router
	content        js.Value
	sideNav        js.Value
	theme          string
	palette        string
	themeCb        js.Func
	paletteCbs     []js.Func
	syncCb         js.Func
	syncBtnCb      js.Func
	syncSelectCb   js.Func
	syncIntervalID js.Value
	termToggleCb   js.Func
}

// New creates a Driver with the given PageRenderer.
func New(renderer controller.PageRenderer) *Driver {
	return &Driver{renderer: renderer}
}

// Init sets up the DOM, registers event listeners, and performs the initial render.
func (d *Driver) Init() {
	d.content = getElementById("content")
	d.sideNav = getElementById("side-nav")

	// Restore theme from localStorage.
	d.theme = getLocalStorage("forge-ui-theme", "light")
	if d.theme == "dark" {
		document().Get("documentElement").Call("setAttribute", "data-theme", "dark")
	}

	// Restore light palette from localStorage.
	d.palette = getLocalStorage("forge-ui-light-palette", "1")
	if d.theme != "dark" && d.palette != "1" {
		document().Get("documentElement").Call("setAttribute", "data-light-palette", d.palette)
	}
	d.updatePaletteActive()

	// Theme toggle button.
	d.themeCb = js.FuncOf(func(this js.Value, args []js.Value) any {
		d.toggleTheme()
		return nil
	})
	getElementById("theme-btn").Call("addEventListener", "click", d.themeCb)

	// Palette dropdown items.
	d.initPaletteDropdown()

	// Auto-sync setup.
	intervalStr := getLocalStorage("forge-ui-sync-interval", "60000")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval < 30000 {
		interval = 60000
	}

	// Sync button.
	d.syncBtnCb = js.FuncOf(func(this js.Value, args []js.Value) any {
		d.sync()
		return nil
	})
	getElementById("sync-btn").Call("addEventListener", "click", d.syncBtnCb)

	// Sync interval dropdown.
	syncSelect := getElementById("sync-select")
	syncSelect.Set("value", strconv.Itoa(interval))
	d.syncSelectCb = js.FuncOf(func(this js.Value, args []js.Value) any {
		val := syncSelect.Get("value").String()
		ms, err := strconv.Atoi(val)
		if err != nil || ms < 30000 {
			ms = 60000
		}
		d.setSyncInterval(ms)
		return nil
	})
	syncSelect.Call("addEventListener", "change", d.syncSelectCb)

	d.startAutoSync(interval)

	// Terminal toggle button dispatches a CustomEvent for terminal.mjs.
	d.termToggleCb = js.FuncOf(func(this js.Value, args []js.Value) any {
		detail := map[string]any{
			"workspace": "default",
		}
		event := js.Global().Get("CustomEvent").New("toggle-terminal", map[string]any{
			"detail": detail,
		})
		document().Call("dispatchEvent", event)
		return nil
	})
	getElementById("terminal-toggle").Call("addEventListener", "click", d.termToggleCb)

	// Hash-based router.
	d.router = NewRouter(func(hash string) {
		d.navigate(hash)
	})
	d.router.Start()
}

// navigate reads the hash, renders the page, and updates the DOM.
// Rendering runs in a goroutine so that net/http fetch calls can yield
// to the JS event loop without deadlocking.
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

	go func() {
		result, err := d.renderer.Render(route)
		if err != nil {
			setInnerHTML(d.content, `<div class="empty-state body-large">Error: `+err.Error()+`</div>`)
			setInnerHTML(d.sideNav, "")
			return
		}
		setInnerHTML(d.sideNav, result.SideNav)
		setInnerHTML(d.content, result.Content)
	}()
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
		docEl.Call("removeAttribute", "data-light-palette")
	} else {
		docEl.Call("removeAttribute", "data-theme")
		if d.palette != "1" {
			docEl.Call("setAttribute", "data-light-palette", d.palette)
		}
	}

	// Re-render with new theme.
	d.navigate(getHash())
}

// sync re-renders the current page.
func (d *Driver) sync() {
	d.navigate(getHash())
}

// startAutoSync starts a periodic timer that calls sync().
func (d *Driver) startAutoSync(ms int) {
	d.syncCb = js.FuncOf(func(this js.Value, args []js.Value) any {
		d.sync()
		return nil
	})
	d.syncIntervalID = jsSetInterval(d.syncCb, ms)
}

// stopAutoSync stops the periodic timer.
func (d *Driver) stopAutoSync() {
	jsClearInterval(d.syncIntervalID)
	d.syncCb.Release()
}

// setSyncInterval changes the auto-sync period.
func (d *Driver) setSyncInterval(ms int) {
	d.stopAutoSync()
	setLocalStorage("forge-ui-sync-interval", strconv.Itoa(ms))
	d.startAutoSync(ms)
}

// initPaletteDropdown registers click handlers on palette dropdown items.
func (d *Driver) initPaletteDropdown() {
	items := document().Call("querySelectorAll", "[data-palette]")
	n := items.Get("length").Int()
	d.paletteCbs = make([]js.Func, n)
	for i := range n {
		item := items.Call("item", i)
		p := item.Call("getAttribute", "data-palette").String()
		cb := js.FuncOf(func(this js.Value, args []js.Value) any {
			d.setPalette(p)
			return nil
		})
		d.paletteCbs[i] = cb
		item.Call("addEventListener", "click", cb)
	}
}

// updatePaletteActive highlights the active palette dropdown item.
func (d *Driver) updatePaletteActive() {
	items := document().Call("querySelectorAll", "[data-palette]")
	n := items.Get("length").Int()
	for i := range n {
		item := items.Call("item", i)
		p := item.Call("getAttribute", "data-palette").String()
		cl := item.Get("classList")
		if p == d.palette {
			cl.Call("add", "palette-dropdown__item--active")
		} else {
			cl.Call("remove", "palette-dropdown__item--active")
		}
	}
}

// setPalette changes the light palette and re-renders.
func (d *Driver) setPalette(p string) {
	d.palette = p
	setLocalStorage("forge-ui-light-palette", p)
	docEl := document().Get("documentElement")
	if p == "1" {
		docEl.Call("removeAttribute", "data-light-palette")
	} else {
		docEl.Call("setAttribute", "data-light-palette", p)
	}
	d.updatePaletteActive()
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
