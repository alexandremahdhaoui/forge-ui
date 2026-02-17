package handler

import (
	"net/http"

	forgepkg "github.com/alexandremahdhaoui/forge-ui/internal/forge"
	gitpkg "github.com/alexandremahdhaoui/forge-ui/internal/git"
	"github.com/alexandremahdhaoui/forge-ui/internal/model"
	"github.com/alexandremahdhaoui/forge-ui/internal/workspace"
)

// HandleWorkspace handles GET /workspaces/{name}.
func (h *Handler) HandleWorkspace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "workspace name required", http.StatusBadRequest)
		return
	}

	data, err := workspace.Get(h.BaseDir, name)
	if err != nil {
		http.Error(w, "workspace not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Enrich each repo with git information.
	for i, repo := range data.Repos {
		gitInfo, err := gitpkg.RepoInfo(repo.Path)
		if err != nil {
			// On git error, leave fields at zero values
			continue
		}
		data.Repos[i].Branch = gitInfo.Branch
		data.Repos[i].IsDirty = gitInfo.IsDirty
		data.Repos[i].StatusFiles = gitInfo.StatusFiles
		data.Repos[i].DiffStat = gitInfo.DiffStat
		data.Repos[i].RecentLogs = gitInfo.RecentLogs
		data.Repos[i].Ahead = gitInfo.Ahead
		data.Repos[i].Behind = gitInfo.Behind
		data.Repos[i].HasUpstream = gitInfo.HasUpstream
	}

	// Load forge data for repos that have forge.yaml and build heatmap data.
	stageSeen := make(map[string]struct{})
	var allStages []string
	var stats model.WorkspaceStats
	stats.TotalRepos = len(data.Repos)

	for _, repo := range data.Repos {
		if !repo.HasForge {
			continue
		}
		stats.ForgeRepos++

		forgeData, err := forgepkg.Load(repo.Path)
		if err != nil {
			continue
		}

		// Build per-stage results map from test reports.
		// Use the newest report per stage (first in slice, since sorted newest-first).
		stageResults := make(map[string]string)
		for _, rpt := range forgeData.TestReports {
			if _, seen := stageResults[rpt.Stage]; !seen {
				stageResults[rpt.Stage] = rpt.Status
			}
			stats.TotalTests += rpt.Stats.Total
			stats.Passed += rpt.Stats.Passed
			stats.Failed += rpt.Stats.Failed
			stats.Skipped += rpt.Stats.Skipped
		}

		// Collect stage names in spec order (preserves forge.yaml ordering).
		for _, ts := range forgeData.Spec.Test {
			if _, seen := stageSeen[ts.Name]; !seen {
				stageSeen[ts.Name] = struct{}{}
				allStages = append(allStages, ts.Name)
			}
		}
		// Also collect stage names from reports (in case report exists for a stage not in spec).
		for _, rpt := range forgeData.TestReports {
			if _, seen := stageSeen[rpt.Stage]; !seen {
				stageSeen[rpt.Stage] = struct{}{}
				allStages = append(allStages, rpt.Stage)
			}
		}

		data.RepoForge = append(data.RepoForge, model.RepoForgeStats{
			RepoName:     repo.Name,
			RepoLink:    repo.RepoLink,
			StageResults: stageResults,
		})
	}

	data.AllStages = allStages
	data.Stats = stats

	h.render(w, "workspace", data)
}
