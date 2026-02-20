package portfolio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/ignore"
)

// createWorkspace creates a directory with a go.work file and optionally a git
// repo subdirectory. This mimics the minimal filesystem layout that
// workspace.List requires to detect workspaces and repos.
func createWorkspace(t *testing.T, dir, wsName string, repoNames ...string) {
	t.Helper()
	wsDir := filepath.Join(dir, wsName)
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "go.work"), []byte("go 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, repo := range repoNames {
		repoDir := filepath.Join(wsDir, repo)
		if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// --- List tests ---

func TestList_EmptyDir(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 0 {
		t.Errorf("got %d portfolios, want 0", len(portfolios))
	}
}

func TestList_OnlyLooseWorkspaces(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	createWorkspace(t, baseDir, "alpha", "repo-a")
	createWorkspace(t, baseDir, "beta", "repo-b")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 1 {
		t.Fatalf("got %d portfolios, want 1", len(portfolios))
	}

	p := portfolios[0]
	if p.Name != "default" {
		t.Errorf("Name = %q, want %q", p.Name, "default")
	}
	if !p.IsDefault {
		t.Error("IsDefault = false, want true")
	}
	if p.Path != baseDir {
		t.Errorf("Path = %q, want %q", p.Path, baseDir)
	}
	if len(p.Workspaces) != 2 {
		t.Errorf("got %d workspaces, want 2", len(p.Workspaces))
	}
}

func TestList_OnlyNamedPortfolios(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create named portfolio "personal" with one workspace inside.
	personalDir := filepath.Join(baseDir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, personalDir, "ws-a", "repo-1")

	// Create named portfolio "work" with one workspace inside.
	workDir := filepath.Join(baseDir, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, workDir, "ws-b", "repo-2")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 2 {
		t.Fatalf("got %d portfolios, want 2", len(portfolios))
	}

	// Named portfolios sorted alphabetically.
	if portfolios[0].Name != "personal" {
		t.Errorf("portfolios[0].Name = %q, want %q", portfolios[0].Name, "personal")
	}
	if portfolios[1].Name != "work" {
		t.Errorf("portfolios[1].Name = %q, want %q", portfolios[1].Name, "work")
	}
	if portfolios[0].IsDefault {
		t.Error("portfolios[0].IsDefault = true, want false")
	}
	if portfolios[1].IsDefault {
		t.Error("portfolios[1].IsDefault = true, want false")
	}

	// Verify workspace contents.
	if len(portfolios[0].Workspaces) != 1 {
		t.Errorf("personal portfolio: got %d workspaces, want 1", len(portfolios[0].Workspaces))
	}
	if len(portfolios[1].Workspaces) != 1 {
		t.Errorf("work portfolio: got %d workspaces, want 1", len(portfolios[1].Workspaces))
	}
}

func TestList_MixedMode(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Loose workspace directly under baseDir.
	createWorkspace(t, baseDir, "loose-ws", "repo-loose")

	// Named portfolio "projects" with a workspace inside.
	projectsDir := filepath.Join(baseDir, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, projectsDir, "ws-inner", "repo-inner")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 2 {
		t.Fatalf("got %d portfolios, want 2", len(portfolios))
	}

	// Named portfolio first, default last.
	if portfolios[0].Name != "projects" {
		t.Errorf("portfolios[0].Name = %q, want %q", portfolios[0].Name, "projects")
	}
	if portfolios[0].IsDefault {
		t.Error("portfolios[0].IsDefault = true, want false")
	}
	if portfolios[1].Name != "default" {
		t.Errorf("portfolios[1].Name = %q, want %q", portfolios[1].Name, "default")
	}
	if !portfolios[1].IsDefault {
		t.Error("portfolios[1].IsDefault = false, want true")
	}
}

func TestList_SkipsHiddenDirs(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create a hidden directory with go.work (should be skipped).
	hiddenDir := filepath.Join(baseDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "go.work"), []byte("go 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a hidden directory that looks like a named portfolio (should be skipped).
	hiddenPortfolio := filepath.Join(baseDir, ".config")
	if err := os.MkdirAll(hiddenPortfolio, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, hiddenPortfolio, "ws-inside", "repo-x")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 0 {
		t.Errorf("got %d portfolios, want 0 (hidden dirs should be skipped)", len(portfolios))
	}
}

func TestList_SkipsNonWorkspaceDirs(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create a directory without go.work and without workspace subdirs.
	emptyDir := filepath.Join(baseDir, "random-dir")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file (not a directory).
	if err := os.WriteFile(filepath.Join(baseDir, "somefile.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 0 {
		t.Errorf("got %d portfolios, want 0", len(portfolios))
	}
}

func TestList_SortOrder(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Loose workspace for the "default" portfolio.
	createWorkspace(t, baseDir, "loose-ws", "repo-loose")

	// Named portfolios in reverse alphabetical order on disk.
	for _, name := range []string{"zeta", "alpha", "mango"} {
		dir := filepath.Join(baseDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		createWorkspace(t, dir, "ws-"+name, "repo-"+name)
	}

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 4 {
		t.Fatalf("got %d portfolios, want 4", len(portfolios))
	}

	// Named portfolios sorted alphabetically.
	expected := []string{"alpha", "mango", "zeta", "default"}
	for i, want := range expected {
		if portfolios[i].Name != want {
			t.Errorf("portfolios[%d].Name = %q, want %q", i, portfolios[i].Name, want)
		}
	}

	// Only the last one is default.
	for i, p := range portfolios {
		wantDefault := i == len(portfolios)-1
		if p.IsDefault != wantDefault {
			t.Errorf("portfolios[%d] (%q): IsDefault = %v, want %v", i, p.Name, p.IsDefault, wantDefault)
		}
	}
}

// --- Get tests ---

func TestGet_NamedPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create named portfolio "forge" with two workspaces.
	forgeDir := filepath.Join(baseDir, "forge")
	if err := os.MkdirAll(forgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, forgeDir, "ws-core", "repo-core")
	createWorkspace(t, forgeDir, "ws-ui", "repo-ui")

	data, err := Get(baseDir, "forge")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if data.Name != "forge" {
		t.Errorf("Name = %q, want %q", data.Name, "forge")
	}
	if data.IsDefault {
		t.Error("IsDefault = true, want false")
	}
	if data.Path != forgeDir {
		t.Errorf("Path = %q, want %q", data.Path, forgeDir)
	}
	if len(data.Workspaces) != 2 {
		t.Fatalf("got %d workspaces, want 2", len(data.Workspaces))
	}

	// Workspaces sorted alphabetically.
	if data.Workspaces[0].Name != "ws-core" {
		t.Errorf("Workspaces[0].Name = %q, want %q", data.Workspaces[0].Name, "ws-core")
	}
	if data.Workspaces[1].Name != "ws-ui" {
		t.Errorf("Workspaces[1].Name = %q, want %q", data.Workspaces[1].Name, "ws-ui")
	}
}

func TestGet_DefaultPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create loose workspaces directly under baseDir.
	createWorkspace(t, baseDir, "ws-one", "repo-1")
	createWorkspace(t, baseDir, "ws-two", "repo-2")

	data, err := Get(baseDir, "default")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if data.Name != "default" {
		t.Errorf("Name = %q, want %q", data.Name, "default")
	}
	if !data.IsDefault {
		t.Error("IsDefault = false, want true")
	}
	if data.Path != baseDir {
		t.Errorf("Path = %q, want %q", data.Path, baseDir)
	}
	if len(data.Workspaces) != 2 {
		t.Fatalf("got %d workspaces, want 2", len(data.Workspaces))
	}
}

func TestGet_NonexistentPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	_, err := Get(baseDir, "nope")
	if err == nil {
		t.Fatal("expected error for nonexistent portfolio, got nil")
	}
}

// --- Ignore integration tests ---

// writeIgnoreFile writes a .forge-workspace-ignore file into dir with the given
// patterns, one per line.
func writeIgnoreFile(t *testing.T, dir string, patterns ...string) {
	t.Helper()
	var content string
	for _, p := range patterns {
		content += p + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, ignore.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestList_IgnoresLooseWorkspace(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	createWorkspace(t, baseDir, "alpha", "repo-a")
	createWorkspace(t, baseDir, "beta", "repo-b")
	createWorkspace(t, baseDir, "gamma", "repo-c")

	writeIgnoreFile(t, baseDir, "beta")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 1 {
		t.Fatalf("got %d portfolios, want 1 (default)", len(portfolios))
	}

	p := portfolios[0]
	if !p.IsDefault {
		t.Fatalf("expected default portfolio, got %q", p.Name)
	}
	if len(p.Workspaces) != 2 {
		t.Fatalf("got %d workspaces, want 2", len(p.Workspaces))
	}

	names := make(map[string]bool)
	for _, ws := range p.Workspaces {
		names[ws.Name] = true
	}
	if names["beta"] {
		t.Error("workspace 'beta' should be ignored but was found")
	}
	if !names["alpha"] {
		t.Error("workspace 'alpha' should be present but was not found")
	}
	if !names["gamma"] {
		t.Error("workspace 'gamma' should be present but was not found")
	}
}

func TestList_IgnoresNamedPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create named portfolio "personal" with one workspace.
	personalDir := filepath.Join(baseDir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, personalDir, "ws-a", "repo-1")

	// Create named portfolio "work" with one workspace.
	workDir := filepath.Join(baseDir, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, workDir, "ws-b", "repo-2")

	writeIgnoreFile(t, baseDir, "work")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 1 {
		t.Fatalf("got %d portfolios, want 1", len(portfolios))
	}
	if portfolios[0].Name != "personal" {
		t.Errorf("portfolios[0].Name = %q, want %q", portfolios[0].Name, "personal")
	}
}

func TestList_IgnoresWorkspaceInNamedPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create named portfolio "personal" with 2 workspaces.
	personalDir := filepath.Join(baseDir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, personalDir, "ws-a", "repo-1")
	createWorkspace(t, personalDir, "ws-b", "repo-2")

	// Write ignore file inside the portfolio dir to ignore ws-b.
	writeIgnoreFile(t, personalDir, "ws-b")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 1 {
		t.Fatalf("got %d portfolios, want 1", len(portfolios))
	}
	if portfolios[0].Name != "personal" {
		t.Errorf("portfolios[0].Name = %q, want %q", portfolios[0].Name, "personal")
	}

	// The "personal" portfolio should contain only ws-a because ws-b is ignored
	// by both isNamedPortfolio (portfolio-level) and workspace.List (workspace-level).
	if len(portfolios[0].Workspaces) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(portfolios[0].Workspaces))
	}
	if portfolios[0].Workspaces[0].Name != "ws-a" {
		t.Errorf("Workspaces[0].Name = %q, want %q", portfolios[0].Workspaces[0].Name, "ws-a")
	}
}

