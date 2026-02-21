package mockcontroller

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/mock"
)

// ForgeService is a mock implementation of controller.ForgeService.
type ForgeService struct {
	mock.Mock
}

func (m *ForgeService) GetForge(baseDir, portfolio, workspace, repo string) (types.ForgePageData, error) {
	args := m.Called(baseDir, portfolio, workspace, repo)
	return args.Get(0).(types.ForgePageData), args.Error(1)
}
