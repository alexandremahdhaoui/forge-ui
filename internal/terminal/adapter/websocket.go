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
	"context"
	"errors"
	"fmt"
	"net"
	"syscall/js"
	"time"
)

var (
	errWSClosed         = errors.New("websocket is closed")
	errWSClosedByServer = errors.New("websocket was closed by server")
)

// Compile-time check: WebSocket implements net.Conn.
var _ net.Conn = (*WebSocket)(nil)

// WebSocket implements net.Conn over the browser WebSocket API.
type WebSocket struct {
	ctx     context.Context
	ws      js.Value
	ch      chan js.Value
	r       []byte
	closeCh chan struct{}
	err     error
	funcs   []js.Func // JS callbacks to release on Close
}

// NewWebSocket creates a browser WebSocket connection and waits for the open
// event before returning. Returns an error if the connection fails.
func NewWebSocket(ctx context.Context, url string) (*WebSocket, error) {
	jsWS := js.Global().Get("WebSocket").New(url)
	ws := &WebSocket{
		ctx:     ctx,
		ws:      jsWS,
		ch:      make(chan js.Value, 4096),
		closeCh: make(chan struct{}),
	}
	errCh := make(chan error, 1)
	onOpen := js.FuncOf(func(this js.Value, args []js.Value) any {
		select {
		case errCh <- nil:
		default:
		}
		return nil
	})
	onError := js.FuncOf(func(this js.Value, args []js.Value) any {
		select {
		case errCh <- fmt.Errorf("websocket error"):
		default:
		}
		ws.Close()
		return nil
	})
	onClose := js.FuncOf(func(this js.Value, args []js.Value) any {
		if !ws.isClosed() {
			ws.err = errWSClosedByServer
		}
		ws.Close()
		return nil
	})
	onMessage := js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		select {
		case <-ws.ctx.Done():
			return ws.Close()
		case ws.ch <- event.Get("data").Call("arrayBuffer"):
		}
		return nil
	})
	ws.funcs = []js.Func{onOpen, onError, onClose, onMessage}
	jsWS.Call("addEventListener", "open", onOpen)
	jsWS.Call("addEventListener", "error", onError)
	jsWS.Call("addEventListener", "close", onClose)
	jsWS.Call("addEventListener", "message", onMessage)
	if err := <-errCh; err != nil {
		return nil, err
	}
	return ws, nil
}

func (ws *WebSocket) isClosed() bool {
	select {
	case <-ws.closeCh:
		return true
	default:
		return false
	}
}

// Close closes the WebSocket connection and releases JS callbacks.
func (ws *WebSocket) Close() error {
	if !ws.isClosed() {
		if ws.err == nil {
			ws.err = errWSClosed
		}
		close(ws.closeCh)
		ws.ws.Call("close")
		for _, f := range ws.funcs {
			f.Release()
		}
		ws.funcs = nil
	}
	return nil
}

func (ws *WebSocket) readChunk() error {
	select {
	case <-ws.ctx.Done():
		return ws.ctx.Err()
	case p := <-ws.ch:
		data, err := jsAwait(p)
		if err != nil {
			return err
		}
		vv := jsUint8Array.New(data)
		n := len(ws.r)
		ws.r = append(ws.r, make([]byte, vv.Length())...)
		js.CopyBytesToGo(ws.r[n:], vv)
		return nil
	case <-ws.closeCh:
		return ws.err
	}
}

// Read reads data from the WebSocket into b.
func (ws *WebSocket) Read(b []byte) (int, error) {
	var err error
	for len(ws.r) == 0 || (len(ws.r) < len(b) && len(ws.ch) > 0) {
		if err = ws.readChunk(); err != nil {
			break
		}
	}
	n := copy(b, ws.r)
	ws.r = ws.r[n:]
	return n, err
}

// Write sends data as binary WebSocket frames, chunking at 4096 bytes.
func (ws *WebSocket) Write(b []byte) (int, error) {
	if ws.isClosed() {
		return 0, ws.err
	}
	n := len(b)
	for len(b) > 0 {
		sz := min(len(b), 4096)
		ws.ws.Call("send", uint8ArrayFromBytes(b[:sz]))
		b = b[sz:]
	}
	return n, nil
}

// LocalAddr returns a dummy address (required by net.Conn).
func (ws *WebSocket) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

// RemoteAddr returns a dummy address (required by net.Conn).
func (ws *WebSocket) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

// SetDeadline is a no-op (required by net.Conn).
func (ws *WebSocket) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is a no-op (required by net.Conn).
func (ws *WebSocket) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is a no-op (required by net.Conn).
func (ws *WebSocket) SetWriteDeadline(t time.Time) error {
	return nil
}