func TestList_IgnoreWithGlobPattern(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	createWorkspace(t, baseDir, "temp-foo", "repo-1")
	createWorkspace(t, baseDir, "temp-bar", "repo-2")
	createWorkspace(t, baseDir, "keeper", "repo-3")

	writeIgnoreFile(t, baseDir, "temp-*")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 1 {
		t.Fatalf("got %d portfolios, want 1 (default)", len(portfolios))
	}

	p := portfolios[0]
	if !p.IsDefault {
		t.Fatalf("expected default portfolio, got %q", p.Name)
	}
	if len(p.Workspaces) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(p.Workspaces))
	}
	if p.Workspaces[0].Name != "keeper" {
		t.Errorf("Workspaces[0].Name = %q, want %q", p.Workspaces[0].Name, "keeper")
	}
}

func TestList_NonMatchingPatternPreservesEntries(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	createWorkspace(t, baseDir, "alpha", "repo-a")

	writeIgnoreFile(t, baseDir, "zzz-nothing")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 1 {
		t.Fatalf("got %d portfolios, want 1 (default)", len(portfolios))
	}

	p := portfolios[0]
	if !p.IsDefault {
		t.Fatalf("expected default portfolio, got %q", p.Name)
	}
	if len(p.Workspaces) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(p.Workspaces))
	}
	if p.Workspaces[0].Name != "alpha" {
		t.Errorf("Workspaces[0].Name = %q, want %q", p.Workspaces[0].Name, "alpha")
	}
}

