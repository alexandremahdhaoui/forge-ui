package adapter

import (
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

type demoDataSource struct {
	portfolios map[string]types.PortfolioSummary
	workspaces map[string]types.WorkspacePageData
	forges     map[string]types.ForgePageData
}

// NewDemoDataSource returns a DataSource backed by static demo data.
func NewDemoDataSource() DataSource {
	return &demoDataSource{
		portfolios: demoPortfolios,
		workspaces: demoWorkspaces,
		forges:     demoForges,
	}
}

func (d *demoDataSource) ListPortfolios(sort string) (types.PortfoliosPageData, error) {
	portfolios := make([]types.PortfolioSummary, 0, len(d.portfolios))
	var stats types.PortfoliosStats

	for _, p := range d.portfolios {
		portfolios = append(portfolios, p)
		stats.TotalWorkspaces += p.Stats.TotalWorkspaces
		stats.TotalRepos += p.Stats.TotalRepos
		stats.DirtyRepos += p.Stats.DirtyRepos
		stats.Passed += p.Stats.Passed
		stats.Failed += p.Stats.Failed
	}
	stats.TotalPortfolios = len(portfolios)

	return types.PortfoliosPageData{
		Portfolios: portfolios,
		Stats:      stats,
		SortMode:   sort,
		HomeURL:    "#/portfolios",
	}, nil
}

func (d *demoDataSource) GetPortfolio(name, sort string) (types.PortfolioPageData, error) {
	p, ok := d.portfolios[name]
	if !ok {
		// Fallback: return empty data; controller will handle the fallback.
		return types.PortfolioPageData{}, nil
	}

	return types.PortfolioPageData{
		Name:       p.Name,
		Path:       p.Path,
		IsDefault:  p.IsDefault,
		Workspaces: p.Workspaces,
		Stats:      p.Stats,
		SortMode:   sort,
		HomeURL:    "#/portfolios",
	}, nil
}

func (d *demoDataSource) GetWorkspace(portfolio, workspace, sort string) (types.WorkspacePageData, error) {
	key := portfolio + "/" + workspace
	ws, ok := d.workspaces[key]
	if !ok {
		return types.WorkspacePageData{}, nil
	}

	ws.SortMode = sort
	ws.HomeURL = "#/portfolios"
	return ws, nil
}

func (d *demoDataSource) GetForge(portfolio, workspace, repo string) (types.ForgePageData, error) {
	key := portfolio + "/" + workspace + "/" + repo
	f, ok := d.forges[key]
	if !ok {
		return types.ForgePageData{}, nil
	}

	f.HomeURL = "#/portfolios"
	return f, nil
}

// --- Demo data ---

var demoPortfolios = map[string]types.PortfolioSummary{
	"infrastructure": {
		Name:      "infrastructure",
		Path:      "/home/user/workspaces/infrastructure",
		IsDefault: false,
		Workspaces: []types.WorkspaceSummary{
			{
				Name:      "platform",
				Path:      "/home/user/workspaces/infrastructure/platform",
				RepoCount: 3,
				Repos: []types.RepoOverview{
					{Name: "forge", WorkspaceName: "platform", Branch: "main", IsDirty: false, Ahead: 0, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", LastCommitTime: time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)},
					{Name: "forge-ui", WorkspaceName: "platform", Branch: "feat/wasm", IsDirty: true, Ahead: 3, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui", LastCommitTime: time.Date(2026, 2, 21, 12, 30, 0, 0, time.UTC)},
					{Name: "config-server", WorkspaceName: "platform", Branch: "main", IsDirty: false, Ahead: 0, Behind: 2, HasUpstream: true, HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server", LastCommitTime: time.Date(2026, 2, 20, 8, 15, 0, 0, time.UTC)},
				},
				AllStages: []string{"lint", "unit", "integration"},
				RepoForge: []types.RepoForgeStats{
					{RepoName: "forge", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"}},
					{RepoName: "forge-ui", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": ""}},
					{RepoName: "config-server", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server", StageResults: map[string]string{"lint": "passed", "unit": "failed", "integration": ""}},
				},
			},
			{
				Name:      "networking",
				Path:      "/home/user/workspaces/infrastructure/networking",
				RepoCount: 2,
				Repos: []types.RepoOverview{
					{Name: "mesh-proxy", WorkspaceName: "networking", Branch: "main", IsDirty: false, HasUpstream: true, HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/networking/repos/mesh-proxy", LastCommitTime: time.Date(2026, 2, 19, 14, 0, 0, 0, time.UTC)},
					{Name: "dns-controller", WorkspaceName: "networking", Branch: "develop", IsDirty: true, Ahead: 1, HasUpstream: true, HasForge: false, RepoLink: "#/portfolios/infrastructure/workspaces/networking/repos/dns-controller", LastCommitTime: time.Date(2026, 2, 21, 9, 0, 0, 0, time.UTC)},
				},
				AllStages: []string{"lint", "unit"},
				RepoForge: []types.RepoForgeStats{
					{RepoName: "mesh-proxy", RepoLink: "#/portfolios/infrastructure/workspaces/networking/repos/mesh-proxy", StageResults: map[string]string{"lint": "passed", "unit": "passed"}},
				},
			},
		},
		Stats: types.WorkspacesStats{TotalWorkspaces: 2, TotalRepos: 5, DirtyRepos: 2, TotalTests: 42, Passed: 38, Failed: 4},
	},
	"default": {
		Name:      "default",
		Path:      "/home/user/workspaces",
		IsDefault: true,
		Workspaces: []types.WorkspaceSummary{
			{
				Name:      "personal",
				Path:      "/home/user/workspaces/personal",
				RepoCount: 2,
				Repos: []types.RepoOverview{
					{Name: "dotfiles", WorkspaceName: "personal", Branch: "main", IsDirty: true, Ahead: 5, HasUpstream: true, HasForge: false, LastCommitTime: time.Date(2026, 2, 21, 11, 0, 0, 0, time.UTC)},
					{Name: "blog", WorkspaceName: "personal", Branch: "main", IsDirty: false, HasUpstream: true, HasForge: false, LastCommitTime: time.Date(2026, 2, 18, 16, 0, 0, 0, time.UTC)},
				},
			},
		},
		Stats: types.WorkspacesStats{TotalWorkspaces: 1, TotalRepos: 2, DirtyRepos: 1},
	},
}

var demoWorkspaces = map[string]types.WorkspacePageData{
	"infrastructure/platform": {
		Name:          "platform",
		PortfolioName: "infrastructure",
		Path:          "/home/user/workspaces/infrastructure/platform",
		Repos: []types.RepoSummary{
			{
				Name: "forge", Branch: "main", IsDirty: false, HasUpstream: true, HasForge: true,
				RepoLink:       "#/portfolios/infrastructure/workspaces/platform/repos/forge",
				LastCommitTime: time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC),
				RecentLogs: []types.LogEntry{
					{Hash: "a1b2c3d", Message: "feat: add generic-builder engine"},
					{Hash: "e4f5g6h", Message: "fix: resolve container build caching"},
					{Hash: "i7j8k9l", Message: "docs: update README with WASM support"},
				},
			},
			{
				Name: "forge-ui", Branch: "feat/wasm", IsDirty: true, Ahead: 3, HasUpstream: true, HasForge: true,
				RepoLink:       "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui",
				LastCommitTime: time.Date(2026, 2, 21, 12, 30, 0, 0, time.UTC),
				StatusFiles: []types.StatusEntry{
					{Code: "M", FilePath: "cmd/forge-ui-wasm/main.go"},
					{Code: "A", FilePath: "web/index.html"},
				},
				DiffStat: " 2 files changed, 450 insertions(+)",
				RecentLogs: []types.LogEntry{
					{Hash: "x1y2z3a", Message: "feat: implement WASM rendering engine"},
					{Hash: "b4c5d6e", Message: "feat: add WASI shim for browser"},
				},
			},
			{
				Name: "config-server", Branch: "main", IsDirty: false, Behind: 2, HasUpstream: true, HasForge: true,
				RepoLink:       "#/portfolios/infrastructure/workspaces/platform/repos/config-server",
				LastCommitTime: time.Date(2026, 2, 20, 8, 15, 0, 0, time.UTC),
				RecentLogs: []types.LogEntry{
					{Hash: "j1k2l3m", Message: "fix: reload config on SIGHUP"},
				},
			},
		},
		Stats:     types.WorkspaceStats{TotalRepos: 3, ForgeRepos: 3, TotalTests: 30, Passed: 27, Failed: 3, Skipped: 2},
		AllStages: []string{"lint", "unit", "integration"},
		RepoForge: []types.RepoForgeStats{
			{RepoName: "forge", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"}},
			{RepoName: "forge-ui", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": ""}},
			{RepoName: "config-server", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server", StageResults: map[string]string{"lint": "passed", "unit": "failed", "integration": ""}},
		},
	},
}

var demoForges = map[string]types.ForgePageData{
	"infrastructure/platform/forge": {
		WorkspaceName: "platform",
		RepoName:      "forge",
		PortfolioName: "infrastructure",
		Spec: types.ForgeSpec{
			Name: "forge",
			Build: []types.BuildSpec{
				{Name: "forge", Src: "./cmd/forge", Dest: "./build/bin", Engine: "go://go-build"},
				{Name: "forge-container", Src: "./containers/forge/Containerfile", Engine: "go://container-build"},
			},
			Test: []types.TestSpec{
				{Name: "lint", Runner: "go://go-lint"},
				{Name: "unit", Runner: "go://go-test"},
				{Name: "integration", Testenv: "kind-cluster", Runner: "go://go-test"},
			},
		},
		Artifacts: []types.Artifact{
			{Name: "forge", Type: "binary", Location: "./build/bin/forge", Timestamp: "2026-02-21T10:00:00Z", Version: "a1b2c3d"},
			{Name: "forge-container", Type: "container", Location: "ghcr.io/user/forge:latest", Timestamp: "2026-02-21T10:05:00Z", Version: "a1b2c3d"},
		},
		TestReports: []types.TestReport{
			{ID: "rpt-001", Stage: "lint", Status: "passed", StartTime: time.Date(2026, 2, 21, 9, 50, 0, 0, time.UTC), Duration: 4.2, Stats: types.TestStats{Total: 1, Passed: 1}},
			{ID: "rpt-002", Stage: "unit", Status: "passed", StartTime: time.Date(2026, 2, 21, 9, 51, 0, 0, time.UTC), Duration: 12.8, Stats: types.TestStats{Total: 87, Passed: 85, Skipped: 2}, Coverage: types.Coverage{Enabled: true, Percentage: 78.4, FilePath: "coverage.out"}},
			{ID: "rpt-003", Stage: "integration", Status: "passed", StartTime: time.Date(2026, 2, 21, 9, 55, 0, 0, time.UTC), Duration: 45.1, Stats: types.TestStats{Total: 12, Passed: 12}},
		},
		TestEnvs: []types.TestEnv{
			{ID: "env-001", Name: "kind-cluster", Status: "passed", CreatedAt: time.Date(2026, 2, 21, 9, 54, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 2, 21, 10, 1, 0, 0, time.UTC), ManagedResources: []string{"kind-cluster/forge-test", "namespace/forge-integration"}},
		},
		Stats:          types.ForgeStats{TotalTests: 100, Passed: 98, Skipped: 2, AvgCoverage: 78.4, HasCoverage: true, StageCount: 3},
		StageStatusMap: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"},
	},
}
