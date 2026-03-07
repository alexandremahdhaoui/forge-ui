//go:build unit

// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/mocks/mockadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func stubOrchestrationMocksForWorkspace() (*mockadapter.MockWsConfigLoader, *mockadapter.MockMetaPlanLoader, *mockadapter.MockRepoPlanLoader) {
	wc := new(mockadapter.MockWsConfigLoader)
	mp := new(mockadapter.MockMetaPlanLoader)
	rp := new(mockadapter.MockRepoPlanLoader)
	wc.On("Load", mock.Anything).Return(types.WsConfig{}, errors.New("none")).Maybe()
	mp.On("LoadAll", mock.Anything).Return(nil, errors.New("none")).Maybe()
	rp.On("LoadSummary", mock.Anything, mock.Anything).Return(types.RepoPlanSummary{}, errors.New("none")).Maybe()
	rp.On("LoadAll", mock.Anything).Return(nil, errors.New("none")).Maybe()
	return wc, mp, rp
}

func TestGetWorkspace_Success(t *testing.T) {
	t.Parallel()

	ws := new(mockadapter.MockWorkspaceDiscovery)
	c := new(mockadapter.MockCache)
	fl := new(mockadapter.MockForgeLoader)

	wsData := types.WorkspacePageData{
		Name: "ws1",
		Repos: []types.RepoSummary{
			{Name: "repo-a", Path: "/path/repo-a"},
			{Name: "repo-b", Path: "/path/repo-b"},
		},
	}

	ws.On("Get", "/base", "ws1").Return(wsData, nil)
	c.On("GetRepoSummary", "default/ws1", mock.Anything).Return(types.RepoSummary{}, false)

	wc, mp, rp := stubOrchestrationMocksForWorkspace()
	svc := NewWorkspaceService(ws, c, fl, wc, mp, rp)
	result, err := svc.GetWorkspace("/base", "default", "ws1", "name")

	require.NoError(t, err)
	assert.Equal(t, "ws1", result.Name)
	assert.Equal(t, "default", result.PortfolioName)
	assert.Equal(t, "name", result.SortMode)
	assert.Equal(t, 2, result.Stats.TotalRepos)
	// Verify repo links
	assert.Equal(t, "/portfolios/default/workspaces/ws1/repos/repo-a", result.Repos[0].RepoLink)
	assert.Equal(t, "/portfolios/default/workspaces/ws1/repos/repo-b", result.Repos[1].RepoLink)

	ws.AssertExpectations(t)
}

