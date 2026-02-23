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
