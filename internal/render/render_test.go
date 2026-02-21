package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestExecute_Portfolios(t *testing.T) {
	data := model.PortfoliosPageData{
		Portfolios: []model.PortfolioSummary{
			{
				Name:      "infra",
				IsDefault: false,
				Workspaces: []model.WorkspaceSummary{
					{
						Name:      "platform",
						Path:      "/home/user/workspaces/infra/platform",
						RepoCount: 2,
						Repos: []model.RepoOverview{
							{Name: "forge", Branch: "main", IsDirty: false, HasUpstream: true, HasForge: true},
							{Name: "api", Branch: "develop", IsDirty: true, Ahead: 1, HasUpstream: true, HasForge: false},
						},
						AllStages: []string{"lint", "unit"},
						RepoForge: []model.RepoForgeStats{
							{RepoName: "forge", StageResults: map[string]string{"lint": "passed", "unit": "passed"}},
						},
					},
				},
				Stats: model.WorkspacesStats{TotalWorkspaces: 1, TotalRepos: 2, DirtyRepos: 1, Passed: 5, Failed: 0},
			},
			{
				Name:      "default",
				IsDefault: true,
				Stats:     model.WorkspacesStats{TotalWorkspaces: 0, TotalRepos: 0},
			},
		},
		Stats: model.PortfoliosStats{
			TotalPortfolios: 2,
			TotalWorkspaces: 1,
			TotalRepos:      2,
			DirtyRepos:      1,
			Passed:          5,
		},
	}

	cmd := Command{
		Action: "render",
		Page:   PagePortfolios,
		Theme:  "light",
		Sort:   "time",
		Data:   mustMarshal(t, data),
	}

	html, err := Execute(cmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Verify key content is rendered
	checks := []string{
		"Portfolios",
		"infra",
		"default",
		"catch-all",
		"platform",
		"/home/user/workspaces/infra/platform",
		"forge",
		"cell-passed",
		"segmented-btn--active",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestExecute_Portfolio(t *testing.T) {
	data := model.PortfolioPageData{
		Name:      "infrastructure",
		IsDefault: false,
		Workspaces: []model.WorkspaceSummary{
			{
				Name:      "platform",
				Path:      "/workspaces/platform",
				RepoCount: 1,
				Repos: []model.RepoOverview{
					{Name: "forge", Branch: "main", HasUpstream: true, HasForge: true},
				},
				AllStages: []string{"lint"},
				RepoForge: []model.RepoForgeStats{
					{RepoName: "forge", StageResults: map[string]string{"lint": "passed"}},
				},
			},
		},
		Stats: model.WorkspacesStats{TotalWorkspaces: 1, TotalRepos: 1, Passed: 3},
	}

	cmd := Command{
		Action: "render",
		Page:   PagePortfolio,
		Theme:  "dark",
		Sort:   "name",
		Data:   mustMarshal(t, data),
	}

	html, err := Execute(cmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	checks := []string{
		"infrastructure",
		"Portfolios",
		"platform",
		"cell-passed",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestExecute_Workspace(t *testing.T) {
	data := model.WorkspacePageData{
		Name:          "platform",
		PortfolioName: "infrastructure",
		Path:          "/workspaces/infrastructure/platform",
		Repos: []model.RepoSummary{
			{
				Name:    "forge",
				Branch:  "main",
				IsDirty: true,
				StatusFiles: []model.StatusEntry{
					{Code: "M", FilePath: "main.go"},
					{Code: "A", FilePath: "new.go"},
				},
				DiffStat:    " 2 files changed, 50 insertions(+), 3 deletions(-)",
				HasForge:    true,
				HasUpstream: true,
				Ahead:       2,
				RecentLogs: []model.LogEntry{
					{Hash: "abc123", Message: "feat: add feature"},
					{Hash: "def456", Message: "fix: resolve bug"},
				},
			},
		},
		Stats: model.WorkspaceStats{TotalRepos: 1, ForgeRepos: 1, TotalTests: 10, Passed: 9, Failed: 1},
		AllStages: []string{"lint", "unit"},
		RepoForge: []model.RepoForgeStats{
			{RepoName: "forge", StageResults: map[string]string{"lint": "passed", "unit": "failed"}},
		},
	}

	cmd := Command{
		Action: "render",
		Page:   PageWorkspace,
		Theme:  "light",
		Sort:   "time",
		Data:   mustMarshal(t, data),
	}

	html, err := Execute(cmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	checks := []string{
		"platform",
		"infrastructure",
		"forge",
		"2 changed",
		"main.go",
		"cell-passed",
		"cell-failed",
		"abc123",
		"feat: add feature",
		"card-elevated",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestExecute_Forge(t *testing.T) {
	data := model.ForgePageData{
		WorkspaceName: "platform",
		RepoName:      "forge",
		PortfolioName: "infrastructure",
		Spec: model.ForgeSpec{
			Name: "forge",
			Build: []model.BuildSpec{
				{Name: "forge", Src: "./cmd/forge", Dest: "./build/bin", Engine: "go://go-build"},
			},
			Test: []model.TestSpec{
				{Name: "lint", Runner: "go://go-lint"},
				{Name: "unit", Runner: "go://go-test"},
			},
		},
		Artifacts: []model.Artifact{
			{Name: "forge", Type: "binary", Location: "./build/bin/forge", Version: "abc123", Timestamp: "2026-02-21T10:00:00Z"},
		},
		TestReports: []model.TestReport{
			{ID: "r1", Stage: "lint", Status: "passed", Duration: 2.5, Stats: model.TestStats{Total: 1, Passed: 1}},
			{ID: "r2", Stage: "unit", Status: "passed", Duration: 8.3, Stats: model.TestStats{Total: 42, Passed: 40, Skipped: 2}, Coverage: model.Coverage{Enabled: true, Percentage: 82.5}},
		},
		Stats: model.ForgeStats{TotalTests: 43, Passed: 41, Failed: 0, Skipped: 2, AvgCoverage: 82.5, HasCoverage: true, StageCount: 2},
		StageStatusMap: map[string]string{"lint": "passed", "unit": "passed"},
	}

	cmd := Command{
		Action: "render",
		Page:   PageForge,
		Theme:  "light",
		Sort:   "",
		Data:   mustMarshal(t, data),
	}

	html, err := Execute(cmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	checks := []string{
		"forge",
		"infrastructure",
		"platform",
		"go://go-build",
		"go://go-lint",
		"go://go-test",
		"binary",
		"abc123",
		"cell-passed",
		"82.5%",
		"Build Targets",
		"Test Reports",
		"Artifacts",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestExecute_UnknownPage(t *testing.T) {
	cmd := Command{
		Action: "render",
		Page:   "nonexistent",
		Data:   json.RawMessage(`{}`),
	}

	_, err := Execute(cmd)
	if err == nil {
		t.Fatal("expected error for unknown page")
	}
	if !strings.Contains(err.Error(), "unknown page") {
		t.Errorf("expected 'unknown page' error, got: %v", err)
	}
}

func TestExecute_InvalidJSON(t *testing.T) {
	cmd := Command{
		Action: "render",
		Page:   PagePortfolios,
		Data:   json.RawMessage(`{invalid}`),
	}

	_, err := Execute(cmd)
	if err == nil {
		t.Fatal("expected error for invalid JSON data")
	}
}

func TestExecute_EmptyPortfolios(t *testing.T) {
	data := model.PortfoliosPageData{
		Stats: model.PortfoliosStats{},
	}

	cmd := Command{
		Action: "render",
		Page:   PagePortfolios,
		Theme:  "light",
		Sort:   "name",
		Data:   mustMarshal(t, data),
	}

	html, err := Execute(cmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(html, "No portfolios found") {
		t.Error("expected empty state message")
	}
}

func TestExecute_DefaultSortMode(t *testing.T) {
	data := model.PortfoliosPageData{
		Stats: model.PortfoliosStats{},
	}

	cmd := Command{
		Action: "render",
		Page:   PagePortfolios,
		Theme:  "light",
		Sort:   "",
		Data:   mustMarshal(t, data),
	}

	html, err := Execute(cmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Default sort should be "time" and the Recent button should be active
	if !strings.Contains(html, "segmented-btn--active\">Recent") {
		t.Error("expected Recent button to be active by default")
	}
}
