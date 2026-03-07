//go:build unit

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

package wssproxy

import (
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyStore_AddAndList(t *testing.T) {
	t.Parallel()

	ks := newKeyStore()
	ks.Add("ws1", "ssh-ed25519 AAAA key1")

	got := ks.List("ws1")
	assert.Equal(t, []string{"ssh-ed25519 AAAA key1"}, got)
}

func TestKeyStore_AddIdempotent(t *testing.T) {
	t.Parallel()

	ks := newKeyStore()
	ks.Add("ws1", "ssh-ed25519 AAAA key1")
	ks.Add("ws1", "ssh-ed25519 AAAA key1")

	got := ks.List("ws1")
	assert.Equal(t, 1, len(got))
	assert.Equal(t, "ssh-ed25519 AAAA key1", got[0])
}

func TestKeyStore_ListUnknownWorkspace(t *testing.T) {
	t.Parallel()

	ks := newKeyStore()

	got := ks.List("unknown")
	assert.NotNil(t, got, "List should return a non-nil empty slice")
	assert.Empty(t, got)
}

func TestKeyStore_MultipleWorkspaces(t *testing.T) {
	t.Parallel()

	ks := newKeyStore()
	ks.Add("ws1", "key-a")
	ks.Add("ws2", "key-b")

	assert.Equal(t, []string{"key-a"}, ks.List("ws1"))
	assert.Equal(t, []string{"key-b"}, ks.List("ws2"))
}

func TestKeyStore_ConcurrentAddAndList(t *testing.T) {
	t.Parallel()

	ks := newKeyStore()
	var wg sync.WaitGroup

	// Concurrent writers.
	for range 50 {
		wg.Go(func() {
			ks.Add("ws1", "key-a")
			ks.Add("ws1", "key-b")
		})
	}

	// Concurrent readers.
	for range 50 {
		wg.Go(func() {
			got := ks.List("ws1")
			// Result should be empty, contain key-a, key-b, or both.
			for _, k := range got {
				if k != "key-a" && k != "key-b" {
					t.Errorf("unexpected key: %s", k)
				}
			}
		})
	}

	wg.Wait()

	// After all goroutines finish, both keys should be present.
	got := ks.List("ws1")
	sort.Strings(got)
	assert.Equal(t, []string{"key-a", "key-b"}, got)
}
