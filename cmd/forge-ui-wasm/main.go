//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/controller"
	"github.com/alexandremahdhaoui/forge-ui/internal/driver/wasm"
)

func main() {
	apiBaseURL := getAPIBaseURL()
	ds := adapter.NewAPIDataSource(apiBaseURL)
	renderer := controller.NewPageRenderer(ds)
	d := wasm.New(renderer)
	d.Init()
	select {} // block forever
}

func getAPIBaseURL() string {
	doc := js.Global().Get("document")
	meta := doc.Call("querySelector", `meta[name="api-base-url"]`)
	if meta.IsNull() || meta.IsUndefined() {
		return "/api/v1" // fallback default
	}
	return meta.Get("content").String()
}
