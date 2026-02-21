package refresher

import (
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/mocks/mockadapter"
	"github.com/stretchr/testify/mock"
)

func TestRefresher_DefaultConfig(t *testing.T) {
	t.Parallel()

	c := new(mockadapter.Cache)
	gi := new(mockadapter.GitInfo)
	pd := new(mockadapter.PortfolioDiscovery)
	ws := new(mockadapter.WorkspaceDiscovery)

	r := New(c, gi, pd, ws, Config{BaseDir: "/nonexistent"})
	if got, want := r.cfg.Interval, 1*time.Minute; got != want {
		t.Errorf("Interval = %v, want %v", got, want)
	}
	if got, want := r.cfg.NumWorkers, 3; got != want {
		t.Errorf("NumWorkers = %d, want %d", got, want)
	}
}

func TestRefresher_CustomConfig(t *testing.T) {
	t.Parallel()

	c := new(mockadapter.Cache)
	gi := new(mockadapter.GitInfo)
	pd := new(mockadapter.PortfolioDiscovery)
	ws := new(mockadapter.WorkspaceDiscovery)

	r := New(c, gi, pd, ws, Config{
		BaseDir:    "/tmp",
		Interval:   30 * time.Second,
		NumWorkers: 5,
	})
	if got, want := r.cfg.Interval, 30*time.Second; got != want {
		t.Errorf("Interval = %v, want %v", got, want)
	}
	if got, want := r.cfg.NumWorkers, 5; got != want {
		t.Errorf("NumWorkers = %d, want %d", got, want)
	}
}

func TestRefresher_StartAndStop(t *testing.T) {
	t.Parallel()

	c := new(mockadapter.Cache)
	gi := new(mockadapter.GitInfo)
	pd := new(mockadapter.PortfolioDiscovery)
	ws := new(mockadapter.WorkspaceDiscovery)

	// Empty portfolio list — no workspaces to refresh.
	pd.On("List", "/base").Return([]types.PortfolioSummary{}, nil)

	r := New(c, gi, pd, ws, Config{
		BaseDir:    "/base",
		Interval:   1 * time.Hour,
		NumWorkers: 1,
	})

	done := make(chan struct{})
	go func() {
		r.Start()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start() did not return within 5 seconds")
	}

	stopDone := make(chan struct{})
	go func() {
		r.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds")
	}

	pd.AssertExpectations(t)
}

