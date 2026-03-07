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

func document() js.Value {
	return js.Global().Get("document")
}

func window() js.Value {
	return js.Global().Get("window")
}

func getElementById(id string) js.Value {
	return document().Call("getElementById", id)
}

func setInnerHTML(el js.Value, html string) {
	el.Set("innerHTML", html)
}

func getHash() string {
	return window().Get("location").Get("hash").String()
}

func setHash(h string) {
	window().Get("location").Set("hash", h)
}

func getLocalStorage(key, fallback string) string {
	val := js.Global().Get("localStorage").Call("getItem", key)
	if val.IsNull() || val.IsUndefined() {
		return fallback
	}
	return val.String()
}

func setLocalStorage(key, value string) {
	js.Global().Get("localStorage").Call("setItem", key, value)
}

func jsSetInterval(fn js.Func, ms int) js.Value {
	return js.Global().Call("setInterval", fn, ms)
}

func jsClearInterval(id js.Value) {
	js.Global().Call("clearInterval", id)
}
