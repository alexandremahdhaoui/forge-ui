//go:build !js || !wasm

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

func stubOrchestrationMocksForPortfolio() (*mockadapter.WsConfigLoader, *mockadapter.MetaPlanLoader, *mockadapter.PortfolioConfigLoader) {
	wc := new(mockadapter.WsConfigLoader)
	mp := new(mockadapter.MetaPlanLoader)
	pc := new(mockadapter.PortfolioConfigLoader)
	pc.On("Load", mock.Anything).Return(types.PortfolioConfig{}, errors.New("none")).Maybe()
	wc.On("Load", mock.Anything).Return(types.WsConfig{}, errors.New("none")).Maybe()
	mp.On("LoadAll", mock.Anything).Return(nil, errors.New("none")).Maybe()
	return wc, mp, pc
}

func TestListPortfolios_SortByTime(t *testing.T) {
	t.Parallel()

	pd := new(mockadapter.PortfolioDiscovery)
	c := new(mockadapter.Cache)
	fl := new(mockadapter.ForgeLoader)

	now := time.Now()
	portfolios := []types.PortfolioSummary{
		{
			Name: "alpha",
			Workspaces: []types.WorkspaceSummary{
				{
					Name:      "ws-old",
					RepoCount: 1,
					Repos: []types.RepoOverview{
						{Name: "repo-a"},
					},
				},
			},
		},
		{
			Name: "beta",
			Workspaces: []types.WorkspaceSummary{
				{
					Name:      "ws-new",
					RepoCount: 1,
					Repos: []types.RepoOverview{
						{Name: "repo-b"},
					},
				},
			},
		},
	}

	pd.On("List", "/base").Return(portfolios, nil)
	c.On("GetRepoOverview", "alpha/ws-old", "repo-a").Return(types.RepoOverview{
		LastCommitTime: now.Add(-2 * time.Hour),
	}, true)
	c.On("GetRepoOverview", "beta/ws-new", "repo-b").Return(types.RepoOverview{
		LastCommitTime: now.Add(-10 * time.Minute),
	}, true)

	wc, mp, pc := stubOrchestrationMocksForPortfolio()

	svc := NewPortfolioService(pd, c, fl, wc, mp, pc)
	result, err := svc.ListPortfolios("/base", "time")

	require.NoError(t, err)
	assert.Equal(t, "time", result.SortMode)
	assert.Equal(t, 2, result.Stats.TotalPortfolios)
	assert.Equal(t, 2, result.Stats.TotalRepos)

	pd.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestListPortfolios_SortByName(t *testing.T) {
	t.Parallel()

	pd := new(mockadapter.PortfolioDiscovery)
	c := new(mockadapter.Cache)
	fl := new(mockadapter.ForgeLoader)

	portfolios := []types.PortfolioSummary{
		{
			Name: "only",
			Workspaces: []types.WorkspaceSummary{
				{
					Name:      "ws1",
					RepoCount: 1,
					Repos:     []types.RepoOverview{{Name: "r1"}},
				},
			},
		},
	}

	pd.On("List", "/base").Return(portfolios, nil)
	c.On("GetRepoOverview", "only/ws1", "r1").Return(types.RepoOverview{}, false)

	wc, mp, pc := stubOrchestrationMocksForPortfolio()
	svc := NewPortfolioService(pd, c, fl, wc, mp, pc)
	result, err := svc.ListPortfolios("/base", "name")

	require.NoError(t, err)
	assert.Equal(t, "name", result.SortMode)
	assert.Len(t, result.Portfolios, 1)

	pd.AssertExpectations(t)
}

func TestListPortfolios_ErrorFromDiscovery(t *testing.T) {
	t.Parallel()

	pd := new(mockadapter.PortfolioDiscovery)
	c := new(mockadapter.Cache)
	fl := new(mockadapter.ForgeLoader)

	pd.On("List", "/base").Return(nil, errors.New("disk error"))

	wc, mp, pc := stubOrchestrationMocksForPortfolio()
	svc := NewPortfolioService(pd, c, fl, wc, mp, pc)
	_, err := svc.ListPortfolios("/base", "time")

	assert.EqualError(t, err, "disk error")

	pd.AssertExpectations(t)
}

func TestGetPortfolio_Success(t *testing.T) {
	t.Parallel()

	pd := new(mockadapter.PortfolioDiscovery)
	c := new(mockadapter.Cache)
	fl := new(mockadapter.ForgeLoader)

	data := types.PortfolioPageData{
		Name: "myportfolio",
		Workspaces: []types.WorkspaceSummary{
			{
				Name:      "ws1",
				RepoCount: 2,
				Repos: []types.RepoOverview{
					{Name: "repo-a"},
					{Name: "repo-b"},
				},
			},
		},
	}

	pd.On("Get", "/base", "myportfolio").Return(data, nil)
	c.On("GetRepoOverview", mock.Anything, mock.Anything).Return(types.RepoOverview{}, false)

	wc, mp, pc := stubOrchestrationMocksForPortfolio()
	svc := NewPortfolioService(pd, c, fl, wc, mp, pc)
	result, err := svc.GetPortfolio("/base", "myportfolio", "name")

	require.NoError(t, err)
	assert.Equal(t, "name", result.SortMode)
	assert.Equal(t, 2, result.Stats.TotalRepos)
	// Verify repo links were rewritten
	assert.Equal(t, "/portfolios/myportfolio/workspaces/ws1/repos/repo-a", result.Workspaces[0].Repos[0].RepoLink)
	assert.Equal(t, "/portfolios/myportfolio/workspaces/ws1/repos/repo-b", result.Workspaces[0].Repos[1].RepoLink)

	pd.AssertExpectations(t)
}

func TestGetPortfolio_NotFound(t *testing.T) {
	t.Parallel()

	pd := new(mockadapter.PortfolioDiscovery)
	c := new(mockadapter.Cache)
	fl := new(mockadapter.ForgeLoader)

	pd.On("Get", "/base", "missing").Return(types.PortfolioPageData{}, errors.New("not found"))

	wc, mp, pc := stubOrchestrationMocksForPortfolio()
	svc := NewPortfolioService(pd, c, fl, wc, mp, pc)
	_, err := svc.GetPortfolio("/base", "missing", "time")

	assert.EqualError(t, err, "not found")
	pd.AssertExpectations(t)
}

func TestGetPortfolio_WithCacheEnrichment(t *testing.T) {
	t.Parallel()

	pd := new(mockadapter.PortfolioDiscovery)
	c := new(mockadapter.Cache)
	fl := new(mockadapter.ForgeLoader)

	commitTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	data := types.PortfolioPageData{
		Name: "p1",
		Workspaces: []types.WorkspaceSummary{
			{
				Name:      "ws1",
				RepoCount: 1,
				Repos: []types.RepoOverview{
					{Name: "repo-a"},
				},
			},
		},
	}

	pd.On("Get", "/base", "p1").Return(data, nil)
	c.On("GetRepoOverview", "p1/ws1", "repo-a").Return(types.RepoOverview{
		Branch:         "main",
		IsDirty:        true,
		Ahead:          3,
		Behind:         1,
		HasUpstream:    true,
		LastCommitTime: commitTime,
	}, true)

	wc, mp, pc := stubOrchestrationMocksForPortfolio()
	svc := NewPortfolioService(pd, c, fl, wc, mp, pc)
	result, err := svc.GetPortfolio("/base", "p1", "time")

	require.NoError(t, err)
	repo := result.Workspaces[0].Repos[0]
	assert.Equal(t, "main", repo.Branch)
	assert.True(t, repo.IsDirty)
	assert.Equal(t, 3, repo.Ahead)
	assert.Equal(t, 1, repo.Behind)
	assert.True(t, repo.HasUpstream)
	assert.Equal(t, commitTime, repo.LastCommitTime)
	assert.Equal(t, 1, result.Stats.DirtyRepos)

	pd.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestGetPortfolio_WithForgeHeatmap(t *testing.T) {
	t.Parallel()

	pd := new(mockadapter.PortfolioDiscovery)
	c := new(mockadapter.Cache)
	fl := new(mockadapter.ForgeLoader)

	data := types.PortfolioPageData{
		Name: "p1",
		Workspaces: []types.WorkspaceSummary{
			{
				Name:      "ws1",
				RepoCount: 1,
				Repos: []types.RepoOverview{
					{Name: "repo-a", HasForge: true, Path: "/repos/repo-a"},
				},
			},
		},
	}

	pd.On("Get", "/base", "p1").Return(data, nil)
	c.On("GetRepoOverview", "p1/ws1", "repo-a").Return(types.RepoOverview{}, false)
	fl.On("Load", "/repos/repo-a").Return(types.ForgePageData{
		Spec: types.ForgeSpec{
			Test: []types.TestSpec{{Name: "unit"}},
		},
		TestReports: []types.TestReport{
			{
				Stage:  "unit",
				Status: "passed",
				Stats:  types.TestStats{Total: 10, Passed: 9, Failed: 1},
			},
		},
	}, nil)

	wc, mp, pc := stubOrchestrationMocksForPortfolio()
	svc := NewPortfolioService(pd, c, fl, wc, mp, pc)
	result, err := svc.GetPortfolio("/base", "p1", "name")

	require.NoError(t, err)
	assert.Equal(t, 10, result.Stats.TotalTests)
	assert.Equal(t, 9, result.Stats.Passed)
	assert.Equal(t, 1, result.Stats.Failed)
	assert.Len(t, result.Workspaces[0].AllStages, 1)
	assert.Equal(t, "unit", result.Workspaces[0].AllStages[0])
	assert.Len(t, result.Workspaces[0].RepoForge, 1)
	assert.Equal(t, "passed", result.Workspaces[0].RepoForge[0].StageResults["unit"])

	pd.AssertExpectations(t)
	c.AssertExpectations(t)
	fl.AssertExpectations(t)
}