func TestRefresher_PopulatesCache(t *testing.T) {
	t.Parallel()

	c := new(mockadapter.Cache)
	gi := new(mockadapter.GitInfo)
	pd := new(mockadapter.PortfolioDiscovery)
	ws := new(mockadapter.WorkspaceDiscovery)

	commitTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	// Portfolio discovery returns a default portfolio with one workspace.
	pd.On("List", "/base").Return([]types.PortfolioSummary{
		{
			Name:      "default",
			IsDefault: true,
			Workspaces: []types.WorkspaceSummary{
				{Name: "ws1"},
			},
		},
	}, nil)

	// Workspace discovery returns one repo.
	ws.On("Get", "/base", "ws1").Return(types.WorkspacePageData{
		Name: "ws1",
		Repos: []types.RepoSummary{
			{Name: "repo-a", Path: "/base/ws1/repo-a", HasForge: true},
		},
	}, nil)

	// Git info returns branch and status for the repo.
	gi.On("RepoInfo", "/base/ws1/repo-a").Return(types.RepoSummary{
		Branch:         "main",
		IsDirty:        false,
		Ahead:          0,
		Behind:         0,
		HasUpstream:    true,
		LastCommitTime: commitTime,
	}, nil)

	// Cache should be populated with the workspace data.
	c.On("SetWorkspace", "default/ws1", mock.MatchedBy(func(data types.CacheWorkspaceData) bool {
		summary, ok := data.Summaries["repo-a"]
		if !ok {
			return false
		}
		if summary.Branch != "main" {
			return false
		}
		if summary.Name != "repo-a" {
			return false
		}
		if summary.Path != "/base/ws1/repo-a" {
			return false
		}
		overview, ok := data.Overviews["repo-a"]
		if !ok {
			return false
		}
		if overview.Branch != "main" {
			return false
		}
		if !overview.LastCommitTime.Equal(commitTime) {
			return false
		}
		return true
	})).Return()

	r := New(c, gi, pd, ws, Config{
		BaseDir:    "/base",
		Interval:   1 * time.Hour,
		NumWorkers: 1,
	})

	r.Start()
	defer r.Stop()

	pd.AssertExpectations(t)
	ws.AssertExpectations(t)
	gi.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestRefresher_PortfolioListError(t *testing.T) {
	t.Parallel()

	c := new(mockadapter.Cache)
	gi := new(mockadapter.GitInfo)
	pd := new(mockadapter.PortfolioDiscovery)
	ws := new(mockadapter.WorkspaceDiscovery)

	pd.On("List", "/base").Return(nil, errRefresherTest)

	r := New(c, gi, pd, ws, Config{
		BaseDir:    "/base",
		Interval:   1 * time.Hour,
		NumWorkers: 1,
	})

	r.Start()
	defer r.Stop()

	// No crash, no cache writes.
	pd.AssertExpectations(t)
	c.AssertNotCalled(t, "SetWorkspace", mock.Anything, mock.Anything)
}

func TestRefresher_WorkspaceGetError(t *testing.T) {
	t.Parallel()

	c := new(mockadapter.Cache)
	gi := new(mockadapter.GitInfo)
	pd := new(mockadapter.PortfolioDiscovery)
	ws := new(mockadapter.WorkspaceDiscovery)

	pd.On("List", "/base").Return([]types.PortfolioSummary{
		{
			Name:      "default",
			IsDefault: true,
			Workspaces: []types.WorkspaceSummary{
				{Name: "ws-broken"},
			},
		},
	}, nil)

	ws.On("Get", "/base", "ws-broken").Return(types.WorkspacePageData{}, errRefresherTest)

	r := New(c, gi, pd, ws, Config{
		BaseDir:    "/base",
		Interval:   1 * time.Hour,
		NumWorkers: 1,
	})

	r.Start()
	defer r.Stop()

	pd.AssertExpectations(t)
	ws.AssertExpectations(t)
	// No cache writes when workspace discovery fails.
	c.AssertNotCalled(t, "SetWorkspace", mock.Anything, mock.Anything)
}

func TestRefresher_RepoInfoError(t *testing.T) {
	t.Parallel()

	c := new(mockadapter.Cache)
	gi := new(mockadapter.GitInfo)
	pd := new(mockadapter.PortfolioDiscovery)
	ws := new(mockadapter.WorkspaceDiscovery)

	pd.On("List", "/base").Return([]types.PortfolioSummary{
		{
			Name:      "default",
			IsDefault: true,
			Workspaces: []types.WorkspaceSummary{
				{Name: "ws1"},
			},
		},
	}, nil)

	ws.On("Get", "/base", "ws1").Return(types.WorkspacePageData{
		Name: "ws1",
		Repos: []types.RepoSummary{
			{Name: "repo-bad", Path: "/base/ws1/repo-bad"},
		},
	}, nil)

	// Git info fails for this repo.
	gi.On("RepoInfo", "/base/ws1/repo-bad").Return(types.RepoSummary{}, errRefresherTest)

	// Cache is still written, but with empty maps (no repos succeeded).
	c.On("SetWorkspace", "default/ws1", mock.MatchedBy(func(data types.CacheWorkspaceData) bool {
		return len(data.Summaries) == 0 && len(data.Overviews) == 0
	})).Return()

	r := New(c, gi, pd, ws, Config{
		BaseDir:    "/base",
		Interval:   1 * time.Hour,
		NumWorkers: 1,
	})

	r.Start()
	defer r.Stop()

	pd.AssertExpectations(t)
	ws.AssertExpectations(t)
	gi.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestRefresher_NamedPortfolio(t *testing.T) {
	t.Parallel()

	c := new(mockadapter.Cache)
	gi := new(mockadapter.GitInfo)
	pd := new(mockadapter.PortfolioDiscovery)
	ws := new(mockadapter.WorkspaceDiscovery)

	pd.On("List", "/base").Return([]types.PortfolioSummary{
		{
			Name:      "myportfolio",
			IsDefault: false,
			Workspaces: []types.WorkspaceSummary{
				{Name: "ws1"},
			},
		},
	}, nil)

	// For named portfolios, workspace base is baseDir/portfolioName.
	ws.On("Get", "/base/myportfolio", "ws1").Return(types.WorkspacePageData{
		Name:  "ws1",
		Repos: []types.RepoSummary{},
	}, nil)

	// Cache key is portfolioName/workspaceName.
	c.On("SetWorkspace", "myportfolio/ws1", mock.MatchedBy(func(data types.CacheWorkspaceData) bool {
		return len(data.Summaries) == 0
	})).Return()

	r := New(c, gi, pd, ws, Config{
		BaseDir:    "/base",
		Interval:   1 * time.Hour,
		NumWorkers: 1,
	})

	r.Start()
	defer r.Stop()

	pd.AssertExpectations(t)
	ws.AssertExpectations(t)
	c.AssertExpectations(t)
}

var errRefresherTest = errMsg("refresher test error")

type errMsg string

func (e errMsg) Error() string { return string(e) }
