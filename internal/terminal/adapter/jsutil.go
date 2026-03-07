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

// Adapted from sshterm (MIT, c2FmZQ/sshterm)
//
// MIT License
//
// Copyright (c) 2024 TTBT Enterprises LLC
// Copyright (c) 2024 Robin Thellend <rthellend@rthellend.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

//go:build js && wasm

package adapter

import (
	"fmt"
	"syscall/js"
)

var (
	jsUint8Array = js.Global().Get("Uint8Array")
	jsError      = js.Global().Get("Error")
	jsPromise    = js.Global().Get("Promise")
)

// uint8ArrayFromBytes copies Go bytes to a JS Uint8Array.
func uint8ArrayFromBytes(in []byte) js.Value {
	out := jsUint8Array.New(js.ValueOf(len(in)))
	js.CopyBytesToJS(out, in)
	return out
}

// uint8ArrayToBytes copies a JS Uint8Array to Go bytes.
func uint8ArrayToBytes(v js.Value) []byte {
	buf := make([]byte, v.Length())
	js.CopyBytesToGo(buf, v)
	return buf
}

// jsAwait awaits a JS Promise and returns the resolved value or an error.
func jsAwait(p js.Value) (js.Value, error) {
	if then := p.Get("then"); then.IsUndefined() || then.Type() != js.TypeFunction {
		return p, nil
	}
	v := make(chan js.Value, 1)
	e := make(chan error, 1)
	resolveFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		v <- args[0]
		return nil
	})
	rejectFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		e <- fmt.Errorf("%s", args[0].Call("toString"))
		return nil
	})
	p.Call("then", resolveFn).
		Call("catch", rejectFn)
	defer resolveFn.Release()
	defer rejectFn.Release()
	select {
	case value := <-v:
		return value, nil
	case err := <-e:
		return js.Value{}, err
	}
}

// jsNewPromise creates a JS Promise from a Go function.
func jsNewPromise(f func() (any, error)) js.Value {
	executor := js.FuncOf(func(this js.Value, args []js.Value) any {
		resolve := args[0]
		reject := args[1]
		go func() {
			v, err := f()
			if err != nil {
				reject.Invoke(jsError.New(err.Error()))
				return
			}
			resolve.Invoke(v)
		}()
		return nil
	})
	p := jsPromise.New(executor)
	executor.Release()
	return p
}

// document returns the JS document object.
func document() js.Value {
	return js.Global().Get("document")
}

// getElementById returns a DOM element by its ID.
func getElementById(id string) js.Value {
	return document().Call("getElementById", id)
}
