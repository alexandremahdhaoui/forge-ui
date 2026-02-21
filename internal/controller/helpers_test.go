package controller

import (
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// --- rewriteRepoLinks tests ---

func TestRewriteRepoLinks_Named(t *testing.T) {
	t.Parallel()

	workspaces := []types.WorkspaceSummary{
		{
			Name: "ws-core",
			Repos: []types.RepoOverview{
				{Name: "repo-a", RepoLink: "/old/path/repo-a"},
				{Name: "repo-b", RepoLink: "/old/path/repo-b"},
			},
		},
		{
			Name: "ws-ui",
			Repos: []types.RepoOverview{
				{Name: "repo-c", RepoLink: "/old/path/repo-c"},
			},
		},
	}

	rewriteRepoLinks(workspaces, "forge")

	tests := []struct {
		wsIdx, repoIdx int
		want           string
	}{
		{0, 0, "/portfolios/forge/workspaces/ws-core/repos/repo-a"},
		{0, 1, "/portfolios/forge/workspaces/ws-core/repos/repo-b"},
		{1, 0, "/portfolios/forge/workspaces/ws-ui/repos/repo-c"},
	}

	for _, tc := range tests {
		got := workspaces[tc.wsIdx].Repos[tc.repoIdx].RepoLink
		if got != tc.want {
			t.Errorf("workspaces[%d].Repos[%d].RepoLink = %q, want %q",
				tc.wsIdx, tc.repoIdx, got, tc.want)
		}
	}
}

func TestRewriteRepoLinks_Default(t *testing.T) {
	t.Parallel()

	workspaces := []types.WorkspaceSummary{
		{
			Name: "ws-loose",
			Repos: []types.RepoOverview{
				{Name: "repo-x", RepoLink: "/old/path"},
			},
		},
	}

	rewriteRepoLinks(workspaces, "default")

	got := workspaces[0].Repos[0].RepoLink
	want := "/portfolios/default/workspaces/ws-loose/repos/repo-x"
	if got != want {
		t.Errorf("RepoLink = %q, want %q", got, want)
	}
}

func TestRewriteRepoLinks_EmptyWorkspaces(t *testing.T) {
	t.Parallel()

	// Must not panic on empty slice.
	rewriteRepoLinks(nil, "anything")
	rewriteRepoLinks([]types.WorkspaceSummary{}, "anything")
}

// --- maxCommitTime tests ---

func TestMaxCommitTime_MultipleRepos(t *testing.T) {
	t.Parallel()

	now := time.Now()
	repos := []types.RepoOverview{
		{Name: "repo-a", LastCommitTime: now.Add(-2 * time.Hour)},
		{Name: "repo-b", LastCommitTime: now.Add(-30 * time.Minute)},
		{Name: "repo-c", LastCommitTime: now.Add(-1 * time.Hour)},
	}

	got := maxCommitTime(repos)
	want := now.Add(-30 * time.Minute)
	if !got.Equal(want) {
		t.Errorf("maxCommitTime = %v, want %v", got, want)
	}
}

func TestMaxCommitTime_EmptySlice(t *testing.T) {
	t.Parallel()

	got := maxCommitTime(nil)
	if !got.IsZero() {
		t.Errorf("maxCommitTime(nil) = %v, want zero time", got)
	}

	got = maxCommitTime([]types.RepoOverview{})
	if !got.IsZero() {
		t.Errorf("maxCommitTime([]) = %v, want zero time", got)
	}
}

// --- enrichWorkspaces tests ---

func TestEnrichWorkspaces_WithCachedData(t *testing.T) {
	t.Parallel()

	c := adapter.NewCache()
	fl := adapter.NewForgeLoader()
	commitTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	c.SetWorkspace("portfolio/ws-core", types.CacheWorkspaceData{
		Summaries: map[string]types.RepoSummary{},
		Overviews: map[string]types.RepoOverview{
			"repo-a": {
				Name:           "repo-a",
				Branch:         "main",
				IsDirty:        true,
				Ahead:          2,
				Behind:         1,
				HasUpstream:    true,
				LastCommitTime: commitTime,
			},
		},
		UpdatedAt: time.Now(),
	})

	workspaces := []types.WorkspaceSummary{
		{
			Name:      "ws-core",
			RepoCount: 1,
			Repos: []types.RepoOverview{
				{Name: "repo-a"},
			},
		},
	}

	cacheKey := func(wsName string) string { return "portfolio/" + wsName }
	totalRepos, dirtyRepos, _, _, _ := enrichWorkspaces(workspaces, c, fl, cacheKey)

	if totalRepos != 1 {
		t.Errorf("totalRepos = %d, want 1", totalRepos)
	}
	if dirtyRepos != 1 {
		t.Errorf("dirtyRepos = %d, want 1", dirtyRepos)
	}

	repo := workspaces[0].Repos[0]
	if repo.Branch != "main" {
		t.Errorf("Branch = %q, want %q", repo.Branch, "main")
	}
	if !repo.IsDirty {
		t.Error("IsDirty = false, want true")
	}
	if repo.Ahead != 2 {
		t.Errorf("Ahead = %d, want 2", repo.Ahead)
	}
	if repo.Behind != 1 {
		t.Errorf("Behind = %d, want 1", repo.Behind)
	}
	if !repo.HasUpstream {
		t.Error("HasUpstream = false, want true")
	}
	if !repo.LastCommitTime.Equal(commitTime) {
		t.Errorf("LastCommitTime = %v, want %v", repo.LastCommitTime, commitTime)
	}
}

func TestEnrichWorkspaces_NoCachedData(t *testing.T) {
	t.Parallel()

	c := adapter.NewCache()
	fl := adapter.NewForgeLoader()

	workspaces := []types.WorkspaceSummary{
		{
			Name:      "ws-empty",
			RepoCount: 2,
			Repos: []types.RepoOverview{
				{Name: "repo-a"},
				{Name: "repo-b"},
			},
		},
	}

	cacheKey := func(wsName string) string { return "p/" + wsName }
	totalRepos, dirtyRepos, totalTests, passed, failed := enrichWorkspaces(workspaces, c, fl, cacheKey)

	if totalRepos != 2 {
		t.Errorf("totalRepos = %d, want 2", totalRepos)
	}
	if dirtyRepos != 0 {
		t.Errorf("dirtyRepos = %d, want 0", dirtyRepos)
	}
	if totalTests != 0 {
		t.Errorf("totalTests = %d, want 0", totalTests)
	}
	if passed != 0 {
		t.Errorf("passed = %d, want 0", passed)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}

	for _, repo := range workspaces[0].Repos {
		if repo.Branch != "" {
			t.Errorf("repo %q: Branch = %q, want empty", repo.Name, repo.Branch)
		}
		if repo.IsDirty {
			t.Errorf("repo %q: IsDirty = true, want false", repo.Name)
		}
	}
}

func TestEnrichWorkspaces_DirtyRepoCount(t *testing.T) {
	t.Parallel()

	c := adapter.NewCache()
	fl := adapter.NewForgeLoader()

	c.SetWorkspace("p/ws-a", types.CacheWorkspaceData{
		Summaries: map[string]types.RepoSummary{},
		Overviews: map[string]types.RepoOverview{
			"repo-1": {Name: "repo-1", Branch: "main", IsDirty: true},
			"repo-2": {Name: "repo-2", Branch: "main", IsDirty: false},
			"repo-3": {Name: "repo-3", Branch: "dev", IsDirty: true},
		},
		UpdatedAt: time.Now(),
	})

	workspaces := []types.WorkspaceSummary{
		{
			Name:      "ws-a",
			RepoCount: 3,
			Repos: []types.RepoOverview{
				{Name: "repo-1"},
				{Name: "repo-2"},
				{Name: "repo-3"},
			},
		},
	}

	cacheKey := func(wsName string) string { return "p/" + wsName }
	totalRepos, dirtyRepos, _, _, _ := enrichWorkspaces(workspaces, c, fl, cacheKey)

	if totalRepos != 3 {
		t.Errorf("totalRepos = %d, want 3", totalRepos)
	}
	if dirtyRepos != 2 {
		t.Errorf("dirtyRepos = %d, want 2", dirtyRepos)
	}
}
