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
	"encoding/json"
	"errors"
	"fmt"
	"syscall/js"

	"github.com/alexandremahdhaoui/forge-ui/internal/terminal/types"
)

const (
	idbStoreName = "store"
	idbVersion   = 2
)

var errNotFound = errors.New("not found")

// Compile-time check: indexedDBKeyStore implements KeyStore.
var _ KeyStore = (*indexedDBKeyStore)(nil)

// indexedDBKeyStore implements KeyStore using browser IndexedDB.
type indexedDBKeyStore struct {
	db js.Value
}

// NewKeyStore opens an IndexedDB database with the given name and returns a
// KeyStore implementation. The object store is created on upgrade if it does
// not exist.
func NewKeyStore(dbName string) (*indexedDBKeyStore, error) {
	req := js.Global().Get("indexedDB").Call("open", js.ValueOf(dbName), js.ValueOf(idbVersion))

	type result struct {
		v   js.Value
		err error
	}
	resCh := make(chan result, 2)

	onUpgrade := js.FuncOf(func(this js.Value, args []js.Value) any {
		db := req.Get("result")
		names := db.Get("objectStoreNames")
		var found bool
		for i := 0; i < names.Length(); i++ {
			if names.Index(i).String() == idbStoreName {
				found = true
				break
			}
		}
		if !found {
			db.Call("createObjectStore", idbStoreName)
		}
		return nil
	})
	onError := js.FuncOf(func(this js.Value, args []js.Value) any {
		resCh <- result{err: errors.New("error opening indexeddb")}
		return nil
	})
	onSuccess := js.FuncOf(func(this js.Value, args []js.Value) any {
		resCh <- result{v: req.Get("result")}
		return nil
	})
	req.Set("onupgradeneeded", onUpgrade)
	req.Set("onerror", onError)
	req.Set("onsuccess", onSuccess)

	r := <-resCh
	onUpgrade.Release()
	onError.Release()
	onSuccess.Release()
	if r.err != nil {
		return nil, r.err
	}
	return &indexedDBKeyStore{db: r.v}, nil
}

// get reads a value from the object store and JSON-deserializes it.
func (s *indexedDBKeyStore) get(key string, value any) error {
	req := s.db.Call("transaction", idbStoreName).Call("objectStore", idbStoreName).Call("get", key)
	errCh := make(chan error, 2)
	onError := js.FuncOf(func(this js.Value, args []js.Value) any {
		errCh <- errors.New("indexeddb transaction error")
		return nil
	})
	onSuccess := js.FuncOf(func(this js.Value, args []js.Value) any {
		errCh <- nil
		return nil
	})
	req.Set("onerror", onError)
	req.Set("onsuccess", onSuccess)
	err := <-errCh
	onError.Release()
	onSuccess.Release()
	if err != nil {
		return err
	}
	v := req.Get("result")
	if v.IsUndefined() {
		return errNotFound
	}
	if err := json.Unmarshal([]byte(v.String()), value); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

// set JSON-serializes a value and writes it to the object store.
func (s *indexedDBKeyStore) set(key string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	t := s.db.Call("transaction", idbStoreName, "readwrite")
	req := t.Call("objectStore", idbStoreName).Call("put", string(b), key)
	errCh := make(chan error, 2)
	onError := js.FuncOf(func(this js.Value, args []js.Value) any {
		errCh <- errors.New("indexeddb transaction error")
		return nil
	})
	onSuccess := js.FuncOf(func(this js.Value, args []js.Value) any {
		errCh <- nil
		return nil
	})
	req.Set("onerror", onError)
	req.Set("onsuccess", onSuccess)
	result := <-errCh
	onError.Release()
	onSuccess.Release()
	return result
}

// ListKeys returns all stored SSH keys.
func (s *indexedDBKeyStore) ListKeys() ([]types.SSHKey, error) {
	var keys []types.SSHKey
	if err := s.get("keys", &keys); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return keys, nil
}

// GetKey returns a single SSH key by name.
func (s *indexedDBKeyStore) GetKey(name string) (types.SSHKey, error) {
	keys, err := s.ListKeys()
	if err != nil {
		return types.SSHKey{}, err
	}
	for _, k := range keys {
		if k.Name == name {
			return k, nil
		}
	}
	return types.SSHKey{}, fmt.Errorf("key %q: %w", name, errNotFound)
}

// SaveKey upserts an SSH key by name.
func (s *indexedDBKeyStore) SaveKey(key types.SSHKey) error {
	keys, err := s.ListKeys()
	if err != nil {
		return err
	}
	found := false
	for i, k := range keys {
		if k.Name == key.Name {
			keys[i] = key
			found = true
			break
		}
	}
	if !found {
		keys = append(keys, key)
	}
	return s.set("keys", keys)
}

// DeleteKey removes an SSH key by name.
func (s *indexedDBKeyStore) DeleteKey(name string) error {
	keys, err := s.ListKeys()
	if err != nil {
		return err
	}
	filtered := make([]types.SSHKey, 0, len(keys))
	for _, k := range keys {
		if k.Name != name {
			filtered = append(filtered, k)
		}
	}
	return s.set("keys", filtered)
}

// ListEndpoints returns all stored terminal endpoints.
func (s *indexedDBKeyStore) ListEndpoints() ([]types.TerminalEndpoint, error) {
	var eps []types.TerminalEndpoint
	if err := s.get("endpoints", &eps); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return eps, nil
}

// SaveEndpoint upserts a terminal endpoint by name.
func (s *indexedDBKeyStore) SaveEndpoint(ep types.TerminalEndpoint) error {
	eps, err := s.ListEndpoints()
	if err != nil {
		return err
	}
	found := false
	for i, e := range eps {
		if e.Name == ep.Name {
			eps[i] = ep
			found = true
			break
		}
	}
	if !found {
		eps = append(eps, ep)
	}
	return s.set("endpoints", eps)
}

// ListHosts returns all stored known hosts.
func (s *indexedDBKeyStore) ListHosts() ([]types.KnownHost, error) {
	var hosts []types.KnownHost
	if err := s.get("hosts", &hosts); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return hosts, nil
}

// SaveHost upserts a known host by name.
func (s *indexedDBKeyStore) SaveHost(host types.KnownHost) error {
	hosts, err := s.ListHosts()
	if err != nil {
		return err
	}
	found := false
	for i, h := range hosts {
		if h.Name == host.Name {
			hosts[i] = host
			found = true
			break
		}
	}
	if !found {
		hosts = append(hosts, host)
	}
	return s.set("hosts", hosts)
}

// GetParams returns stored parameters.
func (s *indexedDBKeyStore) GetParams() (map[string]string, error) {
	var params map[string]string
	if err := s.get("params", &params); err != nil {
		if errors.Is(err, errNotFound) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	return params, nil
}

// SaveParams stores parameters.
func (s *indexedDBKeyStore) SaveParams(params map[string]string) error {
	return s.set("params", params)
}
