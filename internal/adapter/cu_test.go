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
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCompoYAML = `name: my-composition
repos:
  - name: forge-workspace
    url: https://github.com/example/forge-workspace
    managedFiles:
      - go.mod
      - go.sum
  - name: forge-ui
    url: https://github.com/example/forge-ui
    managedFiles:
      - go.mod
`

// initGitRepo initializes a git repo with one commit and an extra branch.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmds := [][]string{
		{"git", "-C", dir, "init"},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "checkout", "-b", "main"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, out)
	}

	// Create a file and commit so branches can be created.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dummy.txt"), []byte("test"), 0o644))

	cmds = [][]string{
		{"git", "-C", dir, "add", "."},
		{"git", "-C", dir, "commit", "-m", "init"},
		{"git", "-C", dir, "branch", "feature-1"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, out)
	}
}

func TestLoadCompo_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "compo.yaml"), []byte(testCompoYAML), 0o644))
	initGitRepo(t, dir)

	adapter := NewCUAdapter()
	state, err := adapter.LoadCompo(dir)
	require.NoError(t, err)

	assert.Equal(t, "my-composition", state.Name)
	assert.Equal(t, "main", state.CurrentBranch)
	assert.Len(t, state.Repos, 2)

	assert.Equal(t, "forge-workspace", state.Repos[0].Name)
	assert.Equal(t, "https://github.com/example/forge-workspace", state.Repos[0].URL)
	assert.Equal(t, []string{"go.mod", "go.sum"}, state.Repos[0].ManagedFiles)

	assert.Equal(t, "forge-ui", state.Repos[1].Name)
	assert.Equal(t, "https://github.com/example/forge-ui", state.Repos[1].URL)
	assert.Equal(t, []string{"go.mod"}, state.Repos[1].ManagedFiles)

	assert.Contains(t, state.Branches, "main")
	assert.Contains(t, state.Branches, "feature-1")
	assert.Len(t, state.Branches, 2)
}

func TestLoadCompo_MissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	adapter := NewCUAdapter()
	_, err := adapter.LoadCompo(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compo.yaml")
}

func TestLoadCompo_InvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "compo.yaml"), []byte("{{invalid yaml:::"), 0o644))

	adapter := NewCUAdapter()
	_, err := adapter.LoadCompo(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestLoadCompo_EmptyRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "compo.yaml"), []byte(testCompoYAML), 0o644))

	// git init but no commits and no branches.
	cmd := exec.Command("git", "-C", dir, "init")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git init failed: %s", out)

	adapter := NewCUAdapter()
	state, err := adapter.LoadCompo(dir)
	require.NoError(t, err)

	assert.Equal(t, "my-composition", state.Name)
	assert.Len(t, state.Repos, 2)
	// Empty repo with no commits: git branch --list returns nothing,
	// but git branch --show-current may return the default branch name
	// (e.g. "main") depending on git version and config.
	assert.Empty(t, state.Branches)
}
