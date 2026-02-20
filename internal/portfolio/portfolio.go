package portfolio

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/alexandremahdhaoui/forge-ui/internal/ignore"
	"github.com/alexandremahdhaoui/forge-ui/internal/model"
	"github.com/alexandremahdhaoui/forge-ui/internal/workspace"
)

// List scans baseDir and returns portfolio summaries. Directories containing
// go.work are classified as loose workspaces (grouped into a "default"
// portfolio). Directories whose subdirectories contain go.work are classified
// as named portfolios. Named portfolios are sorted alphabetically; the
// "default" portfolio is always last.
func List(baseDir string) ([]model.PortfolioSummary, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	patterns := ignore.Load(baseDir)

	var (
		named    []model.PortfolioSummary
		hasLoose bool
	)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name()[0] == '.' {
			continue
		}
		if ignore.IsIgnored(entry.Name(), patterns) {
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
			workspaces, err := workspace.List(childPath)
			if err != nil {
				return nil, fmt.Errorf("listing workspaces in portfolio %q: %w", entry.Name(), err)
			}
			named = append(named, model.PortfolioSummary{
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
		workspaces, err := workspace.List(baseDir)
		if err != nil {
			return nil, fmt.Errorf("listing loose workspaces: %w", err)
		}
		result = append(result, model.PortfolioSummary{
			Name:       "default",
			Path:       baseDir,
			IsDefault:  true,
			Workspaces: workspaces,
		})
	}

	return result, nil
}

// Get returns the portfolio page data for a named portfolio or the "default"
// portfolio. For "default", it lists workspaces directly under baseDir. For
// named portfolios, it lists workspaces under baseDir/name.
func Get(baseDir, name string) (model.PortfolioPageData, error) {
	if name == "default" {
		workspaces, err := workspace.List(baseDir)
		if err != nil {
			return model.PortfolioPageData{}, fmt.Errorf("listing default portfolio: %w", err)
		}
		return model.PortfolioPageData{
			Name:       "default",
			Path:       baseDir,
			IsDefault:  true,
			Workspaces: workspaces,
		}, nil
	}

	portfolioDir := filepath.Join(baseDir, name)
	if _, err := os.Stat(portfolioDir); err != nil {
		return model.PortfolioPageData{}, fmt.Errorf("portfolio %q not found: %w", name, err)
	}

	workspaces, err := workspace.List(portfolioDir)
	if err != nil {
		return model.PortfolioPageData{}, fmt.Errorf("listing portfolio %q: %w", name, err)
	}

	return model.PortfolioPageData{
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
	patterns := ignore.Load(dir)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name()[0] == '.' {
			continue
		}
		if ignore.IsIgnored(entry.Name(), patterns) {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, entry.Name(), "go.work")); err == nil {
			return true
		}
	}
	return false
}
