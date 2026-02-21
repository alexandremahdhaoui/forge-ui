package httpdriver

import (
	"net/http"
	"path/filepath"

	forgepkg "github.com/alexandremahdhaoui/forge-ui/internal/forge"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// HandleForge handles GET /portfolios/{p}/workspaces/{w}/repos/{r}.
func (h *Handler) HandleForge(w http.ResponseWriter, r *http.Request) {
	pName := r.PathValue("p")
	wsName := r.PathValue("w")
	repoName := r.PathValue("r")
	if pName == "" || wsName == "" || repoName == "" {
		http.Error(w, "portfolio, workspace and repo name required", http.StatusBadRequest)
		return
	}

	var repoPath string
	if pName == "default" {
		repoPath = filepath.Join(h.BaseDir, wsName, repoName)
	} else {
		repoPath = filepath.Join(h.BaseDir, pName, wsName, repoName)
	}

	data, err := forgepkg.Load(repoPath)
	if err != nil {
		http.Error(w, "failed to load forge data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data.WorkspaceName = wsName
	data.RepoName = repoName
	data.PortfolioName = pName
	data.HomeURL = h.HomeURL

	// Compute test statistics from reports.
	var stats types.ForgeStats
	var coverageSum float64
	var coverageCount int
	stageSet := make(map[string]struct{})

	// Build stage status map for heatmap (newest status per stage).
	stageStatusMap := make(map[string]string)
	for _, rpt := range data.TestReports {
		stats.TotalTests += rpt.Stats.Total
		stats.Passed += rpt.Stats.Passed
		stats.Failed += rpt.Stats.Failed
		stats.Skipped += rpt.Stats.Skipped
		stageSet[rpt.Stage] = struct{}{}
		if rpt.Coverage.Enabled {
			coverageSum += rpt.Coverage.Percentage
			coverageCount++
		}
		// Record newest status per stage (reports are sorted newest-first).
		if _, seen := stageStatusMap[rpt.Stage]; !seen {
			stageStatusMap[rpt.Stage] = rpt.Status
		}
	}
	if coverageCount > 0 {
		stats.AvgCoverage = coverageSum / float64(coverageCount)
		stats.HasCoverage = true
	}
	stats.StageCount = len(stageSet)
	data.Stats = stats
	data.StageStatusMap = stageStatusMap
	data.DarkMode = isDarkMode(r)

	h.render(w, "forge", data)
}
