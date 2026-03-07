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

// portfolioCreateWorkspace creates a directory with a go.work file and optionally a git
// repo subdirectory.
func portfolioCreateWorkspace(t *testing.T, dir, wsName string, repoNames ...string) {
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

// writePortfolioIgnoreFile writes a .forge-workspace-ignore file into dir with the given
// patterns, one per line.
func writePortfolioIgnoreFile(t *testing.T, dir string, patterns ...string) {
	t.Helper()
	var content string
	for _, p := range patterns {
		content += p + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, ignoreutil.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- List tests ---

func TestPortfolioList_EmptyDir(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	portfolios, err := pd.List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 0 {
		t.Errorf("got %d portfolios, want 0", len(portfolios))
	}
}

func TestPortfolioList_OnlyLooseWorkspaces(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	portfolioCreateWorkspace(t, baseDir, "alpha", "repo-a")
	portfolioCreateWorkspace(t, baseDir, "beta", "repo-b")
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	portfolios, err := pd.List(baseDir)
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

func TestPortfolioList_OnlyNamedPortfolios(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	personalDir := filepath.Join(baseDir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, personalDir, "ws-a", "repo-1")

	workDir := filepath.Join(baseDir, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, workDir, "ws-b", "repo-2")

	portfolios, err := pd.List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 2 {
		t.Fatalf("got %d portfolios, want 2", len(portfolios))
	}

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

	if len(portfolios[0].Workspaces) != 1 {
		t.Errorf("personal portfolio: got %d workspaces, want 1", len(portfolios[0].Workspaces))
	}
	if len(portfolios[1].Workspaces) != 1 {
		t.Errorf("work portfolio: got %d workspaces, want 1", len(portfolios[1].Workspaces))
	}
}

func TestPortfolioList_MixedMode(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	portfolioCreateWorkspace(t, baseDir, "loose-ws", "repo-loose")

	projectsDir := filepath.Join(baseDir, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, projectsDir, "ws-inner", "repo-inner")

	portfolios, err := pd.List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 2 {
		t.Fatalf("got %d portfolios, want 2", len(portfolios))
	}

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

func TestPortfolioList_SkipsHiddenDirs(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	hiddenDir := filepath.Join(baseDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "go.work"), []byte("go 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	hiddenPortfolio := filepath.Join(baseDir, ".config")
	if err := os.MkdirAll(hiddenPortfolio, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, hiddenPortfolio, "ws-inside", "repo-x")

	portfolios, err := pd.List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 0 {
		t.Errorf("got %d portfolios, want 0 (hidden dirs should be skipped)", len(portfolios))
	}
}

func TestPortfolioList_SkipsNonWorkspaceDirs(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	emptyDir := filepath.Join(baseDir, "random-dir")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(baseDir, "somefile.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	portfolios, err := pd.List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 0 {
		t.Errorf("got %d portfolios, want 0", len(portfolios))
	}
}

func TestPortfolioList_SortOrder(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	portfolioCreateWorkspace(t, baseDir, "loose-ws", "repo-loose")

	for _, name := range []string{"zeta", "alpha", "mango"} {
		dir := filepath.Join(baseDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		portfolioCreateWorkspace(t, dir, "ws-"+name, "repo-"+name)
	}

	portfolios, err := pd.List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 4 {
		t.Fatalf("got %d portfolios, want 4", len(portfolios))
	}

	expected := []string{"alpha", "mango", "zeta", "default"}
	for i, want := range expected {
		if portfolios[i].Name != want {
			t.Errorf("portfolios[%d].Name = %q, want %q", i, portfolios[i].Name, want)
		}
	}

	for i, p := range portfolios {
		wantDefault := i == len(portfolios)-1
		if p.IsDefault != wantDefault {
			t.Errorf("portfolios[%d] (%q): IsDefault = %v, want %v", i, p.Name, p.IsDefault, wantDefault)
		}
	}
}

// --- Get tests ---

func TestPortfolioGet_NamedPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	forgeDir := filepath.Join(baseDir, "forge")
	if err := os.MkdirAll(forgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, forgeDir, "ws-core", "repo-core")
	portfolioCreateWorkspace(t, forgeDir, "ws-ui", "repo-ui")

	data, err := pd.Get(baseDir, "forge")
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

	if data.Workspaces[0].Name != "ws-core" {
		t.Errorf("Workspaces[0].Name = %q, want %q", data.Workspaces[0].Name, "ws-core")
	}
	if data.Workspaces[1].Name != "ws-ui" {
		t.Errorf("Workspaces[1].Name = %q, want %q", data.Workspaces[1].Name, "ws-ui")
	}
}

func TestPortfolioGet_DefaultPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	portfolioCreateWorkspace(t, baseDir, "ws-one", "repo-1")
	portfolioCreateWorkspace(t, baseDir, "ws-two", "repo-2")

	data, err := pd.Get(baseDir, "default")
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

func TestPortfolioGet_NonexistentPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	_, err := pd.Get(baseDir, "nope")
	if err == nil {
		t.Fatal("expected error for nonexistent portfolio, got nil")
	}
}

// --- Ignore integration tests ---

func TestPortfolioList_IgnoresLooseWorkspace(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())
	portfolioCreateWorkspace(t, baseDir, "alpha", "repo-a")
	portfolioCreateWorkspace(t, baseDir, "beta", "repo-b")
	portfolioCreateWorkspace(t, baseDir, "gamma", "repo-c")

	writePortfolioIgnoreFile(t, baseDir, "beta")

	portfolios, err := pd.List(baseDir)
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

func TestPortfolioList_IgnoresNamedPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	personalDir := filepath.Join(baseDir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, personalDir, "ws-a", "repo-1")

	workDir := filepath.Join(baseDir, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, workDir, "ws-b", "repo-2")

	writePortfolioIgnoreFile(t, baseDir, "work")

	portfolios, err := pd.List(baseDir)
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

func TestPortfolioList_IgnoresWorkspaceInNamedPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	personalDir := filepath.Join(baseDir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, personalDir, "ws-a", "repo-1")
	portfolioCreateWorkspace(t, personalDir, "ws-b", "repo-2")

	writePortfolioIgnoreFile(t, personalDir, "ws-b")

	portfolios, err := pd.List(baseDir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(portfolios) != 1 {
		t.Fatalf("got %d portfolios, want 1", len(portfolios))
	}
	if portfolios[0].Name != "personal" {
		t.Errorf("portfolios[0].Name = %q, want %q", portfolios[0].Name, "personal")
	}

	if len(portfolios[0].Workspaces) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(portfolios[0].Workspaces))
	}
	if portfolios[0].Workspaces[0].Name != "ws-a" {
		t.Errorf("Workspaces[0].Name = %q, want %q", portfolios[0].Workspaces[0].Name, "ws-a")
	}
}

func TestPortfolioList_IgnoreWithGlobPattern(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())
	portfolioCreateWorkspace(t, baseDir, "temp-foo", "repo-1")
	portfolioCreateWorkspace(t, baseDir, "temp-bar", "repo-2")
	portfolioCreateWorkspace(t, baseDir, "keeper", "repo-3")

	writePortfolioIgnoreFile(t, baseDir, "temp-*")

	portfolios, err := pd.List(baseDir)
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

func TestPortfolioList_NonMatchingPatternPreservesEntries(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())
	portfolioCreateWorkspace(t, baseDir, "alpha", "repo-a")

	writePortfolioIgnoreFile(t, baseDir, "zzz-nothing")

	portfolios, err := pd.List(baseDir)
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

func TestPortfolioList_IgnoreCascades(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())

	personalDir := filepath.Join(baseDir, "personal")
	if err := os.MkdirAll(personalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	portfolioCreateWorkspace(t, personalDir, "ws-a", "repo-1", "repo-2")
	portfolioCreateWorkspace(t, personalDir, "ws-b", "repo-3", "repo-4")

	writePortfolioIgnoreFile(t, personalDir, "ws-b")

	wsADir := filepath.Join(personalDir, "ws-a")
	writePortfolioIgnoreFile(t, wsADir, "repo-2")

	portfolios, err := pd.List(baseDir)
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

	repos := p.Workspaces[0].Repos
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if repos[0].Name != "repo-1" {
		t.Errorf("repos[0].Name = %q, want %q", repos[0].Name, "repo-1")
	}
}

func TestPortfolioGet_IgnoreCascadesInDefaultPortfolio(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	pd := NewPortfolioDiscovery(NewWorkspaceDiscovery())
	portfolioCreateWorkspace(t, baseDir, "ws-keep", "repo-a", "repo-b")
	portfolioCreateWorkspace(t, baseDir, "ws-drop", "repo-c")

	writePortfolioIgnoreFile(t, baseDir, "ws-drop")

	wsKeepDir := filepath.Join(baseDir, "ws-keep")
	writePortfolioIgnoreFile(t, wsKeepDir, "repo-b")

	data, err := pd.Get(baseDir, "default")
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

	repos := data.Workspaces[0].Repos
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if repos[0].Name != "repo-a" {
		t.Errorf("repos[0].Name = %q, want %q", repos[0].Name, "repo-a")
	}
}
