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

import "syscall/js"

// Router handles hash-based navigation by listening for hashchange events.
type Router struct {
	onNavigate func(route string)
	cb         js.Func
}

// NewRouter creates a Router that calls onNavigate when the hash changes.
func NewRouter(onNavigate func(string)) *Router {
	return &Router{onNavigate: onNavigate}
}

// Start registers the hashchange event listener and triggers an initial
// navigation if a hash is already set.
func (r *Router) Start() {
	r.cb = js.FuncOf(func(this js.Value, args []js.Value) any {
		hash := getHash()
		r.onNavigate(hash)
		return nil
	})
	window().Call("addEventListener", "hashchange", r.cb)

	// Set default hash and trigger initial navigation.
	hash := getHash()
	if hash == "" || hash == "#" {
		setHash("#/portfolios")
	}
	r.onNavigate(getHash())
}

// Release frees the JS callback. Call when the router is no longer needed.
func (r *Router) Release() {
	window().Call("removeEventListener", "hashchange", r.cb)
	r.cb.Release()
}
