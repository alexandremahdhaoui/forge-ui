package mockadapter

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/mock"
)

// Cache is a mock implementation of adapter.Cache.
type Cache struct {
	mock.Mock
}

func (m *Cache) SetWorkspace(name string, data types.CacheWorkspaceData) {
	m.Called(name, data)
}

func (m *Cache) GetRepoSummary(workspace, repo string) (types.RepoSummary, bool) {
	args := m.Called(workspace, repo)
	return args.Get(0).(types.RepoSummary), args.Bool(1)
}

func (m *Cache) GetRepoOverview(workspace, repo string) (types.RepoOverview, bool) {
	args := m.Called(workspace, repo)
	return args.Get(0).(types.RepoOverview), args.Bool(1)
}
