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
