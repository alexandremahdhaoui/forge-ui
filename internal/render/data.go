package render

import (
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

// DemoData provides sample data for rendering pages.
// In a full deployment, this would be replaced by live data from an API.
var DemoData = struct {
	Portfolios map[string]model.PortfolioSummary
	Workspaces map[string]model.WorkspacePageData
	Forges     map[string]model.ForgePageData
}{
	Portfolios: map[string]model.PortfolioSummary{
		"infrastructure": {
			Name:      "infrastructure",
			Path:      "/home/user/workspaces/infrastructure",
			IsDefault: false,
			Workspaces: []model.WorkspaceSummary{
				{
					Name:      "platform",
					Path:      "/home/user/workspaces/infrastructure/platform",
					RepoCount: 3,
					Repos: []model.RepoOverview{
						{Name: "forge", WorkspaceName: "platform", Branch: "main", IsDirty: false, Ahead: 0, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", LastCommitTime: time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)},
						{Name: "forge-ui", WorkspaceName: "platform", Branch: "feat/wasm", IsDirty: true, Ahead: 3, Behind: 0, HasUpstream: true, HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui", LastCommitTime: time.Date(2026, 2, 21, 12, 30, 0, 0, time.UTC)},
						{Name: "config-server", WorkspaceName: "platform", Branch: "main", IsDirty: false, Ahead: 0, Behind: 2, HasUpstream: true, HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server", LastCommitTime: time.Date(2026, 2, 20, 8, 15, 0, 0, time.UTC)},
					},
					AllStages: []string{"lint", "unit", "integration"},
					RepoForge: []model.RepoForgeStats{
						{RepoName: "forge", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"}},
						{RepoName: "forge-ui", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": ""}},
						{RepoName: "config-server", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server", StageResults: map[string]string{"lint": "passed", "unit": "failed", "integration": ""}},
					},
				},
				{
					Name:      "networking",
					Path:      "/home/user/workspaces/infrastructure/networking",
					RepoCount: 2,
					Repos: []model.RepoOverview{
						{Name: "mesh-proxy", WorkspaceName: "networking", Branch: "main", IsDirty: false, HasUpstream: true, HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/networking/repos/mesh-proxy", LastCommitTime: time.Date(2026, 2, 19, 14, 0, 0, 0, time.UTC)},
						{Name: "dns-controller", WorkspaceName: "networking", Branch: "develop", IsDirty: true, Ahead: 1, HasUpstream: true, HasForge: false, RepoLink: "#/portfolios/infrastructure/workspaces/networking/repos/dns-controller", LastCommitTime: time.Date(2026, 2, 21, 9, 0, 0, 0, time.UTC)},
					},
					AllStages: []string{"lint", "unit"},
					RepoForge: []model.RepoForgeStats{
						{RepoName: "mesh-proxy", RepoLink: "#/portfolios/infrastructure/workspaces/networking/repos/mesh-proxy", StageResults: map[string]string{"lint": "passed", "unit": "passed"}},
					},
				},
			},
			Stats: model.WorkspacesStats{TotalWorkspaces: 2, TotalRepos: 5, DirtyRepos: 2, TotalTests: 42, Passed: 38, Failed: 4},
		},
		"default": {
			Name:      "default",
			Path:      "/home/user/workspaces",
			IsDefault: true,
			Workspaces: []model.WorkspaceSummary{
				{
					Name:      "personal",
					Path:      "/home/user/workspaces/personal",
					RepoCount: 2,
					Repos: []model.RepoOverview{
						{Name: "dotfiles", WorkspaceName: "personal", Branch: "main", IsDirty: true, Ahead: 5, HasUpstream: true, HasForge: false, LastCommitTime: time.Date(2026, 2, 21, 11, 0, 0, 0, time.UTC)},
						{Name: "blog", WorkspaceName: "personal", Branch: "main", IsDirty: false, HasUpstream: true, HasForge: false, LastCommitTime: time.Date(2026, 2, 18, 16, 0, 0, 0, time.UTC)},
					},
				},
			},
			Stats: model.WorkspacesStats{TotalWorkspaces: 1, TotalRepos: 2, DirtyRepos: 1},
		},
	},

	Workspaces: map[string]model.WorkspacePageData{
		"infrastructure/platform": {
			Name:          "platform",
			PortfolioName: "infrastructure",
			Path:          "/home/user/workspaces/infrastructure/platform",
			Repos: []model.RepoSummary{
				{
					Name: "forge", Branch: "main", IsDirty: false, HasUpstream: true, HasForge: true,
					RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge",
					LastCommitTime: time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC),
					RecentLogs: []model.LogEntry{
						{Hash: "a1b2c3d", Message: "feat: add generic-builder engine"},
						{Hash: "e4f5g6h", Message: "fix: resolve container build caching"},
						{Hash: "i7j8k9l", Message: "docs: update README with WASM support"},
					},
				},
				{
					Name: "forge-ui", Branch: "feat/wasm", IsDirty: true, Ahead: 3, HasUpstream: true, HasForge: true,
					RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui",
					LastCommitTime: time.Date(2026, 2, 21, 12, 30, 0, 0, time.UTC),
					StatusFiles: []model.StatusEntry{
						{Code: "M", FilePath: "cmd/forge-ui-wasm/main.go"},
						{Code: "A", FilePath: "web/index.html"},
					},
					DiffStat: " 2 files changed, 450 insertions(+)",
					RecentLogs: []model.LogEntry{
						{Hash: "x1y2z3a", Message: "feat: implement WASM rendering engine"},
						{Hash: "b4c5d6e", Message: "feat: add WASI shim for browser"},
					},
				},
				{
					Name: "config-server", Branch: "main", IsDirty: false, Behind: 2, HasUpstream: true, HasForge: true,
					RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server",
					LastCommitTime: time.Date(2026, 2, 20, 8, 15, 0, 0, time.UTC),
					RecentLogs: []model.LogEntry{
						{Hash: "j1k2l3m", Message: "fix: reload config on SIGHUP"},
					},
				},
			},
			Stats: model.WorkspaceStats{TotalRepos: 3, ForgeRepos: 3, TotalTests: 30, Passed: 27, Failed: 3, Skipped: 2},
			AllStages: []string{"lint", "unit", "integration"},
			RepoForge: []model.RepoForgeStats{
				{RepoName: "forge", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"}},
				{RepoName: "forge-ui", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": ""}},
				{RepoName: "config-server", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server", StageResults: map[string]string{"lint": "passed", "unit": "failed", "integration": ""}},
			},
		},
	},

	Forges: map[string]model.ForgePageData{
		"infrastructure/platform/forge": {
			WorkspaceName: "platform",
			RepoName:      "forge",
			PortfolioName: "infrastructure",
			Spec: model.ForgeSpec{
				Name: "forge",
				Build: []model.BuildSpec{
					{Name: "forge", Src: "./cmd/forge", Dest: "./build/bin", Engine: "go://go-build"},
					{Name: "forge-container", Src: "./containers/forge/Containerfile", Engine: "go://container-build"},
				},
				Test: []model.TestSpec{
					{Name: "lint", Runner: "go://go-lint"},
					{Name: "unit", Runner: "go://go-test"},
					{Name: "integration", Testenv: "kind-cluster", Runner: "go://go-test"},
				},
			},
			Artifacts: []model.Artifact{
				{Name: "forge", Type: "binary", Location: "./build/bin/forge", Timestamp: "2026-02-21T10:00:00Z", Version: "a1b2c3d"},
				{Name: "forge-container", Type: "container", Location: "ghcr.io/user/forge:latest", Timestamp: "2026-02-21T10:05:00Z", Version: "a1b2c3d"},
			},
			TestReports: []model.TestReport{
				{ID: "rpt-001", Stage: "lint", Status: "passed", StartTime: time.Date(2026, 2, 21, 9, 50, 0, 0, time.UTC), Duration: 4.2, Stats: model.TestStats{Total: 1, Passed: 1}},
				{ID: "rpt-002", Stage: "unit", Status: "passed", StartTime: time.Date(2026, 2, 21, 9, 51, 0, 0, time.UTC), Duration: 12.8, Stats: model.TestStats{Total: 87, Passed: 85, Skipped: 2}, Coverage: model.Coverage{Enabled: true, Percentage: 78.4, FilePath: "coverage.out"}},
				{ID: "rpt-003", Stage: "integration", Status: "passed", StartTime: time.Date(2026, 2, 21, 9, 55, 0, 0, time.UTC), Duration: 45.1, Stats: model.TestStats{Total: 12, Passed: 12}},
			},
			TestEnvs: []model.TestEnv{
				{ID: "env-001", Name: "kind-cluster", Status: "passed", CreatedAt: time.Date(2026, 2, 21, 9, 54, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 2, 21, 10, 1, 0, 0, time.UTC), ManagedResources: []string{"kind-cluster/forge-test", "namespace/forge-integration"}},
			},
			Stats:          model.ForgeStats{TotalTests: 100, Passed: 98, Skipped: 2, AvgCoverage: 78.4, HasCoverage: true, StageCount: 3},
			StageStatusMap: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"},
		},
	},
}
