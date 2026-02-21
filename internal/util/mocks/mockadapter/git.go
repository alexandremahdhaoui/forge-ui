package mockadapter

import (
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/stretchr/testify/mock"
)

// GitInfo is a mock implementation of adapter.GitInfo.
type GitInfo struct {
	mock.Mock
}

func (m *GitInfo) RepoInfo(repoPath string) (types.RepoSummary, error) {
	args := m.Called(repoPath)
	return args.Get(0).(types.RepoSummary), args.Error(1)
}
