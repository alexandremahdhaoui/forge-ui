package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/alexandremahdhaoui/forge-ui/internal/ignore"
	"github.com/alexandremahdhaoui/forge-ui/internal/model"
)

// List scans basedir for directories containing go.work and returns a summary
// for each workspace found. Results are sorted alphabetically by name.
func List(basedir string) ([]model.WorkspaceSummary, error) {
	entries, err := os.ReadDir(basedir)
	if err != nil {
		return nil, err
	}
	patterns := ignore.Load(basedir)

	var workspaces []model.WorkspaceSummary

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

		wsPath := filepath.Join(basedir, entry.Name())

		// Check if go.work exists in this directory.
		if _, err := os.Stat(filepath.Join(wsPath, "go.work")); err != nil {
			continue
		}

		repos := scanRepos(wsPath, entry.Name())

		workspaces = append(workspaces, model.WorkspaceSummary{
			Name:      entry.Name(),
			Path:      wsPath,
			RepoCount: len(repos),
			Repos:     repos,
		})
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].Name < workspaces[j].Name
	})

	return workspaces, nil
}

// Get returns detailed workspace data including all git repos found in the
// workspace directory. The workspace must contain a go.work file.
func Get(basedir, name string) (model.WorkspacePageData, error) {
	wsPath := filepath.Join(basedir, name)

	if _, err := os.Stat(filepath.Join(wsPath, "go.work")); err != nil {
		return model.WorkspacePageData{}, fmt.Errorf("workspace %q not found: no go.work in %s", name, wsPath)
	}

	entries, err := os.ReadDir(wsPath)
	if err != nil {
		return model.WorkspacePageData{}, fmt.Errorf("reading workspace %q: %w", name, err)
	}
	patterns := ignore.Load(wsPath)

	var repos []model.RepoSummary

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

		dirPath := filepath.Join(wsPath, entry.Name())

		// Check if .git/ exists in this directory.
		if _, err := os.Stat(filepath.Join(dirPath, ".git")); err != nil {
			continue
		}

		hasForge := false
		if _, err := os.Stat(filepath.Join(dirPath, "forge.yaml")); err == nil {
			hasForge = true
		}

		repos = append(repos, model.RepoSummary{
			Name:      entry.Name(),
			Path:      dirPath,
			HasForge:  hasForge,
			RepoLink: fmt.Sprintf("/workspaces/%s/repos/%s", name, entry.Name()),
		})
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return model.WorkspacePageData{
		Name:  name,
		Path:  wsPath,
		Repos: repos,
	}, nil
}

// scanRepos finds subdirectories in wsPath that contain a .git/ directory and
// returns a lightweight RepoOverview for each. Git fields (Branch, IsDirty,
// Ahead, Behind) are left at zero values; the handler enriches them.
func scanRepos(wsPath, wsName string) []model.RepoOverview {
	entries, err := os.ReadDir(wsPath)
	if err != nil {
		return nil
	}
	patterns := ignore.Load(wsPath)

	var repos []model.RepoOverview
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		if ignore.IsIgnored(entry.Name(), patterns) {
			continue
		}
		dirPath := filepath.Join(wsPath, entry.Name())
		if _, err := os.Stat(filepath.Join(dirPath, ".git")); err != nil {
			continue
		}
		hasForge := false
		if _, err := os.Stat(filepath.Join(dirPath, "forge.yaml")); err == nil {
			hasForge = true
		}
		repos = append(repos, model.RepoOverview{
			Name:          entry.Name(),
			WorkspaceName: wsName,
			Path:          dirPath,
			HasForge:      hasForge,
			RepoLink:     fmt.Sprintf("/workspaces/%s/repos/%s", wsName, entry.Name()),
		})
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})
	return repos
}
