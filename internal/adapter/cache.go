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
