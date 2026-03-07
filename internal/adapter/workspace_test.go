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

package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/util/ignoreutil"
)

// workspaceMkdirWithFile creates a directory and a marker file inside it.
func workspaceMkdirWithFile(t *testing.T, base, dir, file string) {
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

// writeWorkspaceIgnoreFile writes a .forge-workspace-ignore file in dir with the given lines.
func writeWorkspaceIgnoreFile(t *testing.T, dir string, lines string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ignoreutil.FileName), []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestWorkspaceList_IgnoresPatterns verifies that List() skips workspaces matching ignore patterns.
func TestWorkspaceList_IgnoresPatterns(t *testing.T) {
	ws := NewWorkspaceDiscovery()
	basedir := t.TempDir()

	// Create 3 workspace dirs, each with go.work.
	for _, name := range []string{"ws-a", "ws-b", "ws-c"} {
		workspaceMkdirWithFile(t, basedir, name, "go.work")
	}

	// Ignore ws-b.
	writeWorkspaceIgnoreFile(t, basedir, "ws-b\n")

	workspaces, err := ws.List(basedir)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}

	names := make(map[string]bool)
	for _, w := range workspaces {
		names[w.Name] = true
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

// TestWorkspaceList_NoIgnoreFile verifies that List() returns all workspaces when no ignore file exists.
func TestWorkspaceList_NoIgnoreFile(t *testing.T) {
	ws := NewWorkspaceDiscovery()
	basedir := t.TempDir()

	for _, name := range []string{"ws-a", "ws-b"} {
		workspaceMkdirWithFile(t, basedir, name, "go.work")
	}

	workspaces, err := ws.List(basedir)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(workspaces))
	}

	names := make(map[string]bool)
	for _, w := range workspaces {
		names[w.Name] = true
	}
	if !names["ws-a"] || !names["ws-b"] {
		t.Errorf("expected both ws-a and ws-b, got %v", names)
	}
}

// TestWorkspaceGet_IgnoresRepos verifies that Get() skips repos matching ignore patterns.
func TestWorkspaceGet_IgnoresRepos(t *testing.T) {
	ws := NewWorkspaceDiscovery()
	basedir := t.TempDir()
	wsName := "my-ws"
	wsPath := filepath.Join(basedir, wsName)

	// Create workspace dir with go.work.
	workspaceMkdirWithFile(t, basedir, wsName, "go.work")

	// Create 3 repo dirs, each with .git/.
	for _, repo := range []string{"repo-a", "repo-b", "repo-c"} {
		repoPath := filepath.Join(wsPath, repo)
		if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Ignore repo-b.
	writeWorkspaceIgnoreFile(t, wsPath, "repo-b\n")

	data, err := ws.Get(basedir, wsName)
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

// TestWorkspaceList_ScanReposRespectsIgnore exercises scanRepos() through List() and verifies
// that repos matching ignore patterns inside a workspace are excluded.
func TestWorkspaceList_ScanReposRespectsIgnore(t *testing.T) {
	wsd := NewWorkspaceDiscovery()
	basedir := t.TempDir()
	wsName := "ws-a"
	wsPath := filepath.Join(basedir, wsName)

	// Create workspace dir with go.work.
	workspaceMkdirWithFile(t, basedir, wsName, "go.work")

	// Create 3 repo dirs inside ws-a, each with .git/.
	for _, repo := range []string{"repo-x", "repo-y", "repo-z"} {
		repoPath := filepath.Join(wsPath, repo)
		if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Ignore repo-y inside ws-a.
	writeWorkspaceIgnoreFile(t, wsPath, "repo-y\n")

	workspaces, err := wsd.List(basedir)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}

	w := workspaces[0]
	if w.Name != wsName {
		t.Fatalf("expected workspace %q, got %q", wsName, w.Name)
	}

	if len(w.Repos) != 2 {
		t.Fatalf("expected 2 repos in %s, got %d", wsName, len(w.Repos))
	}

	repoNames := make(map[string]bool)
	for _, r := range w.Repos {
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
