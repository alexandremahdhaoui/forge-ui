//go:build js && wasm

package main

import (
	"time"

	wasm "github.com/alexandremahdhaoui/forge-ui/internal/terminal/driver/wasm"
)

func main() {
	wasm.New()

	// A sleeping goroutine keeps a runtime timer active, which prevents the
	// deadlock detector from firing when other goroutines block on channels
	// resolved by asynchronous JS callbacks (e.g., IndexedDB operations).
	go func() {
		for {
			time.Sleep(time.Hour)
		}
	}()

	select {}
}
