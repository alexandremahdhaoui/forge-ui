package mockadapter

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/mock"
)

// WorkspaceDiscovery is a mock implementation of adapter.WorkspaceDiscovery.
type WorkspaceDiscovery struct {
	mock.Mock
}

func (m *WorkspaceDiscovery) List(basedir string) ([]types.WorkspaceSummary, error) {
	args := m.Called(basedir)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.WorkspaceSummary), args.Error(1)
}

func (m *WorkspaceDiscovery) Get(basedir, name string) (types.WorkspacePageData, error) {
	args := m.Called(basedir, name)
	return args.Get(0).(types.WorkspacePageData), args.Error(1)
}
