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

import "sync"

// keyStore holds registered SSH public keys per workspace.
// Keys are stored in memory and lost on restart. Clients re-register
// on each session start, so persistence is not required.
type keyStore struct {
	mu   sync.RWMutex
	keys map[string]map[string]struct{} // workspace -> set of public keys
}

func newKeyStore() *keyStore {
	return &keyStore{keys: make(map[string]map[string]struct{})}
}

// Add registers a public key for the given workspace. The operation is
// idempotent: adding the same key twice has no additional effect.
func (s *keyStore) Add(workspace, publicKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys[workspace] == nil {
		s.keys[workspace] = make(map[string]struct{})
	}
	s.keys[workspace][publicKey] = struct{}{}
}

// List returns all registered public keys for the given workspace.
// Returns an empty slice if no keys are registered.
func (s *keyStore) List(workspace string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.keys[workspace]))
	for k := range s.keys[workspace] {
		keys = append(keys, k)
	}
	return keys
}
