package mockcontroller

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/mock"
)

// WorkspaceService is a mock implementation of controller.WorkspaceService.
type WorkspaceService struct {
	mock.Mock
}

func (m *WorkspaceService) GetWorkspace(baseDir, portfolio, workspace, sortMode string) (types.WorkspacePageData, error) {
	args := m.Called(baseDir, portfolio, workspace, sortMode)
	return args.Get(0).(types.WorkspacePageData), args.Error(1)
}
