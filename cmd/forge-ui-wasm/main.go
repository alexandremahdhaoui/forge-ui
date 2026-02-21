//go:build js && wasm

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
