package mockadapter

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/mock"
)

// ForgeLoader is a mock implementation of adapter.ForgeLoader.
type ForgeLoader struct {
	mock.Mock
}

func (m *ForgeLoader) Load(repoPath string) (types.ForgePageData, error) {
	args := m.Called(repoPath)
	return args.Get(0).(types.ForgePageData), args.Error(1)
}
