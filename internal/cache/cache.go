package cache

import (
	"sync"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// WorkspaceData holds cached git data for all repos in a single workspace.
type WorkspaceData struct {
	Summaries map[string]types.RepoSummary  // keyed by repo name
	Overviews map[string]types.RepoOverview // keyed by repo name
	UpdatedAt time.Time
}

// Cache is a thread-safe store for per-workspace git data.
// Handlers read from it; the refresher writes to it.
type Cache struct {
	mu         sync.RWMutex
	workspaces map[string]WorkspaceData // keyed by workspace name
}

// New returns an initialized Cache.
func New() *Cache {
	return &Cache{
		workspaces: make(map[string]WorkspaceData),
	}
}

// SetWorkspace atomically replaces all cached data for a workspace.
// All repos in the workspace become visible at once.
func (c *Cache) SetWorkspace(name string, data WorkspaceData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.workspaces[name] = data
}

// GetRepoSummary returns the cached RepoSummary for a specific repo in a workspace.
// The second return value is false if the workspace or repo is not in the cache.
func (c *Cache) GetRepoSummary(workspace, repo string) (types.RepoSummary, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ws, ok := c.workspaces[workspace]
	if !ok {
		return types.RepoSummary{}, false
	}
	s, ok := ws.Summaries[repo]
	return s, ok
}

// GetRepoOverview returns the cached RepoOverview for a specific repo in a workspace.
// The second return value is false if the workspace or repo is not in the cache.
func (c *Cache) GetRepoOverview(workspace, repo string) (types.RepoOverview, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ws, ok := c.workspaces[workspace]
	if !ok {
		return types.RepoOverview{}, false
	}
	o, ok := ws.Overviews[repo]
	return o, ok
}