// --- Cascading ignore integration tests ---

func TestList_IgnoreCascades(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()

	// Create named portfolio "personal" with 2 workspaces.
	personalDir := filepath.Join(baseDir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createWorkspace(t, personalDir, "ws-a", "repo-1", "repo-2")
	createWorkspace(t, personalDir, "ws-b", "repo-3", "repo-4")

	// Ignore ws-b at portfolio level.
	writeIgnoreFile(t, personalDir, "ws-b")

	// Ignore repo-2 at workspace level inside ws-a.
	wsADir := filepath.Join(personalDir, "ws-a")
	writeIgnoreFile(t, wsADir, "repo-2")

	portfolios, err := List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 1 {
		t.Fatalf("got %d portfolios, want 1", len(portfolios))
	}

	p := portfolios[0]
	if p.Name != "personal" {
		t.Fatalf("portfolio name = %q, want %q", p.Name, "personal")
	}
	if len(p.Workspaces) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(p.Workspaces))
	}
	if p.Workspaces[0].Name != "ws-a" {
		t.Errorf("Workspaces[0].Name = %q, want %q", p.Workspaces[0].Name, "ws-a")
	}

	// ws-a should have only repo-1 (repo-2 ignored).
	repos := p.Workspaces[0].Repos
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if repos[0].Name != "repo-1" {
		t.Errorf("repos[0].Name = %q, want %q", repos[0].Name, "repo-1")
	}
}

func TestGet_IgnoreCascadesInDefaultPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	createWorkspace(t, baseDir, "ws-keep", "repo-a", "repo-b")
	createWorkspace(t, baseDir, "ws-drop", "repo-c")

	// Ignore ws-drop at baseDir level.
	writeIgnoreFile(t, baseDir, "ws-drop")

	// Ignore repo-b inside ws-keep.
	wsKeepDir := filepath.Join(baseDir, "ws-keep")
	writeIgnoreFile(t, wsKeepDir, "repo-b")

	data, err := Get(baseDir, "default")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !data.IsDefault {
		t.Fatal("expected default portfolio")
	}
	if len(data.Workspaces) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(data.Workspaces))
	}
	if data.Workspaces[0].Name != "ws-keep" {
		t.Errorf("Workspaces[0].Name = %q, want %q", data.Workspaces[0].Name, "ws-keep")
	}

	// ws-keep should have only repo-a (repo-b ignored).
	repos := data.Workspaces[0].Repos
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if repos[0].Name != "repo-a" {
		t.Errorf("repos[0].Name = %q, want %q", repos[0].Name, "repo-a")
	}
}
