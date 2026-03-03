//go:build js && wasm

package wasm

import (
	"syscall/js"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/controller"
	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
)

// Driver wires terminal adapters into the controller and manages the browser
// lifecycle. It registers forgeTerminal.start and forgeTerminal.stop on the
// JS global object.
type Driver struct {
	controller controller.SessionController
	termIO     adapter.TerminalIO
	startFn    js.Func
	stopFn     js.Func
}

// New creates a Driver and registers forgeTerminal on js.Global() with start
// and stop functions callable from JavaScript.
func New() *Driver {
	d := &Driver{}

	d.startFn = js.FuncOf(func(this js.Value, args []js.Value) any {
		d.start(args)
		return nil
	})

	d.stopFn = js.FuncOf(func(this js.Value, args []js.Value) any {
		d.stop()
		return nil
	})

	js.Global().Set("forgeTerminal", map[string]any{
		"start": d.startFn,
		"stop":  d.stopFn,
	})

	return d
}

// start initializes adapters and starts the terminal session.
// Called from JS: forgeTerminal.start(xtermTerminal, workspace, endpoint)
//
// The entire body runs in a goroutine so the js.FuncOf callback returns
// immediately, unblocking the JS event loop. This is required because
// NewKeyStore opens IndexedDB asynchronously and waits on a channel for the
// result — if the JS event loop is blocked by a synchronous callback, the
// IndexedDB onsuccess handler can never fire, causing a deadlock.
func (d *Driver) start(args []js.Value) {
	// Clean up any previous session before starting a new one.
	d.stop()

	xt := args[0]
	workspace := args[1].String()
	endpoint := args[2].String()

	go func() {
		// Create adapters.
		termIO := adapter.NewTerminalIO(xt)
		sshClient := adapter.NewSSHClient()
		keyStore, err := adapter.NewKeyStore("forge-terminal")
		if err != nil {
			xt.Call("write", "\r\nError: "+err.Error()+"\r\n")
			return
		}

		// Create controller.
		registrar := adapter.NewKeyRegistrar(endpoint)
		ctrl := controller.New(sshClient, keyStore, registrar)

		// Build config.
		cfg := types.TerminalConfig{
			Workspace: workspace,
			Endpoints: []types.TerminalEndpoint{
				{
					Name:    "default",
					URL:     endpoint,
					Default: true,
				},
			},
			AutoConnect: true,
		}

		// Store on driver for stop().
		d.termIO = termIO
		d.controller = ctrl

		if err := d.controller.Start(cfg, d.termIO); err != nil {
			xt.Call("write", "\r\nError: "+err.Error()+"\r\n")
		}
	}()
}

// stop terminates the active terminal session and closes the terminal I/O.
func (d *Driver) stop() {
	if d.controller != nil {
		_ = d.controller.Stop()
	}
	if d.termIO != nil {
		_ = d.termIO.Close()
	}
}
