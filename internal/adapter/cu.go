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

//go:build !js || !wasm

package adapter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"sigs.k8s.io/yaml"
)

var _ CUAdapter = (*fsCUAdapter)(nil)

type fsCUAdapter struct{}

// NewCUAdapter returns a CUAdapter that reads compo.yaml and git branches from the filesystem.
func NewCUAdapter() CUAdapter {
	return &fsCUAdapter{}
}

// LoadCompo reads composition state from compo.yaml and git branch information.
func (a *fsCUAdapter) LoadCompo(cuRepoPath string) (types.CompoState, error) {
	data, err := os.ReadFile(filepath.Join(cuRepoPath, "compo.yaml"))
	if err != nil {
		return types.CompoState{}, fmt.Errorf("read compo.yaml: %w", err)
	}

	type compoRepoYAML struct {
		Name         string   `json:"name"`
		URL          string   `json:"url"`
		ManagedFiles []string `json:"managedFiles"`
	}
	type compoYAML struct {
		Name  string          `json:"name"`
		Repos []compoRepoYAML `json:"repos"`
	}

	var cfg compoYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return types.CompoState{}, fmt.Errorf("unmarshal compo.yaml: %w", err)
	}

	repos := make([]types.CompoRepo, len(cfg.Repos))
	for i, r := range cfg.Repos {
		repos[i] = types.CompoRepo{
			Name:         r.Name,
			URL:          r.URL,
			ManagedFiles: r.ManagedFiles,
		}
	}

	branches, err := listBranches(cuRepoPath)
	if err != nil {
		branches = nil
	}

	currentBranch, err := currentBranchName(cuRepoPath)
	if err != nil {
		currentBranch = ""
	}

	return types.CompoState{
		Name:          cfg.Name,
		Repos:         repos,
		CurrentBranch: currentBranch,
		Branches:      branches,
	}, nil
}

// listBranches returns all local branch names from the git repository at dir.
func listBranches(dir string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "branch", "--list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch --list: %w", err)
	}

	var branches []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// currentBranchName returns the current branch name of the git repository at dir.
func currentBranchName(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git branch --show-current: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