func TestGetWorkspace_WithCachedGitData(t *testing.T) {
	t.Parallel()

	wsMock := new(mockadapter.MockWorkspaceDiscovery)
	c := new(mockadapter.MockCache)
	fl := new(mockadapter.MockForgeLoader)

	commitTime := time.Date(2025, 7, 1, 8, 0, 0, 0, time.UTC)

	wsData := types.WorkspacePageData{
		Name: "ws1",
		Repos: []types.RepoSummary{
			{Name: "repo-a", Path: "/path/repo-a"},
		},
	}

	wsMock.On("Get", "/base/myportfolio", "ws1").Return(wsData, nil)
	c.On("GetRepoSummary", "myportfolio/ws1", "repo-a").Return(types.RepoSummary{
		Branch:         "develop",
		IsDirty:        true,
		StatusFiles:    []types.StatusEntry{{Code: "M", FilePath: "main.go"}},
		DiffStat:       "1 file changed",
		RecentLogs:     []types.LogEntry{{Hash: "abc123", Message: "fix bug"}},
		Ahead:          5,
		Behind:         0,
		HasUpstream:    true,
		LastCommitTime: commitTime,
	}, true)

	wc, mp, rp := stubOrchestrationMocksForWorkspace()
	svc := NewWorkspaceService(wsMock, c, fl, wc, mp, rp)
	result, err := svc.GetWorkspace("/base", "myportfolio", "ws1", "name")

	require.NoError(t, err)
	repo := result.Repos[0]
	assert.Equal(t, "develop", repo.Branch)
	assert.True(t, repo.IsDirty)
	assert.Len(t, repo.StatusFiles, 1)
	assert.Equal(t, "1 file changed", repo.DiffStat)
	assert.Len(t, repo.RecentLogs, 1)
	assert.Equal(t, 5, repo.Ahead)
	assert.True(t, repo.HasUpstream)
	assert.Equal(t, commitTime, repo.LastCommitTime)

	wsMock.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestGetWorkspace_WithForgeHeatmap(t *testing.T) {
	t.Parallel()

	wsMock := new(mockadapter.MockWorkspaceDiscovery)
	c := new(mockadapter.MockCache)
	fl := new(mockadapter.MockForgeLoader)

	wsData := types.WorkspacePageData{
		Name: "ws1",
		Repos: []types.RepoSummary{
			{Name: "repo-a", Path: "/path/repo-a", HasForge: true},
		},
	}

	wsMock.On("Get", "/base", "ws1").Return(wsData, nil)
	c.On("GetRepoSummary", "default/ws1", "repo-a").Return(types.RepoSummary{}, false)
	fl.On("Load", "/path/repo-a").Return(types.ForgePageData{
		Spec: types.ForgeSpec{
			Test: []types.TestSpec{{Name: "unit"}, {Name: "integration"}},
		},
		TestReports: []types.TestReport{
			{Stage: "unit", Status: "passed", Stats: types.TestStats{Total: 20, Passed: 18, Failed: 2}},
			{Stage: "integration", Status: "failed", Stats: types.TestStats{Total: 5, Passed: 3, Failed: 2}},
		},
	}, nil)

	wc, mp, rp := stubOrchestrationMocksForWorkspace()
	svc := NewWorkspaceService(wsMock, c, fl, wc, mp, rp)
	result, err := svc.GetWorkspace("/base", "default", "ws1", "name")

	require.NoError(t, err)
	assert.Equal(t, 25, result.Stats.TotalTests)
	assert.Equal(t, 21, result.Stats.Passed)
	assert.Equal(t, 4, result.Stats.Failed)
	assert.Equal(t, 1, result.Stats.ForgeRepos)
	assert.Len(t, result.AllStages, 2)
	assert.Contains(t, result.AllStages, "unit")
	assert.Contains(t, result.AllStages, "integration")
	assert.Len(t, result.RepoForge, 1)

	fl.AssertExpectations(t)
}

func TestGetWorkspace_SortByTime(t *testing.T) {
	t.Parallel()

	wsMock := new(mockadapter.MockWorkspaceDiscovery)
	c := new(mockadapter.MockCache)
	fl := new(mockadapter.MockForgeLoader)

	now := time.Now()

	wsData := types.WorkspacePageData{
		Name: "ws1",
		Repos: []types.RepoSummary{
			{Name: "old-repo", Path: "/path/old-repo"},
			{Name: "new-repo", Path: "/path/new-repo"},
		},
	}

	wsMock.On("Get", "/base", "ws1").Return(wsData, nil)
	c.On("GetRepoSummary", "default/ws1", "old-repo").Return(types.RepoSummary{
		LastCommitTime: now.Add(-2 * time.Hour),
	}, true)
	c.On("GetRepoSummary", "default/ws1", "new-repo").Return(types.RepoSummary{
		LastCommitTime: now.Add(-10 * time.Minute),
	}, true)

	wc, mp, rp := stubOrchestrationMocksForWorkspace()
	svc := NewWorkspaceService(wsMock, c, fl, wc, mp, rp)
	result, err := svc.GetWorkspace("/base", "default", "ws1", "time")

	require.NoError(t, err)
	assert.Equal(t, "time", result.SortMode)
	// Newer repo should be first
	assert.Equal(t, "new-repo", result.Repos[0].Name)
	assert.Equal(t, "old-repo", result.Repos[1].Name)

	wsMock.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestGetWorkspace_SortByName(t *testing.T) {
	t.Parallel()

	wsMock := new(mockadapter.MockWorkspaceDiscovery)
	c := new(mockadapter.MockCache)
	fl := new(mockadapter.MockForgeLoader)

	wsData := types.WorkspacePageData{
		Name: "ws1",
		Repos: []types.RepoSummary{
			{Name: "zebra", Path: "/path/zebra"},
			{Name: "alpha", Path: "/path/alpha"},
		},
	}

	wsMock.On("Get", "/base", "ws1").Return(wsData, nil)
	c.On("GetRepoSummary", "default/ws1", mock.Anything).Return(types.RepoSummary{}, false)

	wc, mp, rp := stubOrchestrationMocksForWorkspace()
	svc := NewWorkspaceService(wsMock, c, fl, wc, mp, rp)
	result, err := svc.GetWorkspace("/base", "default", "ws1", "name")

	require.NoError(t, err)
	assert.Equal(t, "alpha", result.Repos[0].Name)
	assert.Equal(t, "zebra", result.Repos[1].Name)

	wsMock.AssertExpectations(t)
}

func TestGetWorkspace_NotFound(t *testing.T) {
	t.Parallel()

	wsMock := new(mockadapter.MockWorkspaceDiscovery)
	c := new(mockadapter.MockCache)
	fl := new(mockadapter.MockForgeLoader)

	wsMock.On("Get", "/base", "nonexistent").Return(types.WorkspacePageData{}, errors.New("workspace not found"))

	wc, mp, rp := stubOrchestrationMocksForWorkspace()
	svc := NewWorkspaceService(wsMock, c, fl, wc, mp, rp)
	_, err := svc.GetWorkspace("/base", "default", "nonexistent", "name")

	assert.EqualError(t, err, "workspace not found")
	wsMock.AssertExpectations(t)
}

func TestGetWorkspace_NamedPortfolioPath(t *testing.T) {
	t.Parallel()

	wsMock := new(mockadapter.MockWorkspaceDiscovery)
	c := new(mockadapter.MockCache)
	fl := new(mockadapter.MockForgeLoader)

	wsData := types.WorkspacePageData{
		Name:  "ws1",
		Repos: []types.RepoSummary{},
	}

	// When portfolio != "default", basedir should be baseDir/portfolio
	wsMock.On("Get", "/base/myportfolio", "ws1").Return(wsData, nil)

	wc, mp, rp := stubOrchestrationMocksForWorkspace()
	svc := NewWorkspaceService(wsMock, c, fl, wc, mp, rp)
	result, err := svc.GetWorkspace("/base", "myportfolio", "ws1", "name")

	require.NoError(t, err)
	assert.Equal(t, "myportfolio", result.PortfolioName)
	wsMock.AssertExpectations(t)
}
