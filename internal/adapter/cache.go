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

//go:build !js || !wasm

package adapter

import (
	"sync"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// Cache provides thread-safe access to per-workspace cached git data.
type Cache interface {
	SetWorkspace(name string, data types.CacheWorkspaceData)
	GetRepoSummary(workspace, repo string) (types.RepoSummary, bool)
	GetRepoOverview(workspace, repo string) (types.RepoOverview, bool)
}

type inMemoryCache struct {
	mu         sync.RWMutex
	workspaces map[string]types.CacheWorkspaceData
}

// NewCache returns a thread-safe in-memory Cache.
func NewCache() Cache {
	return &inMemoryCache{
		workspaces: make(map[string]types.CacheWorkspaceData),
	}
}

func (c *inMemoryCache) SetWorkspace(name string, data types.CacheWorkspaceData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.workspaces[name] = data
}

func (c *inMemoryCache) GetRepoSummary(workspace, repo string) (types.RepoSummary, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ws, ok := c.workspaces[workspace]
	if !ok {
		return types.RepoSummary{}, false
	}
	s, ok := ws.Summaries[repo]
	return s, ok
}

func (c *inMemoryCache) GetRepoOverview(workspace, repo string) (types.RepoOverview, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ws, ok := c.workspaces[workspace]
	if !ok {
		return types.RepoOverview{}, false
	}
	o, ok := ws.Overviews[repo]
	return o, ok
}
