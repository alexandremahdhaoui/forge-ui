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
	"io"
	"sync"
	"syscall/js"
)

// Compile-time check: terminalIO implements TerminalIO.
var _ TerminalIO = (*terminalIO)(nil)

type resizeEvent struct {
	cols int
	rows int
}

// terminalIO bridges xterm.js callbacks to Go channels, implementing the
// TerminalIO interface as a raw I/O pipe for tmux passthrough.
type terminalIO struct {
	xt       js.Value // xterm.js Terminal instance
	dataCh   chan []byte
	resizeCh chan resizeEvent
	closeCh  chan struct{}
	dispose  []js.Value
	funcs    []js.Func // JS callbacks to release on Close
	r        []byte

	mu       sync.Mutex
	onResize func(cols, rows int)
}

// NewTerminalIO creates a new TerminalIO adapter that bridges xterm.js to Go.
func NewTerminalIO(xt js.Value) *terminalIO {
	t := &terminalIO{
		xt:       xt,
		dataCh:   make(chan []byte, 100),
		resizeCh: make(chan resizeEvent, 10),
		closeCh:  make(chan struct{}),
	}

	// Register onData: pushes key data into dataCh.
	onDataFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		key := args[0].String()
		select {
		case <-t.closeCh:
		case t.dataCh <- []byte(key):
		default:
			// Input buffer full, drop data.
		}
		return nil
	})
	t.funcs = append(t.funcs, onDataFn)
	t.dispose = append(t.dispose, xt.Call("onData", onDataFn))

	// Register onResize: pushes resize events into resizeCh.
	onResizeFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		select {
		case <-t.closeCh:
		case t.resizeCh <- resizeEvent{
			cols: event.Get("cols").Int(),
			rows: event.Get("rows").Int(),
		}:
		default:
			// Resize buffer full, drop event.
		}
		return nil
	})
	t.funcs = append(t.funcs, onResizeFn)
	t.dispose = append(t.dispose, xt.Call("onResize", onResizeFn))

	// Goroutine to process resize events and call the onResize callback.
	go func() {
		for {
			select {
			case <-t.closeCh:
				return
			case resize, ok := <-t.resizeCh:
				if !ok {
					return
				}
				t.mu.Lock()
				cb := t.onResize
				t.mu.Unlock()
				if cb != nil {
					cb(resize.cols, resize.rows)
				}
			}
		}
	}()

	return t
}

func (t *terminalIO) isClosed() bool {
	select {
	case <-t.closeCh:
		return true
	default:
		return false
	}
}

func (t *terminalIO) readChunk() error {
	select {
	case b := <-t.dataCh:
		t.r = append(t.r, b...)
		return nil
	case <-t.closeCh:
		return io.EOF
	}
}

// Read reads user input from xterm.js onData events.
func (t *terminalIO) Read(b []byte) (int, error) {
	var err error
	for len(t.r) == 0 || (len(b) > len(t.r) && len(t.dataCh) > 0) {
		if err = t.readChunk(); err != nil {
			break
		}
	}
	n := copy(b, t.r)
	t.r = t.r[n:]
	return n, err
}

// Write sends data to xterm.js and waits for the write callback.
func (t *terminalIO) Write(b []byte) (int, error) {
	if t.isClosed() {
		return 0, io.EOF
	}
	ch := make(chan struct{})
	cb := js.FuncOf(func(this js.Value, args []js.Value) any {
		close(ch)
		return nil
	})
	t.xt.Call("write", uint8ArrayFromBytes(b), cb)
	<-ch
	cb.Release()
	return len(b), nil
}

// OnResize registers a callback invoked when the terminal is resized.
func (t *terminalIO) OnResize(cb func(cols, rows int)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onResize = cb
}

// Cols returns the current terminal column count.
func (t *terminalIO) Cols() int {
	return t.xt.Get("cols").Int()
}

// Rows returns the current terminal row count.
func (t *terminalIO) Rows() int {
	return t.xt.Get("rows").Int()
}

// Close disposes JS event listeners, releases Go callbacks, and closes channels.
func (t *terminalIO) Close() error {
	if t.isClosed() {
		return nil
	}
	for _, d := range t.dispose {
		d.Call("dispose")
	}
	t.dispose = nil
	for _, f := range t.funcs {
		f.Release()
	}
	t.funcs = nil
	close(t.closeCh)
	return nil
}
