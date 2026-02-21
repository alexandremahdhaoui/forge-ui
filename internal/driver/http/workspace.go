package httpdriver

import (
	"net/http"
	"path/filepath"
	"sort"

	forgepkg "github.com/alexandremahdhaoui/forge-ui/internal/forge"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/workspace"
)

// HandleWorkspace handles GET /portfolios/{p}/workspaces/{w}.
func (h *Handler) HandleWorkspace(w http.ResponseWriter, r *http.Request) {
	pName := r.PathValue("p")
	wsName := r.PathValue("w")
	if pName == "" || wsName == "" {
		http.Error(w, "portfolio and workspace name required", http.StatusBadRequest)
		return
	}

	wsBaseDir := h.BaseDir
	if pName != "default" {
		wsBaseDir = filepath.Join(h.BaseDir, pName)
	}

	data, err := workspace.Get(wsBaseDir, wsName)
	if err != nil {
		http.Error(w, "workspace not found: "+err.Error(), http.StatusNotFound)
		return
	}

	sortMode := r.URL.Query().Get("sort")
	if sortMode != "name" {
		sortMode = "time"
	}

	cacheKey := pName + "/" + wsName

	// Enrich each repo with cached git information.
	for i, repo := range data.Repos {
		if cached, ok := h.Cache.GetRepoSummary(cacheKey, repo.Name); ok {
			data.Repos[i].Branch = cached.Branch
			data.Repos[i].IsDirty = cached.IsDirty
			data.Repos[i].StatusFiles = cached.StatusFiles
			data.Repos[i].DiffStat = cached.DiffStat
			data.Repos[i].RecentLogs = cached.RecentLogs
			data.Repos[i].Ahead = cached.Ahead
			data.Repos[i].Behind = cached.Behind
			data.Repos[i].HasUpstream = cached.HasUpstream
			data.Repos[i].LastCommitTime = cached.LastCommitTime
		}
	}

	// Sort by last commit time if requested.
	if sortMode == "time" {
		sort.Slice(data.Repos, func(a, b int) bool {
			return data.Repos[a].LastCommitTime.After(data.Repos[b].LastCommitTime)
		})
	}

	// Rewrite repo links for portfolio-scoped URLs.
	for i := range data.Repos {
		data.Repos[i].RepoLink = "/portfolios/" + pName + "/workspaces/" + wsName + "/repos/" + data.Repos[i].Name
	}

	// Load forge data for repos that have forge.yaml and build heatmap data.
	stageSeen := make(map[string]struct{})
	var allStages []string
	var stats types.WorkspaceStats
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

		data.RepoForge = append(data.RepoForge, types.RepoForgeStats{
			RepoName:     repo.Name,
			RepoLink:     "/portfolios/" + pName + "/workspaces/" + wsName + "/repos/" + repo.Name,
			StageResults: stageResults,
		})
	}

	data.AllStages = allStages
	data.Stats = stats
	data.PortfolioName = pName
	data.HomeURL = h.HomeURL
	data.DarkMode = isDarkMode(r)

	data.SortMode = sortMode
	h.render(w, "workspace", data)
}
