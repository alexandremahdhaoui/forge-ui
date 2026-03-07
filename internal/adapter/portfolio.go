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
	"path/filepath"
	"sort"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/ignoreutil"
)

// PortfolioDiscovery discovers portfolios on the filesystem.
type PortfolioDiscovery interface {
	List(baseDir string) ([]types.PortfolioSummary, error)
	Get(baseDir, name string) (types.PortfolioPageData, error)
}

type portfolioDiscoveryImpl struct {
	ws WorkspaceDiscovery
}

// NewPortfolioDiscovery returns a PortfolioDiscovery backed by real filesystem operations.
// It uses the given WorkspaceDiscovery to list workspaces within each portfolio.
func NewPortfolioDiscovery(ws WorkspaceDiscovery) PortfolioDiscovery {
	return &portfolioDiscoveryImpl{ws: ws}
}

func (pd *portfolioDiscoveryImpl) List(baseDir string) ([]types.PortfolioSummary, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	patterns := ignoreutil.Load(baseDir)

	var (
		named    []types.PortfolioSummary
		hasLoose bool
	)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name()[0] == '.' {
			continue
		}
		if ignoreutil.IsIgnored(entry.Name(), patterns) {
			continue
		}

		childPath := filepath.Join(baseDir, entry.Name())

		// Check if the child itself is a workspace (has go.work).
		if _, err := os.Stat(filepath.Join(childPath, "go.work")); err == nil {
			hasLoose = true
			continue
		}

		// Check if the child is a named portfolio (contains subdirs with go.work).
		if isNamedPortfolio(childPath) {
			workspaces, err := pd.ws.List(childPath)
			if err != nil {
				return nil, fmt.Errorf("listing workspaces in portfolio %q: %w", entry.Name(), err)
			}
			named = append(named, types.PortfolioSummary{
				Name:       entry.Name(),
				Path:       childPath,
				IsDefault:  false,
				Workspaces: workspaces,
			})
		}
	}

	sort.Slice(named, func(i, j int) bool {
		return named[i].Name < named[j].Name
	})

	result := named

	// Build the "default" portfolio from loose workspaces.
	if hasLoose {
		workspaces, err := pd.ws.List(baseDir)
		if err != nil {
			return nil, fmt.Errorf("listing loose workspaces: %w", err)
		}
		result = append(result, types.PortfolioSummary{
			Name:       "default",
			Path:       baseDir,
			IsDefault:  true,
			Workspaces: workspaces,
		})
	}

	return result, nil
}

func (pd *portfolioDiscoveryImpl) Get(baseDir, name string) (types.PortfolioPageData, error) {
	if name == "default" {
		workspaces, err := pd.ws.List(baseDir)
		if err != nil {
			return types.PortfolioPageData{}, fmt.Errorf("listing default portfolio: %w", err)
		}
		return types.PortfolioPageData{
			Name:       "default",
			Path:       baseDir,
			IsDefault:  true,
			Workspaces: workspaces,
		}, nil
	}

	portfolioDir := filepath.Join(baseDir, name)
	if _, err := os.Stat(portfolioDir); err != nil {
		return types.PortfolioPageData{}, fmt.Errorf("portfolio %q not found: %w", name, err)
	}

	workspaces, err := pd.ws.List(portfolioDir)
	if err != nil {
		return types.PortfolioPageData{}, fmt.Errorf("listing portfolio %q: %w", name, err)
	}

	return types.PortfolioPageData{
		Name:       name,
		Path:       portfolioDir,
		IsDefault:  false,
		Workspaces: workspaces,
	}, nil
}

// isNamedPortfolio returns true if dir contains at least one subdirectory with
// a go.work file.
func isNamedPortfolio(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	patterns := ignoreutil.Load(dir)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name()[0] == '.' {
			continue
		}
		if ignoreutil.IsIgnored(entry.Name(), patterns) {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, entry.Name(), "go.work")); err == nil {
			return true
		}
	}
	return false
}
