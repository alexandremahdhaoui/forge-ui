package controller

import (
	"errors"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/mocks/mockadapter"
	"github.com/stretchr/testify/assert"
)

// --- Test data builders ---

func testPortfoliosPageData(sort string) types.PortfoliosPageData {
	return types.PortfoliosPageData{
		Portfolios: []types.PortfolioSummary{
			{
				Name:      "infrastructure",
				Path:      "/home/user/workspaces/infrastructure",
				IsDefault: false,
				Workspaces: []types.WorkspaceSummary{
					{
						Name:      "platform",
						Path:      "/home/user/workspaces/infrastructure/platform",
						RepoCount: 3,
						Repos: []types.RepoOverview{
							{Name: "forge", WorkspaceName: "platform", Branch: "main", HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", LastCommitTime: time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)},
						},
						AllStages: []string{"lint", "unit", "integration"},
						RepoForge: []types.RepoForgeStats{
							{RepoName: "forge", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"}},
							{RepoName: "config-server", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server", StageResults: map[string]string{"lint": "passed", "unit": "failed"}},
						},
					},
					{
						Name:      "networking",
						Path:      "/home/user/workspaces/infrastructure/networking",
						RepoCount: 2,
					},
				},
				Stats: types.WorkspacesStats{TotalWorkspaces: 2, TotalRepos: 5, DirtyRepos: 2, TotalTests: 42, Passed: 38, Failed: 4},
			},
			{
				Name:      "default",
				Path:      "/home/user/workspaces",
				IsDefault: true,
				Workspaces: []types.WorkspaceSummary{
					{Name: "personal", Path: "/home/user/workspaces/personal", RepoCount: 2},
				},
				Stats: types.WorkspacesStats{TotalWorkspaces: 1, TotalRepos: 2, DirtyRepos: 1},
			},
		},
		Stats: types.PortfoliosStats{
			TotalPortfolios: 2,
			TotalWorkspaces: 3,
			TotalRepos:      7,
			DirtyRepos:      3,
			Passed:          38,
			Failed:          4,
		},
		SortMode: sort,
		HomeURL:  "#/portfolios",
	}
}

func testPortfolioPageData(sort string) types.PortfolioPageData {
	return types.PortfolioPageData{
		Name:      "infrastructure",
		Path:      "/home/user/workspaces/infrastructure",
		IsDefault: false,
		Workspaces: []types.WorkspaceSummary{
			{
				Name:      "platform",
				Path:      "/home/user/workspaces/infrastructure/platform",
				RepoCount: 3,
				Repos: []types.RepoOverview{
					{Name: "forge", WorkspaceName: "platform", Branch: "main", HasForge: true, RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", LastCommitTime: time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)},
				},
				AllStages: []string{"lint", "unit", "integration"},
				RepoForge: []types.RepoForgeStats{
					{RepoName: "forge", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"}},
				},
			},
			{
				Name:      "networking",
				Path:      "/home/user/workspaces/infrastructure/networking",
				RepoCount: 2,
			},
		},
		Stats:    types.WorkspacesStats{TotalWorkspaces: 2, TotalRepos: 5, DirtyRepos: 2, Passed: 38, Failed: 4},
		SortMode: sort,
		HomeURL:  "#/portfolios",
	}
}

func testWorkspacePageData(sort string) types.WorkspacePageData {
	return types.WorkspacePageData{
		Name:          "platform",
		PortfolioName: "infrastructure",
		Path:          "/home/user/workspaces/infrastructure/platform",
		Repos: []types.RepoSummary{
			{
				Name: "forge", Branch: "main", HasForge: true,
				RepoLink:       "#/portfolios/infrastructure/workspaces/platform/repos/forge",
				LastCommitTime: time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC),
				RecentLogs: []types.LogEntry{
					{Hash: "a1b2c3d", Message: "feat: add generic-builder engine"},
				},
			},
			{
				Name: "forge-ui", Branch: "feat/wasm", IsDirty: true, Ahead: 3, HasForge: true,
				RepoLink:       "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui",
				LastCommitTime: time.Date(2026, 2, 21, 12, 30, 0, 0, time.UTC),
				StatusFiles: []types.StatusEntry{
					{Code: "M", FilePath: "cmd/forge-ui-wasm/main.go"},
					{Code: "A", FilePath: "web/index.html"},
				},
				DiffStat: " 2 files changed, 450 insertions(+)",
			},
			{
				Name: "config-server", Branch: "main", Behind: 2, HasForge: true,
				RepoLink:       "#/portfolios/infrastructure/workspaces/platform/repos/config-server",
				LastCommitTime: time.Date(2026, 2, 20, 8, 15, 0, 0, time.UTC),
			},
		},
		Stats:     types.WorkspaceStats{TotalRepos: 3, ForgeRepos: 3, TotalTests: 30, Passed: 27, Failed: 3, Skipped: 2},
		AllStages: []string{"lint", "unit", "integration"},
		RepoForge: []types.RepoForgeStats{
			{RepoName: "forge", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge", StageResults: map[string]string{"lint": "passed", "unit": "passed", "integration": "passed"}},
			{RepoName: "forge-ui", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui", StageResults: map[string]string{"lint": "passed", "unit": "passed"}},
			{RepoName: "config-server", RepoLink: "#/portfolios/infrastructure/workspaces/platform/repos/config-server", StageResults: map[string]string{"lint": "passed", "unit": "failed"}},
		},
		SortMode: sort,
		HomeURL:  "#/portfolios",
	}
}

func testForgePageData() types.ForgePageData {
	return types.ForgePageData{
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
		HomeURL:        "#/portfolios",
		SiblingRepos: []types.SideNavItem{
			{Name: "forge", Link: "#/portfolios/infrastructure/workspaces/platform/repos/forge", IsActive: true},
			{Name: "forge-ui", Link: "#/portfolios/infrastructure/workspaces/platform/repos/forge-ui"},
			{Name: "config-server", Link: "#/portfolios/infrastructure/workspaces/platform/repos/config-server"},
		},
	}
}

// --- Rendering tests using mock DataSource ---

func newMockRenderer(ds *mockadapter.DataSource) *renderer {
	funcMap := template.FuncMap{
		"percent": func(done, total int) int {
			if total == 0 {
				return 0
			}
			return (done * 100) / total
		},
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		panic("failed to parse embedded templates: " + err.Error())
	}
	return &renderer{ds: ds, templates: tmpl}
}

func TestRender_Portfolios(t *testing.T) {
	ds := mockadapter.NewDataSource(t)
	ds.On("ListPortfolios", "time").Return(testPortfoliosPageData("time"), nil)
	r := newMockRenderer(ds)

	result, err := r.Render("/portfolios?sort=time")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	checks := []string{
		"Portfolios",
		"infrastructure",
		"default",
		"catch-all",
		"platform",
		"/home/user/workspaces/infrastructure/platform",
		"cell-passed",
		"segmented-btn--active",
		"outlook-table",
		"data-table--compact",
		"recent-active",
		"recent-card",
	}

	for _, check := range checks {
		if !strings.Contains(result.Content, check) {
			t.Errorf("expected Content to contain %q", check)
		}
	}

	// Verify side nav contains portfolio names.
	sideNavChecks := []string{
		"infrastructure",
		"default",
		"#/portfolios/infrastructure",
		"#/portfolios/default",
		"side-nav__item",
	}
	for _, check := range sideNavChecks {
		if !strings.Contains(result.SideNav, check) {
			t.Errorf("expected SideNav to contain %q", check)
		}
	}
}

func TestRender_Portfolio(t *testing.T) {
	ds := mockadapter.NewDataSource(t)
	ds.On("GetPortfolio", "infrastructure", "name").Return(testPortfolioPageData("name"), nil)
	r := newMockRenderer(ds)

	result, err := r.Render("/portfolios/infrastructure?sort=name")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	checks := []string{
		"infrastructure",
		"Portfolios",
		"platform",
		"cell-passed",
		"outlook-table",
		"data-table--compact",
		"recent-active",
		"recent-card",
	}

	for _, check := range checks {
		if !strings.Contains(result.Content, check) {
			t.Errorf("expected Content to contain %q", check)
		}
	}

	// Verify side nav contains workspace names and links.
	sideNavChecks := []string{
		"platform",
		"networking",
		"#/portfolios/infrastructure/workspaces/platform",
		"#/portfolios/infrastructure/workspaces/networking",
		"side-nav__item",
		"side-nav__breadcrumb",
	}
	for _, check := range sideNavChecks {
		if !strings.Contains(result.SideNav, check) {
			t.Errorf("expected SideNav to contain %q", check)
		}
	}
}

func TestRender_Workspace(t *testing.T) {
	ds := mockadapter.NewDataSource(t)
	ds.On("GetWorkspace", "infrastructure", "platform", "time").Return(testWorkspacePageData("time"), nil)
	r := newMockRenderer(ds)

	result, err := r.Render("/portfolios/infrastructure/workspaces/platform?sort=time")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	checks := []string{
		"platform",
		"infrastructure",
		"forge",
		"2 changed",
		"cmd/forge-ui-wasm/main.go",
		"cell-passed",
		"cell-failed",
		"a1b2c3d",
		"feat: add generic-builder engine",
		"card-elevated",
		"outlook-table",
		"data-table--compact",
		"recent-active",
		"recent-card",
	}

	for _, check := range checks {
		if !strings.Contains(result.Content, check) {
			t.Errorf("expected Content to contain %q", check)
		}
	}

	// Verify side nav contains repo names and links.
	sideNavChecks := []string{
		"forge",
		"forge-ui",
		"config-server",
		"#/portfolios/infrastructure/workspaces/platform/repos/forge",
		"side-nav__item",
		"side-nav__breadcrumb",
	}
	for _, check := range sideNavChecks {
		if !strings.Contains(result.SideNav, check) {
			t.Errorf("expected SideNav to contain %q", check)
		}
	}
}

func TestRender_Forge(t *testing.T) {
	ds := mockadapter.NewDataSource(t)
	ds.On("GetForge", "infrastructure", "platform", "forge").Return(testForgePageData(), nil)
	r := newMockRenderer(ds)

	result, err := r.Render("/portfolios/infrastructure/workspaces/platform/repos/forge")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	checks := []string{
		"forge",
		"infrastructure",
		"platform",
		"go://go-build",
		"go://go-lint",
		"go://go-test",
		"binary",
		"a1b2c3d",
		"cell-passed",
		"78.4%",
		"Build Targets",
		"Test Reports",
		"Artifacts",
	}

	for _, check := range checks {
		if !strings.Contains(result.Content, check) {
			t.Errorf("expected Content to contain %q", check)
		}
	}

	// Verify side nav contains sibling repo names.
	sideNavChecks := []string{
		"forge",
		"forge-ui",
		"config-server",
		"#/portfolios/infrastructure/workspaces/platform/repos/forge",
		"side-nav__item",
		"side-nav__item--active",
		"side-nav__breadcrumb",
	}
	for _, check := range sideNavChecks {
		if !strings.Contains(result.SideNav, check) {
			t.Errorf("expected SideNav to contain %q", check)
		}
	}
}

func TestRender_UnknownPortfolio_FallsBackToPortfolios(t *testing.T) {
	ds := mockadapter.NewDataSource(t)
	// GetPortfolio returns empty Name -> triggers fallback to ListPortfolios.
	ds.On("GetPortfolio", "nonexistent", "time").Return(types.PortfolioPageData{}, nil)
	ds.On("ListPortfolios", "time").Return(testPortfoliosPageData("time"), nil)
	r := newMockRenderer(ds)

	result, err := r.Render("/portfolios/nonexistent")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(result.Content, "Portfolios") {
		t.Error("expected fallback to portfolios page")
	}
}

func TestRender_DefaultRoute(t *testing.T) {
	ds := mockadapter.NewDataSource(t)
	ds.On("ListPortfolios", "time").Return(testPortfoliosPageData("time"), nil)
	r := newMockRenderer(ds)

	result, err := r.Render("")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(result.Content, "Portfolios") {
		t.Error("expected default route to render portfolios")
	}
}

func TestRender_SortButtons(t *testing.T) {
	ds := mockadapter.NewDataSource(t)
	ds.On("ListPortfolios", "time").Return(testPortfoliosPageData("time"), nil)
	r := newMockRenderer(ds)

	result, err := r.Render("/portfolios?sort=time")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(result.Content, "segmented-btn--active\">Recent") {
		t.Error("expected Recent button to be active by default")
	}
}

func TestRender_HashRoute(t *testing.T) {
	ds := mockadapter.NewDataSource(t)
	ds.On("GetPortfolio", "infrastructure", "time").Return(testPortfolioPageData("time"), nil)
	r := newMockRenderer(ds)

	result, err := r.Render("#/portfolios/infrastructure")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(result.Content, "infrastructure") {
		t.Error("expected hash route to resolve correctly")
	}
}

// --- parseInput tests ---

func TestParseInput_Defaults(t *testing.T) {
	in := parseInput("")
	if in.Route != "/" {
		t.Errorf("expected route '/', got %q", in.Route)
	}
	if in.Sort != "time" {
		t.Errorf("expected sort 'time', got %q", in.Sort)
	}
	if in.Theme != "light" {
		t.Errorf("expected theme 'light', got %q", in.Theme)
	}
}

func TestParseInput_WithParams(t *testing.T) {
	in := parseInput("/portfolios/infra?sort=name&theme=dark")
	if in.Route != "/portfolios/infra" {
		t.Errorf("expected route '/portfolios/infra', got %q", in.Route)
	}
	if in.Sort != "name" {
		t.Errorf("expected sort 'name', got %q", in.Sort)
	}
	if in.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %q", in.Theme)
	}
}

func TestParseInput_HashPrefix(t *testing.T) {
	in := parseInput("#/portfolios")
	if in.Route != "/portfolios" {
		t.Errorf("expected route '/portfolios', got %q", in.Route)
	}
}

// --- Error cases using mock DataSource ---

func TestRender_ListPortfoliosError(t *testing.T) {
	t.Parallel()

	ds := mockadapter.NewDataSource(t)
	ds.On("ListPortfolios", "time").Return(types.PortfoliosPageData{}, errors.New("list error"))

	r := newMockRenderer(ds)
	_, err := r.Render("/portfolios")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list portfolios")
}

func TestRender_GetWorkspaceError(t *testing.T) {
	t.Parallel()

	ds := mockadapter.NewDataSource(t)
	ds.On("GetWorkspace", "infra", "platform", "time").Return(types.WorkspacePageData{}, errors.New("ws error"))

	r := newMockRenderer(ds)
	_, err := r.Render("/portfolios/infra/workspaces/platform")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get workspace")
}

func TestRender_GetForgeError(t *testing.T) {
	t.Parallel()

	ds := mockadapter.NewDataSource(t)
	ds.On("GetForge", "infra", "platform", "my-repo").Return(types.ForgePageData{}, errors.New("forge error"))

	r := newMockRenderer(ds)
	_, err := r.Render("/portfolios/infra/workspaces/platform/repos/my-repo")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get forge")
}

func TestRender_GetForgeEmptyFallsBackToWorkspace(t *testing.T) {
	t.Parallel()

	ds := mockadapter.NewDataSource(t)
	// GetForge returns empty RepoName, which triggers fallback to workspace view.
	ds.On("GetForge", "infra", "platform", "empty-repo").Return(types.ForgePageData{}, nil)
	// Workspace fallback succeeds.
	ds.On("GetWorkspace", "infra", "platform", "time").Return(types.WorkspacePageData{
		Name: "platform",
	}, nil)

	r := newMockRenderer(ds)
	result, err := r.Render("/portfolios/infra/workspaces/platform/repos/empty-repo")

	assert.NoError(t, err)
	assert.Contains(t, result.Content, "platform")
}

func TestRender_GetWorkspaceEmptyFallsBackToPortfolio(t *testing.T) {
	t.Parallel()

	ds := mockadapter.NewDataSource(t)
	// GetWorkspace returns empty Name, which triggers fallback to portfolio view.
	ds.On("GetWorkspace", "infra", "missing-ws", "time").Return(types.WorkspacePageData{}, nil)
	// Portfolio fallback succeeds.
	ds.On("GetPortfolio", "infra", "time").Return(types.PortfolioPageData{
		Name: "infra",
	}, nil)

	r := newMockRenderer(ds)
	result, err := r.Render("/portfolios/infra/workspaces/missing-ws")

	assert.NoError(t, err)
	assert.Contains(t, result.Content, "infra")
}

func TestRenderTemplate_ExecutionError(t *testing.T) {
	t.Parallel()

	// Create a template that will fail on execution by referencing a missing template.
	tmpl := template.Must(template.New("broken").Parse(`{{template "nonexistent" .}}`))

	ds := mockadapter.NewDataSource(t)
	r := &renderer{ds: ds, templates: tmpl}

	_, err := r.renderTemplate("broken", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execute template broken")
}

func TestRender_GetPortfolioError(t *testing.T) {
	t.Parallel()

	ds := mockadapter.NewDataSource(t)
	ds.On("GetPortfolio", "bad-portfolio", "time").Return(types.PortfolioPageData{}, errors.New("portfolio error"))

	r := newMockRenderer(ds)
	_, err := r.Render("/portfolios/bad-portfolio")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get portfolio")
}

func TestTopRecentPortfolios(t *testing.T) {
	t.Parallel()
	now := time.Now()
	portfolios := []types.PortfolioSummary{
		{Name: "old", Workspaces: []types.WorkspaceSummary{{Repos: []types.RepoOverview{{LastCommitTime: now.Add(-3 * time.Hour)}}}}},
		{Name: "new", Workspaces: []types.WorkspaceSummary{{Repos: []types.RepoOverview{{LastCommitTime: now.Add(-10 * time.Minute)}}}}},
		{Name: "mid", Workspaces: []types.WorkspaceSummary{{Repos: []types.RepoOverview{{LastCommitTime: now.Add(-1 * time.Hour)}}}}},
		{Name: "oldest", Workspaces: []types.WorkspaceSummary{{Repos: []types.RepoOverview{{LastCommitTime: now.Add(-5 * time.Hour)}}}}},
	}
	got := topRecentPortfolios(portfolios, 3)
	assert.Len(t, got, 3)
	assert.Equal(t, "new", got[0].Name)
	assert.Equal(t, "mid", got[1].Name)
	assert.Equal(t, "old", got[2].Name)
}

func TestTopRecentWorkspaces(t *testing.T) {
	t.Parallel()
	now := time.Now()
	workspaces := []types.WorkspaceSummary{
		{Name: "ws-old", Repos: []types.RepoOverview{{LastCommitTime: now.Add(-2 * time.Hour)}}},
		{Name: "ws-new", Repos: []types.RepoOverview{{LastCommitTime: now.Add(-5 * time.Minute)}}},
	}
	got := topRecentWorkspaces(workspaces, 3)
	assert.Len(t, got, 2)
	assert.Equal(t, "ws-new", got[0].Name)
	assert.Equal(t, "ws-old", got[1].Name)
}

func TestTopRecentRepos(t *testing.T) {
	t.Parallel()
	now := time.Now()
	repos := []types.RepoSummary{
		{Name: "r1", LastCommitTime: now.Add(-3 * time.Hour)},
		{Name: "r2", LastCommitTime: now.Add(-10 * time.Minute)},
		{Name: "r3", LastCommitTime: now.Add(-1 * time.Hour)},
		{Name: "r4", LastCommitTime: now.Add(-30 * time.Minute)},
	}
	got := topRecentRepos(repos, 3)
	assert.Len(t, got, 3)
	assert.Equal(t, "r2", got[0].Name)
	assert.Equal(t, "r4", got[1].Name)
	assert.Equal(t, "r3", got[2].Name)
}

func TestTopRecentRepos_Empty(t *testing.T) {
	t.Parallel()
	got := topRecentRepos(nil, 3)
	assert.Nil(t, got)
}
