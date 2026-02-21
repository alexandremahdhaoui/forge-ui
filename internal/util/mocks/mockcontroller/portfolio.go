package mockcontroller

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/mock"
)

// PortfolioService is a mock implementation of controller.PortfolioService.
type PortfolioService struct {
	mock.Mock
}

func (m *PortfolioService) ListPortfolios(baseDir, sortMode string) (types.PortfoliosPageData, error) {
	args := m.Called(baseDir, sortMode)
	return args.Get(0).(types.PortfoliosPageData), args.Error(1)
}

func (m *PortfolioService) GetPortfolio(baseDir, name, sortMode string) (types.PortfolioPageData, error) {
	args := m.Called(baseDir, name, sortMode)
	return args.Get(0).(types.PortfolioPageData), args.Error(1)
}
