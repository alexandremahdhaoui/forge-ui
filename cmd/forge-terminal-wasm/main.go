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
