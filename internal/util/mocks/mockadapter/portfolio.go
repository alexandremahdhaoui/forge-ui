package mockadapter

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/mock"
)

// PortfolioDiscovery is a mock implementation of adapter.PortfolioDiscovery.
type PortfolioDiscovery struct {
	mock.Mock
}

func (m *PortfolioDiscovery) List(baseDir string) ([]types.PortfolioSummary, error) {
	args := m.Called(baseDir)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.PortfolioSummary), args.Error(1)
}

func (m *PortfolioDiscovery) Get(baseDir, name string) (types.PortfolioPageData, error) {
	args := m.Called(baseDir, name)
	return args.Get(0).(types.PortfolioPageData), args.Error(1)
}
