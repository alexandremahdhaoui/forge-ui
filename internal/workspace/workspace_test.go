package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/ignore"
)

// helper creates a directory and a marker file inside it.
func mkdirWithFile(t *testing.T, base, dir, file string) {
	t.Helper()
	p := filepath.Join(base, dir)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if file != "" {
		if err := os.WriteFile(filepath.Join(p, file), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// writeIgnoreFile writes a .forge-workspace-ignore file in dir with the given lines.
func writeIgnoreFile(t *testing.T, dir string, lines string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ignore.FileName), []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestList_IgnoresPatterns verifies that List() skips workspaces matching ignore patterns.
func TestList_IgnoresPatterns(t *testing.T) {
	basedir := t.TempDir()

	// Create 3 workspace dirs, each with go.work.
	for _, name := range []string{"ws-a", "ws-b", "ws-c"} {
		mkdirWithFile(t, basedir, name, "go.work")
	}

	// Ignore ws-b.
	writeIgnoreFile(t, basedir, "ws-b\n")

	workspaces, err := List(basedir)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}

	names := make(map[string]bool)
	for _, ws := range workspaces {
		names[ws.Name] = true
	}
	if names["ws-b"] {
		t.Error("ws-b should be ignored but was returned")
	}
	if !names["ws-a"] {
		t.Error("ws-a should be present but was not returned")
	}
	if !names["ws-c"] {
		t.Error("ws-c should be present but was not returned")
	}
}

// TestList_NoIgnoreFile verifies that List() returns all workspaces when no ignore file exists.
func TestList_NoIgnoreFile(t *testing.T) {
	basedir := t.TempDir()

	for _, name := range []string{"ws-a", "ws-b"} {
		mkdirWithFile(t, basedir, name, "go.work")
	}

	workspaces, err := List(basedir)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}

	names := make(map[string]bool)
	for _, ws := range workspaces {
		names[ws.Name] = true
	}
	if !names["ws-a"] || !names["ws-b"] {
		t.Errorf("expected both ws-a and ws-b, got %v", names)
	}
}

// TestGet_IgnoresRepos verifies that Get() skips repos matching ignore patterns.
func TestGet_IgnoresRepos(t *testing.T) {
	basedir := t.TempDir()
	wsName := "my-ws"
	wsPath := filepath.Join(basedir, wsName)

	// Create workspace dir with go.work.
	mkdirWithFile(t, basedir, wsName, "go.work")

	// Create 3 repo dirs, each with .git/.
	for _, repo := range []string{"repo-a", "repo-b", "repo-c"} {
		repoPath := filepath.Join(wsPath, repo)
		if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Ignore repo-b.
	writeIgnoreFile(t, wsPath, "repo-b\n")

	data, err := Get(basedir, wsName)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if len(data.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(data.Repos))
	}

	names := make(map[string]bool)
	for _, r := range data.Repos {
		names[r.Name] = true
	}
	if names["repo-b"] {
		t.Error("repo-b should be ignored but was returned")
	}
	if !names["repo-a"] {
		t.Error("repo-a should be present but was not returned")
	}
	if !names["repo-c"] {
		t.Error("repo-c should be present but was not returned")
	}
}

// TestList_ScanReposRespectsIgnore exercises scanRepos() through List() and verifies
// that repos matching ignore patterns inside a workspace are excluded.
func TestList_ScanReposRespectsIgnore(t *testing.T) {
	basedir := t.TempDir()
	wsName := "ws-a"
	wsPath := filepath.Join(basedir, wsName)

	// Create workspace dir with go.work.
	mkdirWithFile(t, basedir, wsName, "go.work")

	// Create 3 repo dirs inside ws-a, each with .git/.
	for _, repo := range []string{"repo-x", "repo-y", "repo-z"} {
		repoPath := filepath.Join(wsPath, repo)
		if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Ignore repo-y inside ws-a.
	writeIgnoreFile(t, wsPath, "repo-y\n")

	workspaces, err := List(basedir)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}

	ws := workspaces[0]
	if ws.Name != wsName {
		t.Fatalf("expected workspace %q, got %q", wsName, ws.Name)
	}

	if len(ws.Repos) != 2 {
		t.Fatalf("expected 2 repos in %s, got %d", wsName, len(ws.Repos))
	}

	repoNames := make(map[string]bool)
	for _, r := range ws.Repos {
		repoNames[r.Name] = true
	}
	if repoNames["repo-y"] {
		t.Error("repo-y should be ignored but was returned in ws-a.Repos")
	}
	if !repoNames["repo-x"] {
		t.Error("repo-x should be present but was not returned")
	}
	if !repoNames["repo-z"] {
		t.Error("repo-z should be present but was not returned")
	}
}
